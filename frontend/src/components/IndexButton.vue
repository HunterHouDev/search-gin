<template>
  <q-btn
    :color="statusColor"
    title="索引"
    :size="props.size"
    :label="statusLabel"
    :dense="props.dense"
    :style="props.style"
    @click="refreshIndex"
    @mouseover="startHealthPolling"
    @mouseleave="stopHealthPolling"
    :loading="view.loading"
  >
    <template v-slot:loading>
      <q-spinner-facebook size="xs"></q-spinner-facebook>
      <template v-if="view.scanProgress && view.scanProgress.phase === 'scanning'">
        {{ `扫： ${view.scanProgress.completedDirs}/${view.scanProgress.totalDirs}` }}
      </template>
      <template v-else-if="view.scanProgress && view.scanProgress.phase === 'building'">
        {{ `索: ${view.scanProgress.processedBuckets}/${view.scanProgress.totalBuckets}` }}
      </template>
      <template v-else>
        {{ `S:${view.indexNumber}` }}
      </template>
    </template>
    <q-tooltip v-if="view.healthStatus" class="health-tooltip">
      <div class="health-status-row">
        <span :class="'health-status-badge health-' + view.healthStatus">{{ view.healthStatus }}</span>
      </div>

      <!-- 扫描阶段进度条 -->
      <template v-if="view.scanProgress && view.scanProgress.phase === 'scanning'">
        <div class="health-progress-section">
          <div class="health-progress-label">📂 {{ view.scanProgress.currentPhase }}</div>
          <q-linear-progress :value="scanProgressRatio" stripe size="18px" color="info" class="health-progress-bar q-mt-xs q-mb-xs">
            <div class="absolute-full flex flex-center text-white text-caption">
              {{ view.scanProgress.completedDirs }} / {{ view.scanProgress.totalDirs }} 目录
            </div>
          </q-linear-progress>
          <template v-if="view.scanProgress.currentDir">
            <div class="health-dir-text">当前: {{ view.scanProgress.currentDir }}</div>
          </template>
          <div class="health-dir-text">已扫描 {{ view.scanProgress.scannedFiles }} 个文件</div>
        </div>
      </template>

      <!-- 索引构建阶段进度条 -->
      <template v-else-if="view.scanProgress && view.scanProgress.phase === 'building'">
        <div class="health-progress-section">
          <div class="health-progress-label">⚙️ {{ view.scanProgress.currentPhase }}</div>
          <q-linear-progress :value="scanProgressRatio" stripe size="18px" color="warning" class="health-progress-bar q-mt-xs q-mb-xs">
            <div class="absolute-full flex flex-center text-white text-caption">
              {{ view.scanProgress.processedBuckets }} / {{ view.scanProgress.totalBuckets }} bucket
            </div>
          </q-linear-progress>
        </div>
      </template>

      <hr style="margin: 4px 0; border: none; border-top: 1px solid rgba(255,255,255,0.2)">
      <div class="health-grid">
        <span class="health-label">扫描目录</span>
        <span class="health-value">{{ view.bucketCount }} / {{ view.expectedDirs }}</span>
        <span class="health-label">文件总数</span>
        <span class="health-value">{{ view.totalCount }}</span>
        <span class="health-label">占用空间</span>
        <span class="health-value">{{ view.totalSizeStr }}</span>
        <span class="health-label">作者/标签</span>
        <span class="health-value">{{ view.actorCount }} / {{ view.tagCount }}</span>
        <span class="health-label">分类/系列</span>
        <span class="health-value">{{ view.typeCount }} / {{ view.seriesCount }}</span>
        <span class="health-label">上次扫描</span>
        <span class="health-value">{{ view.lastScanTime }}</span>
      </div>
      <hr v-if="view.recommendations && view.recommendations.length > 0" style="margin: 4px 0; border: none; border-top: 1px solid rgba(255,255,255,0.2)">
      <div v-for="(rec, idx) in view.recommendations" :key="idx" class="health-rec">⚠ {{ rec }}</div>
    </q-tooltip>
  </q-btn>
</template>

<script setup>
import { HeartBeatQuery, RefreshAPI, IndexHealthQuery } from 'components/api/searchAPI';
import { useQuasar } from 'quasar';
import { onMounted, onBeforeUnmount, reactive, computed } from 'vue';
const $q = useQuasar();
const emit = defineEmits(['refreshDone']);
const props = defineProps({
  style: {
    type: Object,
    default: () => {
      return { width: '4rem' };
    },
  },
  dense: { type: Boolean, default: false },
  size: { type: String, default: 'md' },
});
const view = reactive({
  indexNumber: 0,
  loading: false,
  heartBeatRetryCount: 0,
  currentHeartBeatInterval: 200,
  heartBeatTimer: null,
  healthStatus: '',
  totalCount: 0,
  totalSizeStr: '',
  lastScanTime: '',
  actorCount: 0,
  tagCount: 0,
  typeCount: 0,
  seriesCount: 0,
  recommendations: [],
  bucketCount: 0,
  expectedDirs: 0,
  healthPollTimer: null,
  scanProgress: null,
});

const statusColor = computed(() => {
  if (view.scanProgress && view.scanProgress.phase === 'scanning') return 'info';
  if (view.scanProgress && view.scanProgress.phase === 'building') return 'warning';
  if (view.loading) return 'orange';
  if (view.healthStatus === 'error') return 'negative';
  if (view.healthStatus === 'warning') return 'warning';
  if (view.healthStatus === 'healthy') return 'positive';
  return 'grey';
});

const statusLabel = computed(() => {
  if (view.scanProgress && view.scanProgress.phase === 'scanning') return '扫描中';
  if (view.scanProgress && view.scanProgress.phase === 'building') return '索引中';
  if (view.loading) return '扫描中';
  if (view.indexNumber > 0) return `S:${view.indexNumber}`;
  return '索~';
});

const scanProgressRatio = computed(() => {
  if (!view.scanProgress) return 0;
  const sp = view.scanProgress;
  if (sp.phase === 'scanning' && sp.totalDirs > 0) return sp.completedDirs / sp.totalDirs;
  if (sp.phase === 'building' && sp.totalBuckets > 0) return sp.processedBuckets / sp.totalBuckets;
  return 0;
});

const queryHealth = async () => {
  try {
    const health = await IndexHealthQuery();
    if (health) updateHealth(health);
  } catch (error) {
    console.error('IndexHealthQuery error:', error);
  }
};

const updateHealth = (health) => {
  view.healthStatus = health.status;
  view.totalCount = health.totalCount;
  view.totalSizeStr = health.totalSizeStr || '';
  view.lastScanTime = health.lastScanTime || '';
  view.actorCount = health.actorCount || 0;
  view.tagCount = health.tagCount || 0;
  view.typeCount = health.typeCount || 0;
  view.seriesCount = health.seriesCount || 0;
  view.recommendations = health.recommendations || [];
  view.bucketCount = health.bucketCount;
  view.expectedDirs = health.expectedDirs;
  view.scanProgress = health.scanProgress || null;
};

const BASE_INTERVAL = 200;
const MAX_INTERVAL = 20000;

const scheduleNextHeartBeat = () => {
  view.heartBeatTimer = setTimeout(async () => {
    try {
      const res = await HeartBeatQuery();
      view.indexNumber = res;

      if (res <= 0) {
        emit('refreshDone');
        view.heartBeatRetryCount = 0;
        view.currentHeartBeatInterval = BASE_INTERVAL;
        await queryHealth();
        return;
      }

      if (view.heartBeatRetryCount > 0) {
        view.heartBeatRetryCount = 0;
        view.currentHeartBeatInterval = BASE_INTERVAL;
      }
    } catch (error) {
      console.error('HeartBeatQuery error:', error);
      view.heartBeatRetryCount++;
      view.currentHeartBeatInterval = Math.min(BASE_INTERVAL * Math.pow(2, view.heartBeatRetryCount), MAX_INTERVAL);
    }

    scheduleNextHeartBeat();
  }, view.currentHeartBeatInterval);
};

const refreshProgress = () => {
  if (view.heartBeatTimer) {
    clearTimeout(view.heartBeatTimer);
  }
  view.heartBeatRetryCount = 0;
  view.currentHeartBeatInterval = BASE_INTERVAL;
  scheduleNextHeartBeat();
};

const refreshIndex = async (item) => {
  view.loading = true;
  refreshProgress();
  const { Code, Message } = await RefreshAPI(item.BaseDir);
  if (Code === 200) {
    $q.notify({
      type: 'positive',
      message: Message,
      position: 'bottom-left',
    });
  }
  view.loading = false;
};

onMounted(() => {
  refreshProgress();
  queryHealth();
});

defineExpose({
  refreshIndex,
  refreshProgress,
  queryHealth,
  updateHealth,
});
</script>

<style scoped>
.health-tooltip {
  font-size: 13px;
  line-height: 1.6;
  max-width: 320px;
}
.health-grid {
  display: grid;
  grid-template-columns: auto auto;
  gap: 2px 12px;
  white-space: nowrap;
}
.health-label {
  color: rgba(255,255,255,0.65);
}
.health-value {
  text-align: right;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.health-rec {
  color: #ffc107;
  font-size: 12px;
  padding: 2px 0;
}
.health-status-row {
  margin-bottom: 4px;
}
.health-status-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
}
.health-scanning, .health-building {
  background: #1976d2;
  color: #fff;
}
.health-warning {
  background: #f57c00;
  color: #fff;
}
.health-negative, .health-error {
  background: #d32f2f;
  color: #fff;
}
.health-positive, .health-healthy {
  background: #388e3c;
  color: #fff;
}
.health-progress-section {
  margin: 6px 0;
}
.health-progress-label {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 2px;
}
.health-progress-bar {
  border-radius: 4px;
}
.health-dir-text {
  font-size: 11px;
  color: rgba(255,255,255,0.65);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 280px;
}
</style>
