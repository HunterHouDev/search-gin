<template>
  <div
    class="immersive-container"
    @mousemove="onMouseMove"
    @click.self="togglePlay"
    @dblclick="toggleFullscreen"
  >
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>

    <!-- 左上角返回按钮 -->
    <q-btn
      flat
      color="white"
      icon="arrow_back"
      class="fixed-top-left-btn"
      @click.stop="goBack"
    >
      <q-tooltip class="bg-dark text-white">返回</q-tooltip>
    </q-btn>

    <!-- 右上角搜索按钮 -->
    <q-btn
      flat
      glossy
      :color="showMenuControls ? 'white' : 'blue'"
      :icon="showMenuControls ? 'close' : 'search'"
      class="fixed-top-right-btn"
      @click="showMenuInfo"
    >
      <q-tooltip class="bg-dark text-white">搜索</q-tooltip>
    </q-btn>

    <!-- 播放列表轮播 -->
    <transition name="slide-down">
      <div v-show="showMenuControls">
        <div
          class="carousel-banner"
          @mouseenter="searchDialog = true"
          @mouseleave="handleBannerMouseLeave"
        >
          <q-btn
            flat
            round
            dense
            color="white"
            icon="chevron_left"
            class="carousel-arrow carousel-arrow-left"
            @click.stop="prevPage"
          />
          <div
            class="carousel-track"
            ref="carouselTrack"
            v-show="playlist.length > 0"
          >
            <div
              v-for="(item, index) in playlist"
              :key="item.Id || index"
              class="carousel-item"
              :class="{ 'carousel-item-active': index === currentIndex }"
              @click.stop="switchToItem(index)"
            >
              <q-img
                :src="item.CoverUrl || getPng(item.Id)"
                fit="cover"
                class="carousel-thumb"
                :ratio="3 / 4"
              >
                <template v-slot:error>
                  <div class="carousel-thumb-placeholder">
                    <q-icon name="movie" size="20px" color="grey-5" />
                  </div>
                </template>
              </q-img>
              <div class="carousel-item-label">
                {{ item.Title || item.Name || `#${index + 1}` }}
              </div>
              <div
                class="carousel-item-active-indicator"
                v-if="index === currentIndex"
              >
                <q-icon name="play_arrow" size="12px" color="white" />
              </div>
            </div>
          </div>
          <q-btn
            flat
            round
            dense
            color="white"
            icon="chevron_right"
            class="carousel-arrow carousel-arrow-right"
            @click.stop="nextPage"
          />
        </div>
        <!-- 搜索面板 - 合并到轮播中 -->
        <div class="search-panel" v-show="searchDialog" @click.stop @mouseleave="searchDialog = false">
          <!-- 头部 -->
          <div class="search-panel-header">
            <div class="search-panel-title">
              <q-input
                v-model="searchParams.Keyword"
                placeholder="输入关键词..."
                dark
                dense
                outlined
                color="indigo-4"
                class="search-input"
                @keyup.enter="fetchSearch"
                @change="fetchSearch"
              >
                <template v-slot:prepend>
                  <q-icon name="manage_search" color="indigo-4" size="18px" />
                </template>
                <template v-slot:append v-if="searchParams.Keyword">
                  <q-btn
                    flat
                    round
                    dense
                    icon="clear"
                    color="grey-5"
                    size="xs"
                    @click="
                      searchParams.Keyword = '';
                      fetchSearch();
                    "
                  />
                </template>
              </q-input>
            </div>
            <q-btn
              flat
              round
              dense
              color="grey-4"
              icon="close"
              @click="searchDialog = false"
            />
          </div>

          <!-- 搜索条件 -->
          <div class="search-conditions">
            <div class="filter-item">
              <div class="filter-row">
                <span class="filter-label">类型</span>
                <q-btn-toggle
                  v-model="searchParams.MovieType"
                  :options="MovieTypeSelects"
                  no-caps
                  glossy
                  toggle-color="indigo-6"
                  color="dark"
                  text-color="grey-4"
                  @update:model-value="fetchSearch"
                />
              </div>

              <div class="filter-row">
                <span class="filter-label">排序</span>
                <q-btn-toggle
                  v-model="searchParams.SortField"
                  :options="FieldEnum"
                  no-caps
                  glossy
                  toggle-color="indigo-6"
                  color="dark"
                  text-color="grey-4"
                  @update:model-value="fetchSearch"
                />
              </div>

              <div class="filter-row">
                <span class="filter-label">顺序</span>
                <q-btn-toggle
                  v-model="searchParams.SortType"
                  :options="DescEnum"
                  no-caps
                  glossy
                  toggle-color="indigo-6"
                  color="dark"
                  text-color="grey-4"
                  @update:model-value="fetchSearch"
                />
              </div>
            </div>
          </div>

          <!-- 搜索结果 -->
          <div class="search-results" ref="searchResultsRef">
            <div v-if="searchLoading" class="search-loading">
              <q-spinner-dots size="40px" color="indigo-4" />
              <p class="text-grey-5 q-mt-sm text-caption">加载中...</p>
            </div>

            <template
              v-else-if="searchResults.Data && searchResults.Data.length > 0"
            >
              <div class="search-cards">
                <div
                  v-for="item in searchResults.Data"
                  :key="item.Id"
                  class="search-card"
                >
                  <div class="search-card-thumb">
                    <q-img
                      :src="getPng(item.Id)"
                      fit="cover"
                      class="search-card-img"
                      :ratio="3 / 4"
                      @click="playFromSearch(item)"
                    >
                      <template v-slot:error>
                        <div class="search-card-placeholder">
                          <q-icon name="movie" color="grey-6" size="28px" />
                        </div>
                      </template>
                    </q-img>
                    <div class="search-card-play-overlay">
                      <q-icon
                        name="play_circle_filled"
                        size="28px"
                        color="white"
                         @click="playFromSearch(item)"
                      />
                    </div>
                  </div>

                  <div class="search-card-info">
                    <div class="search-card-title">
                      {{ formatTitle(item.Title, 24) }}
                    </div>
                    <div class="search-card-tags">
                      <span
                        class="tag tag-actress"
                        v-if="item.Actress"
                        @click="fetchKeyword(item.Actress)"
                        >{{ item.Actress }}</span
                      >
                      <span
                        class="tag tag-code"
                        v-if="item.Code"
                        @click="fetchKeyword(item.Code)"
                        >{{ item.Code }}</span
                      >
                    </div>
                    <div class="search-card-meta">
                      <span class="meta-item">
                        <q-icon name="data_usage" size="10px" />
                        {{ humanStorageSize(item.Size) }}
                      </span>
                      <span class="meta-item">
                        <q-icon name="schedule" size="10px" />
                        {{ getTimeAgo(item.MTime) }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </template>

            <div v-else class="search-empty">
              <q-icon name="search_off" size="48px" color="grey-7" />
              <p class="text-grey-6 q-mt-sm">暂无搜索结果</p>
            </div>
          </div>

          <!-- 分页 -->
          <div class="search-pagination" v-if="searchResults.TotalPage > 1">
            <q-btn
              flat
              dense
              round
              color="indigo-4"
              icon="chevron_left"
              size="sm"
              :disable="searchParams.Page <= 1"
              @click="changePage(-1)"
            />
            <div class="pagination-info">
              <span class="page-current">{{ searchParams.Page }}</span>
              <span class="page-sep">/</span>
              <span class="page-total">{{ searchResults.TotalPage }}</span>
            </div>
            <q-btn
              flat
              dense
              round
              color="indigo-4"
              icon="chevron_right"
              size="sm"
              :disable="searchParams.Page >= searchResults.TotalPage"
              @click="changePage(1)"
            />
          </div>
        </div>
      </div>
    </transition>

    <!-- 视频区域 -->
    <div class="video-wrapper" v-show="videoLoaded" >
      <video
        ref="videoRef"
        id="immersiveVideo"
        :src="currentVideoSrc"
        :poster="currentPoster"
        preload="auto"
        autoplay
        playsinline
        crossorigin="anonymous"
        @timeupdate="onTimeUpdate"
        @loadedmetadata="onMetadataLoaded"
        @play="onPlay"
        @pause="onPause"
        @ended="onEnded"
        @waiting="onWaiting"
        @canplay="onCanPlay"
        @error="onVideoError"
      ></video>
      <!-- 缓冲 loading 遮罩 -->
      <transition name="fade">
        <div class="video-buffering" v-if="isBuffering">
          <q-spinner-gears size="56px" color="indigo-3" />
        </div>
      </transition>
    </div>

    <!-- 拖拽上传区域 -->
    <transition name="fade">
      <div
        v-if="!videoLoaded && !torrentLoading"
        class="drop-zone"
        :class="{ 'drop-zone-active': isDragOver }"
        @dragover.prevent="isDragOver = true"
        @dragleave="isDragOver = false"
        @drop="handleDrop"
      >
        <div class="drop-content">
          <div class="drop-icon-wrapper">
            <q-icon name="movie" size="56px" color="indigo-3" />
            <div class="drop-icon-ring"></div>
          </div>
          <p class="drop-title">拖拽视频文件到此处</p>
          <p class="drop-subtitle">支持 MP4、MKV、AVI、MOV 等格式</p>
        </div>
      </div>
    </transition>

    <!-- 磁力链输入区 -->
    <transition name="slide-up">
      <div v-if="!videoLoaded && !torrentLoading" class="magnet-input-area">
        <div
          class="magnet-input-wrapper"
          :class="{ 'magnet-focused': magnetFocused }"
        >
          <q-icon
            name="link"
            color="indigo-4"
            size="20px"
            class="magnet-icon"
          />
          <q-input
            v-model="magnetURI"
            placeholder="粘贴磁力链 magnet:?xt=urn:btih:..."
            dark
            dense
            borderless
            class="magnet-input"
            @keyup.enter="submitMagnet"
            @focus="magnetFocused = true"
            @blur="magnetFocused = false"
          />
          <q-btn
            flat
            round
            dense
            color="indigo-4"
            icon="play_circle_filled"
            size="md"
            @click="submitMagnet"
            :disable="!magnetURI.trim()"
            class="magnet-submit-btn"
          >
            <q-tooltip class="bg-dark text-white">播放磁力链</q-tooltip>
          </q-btn>
        </div>
      </div>
    </transition>

    <!-- 磁力链文件选择 -->
    <transition name="fade">
      <div v-if="showTorrentFiles" class="torrent-files-dialog">
        <div class="torrent-files-card">
          <div class="torrent-files-header">
            <q-icon name="folder_open" size="24px" color="indigo-4" />
            <span class="torrent-files-title">{{ torrentName }}</span>
            <span class="torrent-files-hint">选择要播放的文件</span>
          </div>
          <div class="torrent-files-list">
            <div
              v-for="(file, index) in torrentFiles"
              :key="index"
              class="torrent-file-item"
              :class="{ 'torrent-file-selected': selectedTorrentFile === file.path }"
              @click="selectTorrentFile(file)"
            >
              <q-icon
                :name="getFileIcon(file.name)"
                size="20px"
                class="torrent-file-icon"
              />
              <div class="torrent-file-info">
                <span class="torrent-file-name">{{ file.name }}</span>
                <span class="torrent-file-size">{{ humanStorageSize(file.length) }}</span>
              </div>
              <q-icon
                v-if="selectedTorrentFile === file.path"
                name="play_circle_filled"
                size="24px"
                color="indigo-4"
              />
            </div>
          </div>
          <div class="torrent-files-actions">
            <q-btn
              flat
              color="grey-5"
              label="取消"
              @click="cancelTorrent"
            />
            <q-btn
              unelevated
              color="indigo-6"
              label="播放选中文件"
              icon="play_arrow"
              :disable="!selectedTorrentFile"
              @click="playSelectedTorrentFile"
            />
          </div>
        </div>
      </div>
    </transition>

    <!-- 种子加载中 -->
    <transition name="fade">
      <div v-if="torrentLoading && !showTorrentFiles" class="torrent-loading">
        <div class="torrent-loading-card">
          <div class="torrent-spinner">
            <q-spinner-gears size="64px" color="indigo-4" />
          </div>
          <p class="torrent-name">{{ torrentName }}</p>
          <div class="torrent-progress-wrap">
            <q-linear-progress
              :value="torrentProgress / 100"
              color="indigo-5"
              track-color="grey-9"
              size="6px"
              rounded
              class="q-mb-sm"
            />
            <div class="torrent-stats">
              <span class="torrent-percent"
                >{{ torrentProgress.toFixed(1) }}%</span
              >
              <span class="torrent-state">{{ torrentState }}</span>
              <span class="torrent-peers" v-if="torrentPeers > 0">
                <q-icon name="people" size="12px" />
                {{ torrentPeers }}
              </span>
            </div>
          </div>
          <q-btn
            unelevated
            color="red-9"
            text-color="red-3"
            label="取消下载"
            icon="cancel"
            size="sm"
            rounded
            @click="cancelTorrent"
            class="q-mt-md"
          />
        </div>
      </div>
    </transition>

    <!-- 底部控制面板 -->
    <transition name="slide-up">
      <div
        class="glass-panel"
        v-show="!controlsHidden || !isPlaying"
        @mouseenter="showControls"
        @mouseleave="startHideTimer"
        @click.stop
      >
        <!-- 进度条 -->
        <div
          class="progress-container"
          ref="progressBar"
          @mousedown="startSeek"
          @mousemove="onProgressHover"
          @mouseleave="hideTooltip"
        >
          <div class="progress-track">
            <!-- 缓冲进度 -->
            <div
              class="progress-buffered"
              :style="{ width: bufferedPercent + '%' }"
            ></div>
            <!-- 播放进度 -->
            <div
              class="progress-fill"
              :style="{ width: progressPercent + '%' }"
            >
              <div class="progress-glow"></div>
            </div>
            <!-- 拖拽手柄 -->
            <div
              class="progress-thumb"
              :style="{ left: progressPercent + '%' }"
              :class="{ seeking: isSeeking }"
            ></div>
          </div>
          <!-- 时间悬浮提示 -->
          <div
            class="progress-tooltip"
            v-if="hoverTime !== null"
            :style="{ left: hoverX + 'px' }"
          >
            {{ hoverTime }}
          </div>
        </div>

        <!-- 控制按钮行 -->
        <div class="control-buttons">
          <!-- 左侧：播放控制 + 时间 -->
          <div class="ctrl-left">
            <q-btn
              flat
              round
              color="white"
              size="sm"
              icon="skip_previous"
              @click="prevItem"
            >
              <q-tooltip class="bg-dark">上一个</q-tooltip>
            </q-btn>
            <q-btn
              flat
              round
              :color="isPlaying ? 'indigo-3' : 'white'"
              size="md"
              :icon="isPlaying ? 'pause_circle' : 'play_circle'"
              class="play-btn"
              @click="togglePlay"
            />
            <q-btn
              flat
              round
              v-if="isPlaying"
              color="white"
              size="md"
              icon="stop"
              class="play-btn"
              @click="stopPlay"
            />

            <q-btn
              flat
              round
              color="white"
              size="sm"
              icon="skip_next"
              @click="nextItem"
            >
              <q-tooltip class="bg-dark">下一个</q-tooltip>
            </q-btn>
            <div class="time-display">
              <span class="time-current">{{ currentTime }}</span>
              <span class="time-sep">/</span>
              <span class="time-total">{{ duration }}</span>
            </div>
            <span class="top-title" v-if="currentVideoName && videoLoaded">
              {{ currentVideoName }}
            </span>
          </div>

          <q-space />

          <!-- 右侧：音量 + 全屏 -->
          <div class="ctrl-right">
            <div
              class="volume-group"
              @mouseenter="showVolume = true"
              @mouseleave="showVolume = false"
            >
              <q-btn
                flat
                round
                color="white"
                size="sm"
                @click="toggleMute"
                :icon="volumeIcon"
              />
              <transition name="fade">
                <q-slider
                  v-show="showVolume"
                  v-model="volume"
                  :min="0"
                  :max="1"
                  :step="0.01"
                  color="indigo-4"
                  track-color="grey-8"
                  class="volume-slider"
                  @update:model-value="setVolume"
                />
              </transition>
            </div>
            <q-btn
              flat
              round
              color="white"
              size="sm"
              :icon="isFullscreen ? 'fullscreen_exit' : 'fullscreen'"
              @click="toggleFullscreen"
            >
              <q-tooltip class="bg-dark">{{
                isFullscreen ? '退出全屏' : '全屏'
              }}</q-tooltip>
            </q-btn>
          </div>
        </div>
      </div>
    </transition>

    <!-- 下载管理器悬浮按钮 -->
    <q-btn
      v-if="activeDownloads.length > 0"
      round
      color="indigo-6"
      icon="download"
      class="download-fab"
      @click="showDownloadManager = true"
    >
      <q-badge color="red" floating rounded>{{ activeDownloads.length }}</q-badge>
      <q-tooltip>下载管理</q-tooltip>
    </q-btn>

    <!-- 下载管理器弹窗 -->
    <q-dialog v-model="showDownloadManager" position="right" full-height>
        <q-card class="download-manager-card">
          <q-card-section class="download-manager-header">
            <div class="download-manager-title">
              <q-icon name="download" size="24px" />
              <span>下载管理器</span>
            </div>
            <q-btn flat round dense icon="close" @click="showDownloadManager = false" />
          </q-card-section>

          <q-card-section class="download-manager-content">
            <div v-if="activeDownloads.length === 0" class="download-empty">
              <q-icon name="cloud_download" size="48px" color="grey-6" />
              <p>暂无下载任务</p>
            </div>

            <div v-else class="download-list">
              <div
                v-for="task in activeDownloads"
                :key="task.infoHash"
                class="download-item"
                :class="{ 'download-item-playing': task.infoHash === currentInfoHash && videoLoaded }"
              >
                <div class="download-item-info">
                  <div class="download-item-name">{{ task.name }}</div>
                  <div class="download-item-meta">
                    <span class="download-item-file" v-if="task.fileName">{{ task.fileName }}</span>
                    <span class="download-item-state" :class="'state-' + task.state">{{ task.state }}</span>
                    <span class="download-item-percent">{{ task.progress.toFixed(1) }}%</span>
                  </div>
                  <q-linear-progress
                    :value="task.progress / 100"
                    color="indigo-5"
                    track-color="grey-9"
                    size="4px"
                    rounded
                    class="q-mt-xs"
                  />
                </div>
                <div class="download-item-actions">
                  <q-btn
                    flat
                    round
                    dense
                    color="green"
                    icon="play_arrow"
                    size="sm"
                    :disable="task.progress < 1"
                    @click="playDownloadTask(task)"
                  >
                    <q-tooltip>播放</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    round
                    dense
                    color="blue"
                    icon="folder_open"
                    size="sm"
                    @click="openDownloadFolder(task)"
                  >
                    <q-tooltip>打开文件夹</q-tooltip>
                  </q-btn>
                  <q-btn
                    flat
                    round
                    dense
                    color="red"
                    icon="close"
                    size="sm"
                    @click="removeDownloadTask(task)"
                  >
                    <q-tooltip>删除</q-tooltip>
                  </q-btn>
                </div>
              </div>
        </div>
      </q-card-section>
    </q-card>
  </q-dialog>
</div>
</template>

<script setup>
import {
  computed,
  onMounted,
  onUnmounted,
  ref,
  reactive,
  watch,
  nextTick,
} from 'vue';
import { format, useQuasar } from 'quasar';
import { useRouter } from 'vue-router';
import axios from 'axios';
import { SearchAPI } from 'components/api/searchAPI';
import { getPng, getFileStream } from 'components/utils/images';
import {
  MovieTypeSelects,
  FieldEnum,
  DescEnum,
  formatTitle,
} from 'components/utils';
import { useSystemProperty } from 'stores/System';

const $q = useQuasar();
const router = useRouter();
const { humanStorageSize } = format;

// ── System Store ───────────────────────────────────────────────────────────────
const systemProperty = useSystemProperty();

// ── DOM refs ─────────────────────────────────────────────────────────────────
const videoRef = ref(null);
const particleCanvas = ref(null);
const progressBar = ref(null);
const carouselTrack = ref(null);
const searchResultsRef = ref(null);
const showMenuControls = ref(false);

// ── 播放状态 ──────────────────────────────────────────────────────────────────
const currentVideoSrc = ref('');
const currentPoster = ref('');
const currentVideoName = ref('');
const videoLoaded = ref(false);
const isPlaying = ref(false);
const isFullscreen = ref(false);
const isBuffering = ref(false);
const currentTime = ref('00:00:00');
const duration = ref('00:00:00');
// 使用 System store 中的音量
const volume = computed({
  get: () => systemProperty.videoOptions.volume,
  set: (val) => {
    systemProperty.videoOptions.volume = val;
    if (videoRef.value) {
      videoRef.value.volume = val;
    }
  },
});
const currentTimeSeconds = ref(0);
const durationSeconds = ref(0);
const bufferedSeconds = ref(0);
const controlsHidden = ref(false);

// ── 进度条拖拽 ────────────────────────────────────────────────────────────────
const isSeeking = ref(false);
const hoverTime = ref(null);
const hoverX = ref(0);

// ── 音量控制 ──────────────────────────────────────────────────────────────────
const showVolume = ref(false);

// ── 磁力链 ────────────────────────────────────────────────────────────────────
const magnetURI = ref('');
const magnetFocused = ref(false);
const isDragOver = ref(false);
const torrentLoading = ref(false);
const torrentName = ref('');
const torrentProgress = ref(0);
const torrentState = ref('');
const torrentPeers = ref(0);
const currentInfoHash = ref('');
const torrentFiles = ref([]);
const showTorrentFiles = ref(false);
const selectedTorrentFile = ref(null);
const showDownloadManager = ref(false);
const activeDownloads = ref([]);
let torrentPollTimer = null;

// ── 播放列表 ──────────────────────────────────────────────────────────────────
const playlist = ref([]);
const currentIndex = ref(-1);

// ── 搜索 ──────────────────────────────────────────────────────────────────────
const searchDialog = ref(false);
const searchLoading = ref(false);
const searchResults = reactive({ Data: [], TotalPage: 0, ResultSize: '' });
const searchParams = reactive({
  Keyword: '',
  MovieType: '',
  SortField: 'MTime',
  SortType: 'desc',
  OnlyRepeat: false,
  Page: 1,
  PageSize: 20,
});

// ── 粒子 / 音频 ───────────────────────────────────────────────────────────────
let audioContext = null;
let analyser = null;
let source = null;
let connectedVideoElement = null; // 记录已连接的视频元素
let animationFrameId = null;
let particles = [];
let hideControlsTimer = null;

// ── 计算属性 ──────────────────────────────────────────────────────────────────
const progressPercent = computed(() => {
  if (durationSeconds.value === 0) return 0;
  return (currentTimeSeconds.value / durationSeconds.value) * 100;
});

const bufferedPercent = computed(() => {
  if (durationSeconds.value === 0) return 0;
  return (bufferedSeconds.value / durationSeconds.value) * 100;
});

const volumeIcon = computed(() => {
  if (volume.value === 0) return 'volume_off';
  if (volume.value < 0.3) return 'volume_mute';
  if (volume.value < 0.7) return 'volume_down';
  return 'volume_up';
});

// ── 导航 ──────────────────────────────────────────────────────────────────────
function goBack() {
  router.back();
}

function showMenuInfo() {
  showMenuControls.value = !showMenuControls.value;
  if( playlist.value.length == 0) {
    fetchSearch();
  }
}

function handleBannerMouseLeave(e) {
  if (e.relatedTarget && e.relatedTarget.closest('.search-panel')) {
    return;
  }
  searchDialog.value = false;
}

// ── 播放列表操作 ──────────────────────────────────────────────────────────────

function switchToItem(index) {
  if (index < 0 || index >= playlist.value.length) return;
  currentIndex.value = index;
  const item = playlist.value[index];
  const src = item.TorrentStream || getFileStream(item.Id);
  loadVideo(
    src,
    item.Title || item.Name || item.Code || `#${index + 1}`,
    getPng(item.Id)
  );
  searchDialog.value = false;
  scrollToActiveItem();
}

function prevItem() {
  if (!playlist.value.length) return;
  switchToItem(
    currentIndex.value > 0 ? currentIndex.value - 1 : playlist.value.length - 1
  );
}

function nextItem() {
  if (!playlist.value.length) return;
  switchToItem(
    currentIndex.value < playlist.value.length - 1 ? currentIndex.value + 1 : 0
  );
}

function prevPage() {
  changePage(-1);
}

function nextPage() {
  changePage(1);
}

function scrollToActiveItem() {
  nextTick(() => {
    if (!carouselTrack.value) return;
    const el = carouselTrack.value.querySelector('.carousel-item-active');
    if (el)
      el.scrollIntoView({
        behavior: 'smooth',
        inline: 'center',
        block: 'nearest',
      });
  });
}

async function playFromSearch(item) {
  const idx = playlist.value.findIndex((p) => p.Id === item.Id);
  switchToItem(idx);
  searchDialog.value = false;
}

function fetchKeyword(keyword) {
  searchParams.Keyword = keyword;
  searchParams.Page = 1;
  fetchSearch();
}

// ── 搜索 ──────────────────────────────────────────────────────────────────────
async function fetchSearch() {
  if (searchLoading.value) return;
  searchLoading.value = true;
  try {
    const data = await SearchAPI(searchParams);
    if (data) {
      playlist.value = [...(data.Data || [])];
      searchResults.Data = data.Data || [];
      searchResults.TotalPage = data.TotalPage || 0;
      searchResults.ResultSize = data.ResultSize || '';
    }
  } catch (e) {
    console.error('搜索请求异常:', e);
    $q.notify({
      type: 'negative',
      message: '搜索失败',
      position: 'top',
      timeout: 2000,
    });
  } finally {
    searchLoading.value = false;
  }
}

function changePage(delta) {
  searchParams.Page += delta;
  fetchSearch();
  if (searchResultsRef.value) searchResultsRef.value.scrollTop = 0;
}

// ── 时间格式化 ────────────────────────────────────────────────────────────────
const today = new Date();
function getTimeAgo(MTime) {
  if (!MTime) return '';
  const days = Math.floor((today - new Date(MTime)) / 86400000);
  if (days > 365) return `${Math.floor(days / 365)}年前`;
  if (days > 30) return `${Math.floor(days / 30)}个月前`;
  if (days > 0) return `${days}天前`;
  return '今天';
}

function parseTime(seconds) {
  if (isNaN(seconds) || seconds < 0) return '00:00:00';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(
    s
  ).padStart(2, '0')}`;
}

// ── 粒子系统 ──────────────────────────────────────────────────────────────────
class Particle {
  constructor(canvas) {
    this.canvas = canvas;
    this.reset();
  }

  reset() {
    this.x = Math.random() * this.canvas.width;
    this.y = Math.random() * this.canvas.height;
    this.size = Math.random() * 2.5 + 0.5;
    this.baseSize = this.size;
    this.speedX = (Math.random() - 0.5) * 0.4;
    this.speedY = (Math.random() - 0.5) * 0.4;
    this.opacity = Math.random() * 0.4 + 0.15;
    this.hue = Math.random() * 60 + 230;
    this.pulsePhase = Math.random() * Math.PI * 2;
    this.pulseSpeed = Math.random() * 0.018 + 0.008;
  }

  update(audioData = null) {
    this.pulsePhase += this.pulseSpeed;
    const pulse = Math.sin(this.pulsePhase) * 0.5 + 0.5;

    if (audioData) {
      const { bass = 0, mid = 0, treble = 0 } = audioData;
      this.size = this.baseSize + bass * 4 + pulse * 1.5;
      this.speedX *= 1 + bass * 0.08;
      this.speedY *= 1 + bass * 0.08;
      this.opacity = Math.min(0.85, 0.2 + bass * 0.5 + pulse * 0.25);
      this.hue = 230 + mid * 60 + treble * 30;
    } else {
      this.size = this.baseSize + pulse * 1.2;
      this.opacity = 0.15 + pulse * 0.25;
    }

    this.x += this.speedX;
    this.y += this.speedY;

    if (this.x < 0 || this.x > this.canvas.width) this.speedX *= -1;
    if (this.y < 0 || this.y > this.canvas.height) this.speedY *= -1;
  }

  draw(ctx) {
    ctx.save();
    ctx.globalAlpha = this.opacity;
    ctx.shadowBlur = 12;
    ctx.shadowColor = `hsl(${this.hue}, 80%, 65%)`;
    ctx.fillStyle = `hsl(${this.hue}, 70%, 72%)`;
    ctx.beginPath();
    ctx.arc(this.x, this.y, Math.max(0.1, this.size), 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
  }
}

function initParticles() {
  if (!particleCanvas.value) return;
  const canvas = particleCanvas.value;
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
  particles = [];
  const count = Math.min(
    350,
    Math.floor((canvas.width * canvas.height) / 6000)
  );
  for (let i = 0; i < count; i++) particles.push(new Particle(canvas));
}

function initAudioAnalyser() {
  if (!videoRef.value) return;

  // 如果是同一个视频元素，不需要重新创建 source
  if (connectedVideoElement === videoRef.value && source) {
    // 检查音频上下文状态，如果处于 suspended 则恢复
    if (audioContext && audioContext.state === 'suspended') {
      audioContext
        .resume()
        .catch((e) => console.warn('Failed to resume audio context:', e));
    }
    return;
  }

  // 如果是不同的视频元素，需要关闭旧的 audioContext 并创建新的
  if (audioContext) {
    try {
      audioContext.close();
    } catch (e) {
      console.warn('Error closing audio context:', e);
    }
    audioContext = null;
    analyser = null;
    source = null;
  }

  try {
    audioContext = new (window.AudioContext || window.webkitAudioContext)();
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 256;

    // 创建新的音频源并连接
    source = audioContext.createMediaElementSource(videoRef.value);
    source.connect(analyser);
    analyser.connect(audioContext.destination);

    // 记录已连接的视频元素
    connectedVideoElement = videoRef.value;
  } catch (e) {
    console.warn('Audio analyser unavailable:', e);
  }
}

function getAudioData() {
  if (!analyser) return null;
  const bufferLength = analyser.frequencyBinCount;
  const dataArray = new Uint8Array(bufferLength);
  analyser.getByteFrequencyData(dataArray);
  const bass = dataArray.slice(0, 10).reduce((a, b) => a + b, 0) / 10 / 255;
  const mid = dataArray.slice(10, 40).reduce((a, b) => a + b, 0) / 30 / 255;
  const treble =
    dataArray.slice(40).reduce((a, b) => a + b, 0) / (bufferLength - 40) / 255;
  return { bass, mid, treble };
}

function animate() {
  if (!particleCanvas.value) return;
  const canvas = particleCanvas.value;
  const ctx = canvas.getContext('2d');
  ctx.fillStyle = 'rgba(8, 8, 14, 0.12)';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  const audioData = isPlaying.value ? getAudioData() : null;
  particles.forEach((p) => {
    p.update(audioData);
    p.draw(ctx);
  });
  animationFrameId = requestAnimationFrame(animate);
}

// ── 文件拖拽 ──────────────────────────────────────────────────────────────────
function handleDrop(e) {
  e.preventDefault();
  isDragOver.value = false;
  const file = e.dataTransfer.files[0];
  if (file && file.type.startsWith('video/')) {
    loadVideo(URL.createObjectURL(file), file.name);
  } else {
    $q.notify({ type: 'warning', message: '仅支持视频文件', position: 'top' });
  }
}

function loadVideo(src, name, poster) {
  currentVideoSrc.value = src;
  currentVideoName.value = name || '未知视频';
  currentPoster.value = poster || '';
  videoLoaded.value = true;

  setTimeout(() => {
    if (videoRef.value) {
      videoRef.value.volume = volume.value;

      // 确保音频上下文正常工作
      if (audioContext && audioContext.state === 'suspended') {
        audioContext
          .resume()
          .then(() => {
            initAudioAnalyser();
          })
          .catch((e) => {
            console.warn('Failed to resume audio:', e);
            initAudioAnalyser();
          });
      } else {
        initAudioAnalyser();
      }

      if (!animationFrameId) animate();
    }
  }, 100);
}

// ── 磁力链 ────────────────────────────────────────────────────────────────────
async function submitMagnet() {
  const uri = magnetURI.value.trim();
  if (!uri.startsWith('magnet:')) {
    $q.notify({
      type: 'negative',
      message: '请输入有效的磁力链',
      position: 'top',
    });
    return;
  }
  torrentLoading.value = true;
  torrentProgress.value = 0;
  torrentState.value = '正在解析磁力链...';
  torrentName.value = '获取种子信息中...';
  try {
    const res = await axios.post('/api/torrent/add', { magnetURI: uri });
    console.log(res.data);
    if (res.data?.Code === 200) {
      const data = res.data.Data;
      currentInfoHash.value = data.infoHash;
      torrentName.value = data.name || '未知种子';
      torrentFiles.value = data.files || [];
      if (torrentFiles.value.length > 0) {
        showTorrentFiles.value = true;
        torrentLoading.value = false;
      } else {
        $q.notify({
          type: 'warning',
          message: '未解析到文件',
          position: 'top',
        });
        torrentLoading.value = false;
      }
    } else {
      $q.notify({
        type: 'negative',
        message: res.data?.message || '添加磁力链失败',
        position: 'top',
      });
      torrentLoading.value = false;
    }
  } catch (err) {
    $q.notify({
      type: 'negative',
      message: '请求失败: ' + (err.response?.data?.message || err.message),
      position: 'top',
    });
    torrentLoading.value = false;
  }
}

function selectTorrentFile(file) {
  selectedTorrentFile.value = file.path;
}

async function playSelectedTorrentFile() {
  if (!selectedTorrentFile.value || !currentInfoHash.value) return;
  torrentLoading.value = true;
  showTorrentFiles.value = false;
  torrentState.value = '正在开始下载...';
  torrentProgress.value = 0;
  const fileName = torrentFiles.value.find(f => f.path === selectedTorrentFile.value)?.name || '未知文件';
  try {
    await axios.post('/api/torrent/startDownload', {
      infoHash: currentInfoHash.value,
      filePath: selectedTorrentFile.value,
    });
    const newTask = {
      infoHash: currentInfoHash.value,
      name: torrentName.value,
      fileName: fileName,
      filePath: selectedTorrentFile.value,
      progress: 0,
      state: '准备下载',
      peers: 0,
    };
    activeDownloads.value.push(newTask);
    startPolling(currentInfoHash.value, newTask);
    const streamUrl = `/api/torrent/stream/${currentInfoHash.value}?file=${encodeURIComponent(selectedTorrentFile.value)}`;
    loadVideo(streamUrl, fileName);
  } catch (err) {
    $q.notify({
      type: 'negative',
      message: '启动下载失败: ' + (err.response?.data?.message || err.message),
      position: 'top',
    });
    torrentLoading.value = false;
  }
  selectedTorrentFile.value = null;
}

function getFileIcon(fileName) {
  const ext = fileName.split('.').pop()?.toLowerCase();
  const videoExts = ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v', 'mpg', 'mpeg'];
  const audioExts = ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a'];
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'];
  if (videoExts.includes(ext)) return 'movie';
  if (audioExts.includes(ext)) return 'audio_file';
  if (imageExts.includes(ext)) return 'image';
  return 'insert_drive_file';
}

function startPolling(infoHash, task) {
  stopPolling();
  torrentPollTimer = setInterval(async () => {
    try {
      const res = await axios.get(`/api/torrent/status/${infoHash}`);
      if (res.data?.code === 200) {
        const d = res.data.data;
        torrentName.value = d.name;
        torrentProgress.value = d.progress;
        torrentState.value = d.state;
        torrentPeers.value = d.peers;
        if (task) {
          task.progress = d.progress;
          task.state = d.state;
          task.peers = d.peers;
        }
        if (d.progress >= 3 && !videoLoaded.value && !showTorrentFiles.value) {
          torrentState.value = '缓冲就绪，开始播放';
          const streamUrl = `/api/torrent/stream/${infoHash}`;
          const newIdx = playlist.value.findIndex((p) => p.Id === infoHash);
          currentIndex.value = newIdx;
          loadVideo(streamUrl, d.videoFile || d.name);
          stopPolling();
        }
      }
    } catch (err) {
      console.warn('轮询状态失败:', err);
    }
  }, 2000);
}

function stopPolling() {
  if (torrentPollTimer) {
    clearInterval(torrentPollTimer);
    torrentPollTimer = null;
  }
}

async function cancelTorrent() {
  stopPolling();
  if (currentInfoHash.value) {
    try {
      await axios.delete(`/api/torrent/${currentInfoHash.value}`);
    } catch {
      /* ignore */
    }
    activeDownloads.value = activeDownloads.value.filter(t => t.infoHash !== currentInfoHash.value);
  }
  torrentLoading.value = false;
  torrentProgress.value = 0;
  torrentState.value = '';
  torrentName.value = '';
  currentInfoHash.value = '';
  torrentFiles.value = [];
  showTorrentFiles.value = false;
  selectedTorrentFile.value = null;
}

// ── 下载管理器 ─────────────────────────────────────────────────────────────────
function playDownloadTask(task) {
  const streamUrl = `/api/torrent/stream/${task.infoHash}?file=${encodeURIComponent(task.filePath)}`;
  currentInfoHash.value = task.infoHash;
  loadVideo(streamUrl, task.fileName);
}

function openDownloadFolder(task) {
  window.open(`/api/openFolder/${task.infoHash}`, '_blank');
}

function removeDownloadTask(task) {
  axios.delete(`/api/torrent/${task.infoHash}`).catch(() => {
    /* ignore */
  });
  activeDownloads.value = activeDownloads.value.filter(t => t.infoHash !== task.infoHash);
  if (currentInfoHash.value === task.infoHash) {
    currentInfoHash.value = '';
    torrentLoading.value = false;
  }
}

// ── 播放控制 ──────────────────────────────────────────────────────────────────
function togglePlay() {
  if (!videoRef.value || !videoLoaded.value) return;
  isPlaying.value ? videoRef.value.pause() : videoRef.value.play();
}

function stopPlay() {
  if (!videoRef.value || !videoLoaded.value) return;
  currentVideoSrc.value = '';
  currentVideoName.value = '暂无视频';
  currentPoster.value = '';
  videoLoaded.value = false;
  isPlaying.value = false;
}

function onPlay() {
  isPlaying.value = true;
  resetControlsTimer();
}
function onPause() {
  isPlaying.value = false;
  showControls();
}
function onWaiting() {
  isBuffering.value = true;
}
function onCanPlay() {
  isBuffering.value = false;
}

function onEnded() {
  isPlaying.value = false;
  controlsHidden.value = false;
  if (
    playlist.value.length > 0 &&
    currentIndex.value < playlist.value.length - 1
  )
    nextItem();
}

// 添加视频错误处理
function onVideoError(e) {
  console.error('Video playback error:', e);
  // $q.notify({ type: 'negative', message: '视频播放失败', position: 'top' });
}

function onTimeUpdate() {
  if (!videoRef.value) return;
  currentTimeSeconds.value = videoRef.value.currentTime;
  currentTime.value = parseTime(videoRef.value.currentTime);
  // 更新缓冲进度
  const buf = videoRef.value.buffered;
  if (buf.length > 0) bufferedSeconds.value = buf.end(buf.length - 1);
}

function onMetadataLoaded() {
  if (!videoRef.value) return;
  durationSeconds.value = videoRef.value.duration;
  duration.value = parseTime(videoRef.value.duration);
}

// ── 进度条拖拽 ────────────────────────────────────────────────────────────────
function startSeek(e) {
  isSeeking.value = true;
  doSeek(e);
  document.addEventListener('mousemove', doSeek);
  document.addEventListener('mouseup', endSeek);
}

function doSeek(e) {
  if (!progressBar.value || !durationSeconds.value) return;
  const rect = progressBar.value.getBoundingClientRect();
  const pct = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
  if (videoRef.value) videoRef.value.currentTime = pct * durationSeconds.value;
}

function endSeek() {
  isSeeking.value = false;
  document.removeEventListener('mousemove', doSeek);
  document.removeEventListener('mouseup', endSeek);
}

function onProgressHover(e) {
  if (!progressBar.value || !durationSeconds.value) return;
  const rect = progressBar.value.getBoundingClientRect();
  const pct = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
  hoverTime.value = parseTime(pct * durationSeconds.value);
  hoverX.value = e.clientX - rect.left;
}

function hideTooltip() {
  hoverTime.value = null;
}

// ── 音量 ──────────────────────────────────────────────────────────────────────
function toggleMute() {
  if (!videoRef.value) return;
  if (volume.value > 0) {
    videoRef.value.volume = 0;
    volume.value = 0;
  } else {
    volume.value = 0.8;
    videoRef.value.volume = 0.8;
  }
}

function setVolume(val) {
  if (!videoRef.value) return;
  volume.value = val;
}

// ── 全屏 ──────────────────────────────────────────────────────────────────────
function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement
      .requestFullscreen()
      .then(() => {
        isFullscreen.value = true;
      })
      .catch((e) => {
        console.warn('全屏失败:', e);
      });
  } else {
    document
      .exitFullscreen()
      .then(() => {
        isFullscreen.value = false;
      })
      .catch((e) => {
        console.warn('退出全屏失败:', e);
      });
  }
}

// ── 控制栏显隐 ────────────────────────────────────────────────────────────────
function showControls() {
  controlsHidden.value = false;
  clearHideTimer();
}

function startHideTimer() {
  if (isPlaying.value) {
    hideControlsTimer = setTimeout(() => {
      controlsHidden.value = true;
    }, 3000);
  }
}

function clearHideTimer() {
  if (hideControlsTimer) {
    clearTimeout(hideControlsTimer);
    hideControlsTimer = null;
  }
}

function resetControlsTimer() {
  clearHideTimer();
  hideControlsTimer = setTimeout(() => {
    controlsHidden.value = true;
  }, 3000);
}

function onMouseMove() {
  showControls();
  if (isPlaying.value) resetControlsTimer();
}

// ── 键盘快捷键 ────────────────────────────────────────────────────────────────
function handleKeydown(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  switch (e.code) {
    case 'Space':
      e.preventDefault();
      togglePlay();
      break;
    case 'ArrowLeft':
      e.preventDefault();
      if (videoRef.value)
        videoRef.value.currentTime = Math.max(
          0,
          videoRef.value.currentTime - 5
        );
      break;
    case 'ArrowRight':
      e.preventDefault();
      if (videoRef.value)
        videoRef.value.currentTime = Math.min(
          durationSeconds.value,
          videoRef.value.currentTime + 5
        );
      break;
    case 'ArrowUp':
      e.preventDefault();
      setVolume(Math.min(1, volume.value + 0.1));
      break;
    case 'ArrowDown':
      e.preventDefault();
      setVolume(Math.max(0, volume.value - 0.1));
      break;
    case 'KeyF':
      toggleFullscreen();
      break;
    case 'KeyM':
      toggleMute();
      break;
  }
}

// ── 窗口 resize ───────────────────────────────────────────────────────────────
function handleResize() {
  if (particleCanvas.value) {
    particleCanvas.value.width = window.innerWidth;
    particleCanvas.value.height = window.innerHeight;
  }
}

// ── 监听 ──────────────────────────────────────────────────────────────────────
watch(searchDialog, (val) => {
  if (val && searchResults.Data.length === 0) fetchSearch();
});

watch(isPlaying, (playing) => {
  if (playing) {
    resetControlsTimer();
  } else {
    controlsHidden.value = false;
    clearHideTimer();
  }
});

// ── 生命周期 ──────────────────────────────────────────────────────────────────
onMounted(() => {
  initParticles();
  animate();
  document.addEventListener('fullscreenchange', () => {
    isFullscreen.value = !!document.fullscreenElement;
  });
  window.addEventListener('resize', handleResize);
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  if (animationFrameId) cancelAnimationFrame(animationFrameId);
  clearHideTimer();
  endSeek();
  if (audioContext) audioContext.close();
  stopPolling();
  if (currentInfoHash.value) {
    axios.delete(`/api/torrent/${currentInfoHash.value}`).catch((e) => {
      console.warn('删除种子失败:', e);
    });
  }
  window.removeEventListener('resize', handleResize);
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<style scoped>
/* ── 容器 ──────────────────────────────────────────────────────────────────── */
.immersive-container {
  position: fixed;
  inset: 0;
  background: radial-gradient(ellipse at 30% 40%, #12122a 0%, #080810 100%);
  overflow: hidden;
  cursor: none;
  user-select: none;
}

.immersive-container:hover {
  cursor: default;
}

/* ── 粒子画布 ─────────────────────────────────────────────────────────────── */
.particle-canvas {
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
}

.top-title {
  font-size: 0.95rem;
  color: rgba(255, 255, 255, 0.8);
  font-weight: 500;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  padding: 0 20px;
  text-shadow: 0 1px 8px rgba(99, 102, 241, 0.6);
}

/* ── 固定位置按钮 ──────────────────────────────────────────────────────────── */
.fixed-top-left-btn {
  position: fixed;
  top: 16px;
  left: 16px;
  z-index: 1000;
  background: rgba(255, 255, 255, 0.06) !important;
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s;
}

.fixed-top-left-btn:hover {
  background: rgba(99, 102, 241, 0.25) !important;
  border-color: rgba(99, 102, 241, 0.5);
  box-shadow: 0 0 18px rgba(99, 102, 241, 0.35);
}

.fixed-top-right-btn {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 1000;
  background: rgba(255, 255, 255, 0.06) !important;
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s;
}

.fixed-top-right-btn:hover {
  background: rgba(99, 102, 241, 0.25) !important;
  border-color: rgba(99, 102, 241, 0.5);
  box-shadow: 0 0 18px rgba(99, 102, 241, 0.35);
}

.top-action-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s;
  flex-shrink: 0;
}

.top-action-btn:hover {
  background: rgba(99, 102, 241, 0.25);
  border-color: rgba(99, 102, 241, 0.5);
  box-shadow: 0 0 18px rgba(99, 102, 241, 0.35);
}

/* ── 轮播播放列表 ──────────────────────────────────────────────────────────── */
.carousel-banner {
  position: relative;
  width: 80%;
  margin: 0 auto;
  padding-left: 52px;
  padding-right: 52px;
  z-index: 222;
  display: flex;
  align-items: center;
  background: rgba(8, 8, 20, 0.55);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
}

.carousel-track {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  scroll-behavior: smooth;
  scroll-snap-type: x mandatory;
  justify-content: space-evenly;
  flex: 1;
  padding: 4px 2px;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.carousel-track::-webkit-scrollbar {
  display: none;
}

.carousel-item {
  flex-shrink: 0;
  width: 46px;
  height: 64px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  scroll-snap-align: center;
  border: 2px solid transparent;
  transition: transform 0.25s ease, border-color 0.25s ease,
    box-shadow 0.25s ease;
  position: relative;
}

.carousel-item:hover {
  border-color: rgba(99, 102, 241, 0.55);
  transform: scale(1.1) translateY(-2px);
}

.carousel-item-active {
  border-color: rgba(139, 92, 246, 0.95);
  box-shadow: 0 0 14px rgba(139, 92, 246, 0.65),
    0 0 28px rgba(99, 102, 241, 0.25);
  transform: scale(1.12) translateY(-2px);
}

.carousel-thumb {
  width: 100%;
  height: 100%;
}

.carousel-thumb-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(25, 25, 45, 0.9);
}

.carousel-item-label {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  font-size: 7px;
  color: rgba(255, 255, 255, 0.85);
  text-align: center;
  padding: 2px 2px;
  background: rgba(0, 0, 0, 0.75);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.carousel-item-active-indicator {
  position: absolute;
  top: 2px;
  right: 2px;
  background: rgba(139, 92, 246, 0.85);
  border-radius: 50%;
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.carousel-arrow {
  position: absolute;
  z-index: 23;
  background: rgba(10, 10, 22, 0.55);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(99, 102, 241, 0.25);
  transition: background 0.2s;
}

.carousel-arrow:hover {
  background: rgba(99, 102, 241, 0.3);
  border-color: rgba(99, 102, 241, 0.55);
}

.carousel-arrow-left {
  left: 8px;
}
.carousel-arrow-right {
  right: 8px;
}

/* ── 视频区域 ─────────────────────────────────────────────────────────────── */
.video-wrapper {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  justify-content: center;
  align-items: center;
}

.video-wrapper video {
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 6px;
  box-shadow: 0 0 80px rgba(80, 60, 180, 0.25);
}

.video-buffering {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  z-index: 5;
  backdrop-filter: blur(4px);
}

/* ── 拖拽区域 ─────────────────────────────────────────────────────────────── */
.drop-zone {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 52vw;
  height: 30vh;
  min-width: 320px;
  border: 2px dashed rgba(99, 102, 241, 0.4);
  border-radius: 24px;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 4;
  transition: border-color 0.3s, background 0.3s, box-shadow 0.3s;
}

.drop-zone:hover,
.drop-zone-active {
  border-color: rgba(139, 92, 246, 0.8);
  background: rgba(99, 102, 241, 0.07);
  box-shadow: 0 0 50px rgba(99, 102, 241, 0.2),
    inset 0 0 40px rgba(99, 102, 241, 0.06);
}

.drop-content {
  text-align: center;
  pointer-events: none;
}

.drop-icon-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.drop-icon-ring {
  position: absolute;
  width: 90px;
  height: 90px;
  border-radius: 50%;
  border: 2px solid rgba(99, 102, 241, 0.3);
  animation: pulse-ring 2.5s ease-in-out infinite;
}

@keyframes pulse-ring {
  0%,
  100% {
    transform: scale(0.9);
    opacity: 0.4;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
}

.drop-title {
  font-size: 1.1rem;
  color: rgba(196, 181, 253, 0.9);
  margin: 0 0 8px;
  font-weight: 500;
}

.drop-subtitle {
  font-size: 0.82rem;
  color: rgba(129, 140, 248, 0.55);
  margin: 0;
}

/* ── 磁力链输入 ──────────────────────────────────────────────────────────── */
.magnet-input-area {
  position: absolute;
  top: 88px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 5;
  width: 64vw;
  max-width: 680px;
  min-width: 280px;
}

.magnet-input-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(12, 12, 24, 0.75);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 40px;
  padding: 8px 8px 8px 18px;
  transition: border-color 0.3s, box-shadow 0.3s;
}

.magnet-input-wrapper.magnet-focused {
  border-color: rgba(139, 92, 246, 0.65);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.12),
    0 0 30px rgba(99, 102, 241, 0.18);
}

.magnet-icon {
  flex-shrink: 0;
  opacity: 0.8;
}

.magnet-input {
  flex: 1;
}

.magnet-input :deep(.q-field__control) {
  background: transparent;
  border: none;
}
.magnet-input :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.88rem;
}
.magnet-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.38);
}

.magnet-submit-btn {
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  transition: background 0.2s, box-shadow 0.2s;
}

.magnet-submit-btn:hover:not([disabled]) {
  background: rgba(99, 102, 241, 0.4);
  box-shadow: 0 0 16px rgba(99, 102, 241, 0.4);
}

/* ── 磁力链文件选择 ───────────────────────────────────────────────────────── */
.torrent-files-dialog {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
  background: rgba(0, 0, 0, 0.6);
}

.torrent-files-card {
  background: rgba(10, 10, 22, 0.92);
  backdrop-filter: blur(28px);
  -webkit-backdrop-filter: blur(28px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 16px;
  width: 520px;
  max-width: 90vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.torrent-files-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 20px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
}

.torrent-files-title {
  font-size: 1rem;
  font-weight: 600;
  color: #e0e7ff;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.torrent-files-hint {
  font-size: 0.75rem;
  color: #818cf8;
}

.torrent-files-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.torrent-file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s;
  border: 1px solid transparent;
  margin-bottom: 6px;
}

.torrent-file-item:hover {
  background: rgba(99, 102, 241, 0.12);
  border-color: rgba(99, 102, 241, 0.25);
}

.torrent-file-selected {
  background: rgba(99, 102, 241, 0.2) !important;
  border-color: rgba(99, 102, 241, 0.5) !important;
}

.torrent-file-icon {
  color: #818cf8;
  flex-shrink: 0;
}

.torrent-file-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.torrent-file-name {
  font-size: 0.875rem;
  color: #e0e7ff;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.torrent-file-size {
  font-size: 0.7rem;
  color: #6b7280;
}

.torrent-files-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid rgba(99, 102, 241, 0.18);
}

/* ── 种子加载 ─────────────────────────────────────────────────────────────── */
.torrent-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 6;
}

.torrent-loading-card {
  background: rgba(10, 10, 22, 0.85);
  backdrop-filter: blur(28px);
  -webkit-backdrop-filter: blur(28px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 20px;
  padding: 32px 40px;
  min-width: 340px;
  text-align: center;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.torrent-spinner {
  margin-bottom: 20px;
}

.torrent-name {
  font-size: 0.95rem;
  color: rgba(255, 255, 255, 0.85);
  margin: 0 0 16px;
  line-height: 1.4;
  word-break: break-all;
}

.torrent-progress-wrap {
  margin-bottom: 4px;
}

.torrent-stats {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  font-size: 12px;
}

.torrent-percent {
  color: #a5b4fc;
  font-weight: 600;
}
.torrent-state {
  color: rgba(134, 239, 172, 0.8);
}
.torrent-peers {
  color: rgba(165, 148, 249, 0.7);
  display: flex;
  align-items: center;
  gap: 3px;
}

/* ── 底部控制面板 ─────────────────────────────────────────────────────────── */
.glass-panel {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 20;
  padding: 16px 24px 24px;
  background: linear-gradient(
    to top,
    rgba(6, 6, 16, 0.95) 0%,
    rgba(6, 6, 16, 0.6) 60%,
    transparent 100%
  );
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

/* ── 进度条 ───────────────────────────────────────────────────────────────── */
.progress-container {
  height: 32px;
  display: flex;
  align-items: center;
  cursor: pointer;
  margin-bottom: 10px;
  position: relative;
}

.progress-track {
  position: relative;
  width: 100%;
  height: 4px;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 4px;
  overflow: visible;
  transition: height 0.2s ease;
}

.progress-container:hover .progress-track {
  height: 6px;
}

.progress-buffered {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: rgba(255, 255, 255, 0.18);
  border-radius: 4px;
  transition: width 0.5s ease;
}

.progress-fill {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: linear-gradient(90deg, #6366f1, #8b5cf6, #f472b6);
  border-radius: 4px;
  transition: width 0.1s linear;
  overflow: hidden;
}

.progress-glow {
  position: absolute;
  top: 0;
  right: 0;
  width: 40px;
  height: 100%;
  background: rgba(255, 255, 255, 0.4);
  filter: blur(4px);
}

.progress-thumb {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%) scale(0.6);
  width: 14px;
  height: 14px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 0 10px rgba(139, 92, 246, 0.8), 0 0 20px rgba(99, 102, 241, 0.4);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  pointer-events: none;
}

.progress-container:hover .progress-thumb,
.progress-thumb.seeking {
  transform: translate(-50%, -50%) scale(1);
  box-shadow: 0 0 14px rgba(139, 92, 246, 1), 0 0 28px rgba(99, 102, 241, 0.5);
}

.progress-tooltip {
  position: absolute;
  top: -34px;
  transform: translateX(-50%);
  background: rgba(20, 20, 40, 0.92);
  color: #c4b5fd;
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 6px;
  white-space: nowrap;
  pointer-events: none;
  border: 1px solid rgba(99, 102, 241, 0.3);
  backdrop-filter: blur(8px);
}

/* ── 控制按钮行 ──────────────────────────────────────────────────────────── */
.control-buttons {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ctrl-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.ctrl-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.play-btn {
  transition: transform 0.15s ease, opacity 0.15s ease;
}

.play-btn:hover {
  transform: scale(1.1);
}

.time-display {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.82rem;
  font-variant-numeric: tabular-nums;
  margin-left: 8px;
}

.time-current {
  color: rgba(255, 255, 255, 0.9);
}
.time-sep {
  color: rgba(255, 255, 255, 0.3);
}
.time-total {
  color: rgba(255, 255, 255, 0.45);
}

.volume-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.volume-slider {
  width: 80px;
  transition: width 0.3s ease;
}

/* ── 搜索侧面板 ──────────────────────────────────────────────────────────── */
.search-panel {
  position: relative;
  height: 800px;
  margin: 0 auto;
  width: 80%;
  /* margin-top: 78px; */
  background: rgba(9, 9, 22, 0.92);
  backdrop-filter: blur(32px);
  -webkit-backdrop-filter: blur(32px);
  border-left: 1px solid rgba(99, 102, 241, 0.25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -10px 0 40px rgba(0, 0, 0, 0.4);
  z-index: 1000;
}

.search-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 18px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(99, 102, 241, 0.06);
}

.search-panel-title {
  font-size: 1rem;
  font-weight: 600;
  color: #a5b4fc;
  display: flex;
  align-items: center;
  letter-spacing: 0.03em;
  width: 100%;
}

.search-conditions {
  padding: 14px 16px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.12);
  display: flex;
  flex-direction: column;
  gap: 10px;
  .filter-item {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }
}

.search-input :deep(.q-field__control) {
  background: rgba(25, 25, 48, 0.65);
  border-radius: 8px;
  width: 72vw;
}
.search-input :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.88rem;
}
.search-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.35);
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-label {
  font-size: 11px;
  color: #818cf8;
  min-width: 28px;
  letter-spacing: 0.03em;
}

.result-size-badge {
  font-size: 11px;
  color: rgba(134, 239, 172, 0.7);
  background: rgba(134, 239, 172, 0.08);
  border: 1px solid rgba(134, 239, 172, 0.2);
  padding: 1px 8px;
  border-radius: 10px;
}

.search-results {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
  scrollbar-width: thin;
  scrollbar-color: rgba(99, 102, 241, 0.25) transparent;
}

.search-results::-webkit-scrollbar {
  width: 4px;
}
.search-results::-webkit-scrollbar-thumb {
  background: rgba(99, 102, 241, 0.25);
  border-radius: 2px;
}

.search-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 0;
}

.search-cards {
  columns: 3;
  gap: 10px;
}

.search-card {
  margin-bottom: 4px;
  display: flex;
  gap: 10px;
  padding: 10px;
  border-radius: 12px;
  background: rgba(22, 22, 45, 0.55);
  border: 1px solid rgba(99, 102, 241, 0.12);
  cursor: pointer;
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s,
    transform 0.2s;
  position: relative;
  overflow: hidden;
}

.search-card::after {
  /* content: ''; */
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.07) 0%,
    transparent 60%
  );
  opacity: 0;
  transition: opacity 0.3s;
}

.search-card:hover::after {
  opacity: 1;
}

.search-card:hover {
  border-color: rgba(139, 92, 246, 0.4);
  background: rgba(32, 32, 60, 0.65);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.15);
  transform: translateX(3px);
}

.search-card-thumb {
  position: relative;
  flex-shrink: 0;
  width: 72px;
  height: 100px;
  border-radius: 8px;
  overflow: hidden;
}

.search-card-img {
  width: 100%;
  height: 100%;
}

.search-card-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(25, 25, 48, 0.8);
}

.search-card-play-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  opacity: 0;
  transition: opacity 0.25s;
}

.search-card:hover .search-card-play-overlay {
  opacity: 1;
}

.search-card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow: hidden;
  min-width: 0;
}

.search-card-title {
  font-size: 12.5px;
  color: #dde5ff;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.search-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  line-height: 1.5;
}

.tag-actress {
  color: #a78bfa;
  background: rgba(139, 92, 246, 0.14);
  border: 1px solid rgba(139, 92, 246, 0.25);
}

.tag-code {
  color: #f472b6;
  background: rgba(244, 114, 182, 0.12);
  border: 1px solid rgba(244, 114, 182, 0.22);
}

.search-card-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: auto;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 10px;
  color: rgba(165, 180, 252, 0.55);
}

.search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
}

/* ── 分页 ─────────────────────────────────────────────────────────────────── */
.search-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 10px 20px;
  border-top: 1px solid rgba(99, 102, 241, 0.15);
  background: rgba(8, 8, 20, 0.6);
}

.pagination-info {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.page-current {
  color: #c4b5fd;
  font-weight: 600;
}
.page-sep {
  color: rgba(255, 255, 255, 0.25);
}
.page-total {
  color: rgba(165, 180, 252, 0.45);
}

/* ── 过渡动画 ─────────────────────────────────────────────────────────────── */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-down-enter-active,
.slide-down-leave-active {
  transition: transform 0.35s ease, opacity 0.35s ease;
}
.slide-down-enter-from,
.slide-down-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.35s ease, opacity 0.35s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(20px);
  opacity: 0;
}

/* ── 响应式 ───────────────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .glass-panel {
    padding: 12px 16px 20px;
  }
  .volume-slider {
    width: 60px;
  }
  .top-title {
    font-size: 0.85rem;
  }
  .magnet-input-area {
    width: 88vw;
    bottom: 78px;
  }
  .search-panel {
    width: 100vw;
    max-width: 100vw;
  }
  .carousel-banner {
    padding: 8px 46px;
    height: 74px;
  }
  .carousel-item {
    width: 40px;
    height: 56px;
  }
  .drop-zone {
    width: 80vw;
    height: 42vh;
  }
  .time-display {
    font-size: 0.75rem;
  }
}

/* ── 下载管理器 ─────────────────────────────────────────────────────────────── */
.download-fab {
  position: fixed !important;
  bottom: 80px;
  right: 24px;
  z-index: 1000;
  box-shadow: 0 4px 16px rgba(99, 102, 241, 0.4);
}

.download-manager-card {
  width: 380px;
  max-width: 90vw;
  height: 100vh;
  background: rgba(9, 9, 22, 0.95);
  backdrop-filter: blur(32px);
  -webkit-backdrop-filter: blur(32px);
  border-left: 1px solid rgba(99, 102, 241, 0.25);
}

.download-manager-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(99, 102, 241, 0.2);
  padding: 16px 20px;
}

.download-manager-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 1.1rem;
  font-weight: 600;
  color: #e0e7ff;
}

.download-manager-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.download-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #6b7280;
}

.download-empty p {
  margin-top: 12px;
  font-size: 0.9rem;
}

.download-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.download-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: rgba(30, 30, 50, 0.6);
  border: 1px solid rgba(99, 102, 241, 0.15);
  border-radius: 12px;
  transition: all 0.2s ease;
}

.download-item:hover {
  background: rgba(40, 40, 65, 0.7);
  border-color: rgba(99, 102, 241, 0.3);
}

.download-item-playing {
  border-color: rgba(16, 185, 129, 0.5);
  background: rgba(16, 185, 129, 0.1);
}

.download-item-info {
  flex: 1;
  min-width: 0;
}

.download-item-name {
  font-size: 0.9rem;
  font-weight: 500;
  color: #e0e7ff;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.download-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 0.7rem;
  color: #6b7280;
}

.download-item-file {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #9ca3af;
}

.download-item-state {
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(99, 102, 241, 0.2);
  color: #818cf8;
}

.download-item-state.state-已完成 {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
}

.download-item-state.state-下载中,
.download-item-state.state-正在连接 {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.download-item-percent {
  min-width: 45px;
  text-align: right;
}

.download-item-actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
</style>
