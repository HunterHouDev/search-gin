<template>
  <q-dialog
    ref="dialogRef"
    @escape-key="onDialogClose"
    @before-hide="onDialogClose"
    @hide="onDialogClose"
    style="width: 80vw !important"
    v-model:model-value="showDialog"
  >
    <q-card
      class="q-dialog-plugin q-pa-md"
      :style="{
        height: '100%',
        width: '100%',
        padding: '4px',
        lineHeight: '32px',
        maxWidth: '80vw !important',
      }"
    >
      <div style="margin-top: 0; height: 96%; overflow: auto">
        <div v-for="item in view.prewiewImages" :key="item">
          <q-img
            fit="fill"
            v-if="item.endsWith('.jpg')"
            :src="GetFileByPathUseEncode(item)"
            @click="gotoSearch(item)"
            style="width: 100%; height: auto; max-height: 600px"
          >
            <template v-slot:error>
              <!-- 图片加载失败时显示的占位图 -->
              <div class="text-subtitle1 text-white">
                <q-icon name="image_not_supported" size="8em"></q-icon>
                <span
                  style="
                    z-index: 99;
                    color: red;
                    background-color: rgba(250, 250, 250, 0.7);
                    text-align: center;
                    font-size: 16px;
                    font-weight: 550;
                    height: 100%;
                  "
                  v-if="item.endsWith('.jpg')"
                  >{{ item }}</span
                >
              </div>
            </template>
          </q-img>
        </div>
      </div>
    </q-card>
  </q-dialog>
</template>
<script setup>
import { useDialogPluginComponent } from 'quasar';
import { reactive, ref } from 'vue';
import { GetFileByPathUseEncode } from 'src/components/utils/images';
import { useRouter } from 'vue-router';
import { useSystemProperty } from 'stores/System';

const systemProperty = useSystemProperty();
const showDialog = ref(false);

const view = reactive({
  item: {},
  prewiewImages: [],
});

defineEmits([
  // REQUIRED; 需要明确指出
  // 组件通过 useDialogPluginComponent() 暴露哪些事件
  ...useDialogPluginComponent.emits,
]);

const open = (data) => {
  const item = data;
  view.prewiewImages = [];
  view.item = { ...item };
  dialogRef.value.show();
  view.prewiewImages = item.Images;
};

// onDialogOK, onDialogCancel
const { dialogRef, onDialogHide } = useDialogPluginComponent();

const onDialogClose = () => {
  showDialog.value = false;
  onDialogHide();
};

const reg2 = /\[\S+\]/g;
const { push } = useRouter();

const reg = /\w+[-_]\d+/;
const reg_1 = /\w+\d+/;

const gotoSearch = (item) => {
  const author = item.match(reg2);
  let keyword = '';
  if (author && author[0] && author[0].length > 0) {
    keyword += author[0]
      .replaceAll('[', ' ')
      .replaceAll(']', ' ')
      .replaceAll('  ', ' ')
      .trim();
  }
  let originalCode = item.match(reg);
  console.log('originalCode', originalCode);
  if (!originalCode) {
    originalCode = item.match(reg_1);
  }
  keyword += ' ' + originalCode;
  systemProperty.setPage(1);
  systemProperty.FileSearchParam.Keyword = keyword;
  systemProperty.setMovieType('');
  push('/search?from=index');
};

defineExpose({
  open,
});
</script>
