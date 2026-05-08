<template>
  <q-layout
    view="lHh lpr lFf"
    container
    style="height: 93vh"
    class="shadow-2 rounded-borders"
    :style="themeStyle"
  >
    <!-- 头部 -->
    <q-header
      elevated
      class="q-gutter-sm flex justify-center"
      style="
        backdrop-filter: blur(5px);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      "
    >
      <q-tabs
        v-model="tab"
        style="width: 100%; position: fixed; z-index: 9"
        :style="{ backgroundColor: systemProperty.isDark ? 'black-4' : 'var(--q-primary)' }"
        no-caps
        glossy
        inline-label
        class="shadow-1"
        active-color="red"
        indicator-color="black"
        align="justify"
      >
        <q-tab name="search" label="搜索设置" />
        <q-tab name="base" label="基础设置" />
        <q-tab name="note" label="网络设置" />
        <q-tab name="system" label="系统设置" />
      </q-tabs>
    </q-header>
    <q-page-container class="scroll" style="margin-top: 4rem">
      
      <q-tab-panels v-model="tab" animated>
        <q-tab-panel name="search">
          <q-field color="purple-12" label="定时扫描" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.EnableTimeScan"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="true"
                  label="开启"
                />
                <q-radio
                  v-model="view.settingInfo.EnableTimeScan"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="false"
                  label="关闭"
                />
              </div>
            </template>
          </q-field>
          <q-field color="purple-12" label="转码删除原文件" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.CutThenDelete"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="true"
                  label="是"
                />
                <q-radio
                  v-model="view.settingInfo.CutThenDelete"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="false"
                  label="否"
                />
              </div>
            </template>
          </q-field>
          
          <q-field color="purple-12" label="系统播放" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.SystemPlayer"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  val="ffplay"
                  label="ffplay"
                />
                <q-radio
                  v-model="view.settingInfo.SystemPlayer"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  val="system"
                  label="system"
                />
              </div>
            </template>
          </q-field>
          <q-field color="purple-12" label="Buttons" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Buttons"
                :options="buttonEnum"
                @onchange="(arr) => (view.settingInfo.Buttons = arr)"
              />
            </template>
          </q-field>

          <q-field color="purple-12" label="Dirs" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Dirs"
                :options="view.settingInfo.DirsLib"
                @onchange="(arr) => (view.settingInfo.Dirs = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="MovieTypes" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.MovieTypes"
                @onchange="(arr) => (view.settingInfo.MovieTypes = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="VideoTypes" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.VideoTypes"
                :options="view.settingInfo.Types"
                @onchange="(arr) => (view.settingInfo.VideoTypes = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="Tags" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Tags"
                :options="view.settingInfo.TagsLib"
                @onchange="(arr) => (view.settingInfo.Tags = arr)"
              />
            </template>
          </q-field>
        </q-tab-panel>

        <q-tab-panel name="base">
          <q-input
            v-model="view.settingInfo.SystemPlayerVolumn"
            :max="100"
            :min="0"
            type="number"
            label="系统播放器音量"
          />
          <q-input
            v-model="view.settingInfo.SystemPlayerWidth"
            label="系统播放器宽度"
          />
          <q-input
            v-model="view.settingInfo.ControllerHost"
            label="ControllerHost"
          />
          <q-input v-model="view.settingInfo.ImageHost" label="ImageHost" />
          <q-input v-model="view.settingInfo.StreamHost" label="StreamHost" />
          <q-field color="purple-12" label="DirsLib" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.DirsLib"
                @onchange="(arr) => (view.settingInfo.DirsLib = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="TagsLib" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.TagsLib"
                @onchange="(arr) => (view.settingInfo.TagsLib = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="Types" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.Types"
                @onchange="(arr) => (view.settingInfo.Types = arr)"
              />
            </template>
          </q-field>
          <q-field color="purple-12" label="Pages" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.Pages"
                @onchange="(arr) => (view.settingInfo.Pages = arr)"
              />
            </template>
          </q-field>
        </q-tab-panel>
        <q-tab-panel name="note">
          <q-input v-model="view.settingInfo.BaseUrl" label="BaseUrl" />
          <q-input v-model="view.settingInfo.ImageUrl" label="ImageUrl" />
          <q-input v-model="view.settingInfo.OMUrl" label="OMUrl" />
          <q-input
            type="textarea"
            autogrow
            v-model="view.settingInfo.Remark"
            label="Remark"
          />
        </q-tab-panel>

        <q-tab-panel name="system">
          <q-editor
            v-model="view.settingInfo.SystemHtml"
            :dense="$q.screen.lt.md"
            :toolbar="[
              [
                {
                  label: $q.lang.editor.align,
                  icon: $q.iconSet.editor.align,
                  fixedLabel: true,
                  list: 'only-icons',
                  options: ['left', 'center', 'right', 'justify'],
                },
                {
                  label: $q.lang.editor.align,
                  icon: $q.iconSet.editor.align,
                  fixedLabel: true,
                  options: ['left', 'center', 'right', 'justify'],
                },
              ],
              [
                'bold',
                'italic',
                'strike',
                'underline',
                'subscript',
                'superscript',
              ],
              ['token', 'hr', 'link', 'custom_btn'],
              ['print', 'fullscreen'],
              [
                {
                  label: $q.lang.editor.formatting,
                  icon: $q.iconSet.editor.formatting,
                  list: 'no-icons',
                  options: ['p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'code'],
                },
                {
                  label: $q.lang.editor.fontSize,
                  icon: $q.iconSet.editor.fontSize,
                  fixedLabel: true,
                  fixedIcon: true,
                  list: 'no-icons',
                  options: [
                    'size-1',
                    'size-2',
                    'size-3',
                    'size-4',
                    'size-5',
                    'size-6',
                    'size-7',
                  ],
                },
                {
                  label: $q.lang.editor.defaultFont,
                  icon: $q.iconSet.editor.font,
                  fixedIcon: true,
                  list: 'no-icons',
                  options: [
                    'default_font',
                    'arial',
                    'arial_black',
                    'comic_sans',
                    'courier_new',
                    'impact',
                    'lucida_grande',
                    'times_new_roman',
                    'verdana',
                  ],
                },
                'removeFormat',
              ],
              ['quote', 'unordered', 'ordered', 'outdent', 'indent'],

              ['undo', 'redo'],
              ['viewsource'],
            ]"
            :fonts="{
              arial: 'Arial',
              arial_black: 'Arial Black',
              comic_sans: 'Comic Sans MS',
              courier_new: 'Courier New',
              impact: 'Impact',
              lucida_grande: 'Lucida Grande',
              times_new_roman: 'Times New Roman',
              verdana: 'Verdana',
            }"
          />
        </q-tab-panel>
      </q-tab-panels>
    </q-page-container>
    <q-footer elevated class="glossy">
      <q-btn
        align="evenly"
        :color="systemProperty.isDark ? 'black' : 'primary'"
        glossy
        ripple
        rounded
        class="w100"
        style="height: 100%"
        @click="submitForm"
        >提...交</q-btn
      >
    </q-footer>
  </q-layout>

  <!-- <q-page-sticky position="bottom" :offset="[20, 20]">
    
  </q-page-sticky> -->
</template>

<script setup>
import { useQuasar } from 'quasar';

import { onMounted, reactive, ref, computed } from 'vue';
import {
  GetSettingInfo,
  PostSettingInfo,
  GetIpAddr,
} from '../../components/api/settingAPI';
import MutiSelector from '../../components/MutiSelector.vue';
import MutiInput from '../../components/MutiInput.vue';
import { buttonEnum } from '../../components/model/Setting';
import { useSystemProperty } from '../../stores/System';

const $q = useQuasar();
const tab = ref('search');
const view = reactive({
  settingInfo: {},
  ipAddr: '',
});

const systemProperty = useSystemProperty();

const themeStyle = computed(() => {
  return {
    color: systemProperty.isDark ? 'white' : 'black',
    backgroundColor: systemProperty.isDark
      ? 'rgba(0, 0, 0, 0.6)'
      : 'rgba(255, 255, 255, 0.9)',
  };
});

const submitForm = async () => {
  view.settingInfo.Dirs = view.settingInfo.Dirs.sort();
  view.settingInfo.DirsLib = view.settingInfo.DirsLib.sort();
  view.settingInfo.Types = view.settingInfo.Types.sort();
  view.settingInfo.VideoTypes = view.settingInfo.VideoTypes.sort();

  view.settingInfo.Tags = view.settingInfo.Tags.filter((item) => {
    return view.settingInfo.TagsLib.includes(item);
  });
  const sortedTags = [];
  view.settingInfo.TagsLib.forEach((item) => {
    if (view.settingInfo.Tags.includes(item)) {
      sortedTags.push(item);
    }
  });
  view.settingInfo.Tags = sortedTags;
  view.settingInfo.Dirs = view.settingInfo.Dirs.filter((item) => {
    return view.settingInfo.DirsLib.includes(item);
  });
  view.settingInfo.VideoTypes = view.settingInfo.VideoTypes.filter((item) => {
    return view.settingInfo.Types.includes(item);
  });
  const { Code, Message } = await PostSettingInfo(view.settingInfo);
  console.log(Code, Message);
  if (Code != 200) {
    $q.notify({ message: `${Message}` });
    // window.location.reload()
  } else {
    $q.notify({ message: `${Message}` });
  }
};

const fetchSearch = async () => {
  const { data } = await GetSettingInfo();
  console.log(data);
  view.settingInfo = data;
};

const queryIpAddr = async () => {
  const { Code, Data } = await GetIpAddr();
  if (Code == '200') {
    view.ipAddr = `http://${Data}:10081`;
  }
};

onMounted(() => {
  document.title = '设置';
  fetchSearch();
  queryIpAddr();
});
</script>
<style lang="scss" scoped></style>
