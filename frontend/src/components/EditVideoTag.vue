<template>
  <div
    :style="containerStyle"
    class="edit-video-tag"
  >
    <div class="col">
      <q-btn flat dense> 种草来源 </q-btn>
      <q-radio
        v-model="systemProperty.submitTagFromData"
        checked-icon="task_alt"
        unchecked-icon="panorama_fish_eye"
        :val="true"
        label="标签统计"
        @click="loadTagData"
      />
      <q-radio
        v-model="systemProperty.submitTagFromData"
        checked-icon="task_alt"
        unchecked-icon="panorama_fish_eye"
        :val="false"
        label="标签设置"
        @click="loadTagData"
      />
    </div>
    <div class="row w100">
      <div class="col-12">
        <q-btn flat dense> 转码任务 </q-btn>
        <q-btn @click="toVcode(props.currentData.Id, 'copy')">MP4</q-btn>
        <q-btn @click="toVcode(props.currentData.Id, 'h264')">H264</q-btn>
        <q-btn @click="toVcode(props.currentData.Id, 'h265')">H265</q-btn>
      </div>
    </div>
    <div
      class="justify-start w100"
      style="max-width: 400px; height: auto; min-height: 60px; overflow: auto"
    >
      <q-btn
        icon="ti-minus"
        square
        dense
        size="sm"
        text-color="white"
        color="red"
        class="tag-item glossy"
        v-for="tag in props.currentData?.Tags"
        :key="tag"
        :label="tag"
        :val="tag"
        @click="removePlayingTag(props.currentData.Id, tag)"
      />
    </div>
    <div
      style="max-width: 400px"
      class="row w100 justify-start"
      v-if="!systemProperty.submitMutiTag"
    >
      <q-btn
        size="md"
        icon="ti-plus"
        square
        dense
        text-color="white"
        color="red"
        class="tag-item glossy fixed"
        v-for="tag in view.tagData"
        :key="tag.Name"
        :label="tag.Name"
        :val="tag.Name"
        v-close-popup
        @click="addPlayingTag(props.currentData.Id, tag.Name)"
        :disable="props.currentData?.Tags?.indexOf(tag.Name) >= 0"
      />
    </div>
    <div class="row w100">
      <q-btn
        color="orange"
        class="glossy w100"
        v-close-popup
        @click="addPlayingMutiTag(props.currentData.Id)"
        v-if="systemProperty.submitMutiTag"
        label="提交"
      ></q-btn>
    </div>
    <div
      class="row"
      style="max-width: 400px; max-height: 400px; overflow: auto"
      v-if="systemProperty.submitMutiTag"
    >
      <q-checkbox
        keep-color
        v-model="view.submitMutiTag"
        v-for="tag in view.tagData"
        :key="tag.Name"
        :label="tag.Name.substring(0, 6)"
        :val="tag.Name"
        dense
        :disable="props.currentData?.Tags?.indexOf(tag.Name) >= 0"
        :dark="props.currentData?.Tags?.indexOf(tag.Name) >= 0"
        color="red"
        class="q-pr-md glossy"
      />
    </div>
  </div>
</template>
<script setup>
import { useSystemProperty } from 'src/stores/System';
import {
  AddTag,
  CloseTag,
  TansferFileVcode,
} from './api/searchAPI';
import { onMounted, reactive, inject, computed } from 'vue';
import { useCommonExec } from 'src/composables/useCommonExec';

const systemProperty = useSystemProperty();
const { exec: commonExec } = useCommonExec();
const props = defineProps({
  currentData: {
    type: Object,
    default: () => ({}),
  },
});

const view = reactive({
  tagData: [],
  submitMutiTag: [],
});

const fetchToUpdateList = inject('fetchToUpdateList', () => undefined);

const emmits = defineEmits(['nextOne', 'prevOne']);

// 主题感知容器样式
const containerStyle = computed(() => {
  const isDark = systemProperty.theme === 'star';
  return {
    padding: '12px 4px',
    backgroundColor: isDark ? 'rgba(30, 30, 40, 0.95)' : 'rgba(250, 250, 250, 0.9)',
    maxWidth: '400px',
    maxHeight: '100vh',
    height: 'auto',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'flex-start',
    color: isDark ? 'white' : '#333',
  };
});

const toVcode = async (item, vcode) => {
  if (systemProperty.addPlayingTagGoNext) {
    emmits('nextOne');
  } else {
    emmits('prevOne');
  }
  commonExec(() => TansferFileVcode(item, vcode));
};

const loadTagData = async () => {
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

const addPlayingMutiTag = async (id) => {
  if (view.submitMutiTag.length > 0) {
    const tags = view.submitMutiTag.join(',');
    await addPlayingTag(id, tags);
    view.submitMutiTag = [];
  }
};

const addPlayingTag = async (id, tag) => {
  if (systemProperty.addPlayingTagGoNext) {
    emmits('nextOne');
  } else {
    emmits('prevOne');
  }

  setTimeout(async () => {
    const res = await AddTag(id, tag);
    if (res?.Data) {
      Object.assign(props.currentData, res.Data);
    }
    fetchToUpdateList(props.currentData);
  }, 1000);
};

const removePlayingTag = async (id, tag) => {
  if (systemProperty.addPlayingTagGoNext) {
    emmits('nextOne');
  } else {
    emmits('prevOne');
  }

  setTimeout(async () => {
    const res = await CloseTag(id, tag);
    if (res?.Data) {
      Object.assign(props.currentData, res.Data);
    }
    fetchToUpdateList(props.currentData);
  }, 1000);
};

onMounted(() => {
  loadTagData();
});
</script>
