<template>
  <div class="immersive-container" @mousemove="onMouseMove" @click.self="togglePlay" @dblclick="toggleFullscreen">
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>

    <!-- 左上角返回按钮 -->
    <q-btn flat color="white" icon="arrow_back" class="fixed-top-left-btn" @click.stop="goBack">
      <q-tooltip class="bg-dark text-white">返回</q-tooltip>
    </q-btn>

    <div class="fixed-top-right-btns">
      <q-btn flat v-if="currentData.Id" color="white" size="md" label="编辑" class="fixed-top-right-btn">
        <q-popup-proxy>
          <div class="edit-popup-content" v-if="currentData.MovieType">
            <q-btn-dropdown dense flat size="md" color="white"
              :label="`${currentData.MovieType === '无' ? '分类' : currentData.MovieType}`" no-caps>
              <q-list style="min-width: 60px">
                <q-item v-for="mt in MovieTypeOptions" :key="mt.value" clickable v-close-popup>
                  <q-item-section @click="setMovieType(currentData, mt.value)">{{ mt.label }}</q-item-section>
                </q-item>
              </q-list>
            </q-btn-dropdown>
            <q-btn flat dense color="white" icon="edit" size="md" @click.stop="fileEditRef.open(currentData)">
              <q-tooltip>修改</q-tooltip>
            </q-btn>
            <q-btn flat dense color="white" icon="delete" size="md" @click.stop="deleteVideo(currentData)">
              <q-tooltip>删除</q-tooltip>
            </q-btn>
            <!-- 截图 (非骑兵) -->
            <q-btn flat round color="green" size="md" icon="photo_camera" v-if="currentData.MovieType !== '骑兵'"
              @click="curImage">
              <q-tooltip class="bg-dark">截图</q-tooltip>
            </q-btn>
            <q-btn flat round color="red" size="md" icon="photo_camera" v-if="currentData.MovieType !== '骑兵'"
              @click="curImage('png')">
              <q-tooltip class="bg-dark">Png</q-tooltip>
            </q-btn>
            <!-- 标签 -->
            <q-btn flat round v-if="currentData.Id" color="white" size="md" icon="ti-star">
              <q-popup-proxy>
                <EditVideoTag :current-data="currentData" @next-one="nextItem" @prev-one="prevItem" />
              </q-popup-proxy>
              <q-tooltip class="bg-dark">标签</q-tooltip>
            </q-btn>
            <!-- 剪辑 -->
            <q-btn flat round color="white" size="md" icon="content_cut" v-if="videoLoaded">
              <q-popup-proxy>
                <VideoCutParam :current-data="currentData" :current-time="currentTime" :duration="durationSeconds"
                  @stop-video="videoRef?.pause()" @play-video="videoRef?.play()" @prev-one-video="prevItem"
                  @next-one-video="nextItem" @forward-time="forwardTime" />
              </q-popup-proxy>
              <q-tooltip class="bg-dark">剪辑</q-tooltip>
            </q-btn>
          </div>
        </q-popup-proxy>
        <q-tooltip>编辑</q-tooltip>
      </q-btn>
      <q-btn flat color="white" size="md" :icon="isFullscreen ? 'fullscreen_exit' : 'fullscreen'"
        class="fixed-top-right-btn" @click="toggleFullscreen">
        <q-tooltip class="bg-dark">{{
          isFullscreen ? '退出全屏' : '全屏'
        }}</q-tooltip>
      </q-btn>
    </div>


    <!-- 顶部中央视频信息 -->
    <div class="fixed-top-center" v-if="videoLoaded">
      <span class="meta-item">
        <q-btn color="indigo-6" flat dense grossy @click="fetchKeyword(currentData.Author)">{{ currentData.Author
          }}</q-btn>
      </span>
      <span class="top-video-tag tag tag-level" v-for="(tag, index) in currentData.Tags" :key="tag"
        :style="{ background: getTagColor(index) }">{{ tag }}</span>
      <span class="top-video-name">
        {{ formatTitle(currentData.Name) }}
      </span>
    </div>
    <!-- 底部控制面板 -->
    <transition name="slide-up">
      <div class="glass-panel" @click.stop @touchstart="touchControl = true" @touchend="touchControl = false">
        <!-- 进度条 -->
        <div class="progress-container" ref="progressBar" @mousedown="startSeek" @mousemove="onProgressHover"
          @mouseleave="hideTooltip" @contextmenu.prevent="onProgressContextMenu" @touchstart.stop="onSeekBarTouchStart"
          @touchmove.stop="onSeekBarTouchMove" @touchend.stop="onSeekBarTouchEnd">
          <div class="progress-track">
            <!-- 缓冲进度 -->
            <div class="progress-buffered" :style="{ width: bufferedPercent + '%' }"></div>
            <!-- 播放进度 -->
            <div class="progress-fill" :style="{ width: progressPercent + '%' }">
              <div class="progress-glow"></div>
            </div>
            <!-- 拖拽手柄 -->
            <div class="progress-thumb" :style="{ left: progressPercent + '%' }"
              :class="{ seeking: isSeeking || touchSeekBar }"></div>
          </div>
          <!-- 时间悬浮提示 -->
          <div class="progress-tooltip" v-if="hoverTime !== null" :style="{ left: hoverX + 'px' }">
            {{ hoverTime }}
          </div>
        </div>
        <!-- 控制按钮行 -->
        <div class="control-buttons mobile-compact" :style="{
          display: 'flex',
          flexDirection: isSmall ? 'column' : 'row'
        }">

          <!-- 快进快退按钮组 -->
          <div class="seek-buttons-popup">
            <q-btn flat round color="white" size="md" icon="skip_previous" @click="prevItem">
              <q-tooltip class="bg-dark">上一个</q-tooltip>
            </q-btn>
            <q-btn flat dense v-for="sec in seekSeconds" :key="sec" class="seek-btn"
              :class="{ 'seek-btn-rewind': sec < 0 }" @click.stop="seekBySeconds(sec, false)"
              @contextmenu.prevent.stop="seekBySeconds(sec, true)">
              {{ sec > 0 ? '+' + sec : sec }}s
            </q-btn>
            <q-btn flat round color="white" size="md" icon="skip_next" @click="nextItem">
              <q-tooltip class="bg-dark">下一个</q-tooltip>
            </q-btn>
          </div>
          <!-- 右侧：音量 + 设置 + 剪辑 + 标签 + 全屏 -->
          <div class="ctrl-right">

            <div class="time-display">
              <span class="time-current">{{ currentTime }}</span>
              <span class="time-sep">/</span>
              <span class="time-total">{{ duration }}</span>
            </div>
            <q-btn flat round :color="isPlaying ? 'indigo-3' : 'white'" size="md"
              :icon="isPlaying ? 'pause_circle' : 'play_circle'" class="play-btn" @click="togglePlay" />
            <q-btn flat round v-if="isPlaying" color="white" size="md" icon="stop" class="play-btn" @click="stopPlay" />
            <!-- 画面设置 -->
            <q-btn flat round dense color="white" size="md" icon="settings" v-if="videoLoaded">
              <q-popup-proxy>
                <PlayerSetting />
              </q-popup-proxy>
              <q-tooltip class="bg-dark">画面设置</q-tooltip>
            </q-btn>
            <div class="volume-group" @mouseenter="showVolume = true" @mouseleave="showVolume = false">
              <transition name="fade">
                <q-slider v-show="showVolume" v-model="volume" :min="0" :max="1" :step="0.01" color="indigo-4"
                  track-color="grey-8" class="volume-slider" vertical reverse @update:model-value="setVolume" />
              </transition>
              <q-btn flat round color="white" size="md" @click="toggleMute" :icon="volumeIcon" />
            </div>
          </div>

        </div>
      </div>
    </transition>
    <!-- 播放列表轮播 -->
    <transition name="slide-down">
      <!-- 搜索面板 - 合并到轮播中 -->
      <div class="search-panel" v-show="searchDialog" @click.stop>
        <!-- 头部 -->
        <div class="search-panel-header">
          <div class="search-panel-title">
            <q-input v-model="searchParams.Keyword" placeholder="输入关键词..." dark dense outlined class="search-input"
              @keyup.enter="fetchSearch" @change="fetchSearch">
              <template v-slot:prepend>
                <q-icon name="manage_search" size="18px" />
              </template>
              <template v-slot:append v-if="searchParams.Keyword">
                <q-btn flat round dense icon="clear" color="grey-5" size="xs" @click="
                  searchParams.Keyword = '';
                fetchSearch();
                " />
              </template>
            </q-input>
          </div>
          <q-btn flat dense size="lg" icon="refresh" @refresh-done="fetchSearch" color="indigo-4" />
          <q-btn flat round dense size="lg" color="grey-4" icon="close" @click="searchDialog = false" />
        </div>

        <!-- 搜索条件 -->
        <div class="search-conditions">
          <div class="filter-item">
            <div class="filter-row">
              <q-select v-model="searchParams.MovieType" :options="MovieTypeSelects" dense emit-value map-options
                borderless dark style="min-width: 120px" @update:model-value="fetchSearch" />
            </div>

            <div class="filter-row">
              <q-select v-model="currentSort" :options="sortOptions" dense emit-value map-options borderless dark
                style="min-width: 120px" @update:model-value="fetchSearch" />
            </div>
            <div class="filter-row">
              <IndexButton flat @refresh-done="fetchSearch" color="red" toggle-color="indigo-6" glossy />
            </div>
          </div>
        </div>

        <!-- 搜索结果 -->
        <div class="search-results" ref="searchResultsRef">
          <div v-if="searchLoading" class="search-loading">
            <q-spinner-dots size="40px" color="indigo-4" />
            <p class="text-grey-5 q-mt-sm text-caption">加载中...</p>
          </div>

          <template v-else-if="searchResults.Data && searchResults.Data.length > 0">
            <div class="search-cards">
              <div v-for="item in searchResults.Data" :key="item.Id" class="search-card" :class="{
                'search-card-playing': currentData.Id === item.Id
              }">
                <div class="search-card-thumb">
                  <q-img :src="item.pngUrl" fit="cover" class="search-card-img" :ratio="3 / 4"
                    @click="playFromSearch(item)">
                    <template v-slot:error>
                      <div class="search-card-placeholder">
                        <q-icon name="movie" color="grey-6" size="28px" />
                      </div>
                    </template>
                  </q-img>
                  <div class="search-card-play-overlay">
                    <q-icon name="play_circle_filled" size="28px" color="white" @click="playFromSearch(item)" />
                  </div>
                  <!-- 播放中指示器 -->
                  <div class="search-card-playing-indicator" v-if="currentData.Id === item.Id && videoLoaded">
                    <q-icon name="play_arrow" size="20px" color="white" />
                  </div>
                </div>

                <div class="search-card-info">
                  <div class="search-card-title">
                    {{ formatTitle(item.Title, 24) }}
                  </div>
                  <div class="search-card-tags">
                    <span class="tag tag-author" v-if="item.Author" @click="fetchKeyword(item.Author)">{{
                      item.Author?.substring(0, 10) }}</span>
                    <span class="tag tag-code" v-if="item.Code" @click="fetchKeyword(item.Code)">{{
                      item.Code.substring(0, 10) }}</span>
                    <template v-if="item.Tags">
                      <span v-for="(value, index) in item.Tags" :key="index" class="tag tag-level"
                        @click="fetchKeyword(value)" :style="{ background: getTagColor(index) }">{{
                          value }}</span>
                    </template>
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
                    <q-btn-dropdown dense flat size="md" color="indigo-4"
                      :label="`${item.MovieType === '无' ? '分类' : item.MovieType}`" no-caps>
                      <q-list style="min-width: 60px">
                        <q-item v-for="mt in MovieTypeOptions" :key="mt.value" clickable v-close-popup>
                          <q-item-section @click="setMovieType(item, mt.value)">{{ mt.label }}</q-item-section>
                        </q-item>
                      </q-list>
                    </q-btn-dropdown>
                    <q-btn flat dense color="primary" icon="edit" size="md" label="修改"
                      @click.stop="currentEditItem = item; fileEditRef.open(item)">
                      <q-tooltip>修改</q-tooltip>
                    </q-btn>
                    <q-btn flat dense color="negative" icon="delete" size="md" label="删除"
                      @click.stop="deleteVideo(item)">
                      <q-tooltip>删除</q-tooltip>
                    </q-btn>
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
        <div class="search-pagination" v-if="searchResults.TotalPage > 0">
          <q-pagination v-model="searchParams.Page" @update:model-value="fetchSearch" color="deep-orange"
            :ellipses="true" :max="searchResults.TotalPage || 0" :max-pages="isSmall ? 5 : 8" boundary-numbers
            direction-links></q-pagination>
          <span class="page-count">共 {{ searchResults.TotalCount }} 条</span>
          <q-select size="xs" dense flat @update:model-value="currentPageSizeChange" filled bgColor="orange"
            style="text-align: center; width: 70px" v-model="searchParams.PageSize" :options="pageOptions">
          </q-select>
          <q-input v-model.number="gotoPage" :dense="true" style="text-align: center; width: 60px" bgColor="orange"
            :max="searchResults.TotalPage" :min="1" @change="pageNoGoto" />
        </div>
      </div>

    </transition>

    <!-- 视频区域 -->
    <div class="video-wrapper" v-show="videoLoaded" @touchstart="onContainerTouchStart" @touchend="onContainerTouchEnd"
      @click="onDoubleTap" @wheel.prevent="onWheel">
      <video ref="videoRef" id="immersiveVideo" :src="currentVideoSrc" :poster="currentPoster" playsinline
        @timeupdate="onTimeUpdate" @loadedmetadata="onMetadataLoaded" @play="onPlay" @pause="onPause" @ended="onEnded"
        @waiting="onWaiting" @canplay="onCanPlay" @error="onVideoError" :style="videoStyle"></video>
      <!-- 缓冲 loading 遮罩 -->
      <transition name="fade">
        <div class="video-buffering" v-if="isBuffering">
          <q-spinner-gears size="56px" color="indigo-3" />
        </div>
      </transition>
    </div>

    <!-- 拖拽上传区域 -->
    <transition name="fade">
      <div v-if="!videoLoaded && !torrentLoading" class="drop-zone" :class="{ 'drop-zone-active': isDragOver }"
        @dragover.prevent="isDragOver = true" @dragleave="isDragOver = false" @drop="handleDrop">
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
        <div class="magnet-input-wrapper" :class="{ 'magnet-focused': magnetFocused }">
          <q-icon name="link" color="indigo-4" size="20px" class="magnet-icon" />
          <q-input v-model="magnetURI" placeholder="粘贴磁力链 magnet:?xt=urn:btih:..." dark dense borderless
            class="magnet-input" @keyup.enter="submitMagnet" @focus="magnetFocused = true"
            @blur="magnetFocused = false" />
          <q-btn flat round dense color="indigo-4" icon="play_circle_filled" size="md" @click="submitMagnet"
            :disable="!magnetURI.trim()" class="magnet-submit-btn">
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
            <div v-for="(file, index) in torrentFiles" :key="index" class="torrent-file-item" :class="{
              'torrent-file-selected': selectedTorrentFile === file.path,
            }" @click="selectTorrentFile(file)">
              <q-icon :name="getFileIcon(file.name)" size="20px" class="torrent-file-icon" />
              <div class="torrent-file-info">
                <span class="torrent-file-name">{{ file.name }}</span>
                <span class="torrent-file-size">{{
                  humanStorageSize(file.length)
                  }}</span>
              </div>
              <q-icon v-if="selectedTorrentFile === file.path" name="play_circle_filled" size="24px" color="indigo-4" />
            </div>
          </div>
          <div class="torrent-files-actions">
            <q-btn flat color="grey-5" label="取消" @click="cancelTorrent" />
            <q-btn unelevated color="indigo-6" label="播放选中文件" icon="play_arrow" :disable="!selectedTorrentFile"
              @click="playSelectedTorrentFile" />
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
            <q-linear-progress :value="torrentProgress / 100" color="indigo-5" track-color="grey-9" size="6px" rounded
              class="q-mb-sm" />
            <div class="torrent-stats">
              <span class="torrent-percent">{{ torrentProgress.toFixed(1) }}%</span>
              <span class="torrent-state">{{ torrentState }}</span>
              <span class="torrent-peers" v-if="torrentPeers > 0">
                <q-icon name="people" size="12px" />
                {{ torrentPeers }}
              </span>
            </div>
          </div>
          <q-btn unelevated color="red-9" text-color="red-3" label="取消下载" icon="cancel" size="sm" rounded
            @click="cancelTorrent" class="q-mt-md" />
        </div>
      </div>
    </transition>



    <!-- 下载管理器悬浮按钮 -->
    <q-btn v-if="activeDownloads.length > 0" round color="indigo-6" icon="download" class="download-fab"
      @click="showDownloadManager = true">
      <q-badge color="red" floating rounded>{{
        activeDownloads.length
        }}</q-badge>
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
            <div v-for="task in activeDownloads" :key="task.infoHash" class="download-item" :class="{
              'download-item-playing':
                task.infoHash === currentInfoHash && videoLoaded,
            }">
              <div class="download-item-info">
                <div class="download-item-name">{{ task.name }}</div>
                <div class="download-item-meta">
                  <span class="download-item-file" v-if="task.fileName">{{
                    task.fileName
                    }}</span>
                  <span class="download-item-state" :class="'state-' + task.state">{{ task.state }}</span>
                  <span class="download-item-percent">{{ task.progress.toFixed(1) }}%</span>
                </div>
                <q-linear-progress :value="task.progress / 100" color="indigo-5" track-color="grey-9" size="4px" rounded
                  class="q-mt-xs" />
              </div>
              <div class="download-item-actions">
                <q-btn flat round dense color="green" icon="play_arrow" size="sm" :disable="task.progress < 1"
                  @click="playDownloadTask(task)">
                  <q-tooltip>播放</q-tooltip>
                </q-btn>
                <q-btn flat round dense color="blue" icon="folder_open" size="sm" @click="openDownloadFolder(task)">
                  <q-tooltip>打开文件夹</q-tooltip>
                </q-btn>
                <q-btn flat round dense color="red" icon="close" label="删除" size="sm" @click="removeDownloadTask(task)">
                  <q-tooltip>删除</q-tooltip>
                </q-btn>
              </div>
            </div>
          </div>
        </q-card-section>
      </q-card>
    </q-dialog>

    <!-- 文件编辑对话框 -->
    <FileEdit ref="fileEditRef" @success="executeWithNextItem(currentEditItem, async () => { })" />
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
} from 'vue';
import { format, useQuasar } from 'quasar';
import { useRouter } from 'vue-router';
import axios from 'axios';
import { SearchAPI, DeleteFile, RefreshAPI, ResetMovieType, CutImage } from 'components/api/searchAPI';

import {
  MovieTypeSelects,
  MovieTypeOptions,
  FieldEnum,
  DescEnum,
  formatTitle,
} from 'components/utils';
import { useSystemProperty } from 'stores/System';
import PlayerSetting from 'components/PlayerSetting.vue';
import VideoCutParam from 'components/VideoCutParam.vue';
import EditVideoTag from 'components/EditVideoTag.vue';
import IndexButton from 'components/IndexButton.vue';
import FileEdit from '../file/components/FileEditDialog.vue';


const $q = useQuasar();
const router = useRouter();
const { humanStorageSize } = format;

// ── System Store ───────────────────────────────────────────────────────────────
const systemProperty = useSystemProperty();

// ── DOM refs ─────────────────────────────────────────────────────────────────
const videoRef = ref(null);
const particleCanvas = ref(null);
const progressBar = ref(null);
const fileEditRef = ref(null);
const currentEditItem = ref(null);

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
const currentData = ref({});
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

// ── 进度条拖拽 ────────────────────────────────────────────────────────────────
const isSeeking = ref(false);
const hoverTime = ref(null);
const hoverX = ref(0);
const seekButtonsVisible = ref(false);
const seekSeconds = [-60, -30, 30, 60, 120];

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
const searchResults = reactive({ Data: [], TotalPage: 0, ResultSize: '', TotalCount: 0 });
// 从 Pinia store 初始化搜索参数
const searchParams = reactive({
  ...systemProperty.FileSearchParam,
});
const pageOptions = ref([10, 20, 40, 60]);
const gotoPage = ref(1);

const pageNoGoto = (e) => {
  console.log('pageNoGoto', e);
  const page = Number(gotoPage.value);
  if (page && page >= 1 && page <= searchResults.TotalPage) {
    searchParams.Page = page;
    fetchSearch();
  }
};

const currentPageSizeChange = (size) => {
  if (size) {
    searchParams.PageSize = Number(size);
    searchParams.Page = 1;
    fetchSearch();
  }
};

// ── 粒子 / 音频 ───────────────────────────────────────────────────────────────
let audioContext = null;
let animationFrameId = null;
let particles = [];

// ── 计算属性 ──────────────────────────────────────────────────────────────────
const progressPercent = computed(() => {
  if (durationSeconds.value === 0) return 0;
  return (currentTimeSeconds.value / durationSeconds.value) * 100;
});

// 视频动态样式
const videoStyle = computed(() => {
  const opts = systemProperty.videoOptions;
  return {
    filter: `brightness(${opts.brightness / 100})`,
    objectFit: opts.playerMode || 'contain',
    transform: opts.rotate || 'none',
  };
});

const bufferedPercent = computed(() => {
  if (durationSeconds.value === 0) return 0;
  return (bufferedSeconds.value / durationSeconds.value) * 100;
});

const isSmall = computed(() => {
  return systemProperty.showStyle === 'sm' || $q.screen.lt.sm || $q.platform.is.mobile;
});

const sortOptions = computed(() => {
  const options = [];
  for (const field of FieldEnum) {
    for (const desc of DescEnum) {
      options.push({
        label: `${field.label} ${desc.label}`,
        value: `${field.value}_${desc.value}`
      });
    }
  }
  return options;
});

const currentSort = computed({
  get: () => `${searchParams.SortField}_${searchParams.SortType}`,
  set: (val) => {
    const [field, type] = val.split('_');
    searchParams.SortField = field;
    searchParams.SortType = type;
  }
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


document.addEventListener('contextmenu', function (e) {
  e.preventDefault(); // 阻止默认行为
  searchDialog.value = !searchDialog.value;
});

// ── 播放列表操作 ──────────────────────────────────────────────────────────────

function switchToItem(index) {
  if (index < 0 || index >= playlist.value.length) return;
  currentIndex.value = index;
  const item = playlist.value[index];
  const src = item.TorrentStream || item.streamUrl;
  loadVideo(
    src,
    item.Title || item.Name || item.Code || `#${index + 1}`,
    item.jpgUrl,
    item
  );
  searchDialog.value = false;
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

function forwardTime(seconds) {
  if (!videoRef.value) return;
  videoRef.value.currentTime = Math.max(0, videoRef.value.currentTime + seconds);
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

// ── 标签颜色 ──────────────────────────────────────────────────────────────
function getTagColor(tag) {
  const colorMap = {
    '0': '#ef4444',
    '1': '#f97316',
    '2': '#eab308',
    '3': '#22c55e',
    '4': '#ec4899',
    '5': '#8b5cf6',
    '6': '#3b82f6',
    '7': '#6b7280',
  };
  return colorMap[tag] || '#6b7280';
}

// ── 公共方法：切换到下一个后再执行操作 ───────────────────────────────────────
async function executeWithNextItem(item, action) {
  if (currentData.value.Id === item.Id) {
    nextItem();
    setTimeout(async () => {
      await action();
      await RefreshAPI(item.BaseDir);
      setTimeout(() => {
        fetchSearch();
      }, 500);
    }, 1000);
  } else {
    await action();
    await RefreshAPI(item.BaseDir);
    setTimeout(() => {
      fetchSearch();
    }, 500);
  }
}

// ── 删除视频 ────────────────────────────────────────────────────────────────
async function deleteVideo(item) {
  await executeWithNextItem(item, async () => {
    const res = await DeleteFile(item.Id);
    if (!res || res.Code !== 200) {
      $q.notify({ message: res?.Message || '删除失败', position: 'bottom-left' });
    }
  });
}

// ── 截图 ────────────────────────────────────────────────────────────────────
async function curImage(type) {
  const res = await CutImage(currentData.value.Id, type || 'shot', currentTime.value, false);
  if (res?.Code !== 200) {
    $q.notify({ message: res?.Message || '截图失败', position: 'bottom-left' });
  } else {
    $q.notify({ message: '截图成功', position: 'bottom-left' });
  }
}

// ── 设置类型 ────────────────────────────────────────────────────────────────
async function setMovieType(item, Type) {
  await executeWithNextItem(item, async () => {
    const res = await ResetMovieType(item.Id, Type);
    if (res?.Code === 200) {
      $q.notify({ type: 'negative', message: res.Message, position: 'bottom-left' });
    } else {
      $q.notify({ type: 'warning', message: res?.Message || '设置失败', position: 'bottom-left' });
    }
  });
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
      searchResults.TotalCount = data.ResultCnt || 0;
      // 搜索完成后同步到 Pinia store
      systemProperty.syncSearchParam(searchParams);
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

function animate() {
  if (!particleCanvas.value) return;
  const canvas = particleCanvas.value;
  const ctx = canvas.getContext('2d');
  ctx.fillStyle = 'rgba(8, 8, 14, 0.12)';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  particles.forEach((p) => {
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

function loadVideo(src, name, poster, item = {}) {
  currentVideoSrc.value = src;
  currentVideoName.value = name || '未知视频';
  currentPoster.value = poster || '';
  videoLoaded.value = true;
  currentData.value = item;

  if (videoRef.value) {

    // 监听 loadedmetadata，元数据加载完成后再播放（使用 once 避免重复绑定）
    videoRef.value.addEventListener('loadedmetadata', function onMeta() {
      // 确保元数据加载后音量仍然正确
      videoRef.value.muted = false;
      videoRef.value.volume = volume.value > 0 ? volume.value : 0.8;

      // 程序化 .play()，移动端必须在用户手势下调用才能带声音
      videoRef.value.play().catch((e) => {
        console.warn('Autoplay blocked on mobile, user interaction needed:', e.message);
      });

      if (!animationFrameId) animate();
    }, { once: true });
  }
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
    const code = res.data?.code ?? res.data?.Code;
    const data = res.data?.data ?? res.data?.Data;
    if (code === 200 && data) {
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
        message: res.data?.message ?? res.data?.Message ?? '添加磁力链失败',
        position: 'top',
      });
      torrentLoading.value = false;
    }
  } catch (err) {
    $q.notify({
      type: 'negative',
      message: err.response?.data?.message ?? err.response?.data?.Message ?? '请求失败: ' + err.message,
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
  const fileName =
    torrentFiles.value.find((f) => f.path === selectedTorrentFile.value)
      ?.name || '未知文件';
  try {
    const response = await axios.post('/api/torrent/startDownload', {
      infoHash: currentInfoHash.value,
      filePath: selectedTorrentFile.value,
    });
    const result = response.data?.data ?? response.data?.Data;
    const newTask = {
      infoHash: currentInfoHash.value,
      name: torrentName.value,
      fileName: fileName,
      filePath: selectedTorrentFile.value,
      progress: result?.skipped ? 100 : 0,
      state: result?.skipped ? '已下载' : '准备下载',
      peers: 0,
    };
    activeDownloads.value.push(newTask);
    if (!result?.skipped) {
      startPolling(currentInfoHash.value, newTask, selectedTorrentFile.value);
    }
    const streamUrl = `/api/torrent/stream/${currentInfoHash.value
      }?file=${encodeURIComponent(selectedTorrentFile.value)}`;
    loadVideo(streamUrl, fileName);
    if (result?.skipped) {
      $q.notify({
        type: 'positive',
        message: '文件已存在，无需下载',
        position: 'top',
        timeout: 2000,
      });
    }
  } catch (err) {
    $q.notify({
      type: 'negative',
      message: '启动下载失败: ' + ((err.response?.data?.message ?? err.response?.data?.Message) || err.message),
      position: 'top',
    });
  }
  torrentLoading.value = false;
  selectedTorrentFile.value = null;
}

function getFileIcon(fileName) {
  const ext = fileName.split('.').pop()?.toLowerCase();
  const videoExts = [
    'mp4',
    'mkv',
    'avi',
    'mov',
    'wmv',
    'flv',
    'webm',
    'm4v',
    'mpg',
    'mpeg',
  ];
  const audioExts = ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a'];
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'];
  if (videoExts.includes(ext)) return 'movie';
  if (audioExts.includes(ext)) return 'audio_file';
  if (imageExts.includes(ext)) return 'image';
  return 'insert_drive_file';
}

function startPolling(infoHash, task, filePath) {
  stopPolling();
  const pollStartTime = Date.now();
  const POLL_TIMEOUT_MS = 5 * 60 * 1000; // 5 分钟超时
  torrentPollTimer = setInterval(async () => {
    // 超时检测
    if (Date.now() - pollStartTime > POLL_TIMEOUT_MS) {
      stopPolling();
      $q.notify({
        type: 'warning',
        message: '下载超时，请检查网络或更换磁力链',
        position: 'top',
      });
      return;
    }
    try {
      const res = await axios.get(`/api/torrent/status/${infoHash}`);
      // 兼容大小写字段
      const d = res.data?.data ?? res.data?.Data;
      const code = res.data?.code ?? res.data?.Code;
      if (code === 200 && d) {
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
          const streamUrl = `/api/torrent/stream/${infoHash}?file=${encodeURIComponent(filePath)}`;
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
    activeDownloads.value = activeDownloads.value.filter(
      (t) => t.infoHash !== currentInfoHash.value
    );
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
  const streamUrl = `/api/torrent/stream/${task.infoHash
    }?file=${encodeURIComponent(task.filePath)}`;
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
  activeDownloads.value = activeDownloads.value.filter(
    (t) => t.infoHash !== task.infoHash
  );
  if (currentInfoHash.value === task.infoHash) {
    currentInfoHash.value = '';
    torrentLoading.value = false;
  }
}

// ── 播放控制 ──────────────────────────────────────────────────────────────────
function togglePlay() {
  if (!videoRef.value || !currentVideoSrc.value) {
    // 当前没有播放资源时，打开搜索弹窗
    searchDialog.value = !searchDialog.value;
    return;
  }
  if (isPlaying.value) {
    videoRef.value.pause();
  } else {
    // 确保取消静音
    videoRef.value.muted = false;
    videoRef.value.play().catch((e) => {
      console.warn('Play failed:', e.message);
    });
    // 恢复可能被挂起的音频上下文
    if (audioContext && audioContext.state === 'suspended') {
      audioContext.resume().catch((e) => console.warn('Failed to resume audio context:', e));
    }
  }
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
}
function onPause() {
  isPlaying.value = false;
}
function onWaiting() {
  isBuffering.value = true;
}
function onCanPlay() {
  isBuffering.value = false;
}

function onEnded() {
  isPlaying.value = false;
  if (
    playlist.value.length > 0 &&
    currentIndex.value < playlist.value.length - 1
  )
    nextItem();
}

// 添加视频错误处理
function onVideoError(e) {
  const video = e.target;
  // 如果视频 src 为空，则忽略错误（页面初始化时可能出现）
  if (!currentVideoSrc.value) {
    return;
  }

  const error = video?.error;
  let errorMsg = '视频播放失败';

  if (error) {
    switch (error.code) {
      case 1: // MEDIA_ERR_ABORTED
        errorMsg = '视频加载被中断';
        break;
      case 2: // MEDIA_ERR_NETWORK
        errorMsg = '网络错误，请检查网络连接';
        break;
      case 3: // MEDIA_ERR_DECODE
        errorMsg = '视频解码失败，文件可能损坏';
        break;
      case 4: // MEDIA_ERR_SRC_NOT_SUPPORTED
        errorMsg = '视频格式不支持';
        break;
      default:
        errorMsg = `视频播放失败 (错误码: ${error.code})`;
    }
    // 添加具体的错误信息
    if (error.message) {
      console.error('Video playback error:', error.code, error.message);
    } else {
      console.error('Video playback error:', error.code, errorMsg);
    }
  } else {
    console.error('Video playback error:', e);
  }

  if (currentInfoHash.value) {
    // 磁力链播放失败，提示等待缓冲
    $q.notify({
      type: 'warning',
      message: '缓冲不足，请等待下载进度增加',
      position: 'bottom-left',
      timeout: 3000,
    });
  } else {
    $q.notify({ type: 'negative', message: errorMsg, position: 'bottom-left' });
  }
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
  if (e.button === 2) return; // 右键不处理，由 onProgressContextMenu 处理
  isSeeking.value = true;
  seekButtonsVisible.value = true;
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

// ── 快进快退 ──────────────────────────────────────────────────────────────────
function onProgressContextMenu(e) {
  // 右键显示快进快退按钮
  if (!progressBar.value || !durationSeconds.value) return;
  e.preventDefault();
}

function seekBySeconds(sec, isRightClick = false) {
  if (!videoRef.value) return;
  // 右键点击时反转方向（快退）
  const delta = isRightClick ? -Math.abs(sec) : sec;
  const newTime = Math.max(0, Math.min(videoRef.value.duration, videoRef.value.currentTime + delta));
  videoRef.value.currentTime = newTime;
  seekButtonsVisible.value = false;
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
  // 确保不是静音状态
  videoRef.value.muted = false;
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


// ── 移动端触摸手势 ─────────────────────────────────────────────────────────
let touchStartX = 0;
let touchStartY = 0;
let touchStartTime = 0;
let touchControl = ref(false); // 是否在触摸控制条
let touchSeekBar = ref(false); // 是否在触摸进度条

function onContainerTouchStart(e) {
  if (e.touches.length !== 1) return;
  touchStartX = e.touches[0].clientX;
  touchStartY = e.touches[0].clientY;
  touchStartTime = videoRef.value?.currentTime || 0;
}

function onContainerTouchEnd(e) {
  const touch = e.changedTouches[0];
  const deltaX = touch.clientX - touchStartX;
  const deltaY = touch.clientY - touchStartY;
  const absDeltaX = Math.abs(deltaX);
  const absDeltaY = Math.abs(deltaY);

  // 如果是点击控制区域，不处理手势
  if (touchControl.value) return;

  // 判断是否为点击（移动距离小）
  if (absDeltaX < 30 && absDeltaY < 30) {
    // 点击：切换控制栏显示
    togglePlay();
    return;
  }

  // 判断滑动方向：水平滑动快进快退，垂直滑动上下切换视频
  if (absDeltaX > absDeltaY && absDeltaX > 50) {
    // 水平滑动：快进快退（滑动屏幕宽度的50% = 10秒）
    const seekDelta = (deltaX / window.innerWidth) * 200;
    if (videoRef.value) {
      videoRef.value.currentTime = Math.max(0, Math.min(
        videoRef.value.duration,
        touchStartTime + seekDelta
      ));
    }
    $q.notify({
      type: 'info',
      message: `快${deltaX > 0 ? '进' : '退'} ${seekDelta.toFixed(0)}秒`,
      position: 'center',
      timeout: 800,
    });
  } else if (absDeltaY > 50) {
    // 垂直滑动：上滑下一个，下滑上一个
    if (deltaY < 0) {
      nextItem();
    } else {
      prevItem();
    }
  }
}

// 鼠标滚轮：上滚上一个，下滚下一个（防抖 500ms）
let wheelTimer = null;
function onWheel(e) {
  if (touchControl.value) return;
  if (wheelTimer) return;
  wheelTimer = setTimeout(() => { wheelTimer = null; }, 500);

  if (e.deltaY > 0) {
    // 向下滚 → 下一个
    nextItem();
  } else {
    // 向上滚 → 上一个
    prevItem();
  }
}

function onSeekBarTouchStart(e) {
  touchSeekBar.value = true;
  e.stopPropagation();
}

function onSeekBarTouchMove(e) {
  if (!touchSeekBar.value || !progressBar.value) return;
  const touch = e.touches[0];
  const rect = progressBar.value.getBoundingClientRect();
  const pct = Math.min(1, Math.max(0, (touch.clientX - rect.left) / rect.width));
  if (videoRef.value) {
    videoRef.value.currentTime = pct * videoRef.value.duration;
  }
  e.stopPropagation();
}

function onSeekBarTouchEnd(e) {
  touchSeekBar.value = false;
  e.stopPropagation();
}

// 双击左右区域快进快退
let lastTap = 0;
function onDoubleTap(e) {
  const now = Date.now();
  const rect = e.target.getBoundingClientRect();
  const tapX = e.clientX - rect.left;
  const halfWidth = rect.width / 2;

  if (now - lastTap < 300) {
    // 双击
    if (tapX < halfWidth) {
      seekBySeconds(-10, false);
    } else {
      seekBySeconds(10, false);
    }
    lastTap = 0;
  } else {
    lastTap = now;
  }
}

function onMouseMove() {

  if (systemProperty.SettingInfo.Pages && systemProperty.SettingInfo.Pages.length > 0) {
    pageOptions.value = systemProperty.SettingInfo.Pages.map((item) => {
      return Number(item);
    });
  }
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
  display: flex;
  gap: 8px;
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

.fixed-top-right-btn:hover {
  background: rgba(99, 102, 241, 0.25) !important;
}

.fixed-top-center {
  position: fixed;
  top: 1px;
  left: 60px;
  z-index: 9;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px 18px;
  border-radius: 24px;
  max-width: 90vw;
}

.top-video-name {
  font-size: 0.95rem;
  color: rgba(255, 255, 255, 0.95);
  max-width: 500px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.top-video-tag {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 600;
}

.fixed-top-right-btns {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 1000;
  display: flex;
  flex-direction: row;
  justify-content: flex-end;
  gap: 8px;
}

.fixed-top-right-btn {
  z-index: 1000;
  background: rgba(255, 255, 255, 0.06) !important;
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s;
}

.edit-popup-content {
  display: flex;
  flex-direction: row;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(30, 30, 50, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.12);
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
  width: 100%;
  height: 100%;
  border-radius: 6px;
  box-shadow: 0 0 80px rgba(80, 60, 180, 0.25);
  transition: filter 0.3s ease, transform 0.3s ease;
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
  padding: 8px 12px 12px;
  background: linear-gradient(to top,
      rgba(6, 6, 16, 0.95) 0%,
      rgba(6, 6, 16, 0.6) 60%,
      transparent 100%);
  /* touch-action 支持触摸滑动 */
  touch-action: none;
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

/* ── 快进快退按钮 ─────────────────────────────────────────────────────────── */
.seek-buttons-popup {
  display: flex;
  gap: 4px;
  border-radius: 8px;
  padding: 6px 8px;
  backdrop-filter: blur(12px);
  z-index: 100;
}

.seek-btn {
  margin: 2px 4px;
  color: #c4b5fd;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 42px;
  text-align: center;
  align-items: center;
}

.seek-btn:hover {
  background: rgba(99, 102, 241, 0.5);
  color: #fff;
  transform: scale(1.1);
}

.seek-btn:active {
  transform: scale(0.95);
}

.seek-btn.seek-btn-rewind {
  color: #fca5a5;
}

.seek-btn.seek-btn-rewind:hover {
  background: rgba(239, 68, 68, 0.5);
  color: #fff;
}

/* ── 控制按钮行 ──────────────────────────────────────────────────────────── */
.control-buttons {
  display: flex;
  justify-content: space-between;
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
  flex-direction: column;
  align-items: center;
  position: relative;
}

.volume-slider {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  height: 120px;
  transition: height 0.3s ease, opacity 0.3s ease;
}

/* ── 搜索侧面板 ──────────────────────────────────────────────────────────── */
.search-panel {
  position: relative;
  height: 88vh;
  margin: 20px auto;
  width: 88%;
  border-radius: 20px;
  border: #10b981 1px solid;
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
}

.filter-item {
  display: flex;
  flex-direction: row;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
}

.search-input :deep(.q-field__control) {
  border-radius: 8px;
  width: 60vw;
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
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  justify-content: space-evenly;
}

.search-card {
  margin-bottom: 8px;
  display: flex;
  padding: 8px;
  border-radius: 12px;
  background: rgba(22, 22, 45, 0.55);
  border: 1px solid rgba(99, 102, 241, 0.12);
  cursor: pointer;
  transition: background 0.25s, border-color 0.25s, box-shadow 0.25s,
    transform 0.2s;
  position: relative;
  overflow: hidden;
  width: calc(100% - 20px);
  max-width: 334px;
}

.search-card::after {
  /* content: ''; */
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg,
      rgba(99, 102, 241, 0.07) 0%,
      transparent 60%);
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

/* 当前播放卡片高亮 */
.search-card-playing {
  border-color: rgba(139, 92, 246, 0.9) !important;
  background: rgba(45, 35, 80, 0.75) !important;
  box-shadow: 0 0 16px rgba(139, 92, 246, 0.45),
    0 0 32px rgba(99, 102, 241, 0.2);
}

.search-card-playing-indicator {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(139, 92, 246, 0.55);
  animation: playing-pulse 1.5s ease-in-out infinite;
}

@keyframes playing-pulse {

  0%,
  100% {
    background: rgba(139, 92, 246, 0.5);
  }

  50% {
    background: rgba(139, 92, 246, 0.3);
  }
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
  line-clamp: 2;
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

.tag-author {
  color: #a78bfa;
  background: rgba(139, 92, 246, 0.14);
  border: 1px solid rgba(139, 92, 246, 0.25);
}

.tag-code {
  color: #f472b6;
  background: rgba(244, 114, 182, 0.12);
  border: 1px solid rgba(244, 114, 182, 0.22);
}

.tag-level {
  color: #fff;
  font-weight: 600;
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
  gap: 8px;
  padding: 10px 20px;
  border-top: 1px solid rgba(99, 102, 241, 0.15);
  background: rgba(8, 8, 20, 0.6);
  flex-wrap: wrap;
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

.page-count {
  color: rgba(255, 255, 255, 0.5);
  margin-left: 4px;
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
    padding: 10px 12px 24px;
  }

  /* 移动端进度条更大 */
  .progress-track {
    height: 8px !important;
  }

  .progress-container {
    height: 40px;
    margin-bottom: 8px;
  }

  .progress-thumb {
    width: 20px !important;
    height: 20px !important;
    transform: translate(-50%, -50%) scale(1) !important;
  }

  .volume-slider {
    height: 100px;
  }

  .top-title {
    font-size: 0.85rem;
  }

  .fixed-top-center {
    max-width: 80vw;
    gap: 6px;
    display: flex;
    flex-wrap: wrap;
  }

  .top-video-name {
    max-width: 300px;
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

  /* 移动端紧凑布局：单行排列 */
  .control-buttons.mobile-compact {
    flex-direction: row;
    justify-content: space-between;
    gap: 4px;
  }

  .ctrl-left,
  .ctrl-right {
    display: flex;
    align-items: center;
    gap: 2px;
  }


  /* 移动端按钮更大 */
  .ctrl-left .q-btn,
  .ctrl-right .q-btn {
    min-width: 44px;
    min-height: 44px;
  }

  .play-btn {
    min-width: 56px !important;
    min-height: 56px !important;
  }

  .play-btn .q-icon {
    font-size: 32px;
  }

  .volume-group .volume-slider {
    display: none;
  }
}

/* 极小屏幕 (< 400px) */
@media (max-width: 400px) {
  .glass-panel {
    padding: 8px 8px 28px;
  }

  .ctrl-left .q-btn:nth-child(-n+3) {
    display: none;
  }

  .time-display {
    font-size: 0.7rem;
    margin-left: 4px;
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
