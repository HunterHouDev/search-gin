<template>
  <div
    style="
      width: 320px;
      height: fit-content;
      background-color: rgba(250, 250, 250, 0.8);
    "
  >
    <div class="row justify-between w100" style="flex: 1">
      <q-btn
        glossy
        v-if="props.restartHidden"
        size="md"
        color="orange"
        label="<重启>"
        v-close-popup
        @click="restartVideo"
      />
      <q-btn
        glossy
        size="md"
        color="orange"
        label="<JPG>"
        @click="previewImg"
      />
      <q-btn
        glossy
        size="md"
        color="orange"
        label="<PNG>"
        @click="previewPng"
      />
      <q-btn glossy color="orange" label="<截图>" @click="curImage" />
    </div>
    <div class="row justify-between w100 flex-1 q-mt-xs">
      <q-btn
        glossy
        v-close-popup
        color="orange"
        label="toMP4"
        @click="toVcode('copy')"
      />
      <q-btn
        glossy
        v-close-popup
        color="orange"
        label="toH264"
        @click="toVcode('h264')"
      />
      <q-btn
        glossy
        v-close-popup
        color="orange"
        label="toH265"
        @click="toVcode('h265')"
      />
    </div>
    <div class="row justify-between w100 flex-1">
      <q-btn
        flat
        outline
        v-close-popup
        icon="ti-control-skip-backward"
        color="red"
        @click="prevOneVideo"
      >
        <q-tooltip class="bg-white text-primary">上一集</q-tooltip>
      </q-btn>
      <q-btn
        dense
        flat
        icon="ti-control-play"
        color="red"
        v-if="!systemProperty.playerRunning"
        @click="playVideo"
      ></q-btn>
      <q-btn
        dense
        flat
        icon="ti-control-pause"
        v-if="systemProperty.playerRunning"
        color="red"
        @click="stopVideo"
      ></q-btn>
      <q-btn
        flat
        icon="ti-control-skip-forward"
        color="red"
        v-close-popup
        @click="nextOneVideo"
      >
        <q-tooltip class="bg-white text-primary">下一集</q-tooltip>
      </q-btn>
    </div>
    <div class="row justify-between w100">
      <q-btn
        dense
        flat
        v-for="item in fowartBtn"
        :key="item"
        :label="item"
        color="red"
        @click="forwardTime(item)"
        @contextmenu="
          (e) => {
            forwardTime(-item);
            e.returnValue = false;
          }
        "
      ></q-btn>
    </div>
    <div class="row justify-evenly w100">
      <q-input
        outlined
        style="width: 48%"
        v-model="view.startTime"
        label="开始时间"
        @focus="
          view.startTime = props.currentTime;
          stopVideo();
        "
      />
      <q-input
        outlined
        align="center"
        style="width: 48%"
        v-model="view.endTime"
        label="结束时间"
        @focus="
          view.endTime = props.currentTime;
          stopVideo();
        "
      />
    </div>
    <div class="row justify-between w100">
      <q-btn
        glossy
        v-close-popup
        color="primary"
        style="width: 99%"
        :disabled="
          view.startTime >= view.endTime ||
          (view.startTime == TIME_START && view.endTime == TIME_END)
        "
        label="剪辑"
        @click="CutFromTo"
      />
      
    </div>
  </div>
</template>
<script setup>
import { reactive } from 'vue';
import { CutFile, CutImage } from './api/searchAPI';
import { useQuasar } from 'quasar';
import { TansferFileVcode } from './api/searchAPI';
import { useSystemProperty } from 'src/stores/System';
const TIME_START = '00:00:00';
const TIME_END = '99:00:00';

const systemProperty = useSystemProperty();
const fowartBtn = [-60, -30, -5, 30, 120, 240];
const $q = useQuasar();
const props = defineProps({
  currentData: {
    type: Object,
    default: () => ({}),
  },
  currentTime: {
    type: String,
    default: '00:00:00',
  },
  restartHidden: {
    type: Boolean,
    default: false,
  },

  duration: {
    type: Number,
    default: 0,
  },
});

const emmits = defineEmits([
  'restartVideo',
  'prevOneVideo',
  'nextOneVideo',
  'stopVideo',
  'playVideo',
  'forwardTime',
]);

const view = reactive({
  startTime: TIME_START,
  endTime: TIME_END,
});

const prevOneVideo = () => {
  emmits('prevOneVideo');
};

const nextOneVideo = () => {
  emmits('nextOneVideo');
};

const restartVideo = () => {
  emmits('restartVideo');
};
const stopVideo = () => {
  emmits('stopVideo');
};

const playVideo = () => {
  emmits('playVideo');
};
const forwardTime = (time) => {
  emmits('forwardTime', time);
};

const CutFromTo = async () => {
  const { Code, Message } = await CutFile(
    systemProperty.PlayingMovie.Id,
    view.startTime,
    view.endTime
  );
  view.startTime = TIME_START;
  view.endTime = TIME_END;
  if (Code !== 200) {
    $q.notify({ message: `${Message}`, position: 'top-right' });
  } else {
    $q.notify({ message: `${Message}`, position: 'top-right' });
    emmits('nextOneVideo');
  }
};

const curImage = async () => {
  await CutImage(
    systemProperty.PlayingMovie.Id,
    'shot',
    props.currentTime,
    false
  );
};

const previewImg = async () => {
  await CutImage(
    systemProperty.PlayingMovie.Id,
    'Jpg',
    props.currentTime,
    false
  );
};

const previewPng = async () => {
  await CutImage(
    systemProperty.PlayingMovie.Id,
    'Png',
    props.currentTime,
    false
  );
};

const toVcode = async (vcode) => {
  const Id = systemProperty.PlayingMovie.Id;
  const res = await TansferFileVcode(Id, vcode);
  if (res.Code !== 200) {
    $q.notify({ message: `${res.Message}`, position: 'top-right' });
  } else {
    $q.notify({ message: `${res.Message}`, position: 'top-right' });
    emmits('nextOneVideo');
  }
};
</script>

<style scoped>
/* 视频播放器样式 */
.fts12 {
  font-size: 1.2rem;
  padding: 0px;
  margin-right: 6px;
}
.w100 {
  width: 100%;
}
</style>
