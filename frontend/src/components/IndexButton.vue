<template>
  <q-btn
    color="red"
    title="索引"
    :size="props.size"
    label="索~"
    :dense="props.dense"
    :style="props.style"
    @click="refreshIndex"
    :loading="view.loading"
  >
    <template v-slot:loading>
      <q-spinner-facebook size="xs"></q-spinner-facebook>
      {{ `S:${view.indexNumber}` }}
    </template>
  </q-btn>
</template>

<script setup>
import { HeartBeatQuery, RefreshAPI } from 'components/api/searchAPI';
import { useQuasar } from 'quasar';
import { onMounted, reactive } from 'vue';
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
});

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
});

defineExpose({
  refreshIndex,
  refreshProgress,
});
</script>
