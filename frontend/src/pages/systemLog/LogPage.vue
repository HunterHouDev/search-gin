<template>
  <div class="q-pa-md">
    <q-card class="theme-card">
      <q-card-section>
        <h6 class="text-subtitle1 q-mb-md">系统日志</h6>
        <div class="log-list">
          <div v-for="item in view.logs" :key="item" class="log-item q-py-xs">
            <span class="log-time">{{ item.time.substring(0,19) }}</span>
            <span class="log-separator"> - </span>
            <span class="log-msg">{{ item.msg }}</span>
          </div>
        </div>
      </q-card-section>
    </q-card>
  </div>
</template>

<script setup>
// import {  date } from 'quasar';
import { onMounted, reactive ,onUnmounted} from 'vue';
import { GeMemeryLog } from '../../components/api/settingAPI';
const view = reactive({
  logs: [],
});

const fetchSearch = async () => {
  const { data } = await GeMemeryLog();
  console.log(data);
  view.logs = Array.isArray(data) ? data.reverse() : [];
};
let intervalId;

onMounted(() => {
  document.title = '系统日志';
  fetchSearch();
  intervalId = setInterval(() => {
    fetchSearch();
  }, 5000);

});

onUnmounted(() => {
  clearInterval(intervalId);
});

</script>
<style lang="scss" scoped>
.theme-card {
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  color: var(--q-text-primary);
}

.text-subtitle1 {
  color: var(--q-text-secondary);
}

.log-list {
  max-height: 70vh;
  overflow-y: auto;
}

.log-item {
  border-bottom: 1px solid var(--q-border-light);
  font-family: monospace;
  font-size: 0.9rem;
}

.log-time {
  color: var(--q-text-secondary);
}

.log-separator {
  color: var(--q-border);
}

.log-msg {
  color: var(--q-text-primary);
}
</style>
