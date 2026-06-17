<template>
  <div
    class="tag-popup"
    style="
      max-width: 400px;
      max-height: 100vh;
      padding: 4px;
      background-color: rgba(250, 250, 250, 0.5);
    "
  >
    <div class="row w100">
      <div class="col">
        <q-btn flat dense> 种草来源 </q-btn>
        <q-radio
          v-model="systemProperty.submitTagFromData"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="true"
          label="统计"
          @click="loadTagSize"
        />
        <q-radio
          v-model="systemProperty.submitTagFromData"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="false"
          label="字典"
          @click="loadTagSize"
        />
        <q-checkbox
          v-model="view.chooseInput"
          checked-icon="task_alt"
          unchecked-icon="panorama_fish_eye"
          :val="false"
          label="输入"
          @click="chooseInput"
        />
      </div>
    </div>
    <div v-if="!view.chooseInput">
      <div class="justify-start" style="height: 80px">
        <q-btn
          icon="ti-minus"
          square
          dense
          size="sm"
          text-color="white"
          color="red"
          class="tag-item glossy"
          v-for="tag in props.currentTag"
          :key="tag"
          :label="tag"
          :val="tag"
          @click="doCloseTag(tag)"
        />
      </div>
      <div v-if="!systemProperty.submitMutiTag">
        <q-btn
          icon="ti-plus"
          square
          dense
          text-color="white"
          color="red"
          class="tag-item"
          v-for="tag in view.tagData"
          :key="tag.Name"
          :label="tag.Name"
          :val="tag.Name"
          @click="doAddTag(tag.Name)"
          :disable="props.currentTag?.indexOf(tag.Name) >= 0"
        />
      </div>
      <div class="row" v-if="systemProperty.submitMutiTag">
        <q-btn
          color="orange"
          style="width: 100%"
          label="提交"
          v-close-popup
          class="tag-item glossy"
          @click="addPlayingMutiTag"
        ></q-btn>
      </div>
      <div
        v-if="systemProperty.submitMutiTag"
        style="height: 400px; overflow: auto"
      >
        <q-checkbox
          v-model="view.submitMutiTag"
          v-for="tag in view.tagData"
          :key="tag.Name"
          :val="tag.Name"
          dense
          keep-color
          :disable="props.currentTag?.indexOf(tag.Name) >= 0"
          :dark="props.currentTag?.indexOf(tag.Name) >= 0"
          :label="tag.Name.substring(0, 6)"
          color="red"
          class="q-pr-md glossy"
        />
      </div>
    </div>
  </div>
  <div v-if="view.chooseInput" style="padding: 10px">
    <q-input
      v-model="view.input"
      style="width: 100%"
      label="输入"
      class="inputWords"
    />
    <q-btn
      color="orange"
      style="width: 100%"
      label="提交"
      v-close-popup
      class="tag-item glossy"
      @click="submitInput"
    ></q-btn>
  </div>
</template>

<script setup>

import { CloseTag, AddTag } from 'components/api/searchAPI';
import { useSystemProperty } from 'stores/System';
import { onMounted, reactive } from 'vue';
import { useCommonExec } from 'src/composables/useCommonExec';

const systemProperty = useSystemProperty();
const { exec: commonExec } = useCommonExec();

const view = reactive({
  submitMutiTag: [],
  tagData: [],
  chooseInput: false,
  input: '',
});
const props = defineProps({
  currentData: {
    type: Object,
    default: () => {
      {
      }
    },
  },
  currentTag: {
    type: Array,
    default: () => [],
  },
  delay: {
    type: Number,
    default: 1,
  },
});

const loadTagSize = async () => {
  view.chooseInput = false;
  if (props.currentTag) {
    view.submitMutiTag = props.currentTag;
  }
  if (
    systemProperty.submitTagFromData &&
    systemProperty.tagSizeMap &&
    systemProperty.tagSizeMap.length > 0
  ) {
    view.tagData = systemProperty.tagSizeMap;
  } else {
    systemProperty.SettingInfo.Tags;
    view.tagData = systemProperty.SettingInfo.Tags.map((item) => {
      return { Name: item };
    });
  }
};

const emmits = defineEmits(['doBefore', 'updateTag']);

const doCloseTag = async (tag) => {
  const updated = await commonExec(() => CloseTag(props.currentData.Id, tag));
  if (updated) {
    Object.assign(props.currentData, updated);
    emmits('updateTag', updated.Tags);
  }
};

const doAddTag = async (tag) => {
  const updated = await commonExec(() => AddTag(props.currentData.Id, tag));
  if (updated) {
    Object.assign(props.currentData, updated);
    emmits('updateTag', updated.Tags);
  }
};

const addPlayingMutiTag = async () => {
  if (view.submitMutiTag.length > 0) {
    const tags = view.submitMutiTag.join(',');
    const updated = await commonExec(() => AddTag(props.currentData.Id, tags));
    if (updated) {
      Object.assign(props.currentData, updated);
      emmits('updateTag', updated.Tags);
    }
    view.submitMutiTag = [];
  }
};

const chooseInput = () => {
  setTimeout(() => {
    const inputElement = document.getElementsByClassName('inputWords');
    if (inputElement) {
      inputElement[0].focus();
    }
  }, 100);
};

const submitInput = async () => {
  if (view.input) {
    const updated = await commonExec(() => AddTag(props.currentData.Id, view.input));
    if (updated) {
      Object.assign(props.currentData, updated);
      emmits('updateTag', updated.Tags);
    }
    view.input = '';
  }
};

onMounted(() => {
  loadTagSize();
});
</script>
