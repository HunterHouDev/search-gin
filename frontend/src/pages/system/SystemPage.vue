<template>
  <div class="q-pa-md">
    <q-card class="q-mb-md theme-card">
      <q-card-section>
        <h6 class="text-subtitle1">网络访问</h6>
        <a :href="view.ipAddr" class="text-primary">访问： {{ view.ipAddr }}</a>
      </q-card-section>
    </q-card>
    <q-card class="q-mb-md theme-card">
      <q-card-section>
        <h6 class="text-subtitle1">浏览器信息</h6>
        <p>userAgent : </p>
        <p class="text-wrap">{{ userAgent }}</p>
      </q-card-section>
    </q-card>
    <q-card class="q-mb-md theme-card">
      <q-card-section>
        <h6 class="text-subtitle1">系统信息</h6>
        <p>{{ $q.platform.is }}</p>
      </q-card-section>
    </q-card>
    <q-card class="theme-card">
      <q-card-section>
        <h6 class="text-subtitle1">其他信息</h6>
        <div class="SystemHtml" v-html="view.settingInfo.SystemHtml"></div>
      </q-card-section>
    </q-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive } from 'vue';
import { GetSettingInfo, GetIpAddr } from '../../components/api/settingAPI';
const view = reactive({
  settingInfo: {},
  ipAddr: '',
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

onMounted(() => {
  document.title = '系统信息';
  fetchSearch();
  queryIpAddr();
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
</style>
