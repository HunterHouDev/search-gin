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
        :poster="view.currentData?.JpgUrl"
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
      <!-- 视频链接 / 分片下载：不再弹出播放列表。
           面板常驻（v-show），关掉浮层不会中断正在播放的链接 -->
      <div v-show="view.showLinkPanel" class="link-panel-mask">
        <div class="link-panel-wrap">
          <div class="row items-center q-px-sm q-py-xs">
            <span class="text-subtitle2">视频链接 / 分片下载</span>
            <q-space />
            <q-btn
              flat
              dense
              round
              color="red"
              icon="close"
              @click="view.showLinkPanel = false"
            />
          </div>
          <LinkSourcePanel
            ref="linkPanelRef"
            :get-video-el="getVideoEl"
            :get-volume="getPlayerVolume"
            :on-play="handleLinkPlay"
          />
        </div>
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
            @click="minimal ? null : view.showDrawer = !view.showDrawer"
          >
            <q-tooltip class="bg-white text-primary">缩放</q-tooltip>
          </q-btn>
          <q-btn
            dense
            flat
            v-if="!view.webFullScreen && !minimal"
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
            v-if="!isMobile() && !view.videoFullscreen && !view.webFullScreen && !minimal"
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
            v-if="!minimal"
            icon="ti-plus"
            @click="view.showLinkPanel = true"
            v-touch-pan.prevent.mouse="moveFab"
          >
            <q-tooltip class="bg-white text-primary"
              >在线播放/下载</q-tooltip
            >
          </q-btn>
        </div>
        <div
          v-if="!minimal"
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
            <!-- 链接 / 下载任务播放时没有索引信息（SizeStr / Author / Code 为空）：
                 空值不渲染，避免留一排空白 chip -->
            <q-chip
              v-if="view.currentData.SizeStr"
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
              v-if="view.showExtraInfo && view.currentData.Author"
              square
              dense
              text-color="red"
              class="chip-tag"
              v-touch-pan.prevent.mouse="moveFab"
            >
              <span @click="copyText(view.currentData.Author)">
                {{ view.currentData.Author?.substring(0, 12) }}
              </span>
            </q-chip>
            <q-chip
              v-if="view.showExtraInfo && view.currentData.Code"
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
            v-if="!view.videoFullscreen && !isMobile() && !minimal"
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
            v-if="!minimal"
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
            v-if="view.showExtraInfo && !minimal"
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
            v-if="!minimal"
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
            v-if="view.showExtraInfo && !isMobile() && !minimal"
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
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue';
import { useSystemProperty } from 'stores/System';
import { isMobile } from 'src/boot/platform';


import { VideoClass } from 'components/utils/video';
import { parseTime, formatTitle } from 'components/utils';
import { DeleteFile, FindFileInfo } from 'components/api/searchAPI';
import { onKeyStroke, useDebounceFn, useClipboard } from '@vueuse/core';

import PlayerSetting from 'components/PlayerSetting.vue';
import VideoCutParam from 'components/VideoCutParam.vue';
import EditVideoTag from 'components/EditVideoTag.vue';
import VideoTimeBar from 'components/VideoTimeBar.vue';
import SearchPanel from 'components/SearchPanel.vue';
import FileEdit from 'src/pages/file/components/FileEditDialog.vue';
import VideoVolumnBtn from 'components/VideoVolumnBtn.vue';
import DeleteBtn from 'components/DeleteBtn.vue';
import LinkSourcePanel from 'src/components/LinkSourcePanel.vue';

defineProps({
  minimal: { type: Boolean, default: false },
});

const source = ref('Hello');
const { copy } = useClipboard({ source });

const $q = useQuasar();

const hoverPlayer = ref(null);

let animationFrame = null;
let loadedmetadataHandler = null;
const systemProperty = useSystemProperty();
let showProgress = null;
const searchPanelRef = ref(null);
const fileEditRef = ref(null);
const view = reactive({
  showDrawer: false,
  // 视频链接 / 分片下载浮层（播放走当前小窗播放器）
  showLinkPanel: false,
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
};

const onSearchPanelEdit = (item) => {
  fileEditRef.value?.open(item);
};

const onSearchPanelDelete = async (item) => {
  await DeleteFile(item);
  searchPanelRef.value?.fetchSearch();
};

// ── 视频链接 / 分片下载面板（播放落到当前小窗播放器） ──────────────────────────
const linkPanelRef = ref(null);

/** 当前播放用的 video 元素（小窗播放器本身） */
const getVideoEl = () => document.getElementById('hoverVideoID');

/** 小窗的当前音量，供链接面板起播时沿用 */
const getPlayerVolume = () =>
  getVideoEl()?.volume ?? systemProperty.videoOptions?.volume ?? 0.8;

/**
 * 切换播放源前的收尾：停掉面板持有的 hls 实例，并清空元素上的 src。
 *
 * hls.js 会把 MediaSource 直接写到 video 元素的 src 上（Vue 并不知情），
 * 之后再只改 view.videoUrl 会出问题：
 *   1) 值与上次相同 → Vue 判定无变化不会重新渲染，同一个地址再点播放毫无反应；
 *   2) 值不同虽然会渲染，但残留的 MediaSource 会和新源互相打架。
 * 所以每次切换都先在这里归零，必要时再由调用方强制对齐元素属性。
 */
const resetVideoElement = () => {
  linkPanelRef.value?.destroyHls?.();
  const el = getVideoEl();
  if (!el) return;
  try {
    el.pause();
  } catch {
    /* 忽略暂停异常 */
  }
  el.removeAttribute('src');
  el.load();
};

/**
 * 链接面板的播放请求：一律用当前播放器播放。
 *
 * src 为空表示由面板的 hls.js 实例接管（它会自己写 MediaSource），
 * 此时不能碰元素——置空 view.videoUrl 会让整个小窗被 v-show 隐藏。
 */
const handleLinkPlay = async (src, name) => {
  if (name) {
    // 下载任务 / 链接不是索引里的文件：只保留任务文件名，
    // SizeStr / Author / Code 一律不带（否则会残留上一个文件的元信息）
    view.currentData = { Name: name };
  }
  // 播放一律落到小窗播放器，浮层正挡在前面：起播即收起。
  // 面板是 v-show 常驻，收起不会中断下载，也不会停掉 hls.js 的播放
  view.showLinkPanel = false;
  if (!src) return;

  resetVideoElement();
  view.videoUrl = src;
  await nextTick();
  const el = getVideoEl();
  if (!el) return;
  // 元素上的 src 可能还留着上一次的 MediaSource，强制对齐
  if (el.getAttribute('src') !== src) {
    el.src = src;
  }
  el.volume = systemProperty.videoOptions?.volume ?? el.volume;
  el.play().catch(() => {
    // 自动播放被浏览器拦下时静默：控件已展开，用户点一下即可
  });
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
  // 播放器都没了，链接面板跟着收起（面板常驻，这里只是隐藏）
  view.showLinkPanel = false;
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
  // 获取最新 token 刷新 URL，防止搜索时的 token 过期
  let streamUrl = item.StreamUrl;
  try {
    const fresh = await FindFileInfo(item);
    if (fresh?.StreamUrl) {
      streamUrl = fresh.StreamUrl;
      item.JpgUrl = fresh.JpgUrl;
      item.PngUrl = fresh.PngUrl;
    }
  } catch (_) { /* 刷新失败时继续使用原有的 URL */ }
  item.StreamUrl = streamUrl;
  view.queryParam = queryParam;
  view.currentData = item;
  systemProperty.PlayingMovie = item;
  // 上一个源可能是分片链接（hls.js 直接写在元素上的 MediaSource）：先归零再换源，
  // 否则再播同一个文件时 Vue 认为 src 没变，点了播放也不切换
  resetVideoElement();
  view.videoUrl = streamUrl;

  const videoElement = document.getElementById('hoverVideoID');
  const videoLocation = systemProperty.getPlayerLocation(item.Id);
  if (videoElement) {
    if (videoElement.getAttribute('src') !== streamUrl) {
      videoElement.src = streamUrl;
    }
    videoElement.volume = systemProperty.videoOptions?.volume;
    videoElement.loop = systemProperty.videoOptions?.loop;

    // 监听 loadedmetadata，确保视频元数据加载完成后再设置 currentTime
    if (loadedmetadataHandler) {
      videoElement.removeEventListener('loadedmetadata', loadedmetadataHandler);
    }
    loadedmetadataHandler = () => {
      if (
        videoLocation &&
        systemProperty.playerReLocation &&
        videoElement.duration - videoLocation > 60
      ) {
        videoElement.currentTime = videoLocation;
      }
      videoElement.focus();
    };
    videoElement.addEventListener('loadedmetadata', loadedmetadataHandler);

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
  resetVideoElement();
  view.videoUrl = null;
  const time1 = setTimeout(() => {
    view.videoUrl = view.currentData.StreamUrl;
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
  hoverPlayer.value = new VideoClass('hoverVideoID');
  // PiP 事件监听必须在 DOM 就绪后注册
  const pipHandler = () => {
    isPip.value = false;
    hoverPlayer.value.play();
  };
  document
    .getElementById('hoverVideoID')
    ?.addEventListener('leavepictureinpicture', pipHandler);
  onUnmounted(() => {
    document
      .getElementById('hoverVideoID')
      ?.removeEventListener('leavepictureinpicture', pipHandler);
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

/* 视频链接 / 分片下载浮层：盖在整个页面上，内容超出时内部滚动 */
.link-panel-mask {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  justify-content: center;
  align-items: center;
  background: rgba(0, 0, 0, 0.6);
}

.link-panel-wrap {
  width: min(92vw, 1100px);
  max-height: 80vh;
  overflow: auto;
  background: rgba(12, 12, 24, 0.95);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 12px;
}
</style>
