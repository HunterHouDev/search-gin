<template>
  <q-dialog ref="dialogRef" full-width @hide="onDialogHide">
    <div class="dialog">
      <q-toolbar class="bg-primary rounded-borders justify-between w100" wrap>
        <q-btn color="red" flat icon="ti-shift-left" :size="isMobile ? 'sm' : 'md'" @click="prevOne">
          <q-tooltip class="bg-white text-primary">上一个</q-tooltip>
        </q-btn>
        <span style="color: white; font-size: 18px" v-if="!isMobile">
          <span style="color: orchid; cursor: pointer">
            历史图鉴 ：
            <q-popup-proxy>
              <div style="
                  padding: 10px;
                  display: flex;
                  flex-wrap: wrap;
                  flex-direction: row;
                  background-color: rgba(250, 250, 250, 1);
                  border-radius: 40px;
                " class="q-gutter-md" v-if="systemProperty.lastAuthores && systemProperty.lastAuthores.length > 0">
                <q-btn color="orange" v-close-popup class="glossy" v-for="item in systemProperty.lastAuthores"
                  :key="item" :label="item" @click="systemProperty.lastAuthor = item"></q-btn>
              </div>
            </q-popup-proxy>
          </span>
          <a style="color: green; border-bottom: 1px solid green; cursor: pointer;margin-right: 2rem;"
            v-if="systemProperty.lastAuthor" @click="view.item.Author = systemProperty.lastAuthor">
            {{ systemProperty.lastAuthor }}</a>
        </span>
        <q-btn color="orange" align="evenly" style="width: 6rem;" outline label="改名" glossy @click="editItemSubmit()" />
        <q-btn style="margin-left: 10px;width: 6rem;" color="green" size="md" outline align="evenly" label="移动" glossy
          @click="editMoveout" />
        <q-space />
        <q-btn flat style="margin-right: 10px" color="orange" size="lg" icon="close" @click="onDialogCancel">
          <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
        </q-btn>
        <q-btn color="red" flat icon="ti-shift-right" :size="isMobile ? 'sm' : 'md'" @click="nextOne">
          <q-tooltip class="bg-white text-primary">下一个</q-tooltip>
        </q-btn>
      </q-toolbar>

      <q-card class="file-edit-card">
        <q-form class="q-pa-sm" :style="{ width: showBus ? '50%' : '100%' }">
          <div class="row wrap justify-start q-gutter-sm">
            <q-toggle v-model="systemProperty.fileEditAutoCode" color="green" glossy label="番号自动" left-label dense
              class="taggle" />
            <q-toggle v-model="systemProperty.fileEditAutoJpg" color="green" glossy label="JPG自动" left-label dense
              class="taggle" />
            <q-toggle v-model="systemProperty.fileEditAutoRefresh" color="green" glossy label="刷新" left-label dense
              class="taggle" />
            <q-toggle color="red" dense glossy flat label="下个" left-label v-model="systemProperty.fileEditAutoNext"
              class="taggle" />
            <q-btn color="primary" glossy label="Bus" @click="toggleJavBus" />
          </div>
          <div class="q-gutter-sm q-pa-sm">

            <q-input color="red-12" style="border-radius: 15px; background: rgba(0, 0, 0, 0.1)" autogrow standout
              outlined label="名称" v-model="view.item.Title" :dense="false" clearable @focus="titleChange"
              @change="titleChange">
              <template v-slot:append>
                <q-icon name="style" size="lg" color="red" class="cursor-pointer"
                  @click="pasteFromClipboard('Title')" />
              </template>
            </q-input>

            <q-input outlined label="图鉴" autogrow v-model="view.item.Author" clearable :dense="false">
            </q-input>
            <q-input outlined autogrow label="番号" v-model="view.item.Code" :dense="false" @change="makePreview"
              clearable />
            <q-input :class="isMobile ? '' : 'col-6'" label="JPG地址" autogrow outlined clearable
              :model-value="jpgDisplay" @update:model-value="onJpgInput"
              @clear="systemProperty.fileEditAutoJpg = false" :dense="false">
              <template v-slot:append>
                <q-icon name="style" size="md" class="cursor-pointer" @click="pasteFromClipboard('Jpg')" />
              </template>
            </q-input>
            <q-input :class="isMobile ? '' : 'col-6'" label="PNG地址" autogrow outlined clearable
              :model-value="pngDisplay" @update:model-value="onPngInput" :dense="false">
              <template v-slot:append>
                <q-icon name="style" size="md" class="cursor-pointer" @click="pasteFromClipboard('Png')" />
              </template>
            </q-input>
            <p style="color: grey">
              {{ view.item.Path }}
              <span style="color: red">{{ view.item.SizeStr }}</span>
              <span style="color: green; margin-left: 10px" v-for="tag in view.item.Tags" :key="tag">{{ tag }}</span>
            </p>
          </div>
          <div class="q-pa-sm preview-panel">
            <template v-if="jpgSrc || pngSrc">
              <q-img v-if="jpgSrc" fit="fill" height="180px" :src="jpgSrc"></q-img>
              <q-img v-if="pngSrc" fit="fill" height="180px" :src="pngSrc"></q-img>
            </template>
            <div v-else class="preview-placeholder">
              <q-icon name="image" size="48px" color="grey-5" />
              <p class="text-grey-6">暂无预览图</p>
            </div>
          </div>
        </q-form>
        <div v-if="showBus" class="bus-panel">
          <iframe ref="busIframe" :frameborder="0" :allowfullscreen="true" width="100%" height="100%"
            :src="busUrl"></iframe>
        </div>
      </q-card>
    </div>
  </q-dialog>
</template>

<script setup lang="ts">
import { useDialogPluginComponent, useQuasar } from 'quasar';
import { computed, reactive, ref, watch } from 'vue';

import { FileRename } from 'components/api/searchAPI';
import { formatTitle } from 'components/utils';
import { FileModel } from 'src/components/model/File';
import { useSystemProperty } from 'stores/System';
import { useBreakpoint } from 'src/composables/useBreakpoint';
const systemProperty = useSystemProperty();


const $q = useQuasar();
const { isMobile } = useBreakpoint();
const view = reactive({
  item: null,
  nodeHost: '',
  preview: false,
});

const showBus = ref(false);
const busUrl = ref('');

// ── 内联 base64 图片（data:image/xxx;base64,...） ────────────────
// 有些站点只提供内联 base64 图片，复制出来就是一整串 data URI（动辄上百 KB），
// 直接塞进输入框既没法看也没法改，故单独存放，输入框只显示一个摘要标签。
const jpgDataUrl = ref('');
const pngDataUrl = ref('');

const isDataImage = (v: string): boolean =>
  typeof v === 'string' &&
  v.startsWith('data:image/') &&
  v.includes(';base64,');

// 解码后的大致体积，用于摘要显示
const dataImageSize = (v: string): string => {
  const idx = v.indexOf(';base64,');
  const raw = v.slice(idx + 8);
  const padding = raw.endsWith('==') ? 2 : raw.endsWith('=') ? 1 : 0;
  const bytes = Math.max(0, Math.floor((raw.length * 3) / 4) - padding);
  return bytes >= 1024 * 1024
    ? `${(bytes / 1024 / 1024).toFixed(1)} MB`
    : `${Math.max(1, Math.round(bytes / 1024))} KB`;
};

const dataImageLabel = (v: string): string =>
  `【内联 base64 图片 · ${dataImageSize(v)}】`;

const jpgSrc = computed(() => jpgDataUrl.value || view.item?.Jpg || '');
const pngSrc = computed(() => pngDataUrl.value || view.item?.Png || '');
const jpgDisplay = computed(() =>
  jpgDataUrl.value ? dataImageLabel(jpgDataUrl.value) : view.item?.Jpg || ''
);
const pngDisplay = computed(() =>
  pngDataUrl.value ? dataImageLabel(pngDataUrl.value) : view.item?.Png || ''
);

// 输入/粘贴统一入口：识别到 base64 图片就存到独立字段，普通地址仍走 item
// v 来自 q-input 的 update:model-value，可能是 number / null，统一转成字符串处理
const applyImageField = (
  field: 'Jpg' | 'Png',
  v: string | number | null,
): void => {
  const text = v == null ? '' : String(v);
  if (isDataImage(text)) {
    if (field === 'Jpg') {
      jpgDataUrl.value = text;
      view.item.Jpg = '';
    } else {
      pngDataUrl.value = text;
      view.item.Png = '';
    }
    $q.notify({
      type: 'positive',
      message: `已填入 base64 图片（${dataImageSize(text)}）`,
      position: 'bottom',
    });
    return;
  }
  if (field === 'Jpg') {
    jpgDataUrl.value = '';
    view.item.Jpg = text;
  } else {
    pngDataUrl.value = '';
    view.item.Png = text;
  }
};
const onJpgInput = (v: string | number | null): void =>
  applyImageField('Jpg', v);
const onPngInput = (v: string | number | null): void =>
  applyImageField('Png', v);

const toggleJavBus = () => {
  if (!systemProperty.fileEditAutoCode) {
    systemProperty.fileEditAutoCode = true;
  }
  if (!showBus.value) {
    let vcode = view.item?.Code || '';
    vcode = vcode.replace(/[\r\n\t]+/g, '');
    vcode = vcode.replace(/&nbsp;/g, '');
    vcode = vcode.trimEnd();
    if (vcode.indexOf('-C') > 0) {
      vcode = vcode.substring(0, vcode.indexOf('-C'));
    }
    if (vcode.indexOf('-') === 0) {
      vcode = vcode.substring(1);
    }
    if (vcode.indexOf('@') >= 0) {
      vcode = vcode.substring(0, vcode.indexOf('@'));
    }
    view.item.Code = vcode;
    if (vcode && systemProperty.SettingInfo.BaseUrl) {
      busUrl.value = systemProperty.SettingInfo.BaseUrl + vcode;
    }
    showBus.value = true;
  } else {
    showBus.value = false;
  }
};

// 用户从 JavBus iframe 切回外层窗口时，读取剪贴板内容填入对应字段
let lastClipboardText = '';
const onWindowFocus = async () => {
  if (!showBus.value) return;
  try {
    const text = (await navigator.clipboard.readText() || '').trim();
    if (!text || text.length === 0 || text === lastClipboardText) return;
    lastClipboardText = text;
    // 内联 base64 图片由 applyImageField 自行提示，这里只提示 http 地址
    if (isDataImage(text)) {
      applyImageField('Jpg', text);
    } else if (text.startsWith('http')) {
      view.item.Jpg = text;
      $q.notify({ type: 'positive', message: `已填入JPG: ${text.slice(0, 50)}`, position: 'bottom' });
    } else {
      view.item.Title = text;
      titleChange(view.item.Title);
      $q.notify({ type: 'positive', message: `已填入名称: ${text.slice(0, 50)}`, position: 'bottom' });
    }
  } catch (_) { /* 剪贴板无权限 */ }
};

watch(showBus, (val) => {
  if (val) {
    lastClipboardText = '';
    window.addEventListener('focus', onWindowFocus);
  } else {
    window.removeEventListener('focus', onWindowFocus);
  }
});

const emits = defineEmits([
  'success',
  'next-one',
  'prev-one',
  'update:modelValue',
  ...useDialogPluginComponent.emits,
]);

const makePreview = () => {
  if (!view.item?.Code) return;
  if (
    (view.item.MovieType === '骑兵' || view.item.MovieType === '无') &&
    systemProperty.fileEditAutoJpg
  ) {
    const uriCode = view.item.Code.toLowerCase().trim().replace('-', '00');
    // 自动封面会覆盖手填的内联图片
    jpgDataUrl.value = '';
    view.item.Jpg =
      systemProperty.SettingInfo.ImageUrl + `${uriCode}/${uriCode}pl.jpg`;
  }
};

const reg = /\w+[-_]\d+/;
const reg_1 = /\w+\d+/;

const titleChange = (v) => {
  if (!v || v.length === 0 || !systemProperty.fileEditAutoCode) return;
  v = String(v);
  v = v.replace(/[\r\n\t]+/g, '');
  v = v.replace(/&nbsp;/g, '');
  v = v.trimEnd();
  let originalCode = v.match(reg);
  if (!originalCode) {
    originalCode = v.match(reg_1);
  }
  if (originalCode && originalCode[0] && originalCode[0].length > 0) {
    let ncode = originalCode[0].toUpperCase();
    if (ncode.indexOf('-') < 0 && ncode.indexOf('_') < 0) {
      // 字母和数字之间插入 -
      ncode = ncode.replace(/([a-zA-Z])(\d)/g, '$1-$2');
    }
    view.item.Code = ncode;
    if (view.item.MovieType === '骑兵' || view.item.MovieType === '无') {
      makePreview();
    }
  }
  const arr = v.split(' ');
  if (arr && arr.length > 2) {
    view.item.Author = arr[arr.length - 1];
    view.item.Author = view.item.Author.trim();
  }
  // 从 Title 中移除 originalCode（忽略大小写）
  if (originalCode && originalCode[0]) {
    const codePattern = originalCode[0].replace(/[-_\s]/g, '[-_\\s]?');
    view.item.Title = view.item.Title.replace(new RegExp(codePattern, 'i'), '');
  }
  view.item.Title = view.item.Title.replace(/[：:{{}}《》]/g, ' ');
};

const pasteFromClipboard = async (field) => {
  const text = await navigator.clipboard.readText();
  if (field === 'Jpg' || field === 'Png') {
    applyImageField(field, text);
    return;
  }
  view.item[field] = text;
  if (field === 'Title') {
    titleChange(text);
  }
};

const open = (item) => {
  showBus.value = false;
  jpgDataUrl.value = '';
  pngDataUrl.value = '';
  view.item = new FileModel().fromObject(item);
  view.nodeHost = item.NodeHost || '';
  view.item.Jpg = null;
  view.item.Png = null;
  view.item.MovieType = item.MovieType;
  view.item.Code = item.Code?.toUpperCase();
  view.item.Title = formatTitle(item.Title);
  dialogRef.value.show();
};

const editMoveout = async () => {
  await editItemSubmit(true);
};

const editItemSubmit = async (MoveOut = false) => {
  const { Id, Title, Code, Author, FileType, MovieType, Tags } = view.item;
  // 内联 base64 图片存在独立字段，优先于输入框里的地址
  const Jpg = jpgDataUrl.value || view.item.Jpg || '';
  const Png = pngDataUrl.value || view.item.Png || '';
  let code = Code.trim().toUpperCase();
  if (code && code.indexOf('-') < 0) {
    code = '-' + code;
  }
  let name = '';
  if (Author.length !== 0) {
    name += '[' + Author.trim() + ']';
  }
  if (code.length !== 0) {
    name += ' [' + code.trim() + ']';
  }

  const arr = (Title || '').trim().split('.');
  const arrLength = arr.length;
  for (let idx = 0; idx < arrLength; idx++) {
    const str = arr[idx];
    if (!str) continue;
    name += str.charAt(0).toUpperCase() + str.slice(1) + ' ';
  }
  if (Tags && Tags.length > 0) {
    name += `《${Tags.join(',')}》`;
  }
  if (MovieType && MovieType !== '无') {
    if (name.indexOf('{{') < 0) {
      name += `{{${MovieType}}}`;
    }
  }
  if (name.indexOf('.' + FileType) < 0) {
    name += '.' + FileType;
  }
  const param = {
    Id,
    Name: name,
    Code: code,
    Title,
    Author,
    MoveOut,
    MovieType,
    Jpg,
    Png,
    NoRefresh: true,
    Host: view.nodeHost,
  };
  if (systemProperty.fileEditAutoNext) {
    await emits('next-one');
  } else {
    onDialogOK();
  }
  systemProperty.lastAuthor = Author;
  if (systemProperty.lastAuthores.indexOf(Author) >= 0) {
    systemProperty.lastAuthores.splice(
      systemProperty.lastAuthores.indexOf(Author),
      1
    );
  }
  if (systemProperty.lastAuthores.length >= 5) {
    systemProperty.lastAuthores.pop();
  }
  systemProperty.lastAuthores = [Author, ...systemProperty.lastAuthores];
  // 超时 / 网络 / HTTP 错误由 axios 拦截器统一提示，未捕获的 rejection 有全局兜底，
  // 这里只处理业务结果
  const res = await FileRename(param);
  if (res?.Code === 200) {
    if (systemProperty.fileEditAutoRefresh) {
      emits('success', Id);
    }
  } else {
    $q.notify({
      type: 'negative',
      message: res?.Message || '重命名失败',
      position: 'bottom-left',
    });
  }
};

const prevOne = async () => {
  await emits('prev-one');
};

const nextOne = async () => {
  await emits('next-one');
};

const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } =
  useDialogPluginComponent();
// dialogRef      - 用在 QDialog 上的 Vue ref 模板引用
// onDialogHide   - 处理 QDialog 上 @hide 事件的函数
defineExpose({
  open,
});
</script>
<style lang="css" scoped>
.taggle {
  border: 1px dotted rgb(197, 131, 50);
  margin-right: 4px;
  padding: 8px;
  border-radius: 8%;
}

.dialog {
  display: flex;
  flex-direction: column;
  background-color: bisque;
  min-height: 66vh;
}

.file-edit-card {
  background-color: rgba(250, 250, 250, 1);
  display: flex;
  justify-content: flex-end;
  width: 100%;
  height: 66vh;
  flex-direction: row;
  overflow: auto;
}

.bus-panel {
  width: 50%;
  height: 100%;
}

.preview-panel {
  border-radius: 5px;
}

.preview-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 60px;
  border: 2px dashed rgba(0, 0, 0, 0.1);
  border-radius: 8px;
}

.preview-placeholder p {
  margin-top: 8px;
}

@media (max-width: 768px) {
  .file-edit-card {
    max-width: 96vw;
    width: 96vw;
  }

  .file-edit-card :deep(.q-toolbar) {
    gap: 4px;
    padding: 4px;
  }

  .file-edit-card :deep(.q-toolbar .q-btn) {
    font-size: 12px;
  }

  .preview-placeholder {
    height: 120px;
  }
}
</style>
