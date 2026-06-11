<template>
  <div
    style="
      position: fixed;
      left: 0px;
      width: 100%;
      z-index: 999;
      background-color: rgba(0, 0, 0, 0.9);
    "
    class="row q-gutter-xs"
    :style="{ justifyContent: rightClose ? 'flex-end' : 'flex-start' }"
  >
    <q-btn
      flat
      glossy
      color="red"
      v-if="!rightClose"
      label="x"
      @click="closeThis"
    />
    <q-btn-toggle
      v-model="view.currentDiv"
      color="black"
      outlined
      glossy
      dense
      text-color="white"
      :options="[
        { value: 'movielist', label: '文件列表' },
        { value: 'picturelist', label: '关联图片' },
      ]"
      @update:model-value="changeDiv"
    />
    <q-btn
      flat
      glossy
      color="orange"
      v-if="view.currentDiv == 'picturelist'"
      label="截图"
      @click="curImage"
    />
    <q-btn
      flat
      glossy
      color="red"
      v-if="rightClose"
      label="x"
      @click="closeThis"
    />
    <div
      v-if="view.currentDiv == 'picturelist'"
      style="
        padding: 0;
        position: fixed;
        left: 3rem;
        margin-top: 3rem;
        float: right;
        z-index: 9999;
      "
    >
      <q-btn
        flat
        color="red"
        v-for="item in fowartBtn"
        :key="item"
        :label="item"
        @click="forwardTime(item)"
        @contextmenu="
          (e) => {
            forwardTime(-item);
            e.returnValue = false;
          }
        "
        class="q-pa-sm fts12"
      ></q-btn>
      {{ props.currentTime }}
    </div>
  </div>
  <div
    style="
      margin: 0;
      padding: 0;
      background-color: rgba(0, 0, 0, 0.8);
      width: 100%;
    "
  >
    <div style="margin-top: 20px; position: relative">
      <div style="height: 40px"></div>
      <q-list
        bordered
        separator
        v-if="view.currentDiv == 'movielist'"
        :style="{ height: props.detailHeight * 0.9 + 'px', overflow: 'auto' }"
      >
        <div
          v-for="item in view.resultData?.Data"
          :key="item.Id"
          :id="item.Id"
          :style="{
            backgroundColor:
              item.Id == props.currentId ? 'rgba(0, 0, 0, 0.3)' : '',
          }"
          @click="
            openVideo({ item, queryParam: view.queryParam });
            closeThis();
          "
        >
          <q-expansion-item dense hideExpandIcon>
            <template v-slot:header>
              <q-item-section avatar>
                <q-img
                  fit="scale-dowmn"
                  easier
                  :src="item.pngUrl"
                  style="
                    width: 80px;
                    height: 80px;
                    border-radius: 6px 6px 0 0;
                    background: linear-gradient(45deg, #f5f5f5, #e0e0e0);
                    overflow: hidden;
                  "
                >
                  <template v-slot:loading>
                    <q-spinner-ios color="white" size="2em"
                      >Loading...</q-spinner-ios
                    >
                  </template>
                  <template v-slot:error>
                    <!-- 图片加载失败时显示的占位图 -->
                    <div class="text-subtitle1 text-white">
                      <q-icon name="image_not_supported" size="2em"></q-icon>
                    </div>
                  </template>
                </q-img>
              </q-item-section>

              <q-item-section>
                <p
                  style="
                    display: -webkit-box; /* 将对象作为弹性伸缩盒子模型显示 */
                    -webkit-box-orient: vertical; /* 设置子元素的排列方式为垂直方向 */
                    line-clamp: 3; /* 设置显示的行数 */
                    overflow: hidden; /* 隐藏溢出文本 */
                    text-overflow: ellipsis; /* 显示省略号 */
                  "
                >
                  <a
                    style="color: #9e089e; background-color: rgba(0, 0, 0, 0.1)"
                    class="mr10 cursor-pointer text-title"
                    target="_blank"
                    >{{ item.Actress?.substring(0, 6) }}</a
                  >
                  <a
                    style="
                      color: rgb(239, 30, 30);
                      background-color: rgba(0, 0, 0, 0.1);
                    "
                    class="mr10 cursor-pointer"
                    >{{ formatCode(item.Code) }}</a
                  >
                  <a
                    style="
                      color: rgb(22, 26, 227);
                      background-color: rgba(0, 0, 0, 0.1);
                    "
                    class="mr10 cursor-pointer"
                    >{{ item.SizeStr }}</a
                  >
                  <q-chip
                    square
                    dense
                    text-color="white"
                    :label="`P${item.PageNo}`"
                    class="chip-tag"
                  ></q-chip>
                  <!-- 标签列表 -->
                  <q-chip
                    size="md"
                    square
                    dense
                    text-color="white"
                    v-for="tag in item.Tags"
                    :key="tag"
                    class="chip-tag"
                  >
                    <span>{{ tag?.substring(0, 4) }}</span>
                  </q-chip>
                  <span class="text-subtitle">{{
                    formatTitle(item.Name)
                  }}</span>
                </p>
              </q-item-section>
            </template>
          </q-expansion-item>
        </div>
        <div
          v-intersection="onIntersection"
          style="height: 8vh; color: #9e089e"
          @click="
            () => {
              pullPage(1);
            }
          "
        >
          点击可加载更多数据
        </div>
      </q-list>
      <q-list
        bordered
        separator
        v-if="view.currentDiv == 'picturelist'"
        :style="{ height: props.detailHeight * 0.9 + 'px', overflow: 'auto' ,padding: '10px' }"
      >
        <q-img
          fit="fill"
          v-for="item in view.prewiewImages"
          :key="item.Id"
          :src="getTempImage(item.Id)"
          width="400px"
        >
          <div style="padding: 0; position: relative; float: right">
            <q-btn
              color="rgba(0,0,0,0.5)"
              dense
              ripple
              @click="deleteTemp(item.Path)"
              icon="ti-trash"
            >
              <q-tooltip class="bg-white text-primary">删除</q-tooltip>
            </q-btn>
          </div>
          <template v-slot:error>
            <!-- 图片加载失败时显示的占位图 -->
            <div class="text-subtitle1 text-white">
              <q-icon name="image_not_supported" size="8em"></q-icon>
              <div>图片加载失败</div>
              <q-btn
                color="rgba(0,0,0,0.5)"
                dense
                ripple
                @click="deleteTemp(item.Path)"
                icon="ti-trash"
              >
                <q-tooltip class="bg-white text-primary">删除</q-tooltip>
              </q-btn>
            </div>
          </template>
        </q-img>
      </q-list>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { getTempImage } from 'components/utils/images';
import { formatCode, formatTitle } from 'components/utils';
import {
  SearchAPI,
  QueryDirImageBase64,
  DeleteFileByPathUseEncode,
  CutImage,
} from 'components/api/searchAPI';
import { useThrottleFn } from '@vueuse/core';
import { useSystemProperty } from 'stores/System';

const systemProperty = useSystemProperty();
const props = defineProps({
  currentTime: {
    type: String,
    default: '00:00:00',
  },
  rightClose: {
    type: Boolean,
    default: false,
  },
  currentId: {
    type: String,
    default: '',
  },
  detailHeight: {
    type: Number,
    default: 200,
  },
});

const isLoading = ref(false);
const view = reactive({
  currentDiv: 'movielist',
  resultData: null,
  queryParam: null,
  prewiewImages: [],
});

const changeDiv = (v) => {
  if (v == 'picturelist') {
    loadDirImage();
  } else {
    pullPage();
  }
};

const loadDirImage = () => {
  QueryDirImageBase64(props.currentId, 'desc').then((res) => {
    view.prewiewImages = [...res.data];
  });
};

const deleteTemp = async (path) => {
  await DeleteFileByPathUseEncode(path);
  loadDirImage();
};

const curImage = async () => {
  view.showDrawer = true;
  view.drawerType = 'img';
  await CutImage(props.currentId, 'shot', props.currentTime, false);
  loadDirImage();
};
const pullPage = async (n) => {
  if (!view.queryParam) {
    return;
  }
  if (
    !view.resultData?.TotalPage ||
    (view.queryParam?.Page < view.resultData?.TotalPage && !isLoading.value)
  ) {
    if (!n) {
      n = 0;
    }
    isLoading.value = true;
    view.queryParam.Page = view.queryParam.Page + n;
    const data = await SearchAPI(view.queryParam);
    if (!view.resultData?.Data) {
      view.resultData = data;
    } else {
      view.resultData.Data.push(...data.Data);
    }
    isLoading.value = false;
  }
};

const throttledOnIntersection = useThrottleFn(() => {
  pullPage(1);
}, 1000);

const onIntersection = async (entry) => {
  if (entry.isIntersecting && !isLoading.value) {
    throttledOnIntersection();
  }
};

const emmits = defineEmits(['closebtn', 'openVideo', 'forwardTime']);
const fowartBtn = [-30, -15, 60, 120, 240];
const forwardTime = (time) => {
  emmits('forwardTime', time);
};

const closeThis = () => {
  emmits('closebtn');
};

const openVideo = (ob) => {
  emmits('openVideo', ob);
};

const refreshData = (tab) => {
  if (tab == 2) {
    view.currentDiv = 'picturelist';
    loadDirImage();
  } else {
    view.currentDiv = 'movielist';
    pullPage();
  }
};

onMounted(() => {
  view.queryParam = systemProperty.FileSearchParam;
});

defineExpose({
  refreshData,
});
</script>
<style lang="css">
.mr10 {
  margin-right: 10px;
}

.chip-tag {
  margin-left: 0;
  padding: 0 4px;
  font-weight: 500;
  width: fit-content;
  background-color: rgba(0, 0, 0, 0.2);
}
</style>
