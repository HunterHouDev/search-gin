<template>
  <div class="q-pa-md">
    <div v-for="item in view.logs" :key="item">
      {{ item.time.substring(0,19) }} - {{ item.msg }}
    </div>
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
  view.logs = data.reverse();
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
.SystemHtml {
  padding: 0rem;
  margin: 0;
}
</style>
