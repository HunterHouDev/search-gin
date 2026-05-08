<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card
      style="
        width: 800px;
        max-width: 90vw;
        background-color: rgba(250, 250, 250, 1);
      "
    >
      <q-toolbar
        class="rounded-borders justify-between"
        style="background-color: rgba(0, 0, 0, 0.9)"
      >
        <q-btn color="red" flat icon="ti-shift-left" @click="prevOne">
          <q-tooltip class="bg-white text-primary">上一个</q-tooltip>
        </q-btn>

        <span style="color: white; font-size: 18px" v-if="!isMobile">
          <span style="color: orchid; cursor: pointer">
            历史图鉴 ：
            <q-popup-proxy>
              <div
                style="
                  padding: 10px;
                  display: flex;
                  flex-wrap: wrap;
                  flex-direction: row;
                  background-color: rgba(250, 250, 250, 1);
                  border-radius: 40px;
                "
                class="q-gutter-md"
                v-if="systemProperty.lastActresses && systemProperty.lastActresses.length>0"
              >
                <q-btn
                  color="orange"
                  v-close-popup
                  class="glossy"
                  v-for="item in systemProperty.lastActresses"
                  :key="item"
                  :label="item"
                  @click="systemProperty.lastActress = item"
                ></q-btn>
              </div>
            </q-popup-proxy>
          </span>
          <a
            style="color: green; border-bottom: 1px solid green; cursor: pointer"
            v-if="systemProperty.lastActress"
            @click="view.item.Actress = systemProperty.lastActress"
          >
            {{ systemProperty.lastActress }}</a
          >
        </span>
        <q-space />
        <q-btn
          style="margin-right: 10px"
          color="orange"
          align="evenly"
          label="改名移动"
          glossy
          @click="editMoveout"
        />
        <q-btn
          style="margin-right: 10px"
          color="green"
          glossy
          align="evenly"
          label="仅改名"
          @click="editItemSubmit(false)"
        />
        <q-btn
          style="margin-right: 10px"
          color="primary"
          icon="close"
          glossy
          dense
          @click="onDialogCancel"
        >
          <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
        </q-btn>
        <q-btn color="red" flat icon="ti-shift-right" @click="nextOne">
          <q-tooltip class="bg-white text-primary">下一个</q-tooltip>
        </q-btn>
      </q-toolbar>
      <q-form class="q-gutter-md row justify-between">
        <div
          class="q-gutter-sm q-pa-sm"
          :style="{
            width: isMobile
              ? '100%'
              : view.item.Jpg || view.item.Png
              ? '60%'
              : '100%',
          }"
        >
          <div>
            <p style="color: grey">
              {{ view.item.Path }}
              <span style="color: red">{{ view.item.SizeStr }}</span>
              <span
                style="color: green; margin-left: 10px"
                v-for="tag in view.item.Tags"
                :key="tag"
                >{{ tag }}</span
              >
            </p>
          </div>
          <q-input
            color="red-12"
            style="border-radius: 15px; background: rgba(0, 0, 0, 0.1)"
            autogrow
            standout
            outlined
            label="名称"
            v-model="view.item.Title"
            :dense="false"
            clearable
            @focus="titleChange"
            @change="titleChange"
          >
            <template v-slot:append>
              <q-icon
                name="style"
                size="lg"
                color="red"
                class="cursor-pointer"
                @click="fromclipboardTitle"
              />
            </template>
          </q-input>

          <q-input
            outlined
            label="图鉴"
            autogrow
            v-model="view.item.Actress"
            clearable
            :dense="false"
          >
          </q-input>
          <q-input
            outlined
            autogrow
            label="番号"
            v-model="view.item.Code"
            :dense="false"
            @change="makePreview"
            clearable
          />
          <q-input
            class="col-8"
            label="JPG地址"
            autogrow
            outlined
            clearable
            v-model="view.item.Jpg"
            :dense="false"
            @clear="systemProperty.fileEditAutoJpg = false"
          >
            <template v-slot:append>
              <q-icon
                name="style"
                size="md"
                class="cursor-pointer"
                @click="fromclipboardJpg"
              />
            </template>
          </q-input>
          <q-input
            label="PNG地址"
            autogrow
            outlined
            v-model="view.item.Png"
            clearable
            :dense="false"
          >
            <template v-slot:append>
              <q-icon
                name="style"
                size="md"
                class="cursor-pointer"
                @click="fromclipboardPng"
              />
            </template>
          </q-input>
          <div class="row w100">
            <q-toggle
              v-model="systemProperty.fileEditAutoCode"
              color="green"
              label="番号自动"
              left-label
              dense
              class="taggle"
            />
            <q-toggle
              v-model="systemProperty.fileEditAutoJpg"
              color="green"
              label="JPG自动"
              left-label
              dense
              class="taggle"
            />
            <q-toggle
              v-model="systemProperty.fileEditAutoRefresh"
              color="green"
              label="刷新自动"
              left-label
              dense
              class="taggle"
            />

            <q-toggle
              color="red"
              dense
              flat
              label="下一个"
              left-label
              v-model="systemProperty.fileEditAutoNext"
              class="taggle"
            />
          </div>
        </div>
        <div
          class="q-pa-sm"
          style="border-radius: 5px; width: 36%"
          v-if="view.item.Jpg || view.item.Png"
        >
          <q-img fit="fill" height="180px" :src="view.item.Jpg"></q-img>
          <q-img fit="fill" height="180px" :src="view.item.Png"></q-img>
        </div>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { useDialogPluginComponent, useQuasar } from 'quasar';
import { computed, reactive } from 'vue';

import { FileRename } from 'components/api/searchAPI';
import { formatTitle } from 'components/utils';
import { FileModel } from 'src/components/model/File';
import { useSystemProperty } from 'stores/System';
// import { useClipboard } from '@vueuse/core';
const systemProperty = useSystemProperty();

// const source = ref('Hello');
// const { copy } = useClipboard({ source });

const $q = useQuasar();
const isMobile = computed(() => {
  return $q?.platform.is.mobile;
});
const view = reactive({
  item: null,
  preview: false,
});

const emits = defineEmits([
  // REQUIRED; 需要明确指出
  // 组件通过 useDialogPluginComponent() 暴露哪些事件
  'success',
  'plus-one',
  'next-one',
  'prev-one',
  'sub-one',
  'update:modelValue',
  ...useDialogPluginComponent.emits,
]);

const makePreview = () => {
  if (
    (view.item.MovieType === '骑兵' || view.item.MovieType === '无') &&
    systemProperty.fileEditAutoJpg
  ) {
    const uriCode = view.item.Code.toLowerCase().trim().replace('-', '00');
    view.item.Jpg =
      systemProperty.SettingInfo.ImageUrl + `${uriCode}/${uriCode}pl.jpg`;
  }
};

const reg = /\w+[-_]\d+/;
const reg_1 = /\w+\d+/;

const titleChange = (v) => {
  if (v && v.length > 0 && systemProperty.fileEditAutoCode) {
    v = v.replace(/[\r\n\t]+/g, '');
    v = v.replace(/&nbsp;/g, '');
    v = v.trimEnd();
    let originalCode = v.match(reg);
    console.log('originalCode', originalCode);
    if (!originalCode) {
      originalCode = v.match(reg_1);
    }
    if (originalCode && originalCode[0] && originalCode[0].length > 0) {
      let ncode = originalCode[0].toUpperCase();
      if (ncode && ncode.indexOf('-') < 0 && ncode.indexOf('_') < 0) {
        // 字母和数字之间插入 -
        ncode = ncode.replace(/([a-zA-Z])(\d)/g, '$1-$2');
        if (ncode && ncode.indexOf('-') < 0 && ncode.indexOf('_') < 0) {
          ncode = '-' + ncode;
        }
      }
      console.log('ncode', ncode);
      view.item.Code = ncode;
      if (view.item.MovieType === '骑兵' || view.item.MovieType === '无') {
        makePreview();
      }
    }
    const arr = v.split(' ');
    if (arr && arr.length > 2) {
      view.item.Actress = arr[arr.length - 1];
      view.item.Actress = view.item.Actress.trim();
    }
    view.item.Title = view.item.Title.replace(originalCode, '');
    view.item.Title = view.item.Title.replace('：', ' ');
    view.item.Title = view.item.Title.replace(':', ' ');
    view.item.Title = view.item.Title.replace('{{', ' ');
    view.item.Title = view.item.Title.replace('}}', ' ');
    view.item.Title = view.item.Title.replace('《', ' ');
    view.item.Title = view.item.Title.replace('》', ' ');
  }
};

const fromclipboardTitle = async () => {
  const text = await navigator.clipboard.readText();
  view.item.Title = text;
  await titleChange(text);
};

const fromclipboardJpg = async () => {
  const text = await navigator.clipboard.readText();
  view.item.Jpg = text;
};

const fromclipboardPng = async () => {
  const text = await navigator.clipboard.readText();
  view.item.Png = text;
};

const open = (item) => {
  view.item = new FileModel().fromObject(item);
  view.item.Jpg = null;
  view.item.Png = null;
  view.item.MovieType = item.MovieType;
  view.item.Code = item.Code?.toUpperCase();
  view.item.Title = formatTitle(item.Title);
  view.previewUrl = null;
  dialogRef.value.show();
};

const editMoveout = async () => {
  await editItemSubmit(true);
};

const editItemSubmit = async (MoveOut) => {
  const { Id, Title, Code, Actress, FileType, MovieType, Jpg, Png, Tags } =
    view.item;
  let code = Code.trim().toUpperCase();
  if (code && code.indexOf('-') < 0) {
    code = '-' + code;
  }
  let name = '';
  if (Actress.length !== 0) {
    name += '[' + Actress.trim() + ']';
  }
  if (code.length !== 0) {
    name += ' [' + code.trim() + ']';
  }

  const arr = (Title || '').trim().split('.');
  const arrLength = arr.length;
  for (let idx = 0; idx < arrLength; idx++) {
    const str = arr[idx];
    const strNew = str.replace(str.charAt(0), str.charAt(0).toUpperCase());
    name += strNew;
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
    Actress,
    MoveOut,
    MovieType,
    Jpg,
    Png,
    NoRefresh: true,
  };
  emits('plus-one');
  if (systemProperty.fileEditAutoNext) {
    await emits('next-one');
  } else {
    onDialogOK();
  }
  systemProperty.lastActress = Actress;
  if (systemProperty.lastActresses.indexOf(Actress) >= 0) {
    systemProperty.lastActresses.splice(
      systemProperty.lastActresses.indexOf(Actress),
      1
    );
  }
  if (systemProperty.lastActresses.length >= 5) {
    systemProperty.lastActresses.pop();
  }
  systemProperty.lastActresses = [Actress, ...systemProperty.lastActresses];
  const res = await FileRename(param);
  if (res.Code === 200) {
    emits('sub-one');
    if (systemProperty.fileEditAutoRefresh) {
      emits('success', Id);
    }
  } else {
    emits('sub-one');
    $q.notify({
      type: 'negative',
      message: res.Message,
      position: 'top-right',
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
defineExpose({
  open,
});
</script>
<style lang="css">
.taggle {
  border: 1px dotted rgb(197, 131, 50);
  margin-right: 4px;
  padding: 8px;
  border-radius: 8%;
}
</style>
