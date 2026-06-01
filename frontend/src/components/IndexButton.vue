<template>
  <q-btn
    :color="statusColor"
    title="索引"
    :size="props.size"
    :label="statusLabel"
    :dense="props.dense"
    :style="props.style"
    @click="refreshIndex"
    :loading="view.loading"
  >
    <template v-slot:loading>
      <q-spinner-facebook size="xs"></q-spinner-facebook>
      {{ `S:${view.indexNumber}` }}
    </template>
    <q-tooltip v-if="view.healthStatus">
      索引状态: {{ view.healthStatus }}
      <br>文件数: {{ view.totalCount }}
      <br v-if="view.recommendations && view.recommendations.length > 0">
      <span v-for="(rec, idx) in view.recommendations" :key="idx">{{ rec }}<br/></span>
    </q-tooltip>
  </q-btn>
</template>

<script setup>
import { HeartBeatQuery, RefreshAPI, IndexHealthQuery } from 'components/api/searchAPI';
import { useQuasar } from 'quasar';
import { onMounted, reactive, computed } from 'vue';
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
  recommendations: [],
  bucketCount: 0,
  expectedDirs: 0,
});

const statusColor = computed(() => {
  if (view.loading) return 'orange';
  if (view.healthStatus === 'error') return 'negative';
  if (view.healthStatus === 'warning') return 'warning';
  if (view.healthStatus === 'healthy') return 'positive';
  return 'grey';
});

const statusLabel = computed(() => {
  if (view.loading) return '扫描中';
  if (view.indexNumber > 0) return `S:${view.indexNumber}`;
  return '索~';
});

const queryHealth = async () => {
  try {
    const health = await IndexHealthQuery();
    if (health) {
      view.healthStatus = health.status;
      view.totalCount = health.totalCount;
      view.recommendations = health.recommendations || [];
      view.bucketCount = health.bucketCount;
      view.expectedDirs = health.expectedDirs;
    }
  } catch (error) {
    console.error('IndexHealthQuery error:', error);
  }
};

const BASE_INTERVAL = 200;
const MAX_INTERVAL = 20000;

const scheduleNextHeartBeat = () => {
  view.heartBeatTimer = setTimeout(async () => {
    try {
      const res = await HeartBeatQuery();
      view.indexNumber = res;
      console.log('res', res);

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
        console.log('HeartBeat recovered, interval reset to', BASE_INTERVAL + 'ms');
      }
    } catch (error) {
      console.error('HeartBeatQuery error:', error);
      view.heartBeatRetryCount++;
      view.currentHeartBeatInterval = Math.min(BASE_INTERVAL * Math.pow(2, view.heartBeatRetryCount), MAX_INTERVAL);
      console.log(`HeartBeat retry ${view.heartBeatRetryCount}, next interval: ${view.currentHeartBeatInterval}ms`);
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
      type: 'negative',
      message: Message,
      position: 'top-right',
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
});
</script>
