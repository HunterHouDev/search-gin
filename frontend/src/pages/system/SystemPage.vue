<template>
  <div class="q-pa-md">
    <a :href="view.ipAddr">访问： {{ view.ipAddr }}</a>
    <hr />
    <span>userAgent : </span>
    <span>{{ userAgent }}</span>
    <hr />
    <span>系统信息：</span>
    <span>{{ $q.platform.is }}</span>
    <hr />
    <div class="SystemHtml" v-html="view.settingInfo.SystemHtml"></div>
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
.SystemHtml {
  padding: 0rem;
  margin: 0;
}
</style>
