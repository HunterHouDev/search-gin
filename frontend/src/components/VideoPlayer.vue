<template>
  <div style="position: fixed; z-index: 1; width: 100%" id="videoPlayerFrame">
    <q-toolbar
      class="row justify-end"
      style="padding: 2px 0px; overflow-y: auto; flex-wrap: wrap"
    >
      <q-toolbar-title>
        <q-btn square dense text-color="red" class="chip-tag">
          <span>{{ view.currentData.Actress?.substring(0, 6) }}</span>
        </q-btn>
        <q-btn square dense text-color="red" class="chip-tag">
          <span>{{ view.currentData.Code?.substring(0, 12) }}</span>
        </q-btn>
        <q-btn
          square
          dense
          text-color="red"
          v-for="tag in view.currentData.Tags"
          :key="tag"
          class="chip-tag"
        >
          <span>{{ tag?.substring(0, 4) }}</span>
        </q-btn>
      </q-toolbar-title>
      <q-btn :label="view.currentTime" color="red" flat>
        <q-popup-proxy
          v-model="view.showCut"
          style="width: 320px; max-height: 50vh"
        >
          <VideoCutParam
            :restartHidden="true"
            :current-time="view.currentTime"
            :duration="videoClass.duration()"
            @prev-one-video="prevOne(-1)"
            @next-one-video="nextOne(1)"
            @playVideo="videoClass.play()"
            @stopVideo="videoClass.pause()"
            @forward-time="forwardTime"
          />
        </q-popup-proxy>
      </q-btn>

      <q-btn flat color="primary" @click="showDrawerFn" label="列表/图库" />
      <q-btn flat color="orange" label="截图" @click="curImage" />
      <q-btn
        flat
        color="red"
        icon="edit"
        label="种草"
        size="md"
        dense
        class="q-mr-sm"
      >
        <q-popup-proxy style="background-color: rgba(250, 250, 250, 0.9)">
          <TagPop
            :currentData="view.currentData"
            :current-tag="view.currentData.Tags"
            @do-before="nextOne(1)"
            :delay="1000"
          />
        </q-popup-proxy>
      </q-btn>
      <q-btn
        flat
        color="red"
        size="md"
        label="设置"
        class="fts12"
        icon="settings"
      >
        <q-popup-proxy>
          <PlayerSetting />
        </q-popup-proxy>
      </q-btn>
      <DeleteBtn
        dense
        flat
        :current-data="view.currentData"
        @next-one="nextOne"
        @prev-one="nextOne(-1)"
      />
      <!--  -->
      <VideoVolumnBtn
        v-if="view.showVolumnBtn"
        color="red"
        centerColor="white"
        @volume-update="volumeUpdate"
        @volume-up="volumeUp"
        style="margin-left: 8px"
      />
      <q-btn
            dense
            flat
            v-if="!view.webFullScreen"
            align="center"
            size="md"
            icon="ti-fullscreen"
            color="red"
            @click="changeVideoScreen"
            v-touch-pan.prevent.mouse="moveFab"
          >
            <q-tooltip class="bg-white text-primary">{{
              view.videoFullscreen ? '还原' : '全屏'
            }}</q-tooltip>
          </q-btn>
      <q-btn
        dense
        v-if="props.closeBtn"
        icon="close"
        color="red"
        @click="close"
        style="margin-left: 10px"
      >
        <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
      </q-btn>
    </q-toolbar>
    <div class="row justify-end q-gutter-sm">
      <VideoTimeBar
        :current-time="view.currentTime"
        @forward-time="forwardTime"
        @time-rate="timeRate"
        @playVideo="videoClass.play()"
        @stopVideo="videoClass.pause()"
        @prev-one="nextOne(-1)"
        @next-one="nextOne(1)"
      />
    </div>
  </div>

  <video
    id="videoPlayerID"
    ref="vue3VideoPlayRef"
    controls
    preload="auto"
    playsinline
    :src="view.videoUrl"
    :poster="view.videoPoster"
    @playing="systemProperty.playerRunning = true"
    @pause="systemProperty.playerRunning = false"
    @loadedmetadata="checkSubtitles"
    style="width: -webkit-fill-available; background-color: rgba(0, 0, 0, 0.9)"
    :style="{
      position: isMobile ? '' : 'fixed',
      height: !props.fullscreen && isMobile ? '70vh' : '-webkit-fill-available',
      'object-fit': systemProperty.videoOptions.playerMode,
      filter: `brightness(${systemProperty.videoOptions.brightness}%)`,
      transform: `${systemProperty.videoOptions.rotate}`,
    }"
  >
    <track
      :src="view.videoSubtitles"
      kind="subtitles"
      srclang="zh"
      label="中文"
      default
      @error="handleSubtitleError"
    />
  </video>
  <q-page-sticky
    position="top-left"
    :offset="[10, -100]"
    :style="{
      zIndex: 9,
      height: view.videoHeight * 0.8 + 'px',
      width: isMobile ? '98vw' : view.videoWidth * 0.4 + 'px',
    }"
  >
    <div
      style="z-index: 9; margin-top: 10rem"
      v-show="view.showDrawer"
      :style="{ width: isMobile ? '98vw' : view.videoWidth * 0.45 + 'px' }"
    >
      <VideoListSide
        rightClose
        :detailHeight="view.videoHeight * 0.9"
        :currentId="view.currentData.Id"
        :currentTime="view.currentTime"
        @open-video="videoSideOpen"
        @closebtn="closeDrawer"
        @forward-time="forwardTime"
        ref="playerlist"
      />
    </div>
  </q-page-sticky>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useSystemProperty } from 'stores/System';
import { CutImage } from 'components/api/searchAPI';
import { VideoClass } from 'components/utils/video';
import {
  getFileStream,
  getJpg,
  getVideoSrt,
} from 'components/utils/images';
import { parseTime } from 'components/utils';

import VideoTimeBar from 'components/VideoTimeBar.vue';
import VideoCutParam from 'components/VideoCutParam.vue';
import VideoVolumnBtn from 'components/VideoVolumnBtn.vue';
import PlayerSetting from 'components/PlayerSetting.vue';
import DeleteBtn from 'components/DeleteBtn.vue';
import VideoListSide from 'components/VideoListSide.vue';
import TagPop from './TagPop.vue';

const $q = useQuasar();
const isMobile = computed(() => {
  return $q.platform.is.mobile;
});

const props = defineProps({
  closeBtn: { type: Boolean, default: false },
  fullscreen: { type: Boolean, default: false },
});

const systemProperty = useSystemProperty();
const vue3VideoPlayRef = ref(null);
const playerlist = ref(null);
const videoClass = new VideoClass('videoPlayerID');
let animationFrameId = null;
let currentItemId = null;

const showProgress = () => {
  if (!currentItemId) return;
  
  view.currentTime = parseTime(videoClass.currentTime());
  if (
    videoClass.currentTime() > 60 &&
    systemProperty.playerReLocation
  ) {
    systemProperty.addPlayerLocation(currentItemId, videoClass.currentTime());
  }
  animationFrameId = requestAnimationFrame(showProgress);
};

const view = reactive({
  showDrawer: false,
  currentData: [],
  showCut: false,
  currentTime: '00:00:00',
  videoWidth: 400,
  videoHeight: 400,
  videoUrl: '',
  videoPoster: '',
  showVolumnBtn: false,
  webFullScreen: false,
});

const showDrawerFn = () => {
  if (view.showDrawer) {
    view.showDrawer = false;
  } else {
    view.showDrawer = true;
    console.log('refreshData', playerlist.value);
    playerlist.value?.refreshData();
  }
};

const changeVideoScreen = () => {
  const videoElement = document.getElementById('videoPlayerID');
  if (videoElement) {
    view.videoFullscreen = !view.videoFullscreen;
    $q.fullscreen.toggle(videoElement);
  }
};

const closeDrawer = () => {
  console.log('closeDrawer');
  view.showDrawer = false;
};

const videoSideOpen = (params) => {
  const { item } = params;
  openVideo(item);
};

const openVideo = async (item) => {
  currentItemId = item.Id;
  view.currentData = item;
  systemProperty.PlayingMovie = item;
  view.videoUrl = getFileStream(item.Id);
  view.videoPoster = getJpg(item.Id);
  view.videoSubtitles = getVideoSrt(item.Srt);
  setTimeout(() => {
    videoClass.play();
    view.showVolumnBtn = true;
    videoClass.volumeUpdate(systemProperty.videoOptions.volume);
    const videoLocation = systemProperty.getPlayerLocation(item.Id);
    console.log('time-left', videoClass.duration() - videoLocation);
    if (
      videoLocation &&
      systemProperty.playerReLocation &&
      videoClass.duration() - videoLocation > 180
    ) {
      videoClass.timeUpdate(videoLocation);
    }
    setVideoWidth();
  }, 500);

  showProgress();
  setVideoWidth();
};

const curImage = async () => {
  showDrawerFn();
  await CutImage(view.currentData.Id, 'shot', view.currentTime, false);
  playerlist.value?.refreshData(2);
};

const forwardTime = (n) => {
  if (view.currentData) {
    videoClass.forwardTime(n);
  }
};

const timeRate = (time) => {
  if (view.currentData) {
    videoClass.timeRate(time);
  }
};

const volumeUp = (val) => {
  if (view.currentData && view.videoUrl) {
    console.log('volumeUp', val);
    videoClass.volumeUp(val);
  }
};

const volumeUpdate = (val) => {
  if (view.currentData && view.videoUrl) {
    systemProperty.videoOptions.volume = videoClass.volumeUpdate(val);
  }
};

const setVideoWidth = () => {
  const time1 = setTimeout(() => {
    const videoElement = document.getElementById('videoPlayerID');
    view.videoHeight = videoElement?.clientHeight;
    view.videoWidth = videoElement?.clientWidth;
    // console.log('clientHeight', videoElement?.clientHeight);
    // console.log('clientWidth', videoElement?.clientWidth);
    clearTimeout(time1);
  }, 500);
};

const emmits = defineEmits(['nextOne', 'close']);

const close = () => {
  view.videoUrl = '';
  view.videoPoster = '';
  view.currentData = {};
  systemProperty.PlayingMovie = {};
  emmits('close');
};

const nextOne = (step) => {
  view.videoUrl = '';
  view.videoPoster = '';
  console.log('nextOne', step);
  emmits('nextOne', step);
};


const handleSubtitleError = (e) => {
  console.error('字幕加载失败:', e);
};

const checkSubtitles = () => {
  const video = document.getElementById('videoPlayerID');
  const track = video.textTracks[0];
  if(track) {
    track.mode = 'showing';
    console.log('字幕轨道已加载');
  } else {
    console.warn('未检测到字幕轨道');
  }
};

const stopProgress = () => {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId);
    animationFrameId = null;
  }
};

watch(() => systemProperty.playerRunning, (isRunning) => {
  if (!isRunning) {
    stopProgress();
  } else if (currentItemId) {
    showProgress();
  }
});

onUnmounted(() => {
  stopProgress();
});

onMounted(() => {
  systemProperty.PlayingMovie = {};
});

defineExpose({
  openVideo,
});
</script>

<style scoped>
.video-player {
  width: 100%;
  height: 100%;
}
video {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>

 