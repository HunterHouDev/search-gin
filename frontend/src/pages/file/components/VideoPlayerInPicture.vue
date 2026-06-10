<template>
  <q-page-sticky
    v-show="view.videoUrl && !isPip"
    style="z-index: 9; max-width: 100vw; display: flex !important"
    position="bottom-right"
    :offset="videoOffset"
    :style="{
      background: 'rgba(0,0,0,0.9)',
      borderRadius: '8px',
      maxWidth: '100vw',
      maxHeight: isMobile() ? '80vh' : '90vh',
      justifyContent: 'center',
      alignItems: 'center',
    }"
  >
    <div
      class="shadow-2 rounded-borders"
      id="videoFrame"
      v-show="view.videoUrl"
    >
      <video
        autoplay
        preload="auto"
        x-webkit-airplay="allow"
        x-webkit-fullscreen="true"
        x-moz-fullscreen="true"
        x-ms-fullscreen="true"
        id="hoverVideoID"
        controls
        :src="view.videoUrl"
        :poster="getJpg(view.currentData?.Id)"
        @ended="nextOne"
        @playing="systemProperty.playerRunning = true"
        @pause="systemProperty.playerRunning = false"
        @wheel.prevent="onWheel"
        v-touch-pan="touchVideo"
        style="
          touch-action: auto;
          pointer-events: auto;
          width: 100%;
          min-width: 500px;
          min-height: 400px;
        "
        :style="{
          'object-fit': systemProperty.videoOptions.playerMode,
          width: computedFrameWidth(),
          height: computedFrameHeight(),
          filter: `brightness(${systemProperty.videoOptions.brightness}%)`,
          transform: `${systemProperty.videoOptions.rotate}`,
        }"
      ></video>
      <div v-if="view.showDrawer" style="position: fixed; inset: 0; display: flex; justify-content: center; align-items: center; z-index: 2000; background: rgba(0,0,0,0.6)">
        <SearchPanel
          ref="searchPanelRef"
          :visible="view.showDrawer"
          :currentId="view.currentData.Id"
          :currentTime="view.currentVideoTime"
          :isPlaying="systemProperty.playerRunning"
          isSmall
          @play="onSearchPanelPlay"
          @close="view.showDrawer = false"
          @keyword="onSearchPanelKeyword"
          @edit="onSearchPanelEdit"
          @delete="onSearchPanelDelete"
        />
      </div>
      <FileEdit ref="fileEditRef" />
      <q-page-sticky position="top-left" :offset="[0, -50]">
        <div class="q-gutter-sm row justify-start">
          <q-btn
            flat
            dense
            style="margin-left: 10px"
            v-if="!view.videoFullscreen && !view.webFullScreen && !isMobile()"
            size="md"
            icon="ti-arrow-top-left"
            color="red"
            v-touch-pan.prevent.mouse="zoomFab"
            @click="view.showDrawer = !view.showDrawer"
          >
            <q-tooltip class="bg-white text-primary">缩放</q-tooltip>
          </q-btn>
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
            flat
            v-if="!isMobile() && !view.videoFullscreen && !view.webFullScreen"
            icon="ti-layout-media-right-alt"
            color="red"
            @click="requestPiP"
            id="movebtn"
            v-touch-pan.prevent.mouse="moveFab"
          >
          </q-btn>
          <q-btn
            flat
            dense
            color="red"
            icon="settings"
            v-touch-pan.prevent.mouse="moveFab"
          >
            <q-popup-proxy>
              <PlayerSetting widescreen />
            </q-popup-proxy>
          </q-btn>

          <q-btn
            flat
            dense
            color="red"
            icon="ti-menu-alt"
            @click="view.showDrawer = true"
            v-touch-pan.prevent.mouse="moveFab"
          ></q-btn>
        </div>
        <div
          style="
            width: auto;
            height: auto;
            max-height: 50vh;
            display: flex;
            flex-direction: column;
            justify-content: start;
            flex-wrap: wrap;
          "
          :style="{
            marginTop:
              isMobile() || !systemProperty.videoOptions.widescreen
                ? '50px'
                : '10px',
            maxHeight: '68%',
          }"
        >
          <div>
            <q-chip
              square
              dense
              text-color="red"
              class="chip-tag"
              v-touch-pan.prevent.mouse="moveFab"
            >
              <span>
                {{ view.currentData.SizeStr }}
              </span>
            </q-chip>

            <q-chip
              v-if="view.showExtraInfo"
              square
              dense
              text-color="red"
              class="chip-tag"
              v-touch-pan.prevent.mouse="moveFab"
            >
              <span @click="copyText(view.currentData.Actress)">
                {{ view.currentData.Actress?.substring(0, 12) }}
              </span>
            </q-chip>
            <q-chip
              v-if="view.showExtraInfo"
              square
              dense
              text-color="red"
              class="chip-tag"
              v-touch-pan.prevent.mouse="moveFab"
            >
              <span @click="copyText(view.currentData.Code)">
                {{ view.currentData.Code?.substring(0, 24) }}
              </span>
              <q-tooltip class="bg-white text-primary">{{  view.currentData.Code  }}</q-tooltip>
            </q-chip>
          </div>

          <div v-for="tag in view.currentData?.Tags" :key="tag">
            <q-chip
              v-if="view.showExtraInfo"
              square
              dense
              text-color="red"
              :key="tag"
              class="chip-tag"
            >
              <span>{{ tag?.substring(0, 4) }}</span>

            </q-chip>
          </div>
        </div>
      </q-page-sticky>
      <q-page-sticky position="top-right" :offset="[0, -50]">
        <div class="row justify-end q-gutter-sm">
          <q-btn flat dense size="md" color="red" icon="ti-star" align="center">
            <q-popup-proxy>
              <EditVideoTag
                :current-data="view.currentData"
                @next-one="nextOne"
                @prev-one="prevOne"
              />
            </q-popup-proxy>
          </q-btn>
          <q-btn
            dense
            flat
            v-if="!view.videoFullscreen && !isMobile()"
            align="center"
            size="md"
            :icon="view.videoFullscreen ? 'ti-zoom-out' : 'ti-zoom-in'"
            color="red"
            v-touch-pan.disable="view.webFullScreen || view.videoFullscreen"
            @click="frameWebFullscreen"
          >
            <q-tooltip class="bg-white text-primary">{{
              view.webFullScreen ? '恢复' : '网页全屏'
            }}</q-tooltip>
          </q-btn>

          <DeleteBtn
            :dense="true"
            flat
            :current-data="view.currentData"
            @next-one="nextOne"
            @prev-one="prevOne"
          />
          <q-btn
            dense
            flat
            size="md"
            align="center"
            icon="ti-control-skip-backward"
            color="red"
            @click="prevOne"
          >
            <q-tooltip class="bg-white text-primary">上集</q-tooltip>
          </q-btn>

          <q-btn
            dense
            flat
            align="center"
            size="md"
            icon="ti-control-skip-forward"
            color="red"
            @click="nextOne"
          >
            <q-tooltip class="bg-white text-primary">下集</q-tooltip>
          </q-btn>

          <q-btn
            flat
            size="md"
            dense
            icon="close"
            color="red"
            align="right"
            v-touch-pan.prevent.mouse="moveFab"
            style="margin-right: 8px"
            @click="closeVideo"
          >
            <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
          </q-btn>
        </div>
      </q-page-sticky>

      <q-page-sticky
        position="top-right"
        :offset="[6, -10]"
        style="z-index: 99"
      >
        <div class="row justify-end flex-wrap q-gutter-sm">
          <VideoTimeBar
            nosize
            :current-time="view.currentVideoTime"
            @forward-time="forwardTime"
            @time-rate="timeRate"
            @playVideo="hoverPlayer.play()"
            @stopVideo="hoverPlayer.pause()"
            @next-one="nextOne"
            @prev-one="prevOne"
          />
          <VideoVolumnBtn
            v-if="view.showExtraInfo"
            style="margin-top: 18px"
            v-touch-pan.prevent.mouse="moveFab"
            @volume-update="volumeUpdate"
            @volume-up="volumeUp"
          />
        </div>
      </q-page-sticky>
      <q-page-sticky position="bottom-right" :offset="[10, 80]">
        <div
          style="
            display: flex;
            flex-direction: column;
            justify-content: flex-end;
            align-items: flex-end;
          "
          class="q-gutter-sm flex"
        >
          <q-btn
            flat
            dense
            color="red"
            class="fts12"
            style="max-width: 10rem"
            v-touch-pan.prevent.mouse="moveFab"
            :label="view.currentVideoTime"
            >::{{ view.progress }}%
            <q-popup-proxy>
              <VideoCutParam
                :current-time="view.currentVideoTime"
                :duration="hoverPlayer.duration()"
                @prev-one-video="prevOne"
                @next-one-video="nextOne"
                @restart-video="restartVideo()"
                @playVideo="hoverPlayer.play()"
                @stopVideo="hoverPlayer.pause()"
                @forward-time="forwardTime"
                restartHidden
              />
            </q-popup-proxy>
          </q-btn>
          <span
            v-if="view.showExtraInfo && !isMobile()"
            square
            dense
            v-touch-pan.prevent.mouse="moveFab"
          >
            {{ formatTitle(view.currentData.Name, 30) }}
            <q-tooltip class="bg-white text-primary">{{  view.currentData.Name  }}</q-tooltip>
          </span>
        </div>
      </q-page-sticky>
    </div>
  </q-page-sticky>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { computed, onMounted, reactive, ref } from 'vue';
import { useSystemProperty } from 'stores/System';
import { isMobile } from 'src/boot/platform';

import { getFileStream, getJpg } from 'components/utils/images';
import { VideoClass } from 'components/utils/video';
import { parseTime, formatTitle } from 'components/utils';
import { DeleteFile } from 'components/api/searchAPI';
import { onKeyStroke, useDebounceFn, useClipboard } from '@vueuse/core';

import PlayerSetting from 'components/PlayerSetting.vue';
import VideoCutParam from 'components/VideoCutParam.vue';
import EditVideoTag from 'components/EditVideoTag.vue';
import VideoTimeBar from 'components/VideoTimeBar.vue';
import SearchPanel from 'components/SearchPanel.vue';
import FileEdit from 'src/pages/file/components/FileEditDialog.vue';
import VideoVolumnBtn from 'components/VideoVolumnBtn.vue';
import DeleteBtn from 'components/DeleteBtn.vue';

const source = ref('Hello');
const { copy } = useClipboard({ source });

const $q = useQuasar();

const hoverPlayer = computed(() => {
  const hoverPlayer = new VideoClass('hoverVideoID');
  return hoverPlayer;
});

let animationFrame = null;
const systemProperty = useSystemProperty();
let showProgress = null;
const searchPanelRef = ref(null);
const fileEditRef = ref(null);
const view = reactive({
  showDrawer: false,
  showExtraInfo: true,
  splitterModel: 100,
  currentData: {},
  queryParam: {
    Keyword: '',
    MovieType: '',
    OnlyRepeat: false,
    Page: 1,
    PageSize: 20,
    SortField: 'MTime',
    SortType: 'desc',
  },
  videoUrl: null,
  videoFullscreen: false,
  webFullScreen: false,
  videoHeight: 500,
  videoWidth: 500,
  videoduration: 0,
  currentVideoTime: '00:00:00',
});

const copyText = async (str) => {
  if (str && str.startsWith('-')) {
    str = str.substring(1);
  }
  console.log(str);
  await copy(str);
  $q.notify({ message: `${str}`, position: 'bottom-left' });
};
const videoOffset = computed(() => {
  if (isMobile()) {
    return [0, 100];
  }
  return systemProperty.pictureInPictureVideoOffset;
});
const videoWidthComputed = computed(() => {
  return systemProperty.pictureInPictureVideoWidth;
});

const computedFrameWidth = () => {
  if (view.videoFullscreen || isMobile()) {
    return '100vw';
  }
  if (view.webFullScreen) {
    return window.innerWidth - 10 + 'px';
  }
  if (!systemProperty.videoOptions.widescreen) {
    return 'auto';
  }
  return videoWidthComputed.value + 80 + 'px';
};

const computedFrameHeight = () => {
  if (view.videoFullscreen) {
    return '96vh';
  }
  if (view.webFullScreen) {
    return window.innerHeight - 64 + 'px';
  }
  if (!systemProperty.videoOptions.widescreen) {
    return window.innerHeight - 180 + 'px';
  }
  return videoWidthComputed.value - 180 + 'px';
};

onKeyStroke(['Escape'], () => {
  if (view.videoFullscreen) {
    exitFullscreen();
  } else if (view.webFullScreen) {
    exitFrameWebFullscreen();
  } else {
    closeVideo();
    cancelAnimationFrame(showProgress);
  }
});

const timeRate = (time) => {
  if (view.currentData && view.videoUrl) {
    hoverPlayer.value.timeRate(time);
  }
};

const forwardTime = (time) => {
  if (view.currentData && view.videoUrl) {
    hoverPlayer.value.forwardTime(time);
  }
};

const onSearchPanelPlay = (item) => {
  openVideo({ item, queryParam: null });
  view.showDrawer = false;
};

const onSearchPanelKeyword = (_keyword) => {
  // 暂时关闭面板
  console.log('onSearchPanelKeyword', _keyword);
};

const onSearchPanelEdit = (item) => {
  fileEditRef.value?.open(item);
};

const onSearchPanelDelete = async (item) => {
  await DeleteFile(item.Id);
  searchPanelRef.value?.fetchSearch();
};

const moveFab = (ev) => {
  systemProperty.pictureInPictureVideoOffset = [
    systemProperty.pictureInPictureVideoOffset[0] - ev.delta.x,
    systemProperty.pictureInPictureVideoOffset[1] - ev.delta.y,
  ];
};

const zoomFab = (ev) => {
  systemProperty.pictureInPictureVideoWidth =
    systemProperty.pictureInPictureVideoWidth - ev.delta.x;
};

const changeVideoScreen = () => {
  const videoElement = document.getElementById('videoFrame');
  if (videoElement) {
    view.videoFullscreen = !view.videoFullscreen;
    $q.fullscreen.toggle(videoElement);
    videoWidthDebounce();
  }
};

const closeVideo = () => {
  exitFrameWebFullscreen();
  exitFullscreen();
  systemProperty.PlayingMovie = {};
  view.videoUrl = null;
  cancelAnimationFrame(animationFrame);
  emmits('close');
};

const exitFullscreen = () => {
  $q.fullscreen.exit();
  view.videoFullscreen = false;
  videoWidthDebounce();
};

// 进入web全屏
const frameWebFullscreen = () => {
  const videoFrame = document.getElementById('videoFrame');
  if (videoFrame) {
    // 调整全屏状态下的尺寸
    if (!view.webFullScreen) {
      systemProperty.pictureInPictureVideoWidthFullBefore =
        systemProperty.pictureInPictureVideoWidth;
      systemProperty.pictureInPictureVideoOffsetFullBefore =
        systemProperty.pictureInPictureVideoOffset;
      systemProperty.pictureInPictureVideoOffset = [8, 8];
      view.webFullScreen = true;
    } else {
      exitFrameWebFullscreen();
    }
  }
  videoWidthDebounce();
};
// 推出web全屏
const exitFrameWebFullscreen = () => {
  const videoFrame = document.getElementById('videoFrame');

  if (videoFrame) {
    // 调整全屏状态下的尺寸
    systemProperty.pictureInPictureVideoWidth =
      systemProperty.pictureInPictureVideoWidthFullBefore;
    systemProperty.pictureInPictureVideoOffset =
      systemProperty.pictureInPictureVideoOffsetFullBefore;
    view.webFullScreen = false;
  }
};

const touchVideo = async (ev) => {
  // 新增右滑手势
  if (ev.evt.srcElement.id && ev.evt.srcElement.id === 'hoverVideoID') {
    if (Math.abs(ev.delta.x) > Math.abs(ev.delta.y)) {
      hoverPlayer.value.forwardTime((ev.delta.x % 50) * 2);
    }
  }
};
const volumeUp = (val) => {
  if (view.currentData && view.videoUrl) {
    systemProperty.videoOptions.volume = hoverPlayer.value.volumeUp(val);
  }
};

const volumeUpdate = (volume) => {
  if (view.currentData && view.videoUrl) {
    systemProperty.videoOptions.volume = hoverPlayer.value.volumeUpdate(volume);
  }
};

const openVideo = async (params) => {
  // console.log('openVideo', params);
  const { item, queryParam, webFullScreen } = params;
  if (!item) {
    return;
  }
  view.queryParam = queryParam;
  view.currentData = item;
  systemProperty.PlayingMovie = item;
  view.videoUrl = getFileStream(item.Id);

  const videoElement = document.getElementById('hoverVideoID');
  const videoLocation = systemProperty.getPlayerLocation(item.Id);
  if (videoElement) {
    videoElement.volume = systemProperty.videoOptions?.volume;
    videoElement.loop = systemProperty.videoOptions?.loop;

    // 监听 loadedmetadata，确保视频元数据加载完成后再设置 currentTime
    videoElement.addEventListener('loadedmetadata', () => {
      if (
        videoLocation &&
        systemProperty.playerReLocation &&
        videoElement.duration - videoLocation > 60
      ) {
        videoElement.currentTime = videoLocation;
      }
      videoElement.focus();
    });

    // 取消旧的 RAF 循环，防止重复调用 openVideo 导致的 RAF 泄漏
    cancelAnimationFrame(animationFrame);
    showProgress = () => {
      if (view.videoUrl) {
        const progress = parseInt(
          (videoElement?.currentTime / videoElement?.duration) * 100
        );
        if (!isNaN(progress) && progress >= 0 && progress <= 100) {
          // 添加数据验证
          view.videoduration = videoElement?.duration;
          view.progress = progress;
        }

        view.currentVideoTime = parseTime(videoElement?.currentTime);
        if (videoElement?.currentTime > 60 && systemProperty.playerReLocation) {
          systemProperty.addPlayerLocation(item.Id, videoElement?.currentTime);
        }
        animationFrame = requestAnimationFrame(showProgress);
      } else {
        cancelAnimationFrame(animationFrame);
        cancelAnimationFrame(showProgress);
      }
    };
    showProgress();
  }
  if (webFullScreen) {
    frameWebFullscreen();
  }
  videoWidthDebounce();

  emmits('chooseData', item);
};

const setVideoWidth = () => {
  const time1 = setTimeout(() => {
    const videoElement = document.getElementById('hoverVideoID');
    view.videoWidth = videoElement?.clientWidth;
    clearTimeout(time1);
  }, 400);
};

const videoWidthDebounce = useDebounceFn(setVideoWidth, 500);

const restartVideo = () => {
  view.videoUrl = null;
  const time1 = setTimeout(() => {
    view.videoUrl = getFileStream(view.currentData.Id);
    const videoLocation = systemProperty.getPlayerLocation(view.currentData.Id);
    if (videoLocation) {
      hoverPlayer.value.timeUpdate(videoLocation);
    }
    clearTimeout(time1);
  }, 1000);
};

const isPip = ref(false);
const requestPiP = () => {
  if (!isPip.value) {
    hoverPlayer.value.requestPictureInPicture();
  } else {
    document.exitPictureInPicture();
  }
  isPip.value = !isPip.value;
};

const emmits = defineEmits(['prevOne', 'nextOne', 'close', 'chooseData']);

const prevOne = async () => {
  view.videoUrl = null;
  emmits('prevOne');
};

const nextOne = async () => {
  view.videoUrl = null;
  emmits('nextOne');
};

let wheelTimer = null;
const onWheel = (e) => {
  if (!view.videoUrl) return;
  if (wheelTimer) return;
  if (e.deltaY < 0) {
    prevOne();
    $q.notify({ type: 'info', message: '上一集', position: 'top', timeout: 800 });
  } else {
    nextOne();
    $q.notify({ type: 'info', message: '下一集', position: 'top', timeout: 800 });
  }
  wheelTimer = setTimeout(() => { wheelTimer = null; }, 600);
};

onMounted(() => {
  systemProperty.PlayingMovie = {};
  // PiP 事件监听必须在 DOM 就绪后注册
  document
    .getElementById('hoverVideoID')
    ?.addEventListener('leavepictureinpicture', () => {
      isPip.value = false;
      hoverPlayer.value.play();
    });
});

defineExpose({
  openVideo,
  closeVideo,
});
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

.chip-tag {
  margin-left: 0;
  padding: 0 4px;
  font-weight: 500;
  width: fit-content;
  background-color: rgba(0, 0, 0, 0.2);
}
</style>
