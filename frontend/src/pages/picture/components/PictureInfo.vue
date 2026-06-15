<template>
  <q-dialog ref="dialogRef" @escape-key="onDialogClose" @hide="onDialogHide" maximized>
    <q-card class="q-dialog-plugin bg-dark">
      <q-bar class="bg-primary text-white q-pa-sm">
        <span class="text-subtitle2">{{ view.item.Name }}</span>
        <q-space />
        <q-btn dense flat icon="close" @click="onDialogClose">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-bar>

      <q-scroll-area class="fit" style="height: calc(100vh - 50px)">
        <div v-if="view.prewiewImages.length === 0" class="column flex-center q-pa-xl">
          <q-icon name="image_not_supported" size="80px" color="grey-6" />
          <span class="text-grey-6 q-mt-md">暂无图片</span>
        </div>

        <div v-else class="image-grid q-pa-md">
          <div v-for="(item, idx) in view.prewiewImages" :key="idx" class="image-item">
            <q-img
              fit="contain"
              :src="GetFileByPathUseEncode(item)"
              @click="gotoSearch(item)"
              class="rounded-borders cursor-pointer"
              style="width: 100%; height: 100%; min-height: 300px"
              loading="lazy"
            >
              <template v-slot:loading>
                <div class="absolute-full flex flex-center bg-dark text-grey-6">
                  <q-spinner-gears size="40px" />
                </div>
              </template>

              <template v-slot:error>
                <div class="absolute-full flex flex-center bg-grey-9 text-grey-5 column q-pa-sm">
                  <q-icon name="broken-image" size="48px" />
                  <span class="text-caption q-mt-xs text-center" style="word-break: break-all">{{ item }}</span>
                </div>
              </template>
            </q-img>
          </div>
        </div>
      </q-scroll-area>
    </q-card>
  </q-dialog>
</template>
<script setup>
import { useDialogPluginComponent } from 'quasar';
import { reactive } from 'vue';
import { GetFileByPathUseEncode } from 'src/components/utils/images';
import { useRouter } from 'vue-router';
import { useSystemProperty } from 'stores/System';

const systemProperty = useSystemProperty();

const view = reactive({
  item: {},
  prewiewImages: [],
});

defineEmits([...useDialogPluginComponent.emits]);

const { dialogRef, onDialogHide } = useDialogPluginComponent();

const onDialogClose = () => {
  dialogRef.value.hide();
  onDialogHide();
};

const open = (data) => {
  view.prewiewImages = [];
  view.item = { ...data };
  view.prewiewImages = data.Images || [];
  dialogRef.value.show();
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

<style lang="scss" scoped>
.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
  align-items: start;
}

.image-item {
  break-inside: avoid;
  border-radius: 8px;
  overflow: hidden;
  background-color: #1d1d1d;
  transition: transform 0.2s ease;

  &:hover {
    transform: scale(1.02);
  }
}

@media (max-width: 768px) {
  .image-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
}
</style>
