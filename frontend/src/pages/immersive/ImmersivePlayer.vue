<template>
  <div class="immersive-container">
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>

    <q-btn
      flat
      round
      color="white"
      icon="arrow_back"
      class="back-btn"
      @click="goBack"
    >
      <q-tooltip class="bg-white text-primary">返回</q-tooltip>
    </q-btn>

    <q-btn
      flat
      round
      color="white"
      icon="search"
      class="search-btn"
      @click="searchDialog = true"
    >
      <q-tooltip class="bg-white text-primary">搜索</q-tooltip>
    </q-btn>

    <div class="carousel-banner" :class="{ 'banner-hidden': controlsHidden && isPlaying }">
      <q-btn flat round dense color="white" icon="chevron_left" class="carousel-arrow carousel-arrow-left" @click="prevItem" />
      <div class="carousel-track" ref="carouselTrack">
        <div
          v-for="(item, index) in playlist"
          :key="item.Id || index"
          class="carousel-item"
          :class="{ 'carousel-item-active': index === currentIndex }"
          @click="switchToItem(index)"
        >
          <q-img
            :src="item.CoverUrl || getJpg(item.Id)"
            fit="cover"
            class="carousel-thumb"
          >
            <template v-slot:error>
              <div class="carousel-thumb-placeholder">
                <q-icon name="movie" size="24px" color="grey-6" />
              </div>
            </template>
          </q-img>
          <div class="carousel-item-label">{{ item.Title || item.Name || `#${index + 1}` }}</div>
        </div>
      </div>
      <q-btn flat round dense color="white" icon="chevron_right" class="carousel-arrow carousel-arrow-right" @click="nextItem" />
    </div>

    <div class="video-wrapper" v-show="videoLoaded">
      <video
        ref="videoRef"
        id="immersiveVideo"
        :src="currentVideoSrc"
        :poster="currentPoster"
        preload="auto"
        playsinline
        @timeupdate="onTimeUpdate"
        @loadedmetadata="onMetadataLoaded"
        @play="onPlay"
        @pause="onPause"
        @ended="onEnded"
      ></video>
    </div>

    <div v-if="!videoLoaded && !torrentLoading && playlist.length === 0" class="drop-zone" @dragover.prevent @drop="handleDrop">
      <div class="drop-content">
        <q-icon name="movie" size="64px" color="grey-6"></q-icon>
        <p class="text-grey-5 text-h6 q-mt-md">拖拽视频文件到此处</p>
        <p class="text-grey-6">支持 MP4, MKV, AVI 等格式</p>
      </div>
    </div>

    <div v-if="torrentLoading" class="torrent-loading">
      <div class="loading-ring">
        <q-spinner-gears size="80px" color="indigo-4" />
      </div>
      <div class="loading-info">
        <p class="text-white text-h6">{{ torrentName }}</p>
        <div class="progress-bar-container">
          <q-linear-progress
            :value="torrentProgress / 100"
            color="indigo-5"
            track-color="grey-9"
            size="8px"
            rounded
          />
        </div>
        <p class="text-grey-4 q-mt-sm">
          {{ torrentProgress.toFixed(1) }}% · {{ torrentState }}
          <span v-if="torrentPeers > 0"> · {{ torrentPeers }} 个节点</span>
        </p>
        <q-btn
          flat
          color="red-4"
          label="取消"
          size="sm"
          @click="cancelTorrent"
          class="q-mt-sm"
        />
      </div>
    </div>

    <div v-if="!videoLoaded && !torrentLoading && playlist.length === 0" class="magnet-input-area">
      <div class="magnet-input-wrapper">
        <q-input
          v-model="magnetURI"
          placeholder="粘贴磁力链 magnet:?xt=urn:btih:..."
          dark
          dense
          outlined
          color="indigo-5"
          class="magnet-input"
          @keyup.enter="submitMagnet"
        >
          <template v-slot:prepend>
            <q-icon name="link" color="indigo-4" />
          </template>
        </q-input>
        <q-btn
          flat
          round
          color="indigo-4"
          icon="play_circle_filled"
          size="lg"
          @click="submitMagnet"
          :disable="!magnetURI.trim()"
        >
          <q-tooltip class="bg-white text-primary">播放磁力链</q-tooltip>
        </q-btn>
      </div>
    </div>

    <div class="glass-panel" :class="{ 'panel-hidden': controlsHidden }" @mouseenter="showControls" @mouseleave="hideControls">
      <div class="video-info" v-if="currentVideoName">
        <p class="video-title">{{ currentVideoName }}</p>
      </div>

      <div class="progress-container" @click="seekVideo" ref="progressBar">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
          <div class="progress-handle" :style="{ left: progressPercent + '%' }"></div>
        </div>
      </div>

      <div class="control-buttons">
        <q-btn flat round color="white" size="sm" @click="prevItem" icon="skip_previous" />
        <q-btn flat round color="white" size="lg" @click="togglePlay" :icon="isPlaying ? 'pause' : 'play_arrow'" />
        <q-btn flat round color="white" size="sm" @click="nextItem" icon="skip_next" />
        <div class="time-display">
          <span>{{ currentTime }}</span>
          <span class="text-grey-5 q-mx-sm">/</span>
          <span class="text-grey-5">{{ duration }}</span>
        </div>
        <q-space />
        <q-btn flat round color="white" size="md" @click="toggleMute" :icon="volumeIcon" />
        <q-slider
          v-model="volume"
          :min="0"
          :max="1"
          :step="0.01"
          color="red"
          track-color="grey-8"
          class="volume-slider"
          @update:model-value="setVolume"
        />
        <q-btn flat round color="white" size="lg" @click="toggleFullscreen" :icon="isFullscreen ? 'fullscreen_exit' : 'fullscreen'" />
      </div>
    </div>

    <q-dialog v-model="searchDialog" position="right" full-height seamless>
      <div class="search-panel">
        <div class="search-panel-header">
          <div class="search-panel-title">
            <q-icon name="search" size="20px" class="q-mr-sm" />
            搜索
          </div>
          <q-btn flat round dense color="white" icon="close" @click="searchDialog = false" />
        </div>

        <div class="search-conditions">
          <q-input
            v-model="searchParams.Keyword"
            placeholder="关键词搜索..."
            dark
            dense
            outlined
            color="indigo-4"
            class="search-input"
            @keyup.enter="fetchSearch"
          >
            <template v-slot:prepend>
              <q-icon name="search" color="indigo-4" />
            </template>
          </q-input>

          <div class="search-condition-row">
            <div class="condition-label">类型</div>
            <q-btn-toggle
              v-model="searchParams.MovieType"
              :options="MovieTypeSelects"
              size="xs"
              no-caps
              dense
              glossy
              toggle-color="indigo-6"
              color="dark"
              text-color="grey-4"
              @update:model-value="fetchSearch"
            />
          </div>

          <div class="search-condition-row">
            <div class="condition-label">排序</div>
            <q-btn-toggle
              v-model="searchParams.SortField"
              :options="FieldEnum"
              size="xs"
              no-caps
              dense
              glossy
              toggle-color="indigo-6"
              color="dark"
              text-color="grey-4"
              @update:model-value="fetchSearch"
            />
          </div>

          <div class="search-condition-row">
            <div class="condition-label">顺序</div>
            <q-btn-toggle
              v-model="searchParams.SortType"
              :options="DescEnum"
              size="xs"
              no-caps
              dense
              glossy
              toggle-color="indigo-6"
              color="dark"
              text-color="grey-4"
              @update:model-value="fetchSearch"
            />
          </div>

          <div class="search-condition-row">
            <q-checkbox
              v-model="searchParams.OnlyRepeat"
              label="去重"
              dense
              dark
              color="indigo-5"
              @update:model-value="fetchSearch"
            />
          </div>
        </div>

        <div class="search-results" ref="searchResultsRef">
          <div v-if="searchLoading" class="search-loading">
            <q-spinner-gears size="40px" color="indigo-4" />
          </div>
          <div v-else-if="searchResults.Data && searchResults.Data.length > 0" class="search-cards">
            <div
              v-for="item in searchResults.Data"
              :key="item.Id"
              class="search-card"
              @click="playFromSearch(item)"
            >
              <q-img
                :src="getJpg(item.Id)"
                fit="cover"
                class="search-card-img"
              >
                <template v-slot:error>
                  <div class="search-card-placeholder">
                    <q-icon name="movie" color="grey-6" />
                  </div>
                </template>
                <div class="search-card-overlay">
                  <q-icon name="play_circle_filled" size="32px" color="white" />
                </div>
              </q-img>
              <div class="search-card-info">
                <div class="search-card-title">{{ formatTitle(item.Title, 20) }}</div>
                <div class="search-card-meta">
                  <span class="meta-actress">{{ item.Actress }}</span>
                  <span class="meta-code">{{ item.Code }}</span>
                </div>
                <div class="search-card-meta">
                  <span class="meta-size">{{ humanStorageSize(item.Size) }}</span>
                  <span class="meta-time">{{ getTimeAgo(item.MTime) }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="search-empty">
            <q-icon name="search_off" size="48px" color="grey-7" />
            <p class="text-grey-6 q-mt-sm">暂无结果</p>
          </div>
        </div>

        <div class="search-pagination" v-if="searchResults.TotalPage > 0">
          <q-btn
            flat
            dense
            color="indigo-4"
            icon="chevron_left"
            :disable="searchParams.Page <= 1"
            @click="searchParams.Page--; fetchSearch()"
          />
          <span class="pagination-text">{{ searchParams.Page }} / {{ searchResults.TotalPage }}</span>
          <q-btn
            flat
            dense
            color="indigo-4"
            icon="chevron_right"
            :disable="searchParams.Page >= searchResults.TotalPage"
            @click="searchParams.Page++; fetchSearch()"
          />
        </div>
      </div>
    </q-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, reactive, watch, nextTick } from 'vue';
import { format, useQuasar } from 'quasar';
import { useRouter } from 'vue-router';
import axios from 'axios';
import { SearchAPI } from 'components/api/searchAPI';
import { getJpg, getFileStream } from 'components/utils/images';
import {
  MovieTypeSelects,
  FieldEnum,
  DescEnum,
  formatTitle,
} from 'components/utils';

const $q = useQuasar();
const router = useRouter();

const { humanStorageSize } = format;

const videoRef = ref(null);
const particleCanvas = ref(null);
const progressBar = ref(null);
const carouselTrack = ref(null);
const searchResultsRef = ref(null);

const currentVideoSrc = ref('');
const currentPoster = ref('');
const currentVideoName = ref('');
const videoLoaded = ref(false);
const isPlaying = ref(false);
const isFullscreen = ref(false);
const currentTime = ref('00:00:00');
const duration = ref('00:00:00');
const volume = ref(0.8);
const currentTimeSeconds = ref(0);
const durationSeconds = ref(0);
const controlsHidden = ref(false);

const magnetURI = ref('');
const torrentLoading = ref(false);
const torrentName = ref('');
const torrentProgress = ref(0);
const torrentState = ref('');
const torrentPeers = ref(0);
const currentInfoHash = ref('');
let torrentPollTimer = null;

const playlist = reactive([]);
const currentIndex = ref(-1);

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

let audioContext = null;
let analyser = null;
let animationFrameId = null;
let particles = [];
let hideControlsTimer = null;

const progressPercent = computed(() => {
  if (durationSeconds.value === 0) return 0;
  return (currentTimeSeconds.value / durationSeconds.value) * 100;
});

const volumeIcon = computed(() => {
  if (volume.value === 0) return 'volume_off';
  if (volume.value < 0.3) return 'volume_mute';
  if (volume.value < 0.7) return 'volume_down';
  return 'volume_up';
});

function goBack() {
  router.back();
}

function addToPlaylist(item) {
  const exists = playlist.some(p => p.Id === item.Id);
  if (!exists) {
    playlist.push(item);
  }
}

function switchToItem(index) {
  if (index < 0 || index >= playlist.length) return;
  currentIndex.value = index;
  const item = playlist[index];
  const src = item.TorrentStream || getFileStream(item.Id);
  loadVideo(src, item.Title || item.Name || item.Code || `#${index + 1}`, getJpg(item.Id));
  scrollToActiveItem();
}

function prevItem() {
  if (playlist.length === 0) return;
  const newIdx = currentIndex.value > 0 ? currentIndex.value - 1 : playlist.length - 1;
  switchToItem(newIdx);
}

function nextItem() {
  if (playlist.length === 0) return;
  const newIdx = currentIndex.value < playlist.length - 1 ? currentIndex.value + 1 : 0;
  switchToItem(newIdx);
}

function scrollToActiveItem() {
  nextTick(() => {
    if (!carouselTrack.value) return;
    const activeEl = carouselTrack.value.querySelector('.carousel-item-active');
    if (activeEl) {
      activeEl.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' });
    }
  });
}

async function playFromSearch(item) {
  addToPlaylist(item);
  const newIdx = playlist.findIndex(p => p.Id === item.Id);
  switchToItem(newIdx);
}

async function fetchSearch() {
  if (searchLoading.value) return;
  searchLoading.value = true;
  try {
    const data = await SearchAPI(searchParams);
    if (data) {
      searchResults.Data = data.Data || [];
      searchResults.TotalPage = data.TotalPage || 0;
      searchResults.ResultSize = data.ResultSize || '';
    }
  } catch (e) {
    console.error('搜索请求异常:', e);
    $q.notify({ type: 'negative', message: '搜索失败', position: 'top' });
  } finally {
    searchLoading.value = false;
  }
}

const today = new Date();
function getTimeAgo(MTime) {
  if (!MTime) return '';
  const days = Math.floor((today - new Date(MTime)) / (1000 * 60 * 60 * 24));
  if (days > 365) return `${Math.floor(days / 365)}年`;
  if (days > 30) return `${Math.floor(days / 30)}月`;
  if (days > 0) return `${days}天`;
  return '今天';
}

class Particle {
  constructor(canvas) {
    this.canvas = canvas;
    this.reset();
  }

  reset() {
    this.x = Math.random() * this.canvas.width;
    this.y = Math.random() * this.canvas.height;
    this.size = Math.random() * 3 + 1;
    this.baseSize = this.size;
    this.speedX = (Math.random() - 0.5) * 0.5;
    this.speedY = (Math.random() - 0.5) * 0.5;
    this.opacity = Math.random() * 0.5 + 0.3;
    this.hue = Math.random() * 60 + 240;
    this.pulsePhase = Math.random() * Math.PI * 2;
    this.pulseSpeed = Math.random() * 0.02 + 0.01;
  }

  update(audioData = null) {
    this.pulsePhase += this.pulseSpeed;
    const pulse = Math.sin(this.pulsePhase) * 0.5 + 0.5;

    if (audioData) {
      const bass = audioData.bass || 0;
      const mid = audioData.mid || 0;
      const treble = audioData.treble || 0;

      this.size = this.baseSize + bass * 5 + pulse * 2;
      this.speedX *= 1 + bass * 0.1;
      this.speedY *= 1 + bass * 0.1;
      this.opacity = Math.min(1, 0.3 + bass * 0.5 + pulse * 0.3);
      this.hue = 240 + mid * 60 + treble * 30;
    } else {
      this.size = this.baseSize + pulse * 1.5;
      this.opacity = 0.3 + pulse * 0.3;
    }

    this.x += this.speedX;
    this.y += this.speedY;

    if (this.x < 0 || this.x > this.canvas.width) this.speedX *= -1;
    if (this.y < 0 || this.y > this.canvas.height) this.speedY *= -1;
  }

  draw(ctx) {
    ctx.save();
    ctx.globalAlpha = this.opacity;
    ctx.shadowBlur = 15;
    ctx.shadowColor = `hsl(${this.hue}, 80%, 60%)`;
    ctx.fillStyle = `hsl(${this.hue}, 80%, 70%)`;
    ctx.beginPath();
    ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
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
  const count = Math.min(400, Math.floor((canvas.width * canvas.height) / 5000));
  for (let i = 0; i < count; i++) {
    particles.push(new Particle(canvas));
  }
}

function initAudioAnalyser() {
  if (!videoRef.value) return;
  try {
    audioContext = new (window.AudioContext || window.webkitAudioContext)();
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 256;
    const source = audioContext.createMediaElementSource(videoRef.value);
    source.connect(analyser);
    analyser.connect(audioContext.destination);
  } catch (e) {
    console.warn('Audio analyser not available:', e);
  }
}

function getAudioData() {
  if (!analyser) return null;
  const bufferLength = analyser.frequencyBinCount;
  const dataArray = new Uint8Array(bufferLength);
  analyser.getByteFrequencyData(dataArray);

  const bass = dataArray.slice(0, 10).reduce((a, b) => a + b, 0) / 10 / 255;
  const mid = dataArray.slice(10, 40).reduce((a, b) => a + b, 0) / 30 / 255;
  const treble = dataArray.slice(40, bufferLength).reduce((a, b) => a + b, 0) / (bufferLength - 40) / 255;

  return { bass, mid, treble };
}

function animate() {
  if (!particleCanvas.value) return;
  const canvas = particleCanvas.value;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = 'rgba(10, 10, 15, 0.1)';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  const audioData = isPlaying.value ? getAudioData() : null;

  particles.forEach(particle => {
    particle.update(audioData);
    particle.draw(ctx);
  });

  animationFrameId = requestAnimationFrame(animate);
}

function parseTime(seconds) {
  if (isNaN(seconds)) return '00:00:00';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

function handleDrop(e) {
  e.preventDefault();
  const file = e.dataTransfer.files[0];
  if (file && file.type.startsWith('video/')) {
    loadVideo(URL.createObjectURL(file), file.name);
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
      initAudioAnalyser();
      if (!animationFrameId) {
        animate();
      }
    }
  }, 100);
}

async function submitMagnet() {
  const uri = magnetURI.value.trim();
  if (!uri.startsWith('magnet:')) {
    $q.notify({ type: 'negative', message: '请输入有效的磁力链', position: 'top' });
    return;
  }

  torrentLoading.value = true;
  torrentProgress.value = 0;
  torrentState.value = '正在连接...';
  torrentName.value = '获取种子信息中...';

  try {
    const res = await axios.post('/api/torrent/add', { magnetURI: uri });
    if (res.data && res.data.code === 200) {
      currentInfoHash.value = res.data.data.infoHash;
      startPolling(currentInfoHash.value);
    } else {
      $q.notify({ type: 'negative', message: res.data?.message || '添加磁力链失败', position: 'top' });
      torrentLoading.value = false;
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: '请求失败: ' + (err.response?.data?.message || err.message), position: 'top' });
    torrentLoading.value = false;
  }
}

function startPolling(infoHash) {
  stopPolling();
  torrentPollTimer = setInterval(async () => {
    try {
      const res = await axios.get(`/api/torrent/status/${infoHash}`);
      if (res.data && res.data.code === 200) {
        const data = res.data.data;
        torrentName.value = data.name;
        torrentProgress.value = data.progress;
        torrentState.value = data.state;
        torrentPeers.value = data.peers;

        if (data.progress >= 3 && !videoLoaded.value) {
          torrentState.value = '缓冲就绪，开始播放';
          const streamUrl = `/api/torrent/stream/${infoHash}`;
          const torrentItem = {
            Id: infoHash,
            Title: data.videoFile || data.name,
            Name: data.videoFile || data.name,
            TorrentStream: streamUrl,
            CoverUrl: '',
          };
          addToPlaylist(torrentItem);
          const newIdx = playlist.findIndex(p => p.Id === infoHash);
          currentIndex.value = newIdx;
          loadVideo(streamUrl, data.videoFile || data.name);
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
    } catch (err) {
      console.warn('取消下载失败:', err);
    }
  }
  torrentLoading.value = false;
  torrentProgress.value = 0;
  torrentState.value = '';
  torrentName.value = '';
  currentInfoHash.value = '';
}

function togglePlay() {
  if (!videoRef.value) return;
  if (isPlaying.value) {
    videoRef.value.pause();
  } else {
    videoRef.value.play();
  }
}

function onPlay() {
  isPlaying.value = true;
  resetControlsTimer();
}

function onPause() {
  isPlaying.value = false;
}

function onEnded() {
  isPlaying.value = false;
  controlsHidden.value = false;
  if (playlist.length > 0 && currentIndex.value < playlist.length - 1) {
    nextItem();
  }
}

function onTimeUpdate() {
  if (!videoRef.value) return;
  currentTimeSeconds.value = videoRef.value.currentTime;
  currentTime.value = parseTime(videoRef.value.currentTime);
}

function onMetadataLoaded() {
  if (!videoRef.value) return;
  durationSeconds.value = videoRef.value.duration;
  duration.value = parseTime(videoRef.value.duration);
}

function seekVideo(e) {
  if (!videoRef.value || !progressBar.value) return;
  const rect = progressBar.value.getBoundingClientRect();
  const percent = (e.clientX - rect.left) / rect.width;
  videoRef.value.currentTime = percent * durationSeconds.value;
}

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
  videoRef.value.volume = val;
}

function toggleFullscreen() {
  const elem = document.documentElement;
  if (!document.fullscreenElement) {
    elem.requestFullscreen().then(() => {
      isFullscreen.value = true;
    });
  } else {
    document.exitFullscreen().then(() => {
      isFullscreen.value = false;
    });
  }
}

function showControls() {
  controlsHidden.value = false;
  if (hideControlsTimer) {
    clearTimeout(hideControlsTimer);
    hideControlsTimer = null;
  }
}

function hideControls() {
  if (isPlaying.value) {
    hideControlsTimer = setTimeout(() => {
      controlsHidden.value = true;
    }, 3000);
  }
}

function resetControlsTimer() {
  if (hideControlsTimer) {
    clearTimeout(hideControlsTimer);
  }
  hideControlsTimer = setTimeout(() => {
    controlsHidden.value = true;
  }, 3000);
}

function handleResize() {
  if (particleCanvas.value) {
    particleCanvas.value.width = window.innerWidth;
    particleCanvas.value.height = window.innerHeight;
  }
}

watch(searchDialog, (val) => {
  if (val && searchResults.Data.length === 0) {
    fetchSearch();
  }
});

onMounted(() => {
  initParticles();
  animate();

  document.addEventListener('fullscreenchange', () => {
    isFullscreen.value = !!document.fullscreenElement;
  });

  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId);
  }
  if (hideControlsTimer) {
    clearTimeout(hideControlsTimer);
  }
  if (audioContext) {
    audioContext.close();
  }
  stopPolling();
  if (currentInfoHash.value) {
    axios.delete(`/api/torrent/${currentInfoHash.value}`).catch(() => { /* ignore */ });
  }
  document.removeEventListener('resize', handleResize);
});

watch(isPlaying, (playing) => {
  if (playing) {
    resetControlsTimer();
  } else {
    controlsHidden.value = false;
    if (hideControlsTimer) {
      clearTimeout(hideControlsTimer);
    }
  }
});
</script>

<style scoped>
.immersive-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: radial-gradient(ellipse at center, #1a1a2e 0%, #0a0a0f 100%);
  overflow: hidden;
}

.back-btn {
  position: absolute;
  top: 20px;
  left: 20px;
  z-index: 30;
  background: rgba(15, 15, 25, 0.5);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(99, 102, 241, 0.3);
  transition: all 0.3s ease;
}

.back-btn:hover {
  background: rgba(99, 102, 241, 0.3);
  border-color: rgba(99, 102, 241, 0.6);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
}

.search-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 30;
  background: rgba(15, 15, 25, 0.5);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(99, 102, 241, 0.3);
  transition: all 0.3s ease;
}

.search-btn:hover {
  background: rgba(99, 102, 241, 0.3);
  border-color: rgba(99, 102, 241, 0.6);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
}

.particle-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1;
}

.carousel-banner {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  padding: 10px 60px;
  height: 80px;
  background: rgba(10, 10, 20, 0.6);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(99, 102, 241, 0.2);
  transition: all 0.4s ease;
}

.banner-hidden {
  transform: translateY(-100%);
}

.carousel-track {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  scroll-behavior: smooth;
  scroll-snap-type: x mandatory;
  flex: 1;
  padding: 4px 0;
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.carousel-track::-webkit-scrollbar {
  display: none;
}

.carousel-item {
  flex-shrink: 0;
  width: 52px;
  height: 60px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  scroll-snap-align: center;
  border: 2px solid transparent;
  transition: all 0.3s ease;
  position: relative;
}

.carousel-item:hover {
  border-color: rgba(99, 102, 241, 0.5);
  transform: scale(1.08);
}

.carousel-item-active {
  border-color: rgba(139, 92, 246, 0.9);
  box-shadow: 0 0 12px rgba(139, 92, 246, 0.6), 0 0 24px rgba(99, 102, 241, 0.3);
  transform: scale(1.1);
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
  background: rgba(30, 30, 50, 0.8);
}

.carousel-item-label {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  font-size: 8px;
  color: white;
  text-align: center;
  padding: 1px 2px;
  background: rgba(0, 0, 0, 0.7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.carousel-arrow {
  position: absolute;
  z-index: 21;
  background: rgba(15, 15, 25, 0.5);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.carousel-arrow:hover {
  background: rgba(99, 102, 241, 0.3);
}

.carousel-arrow-left {
  left: 8px;
}

.carousel-arrow-right {
  right: 8px;
}

.video-wrapper {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 85vw;
  height: 80vh;
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
  border-radius: 8px;
  box-shadow: 0 0 60px rgba(99, 102, 241, 0.3);
}

.drop-zone {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 60vw;
  height: 50vh;
  border: 3px dashed rgba(99, 102, 241, 0.5);
  border-radius: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 2;
  transition: all 0.3s ease;
}

.drop-zone:hover {
  border-color: #6366f1;
  background: rgba(99, 102, 241, 0.1);
  box-shadow: 0 0 40px rgba(99, 102, 241, 0.3);
}

.drop-content {
  text-align: center;
}

.magnet-input-area {
  position: absolute;
  bottom: 100px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 5;
  width: 70vw;
  max-width: 700px;
}

.magnet-input-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(15, 15, 25, 0.7);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 16px;
  padding: 8px 16px;
  transition: all 0.3s ease;
}

.magnet-input-wrapper:hover {
  border-color: rgba(99, 102, 241, 0.6);
  box-shadow: 0 0 30px rgba(99, 102, 241, 0.2);
}

.magnet-input {
  flex: 1;
}

.magnet-input :deep(.q-field__control) {
  background: transparent;
}

.magnet-input :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.9rem;
}

.magnet-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.4);
}

.torrent-loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 5;
  text-align: center;
}

.loading-ring {
  margin-bottom: 24px;
}

.loading-info {
  background: rgba(15, 15, 25, 0.7);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 16px;
  padding: 24px 32px;
  min-width: 320px;
}

.progress-bar-container {
  margin-top: 12px;
}

.glass-panel {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 10;
  padding: 20px 30px 30px;
  background: rgba(15, 15, 25, 0.75);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-top: 1px solid rgba(99, 102, 241, 0.3);
  transition: all 0.4s ease;
}

.panel-hidden {
  transform: translateY(calc(100% - 60px));
}

.panel-hidden .video-info,
.panel-hidden .progress-container,
.panel-hidden .time-display,
.panel-hidden .volume-slider {
  opacity: 0;
  pointer-events: none;
}

.video-info {
  margin-bottom: 15px;
  transition: opacity 0.3s ease;
}

.video-title {
  font-size: 1.1rem;
  color: #fff;
  margin: 0;
  text-shadow: 0 0 10px rgba(99, 102, 241, 0.5);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.progress-container {
  height: 30px;
  display: flex;
  align-items: center;
  cursor: pointer;
  margin-bottom: 15px;
  transition: opacity 0.3s ease;
}

.progress-bar {
  position: relative;
  width: 100%;
  height: 6px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 3px;
  overflow: visible;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #6366f1, #8b5cf6, #f472b6);
  border-radius: 3px;
  transition: width 0.1s linear;
  box-shadow: 0 0 10px rgba(99, 102, 241, 0.5);
}

.progress-handle {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 16px;
  height: 16px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 0 10px rgba(99, 102, 241, 0.8);
  transition: transform 0.2s ease;
}

.progress-container:hover .progress-handle {
  transform: translate(-50%, -50%) scale(1.3);
}

.control-buttons {
  display: flex;
  align-items: center;
  gap: 15px;
}

.time-display {
  font-size: 0.9rem;
  color: #fff;
  transition: opacity 0.3s ease;
}

.volume-slider {
  width: 100px;
}

.search-panel {
  width: 380px;
  max-width: 90vw;
  height: 100vh;
  background: rgba(12, 12, 25, 0.85);
  backdrop-filter: blur(30px);
  -webkit-backdrop-filter: blur(30px);
  border-left: 1px solid rgba(99, 102, 241, 0.3);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.2);
}

.search-panel-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: #c4b5fd;
  display: flex;
  align-items: center;
}

.search-conditions {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.15);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.search-input :deep(.q-field__control) {
  background: rgba(30, 30, 50, 0.6);
}

.search-input :deep(.q-field__native) {
  color: #c4b5fd;
}

.search-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.4);
}

.search-condition-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.condition-label {
  font-size: 12px;
  color: #818cf8;
  min-width: 30px;
}

.search-results {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  -ms-overflow-style: none;
  scrollbar-width: thin;
  scrollbar-color: rgba(99, 102, 241, 0.3) transparent;
}

.search-results::-webkit-scrollbar {
  width: 4px;
}

.search-results::-webkit-scrollbar-thumb {
  background: rgba(99, 102, 241, 0.3);
  border-radius: 2px;
}

.search-loading {
  display: flex;
  justify-content: center;
  padding: 40px 0;
}

.search-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.search-card {
  display: flex;
  gap: 12px;
  padding: 10px;
  border-radius: 12px;
  background: rgba(30, 30, 50, 0.5);
  border: 1px solid rgba(99, 102, 241, 0.15);
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.search-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(99, 102, 241, 0.1), transparent);
  transition: left 0.5s ease;
}

.search-card:hover::before {
  left: 100%;
}

.search-card:hover {
  border-color: rgba(139, 92, 246, 0.5);
  background: rgba(40, 40, 60, 0.6);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.2);
  transform: translateX(4px);
}

.search-card-img {
  width: 80px;
  height: 110px;
  border-radius: 8px;
  flex-shrink: 0;
  overflow: hidden;
}

.search-card-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(30, 30, 50, 0.8);
}

.search-card-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 80px;
  height: 110px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.3);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.search-card:hover .search-card-overlay {
  opacity: 1;
}

.search-card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow: hidden;
}

.search-card-title {
  font-size: 13px;
  color: #e0e7ff;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.search-card-meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  flex-wrap: wrap;
}

.meta-actress {
  color: #a78bfa;
  background: rgba(139, 92, 246, 0.15);
  padding: 1px 6px;
  border-radius: 4px;
}

.meta-code {
  color: #f472b6;
  background: rgba(244, 114, 182, 0.15);
  padding: 1px 6px;
  border-radius: 4px;
}

.meta-size {
  color: #67e8f9;
  background: rgba(103, 232, 249, 0.1);
  padding: 1px 6px;
  border-radius: 4px;
}

.meta-time {
  color: #86efac;
  background: rgba(134, 239, 172, 0.1);
  padding: 1px 6px;
  border-radius: 4px;
}

.search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
}

.search-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 12px 20px;
  border-top: 1px solid rgba(99, 102, 241, 0.2);
  background: rgba(12, 12, 25, 0.5);
}

.pagination-text {
  color: #a5b4fc;
  font-size: 13px;
  min-width: 60px;
  text-align: center;
}

@media (max-width: 768px) {
  .video-wrapper {
    width: 95vw;
    height: 60vh;
  }

  .glass-panel {
    padding: 15px 20px 25px;
  }

  .volume-slider {
    width: 70px;
  }

  .video-title {
    font-size: 0.9rem;
  }

  .magnet-input-area {
    width: 90vw;
    bottom: 80px;
  }

  .search-panel {
    width: 100vw;
    max-width: 100vw;
  }

  .carousel-banner {
    padding: 8px 50px;
    height: 70px;
  }

  .carousel-item {
    width: 44px;
    height: 52px;
  }
}
</style>
