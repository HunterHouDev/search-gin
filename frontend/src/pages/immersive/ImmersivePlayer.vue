<template>
  <q-page class="immersive-container">
    <canvas ref="particleCanvas" class="particle-canvas"></canvas>

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

    <div v-if="!videoLoaded" class="drop-zone" @dragover.prevent @drop="handleDrop">
      <div class="drop-content">
        <q-icon name="movie" size="64px" color="grey-6"></q-icon>
        <p class="text-grey-5 text-h6 q-mt-md">拖拽视频文件到此处</p>
        <p class="text-grey-6">支持 MP4, MKV, AVI 等格式</p>
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
        <q-btn flat round color="white" size="lg" @click="togglePlay" :icon="isPlaying ? 'pause' : 'play_arrow'" />
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

    <div class="particle-info" v-if="videoLoaded">
      <q-badge color="purple" :label="'粒子: ' + particleCount"></q-badge>
    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useQuasar } from 'quasar';

const $q = useQuasar();

const videoRef = ref(null);
const particleCanvas = ref(null);
const progressBar = ref(null);

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
const particleCount = ref(0);

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
  const ctx = canvas.getContext('2d');
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;

  particles = [];
  const count = Math.min(400, Math.floor((canvas.width * canvas.height) / 5000));
  for (let i = 0; i < count; i++) {
    particles.push(new Particle(canvas));
  }
  particleCount.value = count;
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

function loadVideo(src, name) {
  currentVideoSrc.value = src;
  currentVideoName.value = name || '未知视频';
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

.particle-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1;
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
  font-family: 'Orbitron', sans-serif;
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
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9rem;
  color: #fff;
  transition: opacity 0.3s ease;
}

.volume-slider {
  width: 100px;
}

.particle-info {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 20;
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
}
</style>
