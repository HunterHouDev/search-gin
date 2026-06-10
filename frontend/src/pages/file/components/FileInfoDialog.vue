<template>
  <q-dialog
    ref="dialogRef"
    @escape-key="onDialogClose"
    @before-hide="onDialogClose"
    @hide="onDialogClose"
    v-model:model-value="isDialogOpen"
    :fullWidth="true"
    square
  >
    <q-layout
      view="lHh Lpr lFf"
      container
      :style="{
        width: isMobile ? '100vw!important' : '1000px!important',
        maxHeight: isMobile ? '80vh!important' : '900px!important',
      }"
    >
      <q-header bordered class="justify-between w100">
        <q-toolbar class="bg-black text-white shadow-2 rounded-borders">
          <q-btn color="green" flat dense @click="prevOne" icon="ti-shift-left">
            <q-tooltip class="bg-white text-primary">上一个</q-tooltip>
          </q-btn>

          <q-toolbar-title shrink style="width: 50%">
            {{ view.item.Name }}
          </q-toolbar-title>
          <q-space />
          <q-tabs v-model="showDetail" shrink mobile-arrows>
            <q-tab
              v-for="item in ClickButtons"
              :key="item.value"
              :name="item.value"
              :label="item.label"
              @click="tabClick(item.value)"
            />
          </q-tabs>
          <q-space />
          <q-btn dense flat icon="close" @click="onDialogClose">
            <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
          </q-btn>
          <q-btn
            color="green"
            flat
            dense
            icon="ti-shift-right"
            @click="nextOne"
          >
            <q-tooltip class="bg-white text-primary">下一个</q-tooltip>
          </q-btn>
        </q-toolbar></q-header
      >

      <q-tab-panels
        v-model="showDetail"
        style="height: auto; padding-top: 3rem"
      >
        <q-tab-panel name="web">
          <iframe
            :frameborder="0"
            :allowfullscreen="true"
            width="100%"
            :style="{ height: isMobile ? 'calc(100vh - 150px)' : '720px' }"
            :src="`${view.settingInfo.BaseUrl}${view.item.Code}`"
          ></iframe>
        </q-tab-panel>

        <q-tab-panel name="detail">
          <q-img
            fit="fit"
            easier
            draggable
            :src="getJpg(view.item.Id)"
            style="max-height: 560px"
          >
          </q-img>
          <div class="row justify-left q-gutter-md" :class="{ column: isMobile }">
            
            <q-field label="Time" stack-label>
              <template v-slot:control>
                <div class="self-center full-width no-outline" tabindex="0">
                  {{ formatTitle(view.item.MTime) }}
                </div>
              </template>
            </q-field>
            <q-field label="Actress" stack-label>
              <template v-slot:control>
                <div class="self-center full-width no-outline" tabindex="0">
                  {{ view.item.Actress }}
                </div>
              </template>
            </q-field>
            
            <q-field label="Code" stack-label>
              <template v-slot:control>
                <div
                  class="self-center full-width no-outline cursor-pointer"
                  style="color: blue"
                  tabindex="0"
                  @click="searchCode(view.item)"
                >
                  {{ view.item.Code }}
                </div>
              </template>
            </q-field>
            
          </div>
          <div class="row q-pt-sm" :class="{ column: isMobile }">
            <span style="color: orange"> Name: </span>
            {{ formatTitle(view.item.Name) }}
          </div>
<!-- 
          <div class="row q-pt-sm">
            <span align="left" class="full-width no-outline">
              <span style="color: orange"> DirPath: </span
              ><a
                class="cursor-pointer"
                style="color: blue"
                @click="OpenFileFolder(view.item.Id)"
              >
                {{ view.item.DirPath }}
              </a>
            </span>
          </div> -->

          <div class="row q-pt-sm" :class="{ column: isMobile }">
            <span align="left" ripple class="full-width outline">
              <a style="color: grey"  class="cursor-pointer" @click="OpenFileFolder(view.item.Id)">  {{ view.item.Path }} </a>
            </span>
          </div>
        </q-tab-panel>
        <q-tab-panel name="image">
          <q-img
            fit="contain"
            v-for="item in view.prewiewImages"
            :key="item.Id"
            :src="getTempImage(item.Id)"
            style="width: 100%; height: auto; max-height: 500px"
          >
            <template v-slot:error>
              <!-- 图片加载失败时显示的占位图 -->
              <div class="text-subtitle1 text-white">
                <q-icon name="image_not_supported" size="8em"></q-icon>
                <div>图片加载失败</div>
                <q-btn
                  color="rgba(0,0,0,0.5)"
                  size="sm"
                  dense
                  ripple
                  @click="deleteTemp(item.Path)"
                  icon="ti-trash"
                >
                  <q-tooltip class="bg-white text-primary">删除</q-tooltip>
                </q-btn>
              </div>
            </template>
            <div style="padding: 0; position: relative; float: right">
              <q-btn
                color="rgba(0,0,0,0.5)"
                size="sm"
                dense
                ripple
                @click="deleteTemp(item.Path)"
                icon="ti-trash"
              >
                <q-tooltip class="bg-white text-primary">删除</q-tooltip>
              </q-btn>
            </div>
          </q-img></q-tab-panel
        >
        <q-tab-panel
          name="movie"
          class="bg-black"
          style="padding: 8px 1px 1px 1px"
        >
          <VideoPlayer ref="videoRef" @next-one="nextOnePlayer" />
        </q-tab-panel>
      </q-tab-panels>
    </q-layout>
  </q-dialog>
</template>
<script setup>
import VideoPlayer from 'src/components/VideoPlayer.vue';
import { useQuasar } from 'quasar';
import { useDialogPluginComponent } from 'quasar';
import { onMounted, reactive, ref, computed } from 'vue';

import { formatTitle } from 'components/utils';
import { GetSettingInfo } from 'components/api/settingAPI';
import {
  QueryDirImageBase64,
  OpenFileFolder,
  DeleteFileByPathUseEncode,
} from 'components/api/searchAPI';
import { getJpg, getTempImage } from 'src/components/utils/images';
const { dialogRef, onDialogHide } = useDialogPluginComponent();

const $q = useQuasar();
const isMobile = computed(() => {
  return $q.platform.is.mobile;
});

const ClickButtons = [
  { label: '播放', value: 'movie' },
  { label: '详情', value: 'detail' },
  { label: '图层', value: 'image' },
  { label: 'JavBus', value: 'web' },
];

const videoRef = ref(null);
const showDetail = ref('detail');
const isDialogOpen = ref(false);

const view = reactive({
  item: {},
  settingInfo: {},
  prewiewImages: [],
  playList: [],
  menuDrawer: false,
  playerStyle: {
    height: isMobile.value ? '450px' : '800px',
  },
});

const showMovie = () => {
  showDetail.value = 'movie';
  setTimeout(() => {
    videoRef.value.openVideo(view.item);
  }, 100);
};
const tabClick = (value) => {
  showDetail.value = value;
  if (value === 'movie') {
    showMovie();
  }
  if (value === 'image') {
    loadImage(view.item);
  }
};

const emmits = defineEmits([
  ...useDialogPluginComponent.emits,
  'hide',
  'nextOne',
  'prevOne',
]);

const nextOnePlayer = (e) => {
  if (e > 0) {
    emmits('nextOne');
  } else {
    emmits('prevOne');
  }
};

const nextOne = () => {
  console.log('fileinfodiaolg nextOne');
  emmits('nextOne');
};
const prevOne = () => {
  emmits('prevOne');
};

const open = (data) => {
  const { item, playing } = data;
  view.prewiewImages = [];
  view.item = { ...item };
  isDialogOpen.value = true;
  if (playing || showDetail.value == 'movie') {
    showMovie();
  }
  if (showDetail.value === 'image') {
    loadImage(view.item);
  }
};

const loadImage = (item) => {
  if (item) {
    QueryDirImageBase64(item.Id, 'asc').then((res) => {
      view.prewiewImages = res.data;
    });
  }
};

const deleteTemp = async (path) => {
  await DeleteFileByPathUseEncode(path);
  loadImage(view.item);
};

const fetchSetting = async () => {
  const res = await GetSettingInfo();
  view.settingInfo = res.data;
};

const searchCode = (item) => {
  let itemCode = item.Code;
  if (itemCode.indexOf('-C') > 0) {
    itemCode = itemCode.substring(0, itemCode.indexOf('-C'));
  }
  if (itemCode.indexOf('-') === 0) {
    itemCode = itemCode.substring(1);
  }
  const url = `${view.settingInfo.BaseUrl}/${itemCode}`;
  console.log(url);
  if ($q.platform.is.electron) {
    window.electron.createWindow({
      router: url,
      width: 1280,
      height: 1000,
      titleBarStyle: '',
    });
  } else {
    window.open(url);
  }
};

const onDialogClose = () => {
  showDetail.value = 'detail';
  isDialogOpen.value = false;
  emmits('hide');
  onDialogHide();
};

// dialogRef      - 用在 QDialog 上的 Vue ref 模板引用
// onDialogHide   - 处理 QDialog 上 @hide 事件的函数
// onDialogOK     - 对话框结果为 ok 时会调用的函数
//                    示例: onDialogOK() - 不带参数
//                    示例: onDialogOK({ /*.../* }) - 带参数
// onDialogCancel - 对话框结果为 cancel 时调用的函数

// 这是示例的内容，不是必需的
// const onOKClick = () => {
// REQUIRED！ 对话框的结果为 ok 时，必须调用 onDialogOK()  (参数是可选的)
// onDialogOK()
// 带参数的版本: onDialogOK({ ... })
// ...会自动关闭对话框
// }

onMounted(() => {
  fetchSetting();
});

defineExpose({
  open,
});
</script>
<style scoped>
.example-item {
  width: 140px;
  height: auto;
  max-height: 320px;
  overflow: hidden;
}

.item-img {
  width: 140px;
  height: auto;
  max-height: 220px;
}
.chip-tag {
  margin-left: 0;
  padding: 0 4px;
  font-weight: 500;
  width: fit-content;
  background-color: rgba(0, 0, 0, 0.2);
}

/* 移动端适配 */
@media (max-width: 600px) {
  .q-tab-panel {
    padding: 8px 4px;
  }
  .q-field--with-bottom {
    padding-bottom: 4px;
  }
}
</style>
