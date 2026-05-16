<template>
  <div class="q-pa-md">
    <q-tabs 
      v-model="tab" 
      class="q-mb-md setting-tabs"
      align="justify" 
      narrow-indicator 
      active-color="white"
      indicator-color="white"
      glossy
      :style="{ backgroundColor: systemProperty.theme === 'star' ? 'rgba(15, 15, 26, 0.95)' : 'var(--q-primary)' }"
    >
      <q-tab name="info" label="系统信息" />
      <q-tab name="log" label="系统日志" />
    </q-tabs>

    <q-tab-panels v-model="tab" animated>
      <q-tab-panel name="info">
        <q-card class="q-mb-md theme-card">
          <q-card-section>
            <h6 class="text-subtitle1 q-mb-md">功能介绍</h6>
            <div class="SystemHtml" v-html="view.settingInfo.SystemHtml"></div>
          </q-card-section>
        </q-card>
        <q-card class="theme-card">
          <q-card-section>
            <p>网络访问 : </p>
            <a :href="view.ipAddr" class="text-primary">访问： {{ view.ipAddr }}</a>
            <p>userAgent : </p>
            <p class="text-wrap">{{ userAgent }}</p>
            <p>系统信息 : </p>
           <p>{{ $q.platform.is }}</p>
          </q-card-section>
        </q-card>
      </q-tab-panel>

      <q-tab-panel name="log">
        <q-card class="theme-card">
          <q-card-section>
            <div class="log-list">
              <div v-for="(item, index) in view.logs" :key="index" class="log-item q-py-xs">
                <span class="log-time">{{ item.time?.substring(0, 19) }}</span>
                <span class="log-separator"> - </span>
                <span class="log-msg">{{ item.msg }}</span>
              </div>
            </div>
          </q-card-section>
        </q-card>
      </q-tab-panel>
    </q-tab-panels>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import { GetSettingInfo, GetIpAddr, GeMemeryLog } from '../../components/api/settingAPI';
import { useSystemProperty } from '../../stores/System';

const systemProperty = useSystemProperty();
const tab = ref('info');
const view = reactive({
  settingInfo: {},
  ipAddr: '',
  logs: [],
});

const fetchSearch = async () => {
  const { data } = await GetSettingInfo();
  console.log(data);
  view.settingInfo = data;
};

const userAgent = computed(() => {
  return navigator.userAgent;
});

const queryIpAddr = async () => {
  const { Code, Data } = await GetIpAddr();
  if (Code == '200') {
    view.ipAddr = `http://${Data}:10081`;
  }
};

const fetchLogs = async () => {
  const { data } = await GeMemeryLog();
  view.logs = data.reverse();
};

let logIntervalId;

onMounted(() => {
  document.title = '系统信息';
  fetchSearch();
  queryIpAddr();
  fetchLogs();
  logIntervalId = setInterval(() => {
    fetchLogs();
  }, 5000);
});

onUnmounted(() => {
  if (logIntervalId) {
    clearInterval(logIntervalId);
  }
});
</script>
<style lang="scss" scoped>
.setting-tabs {
  border-radius: 8px 8px 0 0;
  
  .q-tab {
    font-weight: 500;
    letter-spacing: 0.5px;
    transition: all 0.3s ease;

    &--active {
      font-weight: 600;
    }
  }

  :deep(.q-tab__indicator) {
    height: 3px;
    border-radius: 3px 3px 0 0;
  }
}

.theme-card {
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  color: var(--q-text-primary);
}

.text-subtitle1 {
  color: var(--q-text-secondary);
}

.text-primary {
  color: var(--q-primary);
}

.text-wrap {
  word-break: break-all;
}

.SystemHtml {
  padding: 0rem;
  margin: 0;
  color: var(--q-text-primary);
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
