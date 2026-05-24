<template>
  <q-dialog v-model="card" title="关机设置">
    <q-card class="my-card">
      <div style="width: 3000px; padding: 20px" class="q-gutter-sm">
        <div class="row justify-between">
          <q-btn color="primary" v-close-popup @click="closeApp">关闭系统</q-btn>
          <q-btn color="primary" v-close-popup @click="closePage()">关闭页面</q-btn>
        </div>
        <div class="text-h6">关机设置</div>
        <q-card-section class="q-pt-none">
          <div class="q-gutter-sm">
            <q-radio v-model="view.shutdownType" val="now" label="立即" />
            <q-radio v-model="view.shutdownType" val="target" label="定时" />
          </div>
          <div v-if="view.shutdownType === 'target'" style="
              display: flex;
              flex-direction: row;
              justify-content: space-between;
            ">
            <q-input class="timeSelect" v-model="view.shutdownHH"></q-input>
            <q-input class="timeSelect" v-model="view.shutdownMM"></q-input>
            <q-input class="timeSelect" v-model="view.shutdownSS"></q-input>
          </div>
        </q-card-section>
      </div>
      <q-card-actions align="right">
        <q-btn color="primary" v-close-popup @click="clearTime()">清除定时</q-btn>
        <q-btn v-close-popup color="primary">取消</q-btn>
        <q-btn v-close-popup color="primary" @click="submitBtn">关机 </q-btn>
        <q-btn color="primary" v-close-popup @click="logout">退出登录</q-btn>
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>
<script setup>
import { reactive, ref } from 'vue';
import { GetShutDown, AppShutDown } from '../components/api/settingAPI';
import { useSystemProperty } from '../stores/System';
import { useQuasar } from 'quasar';

const $q = useQuasar();
const card = ref(false);

const systemProperty = useSystemProperty();

const view = reactive({
  shutdownHH: 0,
  shutdownMM: 0,
  shutdownSS: 0,
  shutdownType: 'now',
  shutdownTime: new Date(),
});

const open = () => {
  card.value = true;
};

const close = () => {
  card.value = false;
};

const clearTime = () => {
  systemProperty.shutdownLeftSecond = null;
};

const closePage = () => {
  window.location.href = "about:blank"; window.close();
}

const closeApp = async () => {

  const res = await AppShutDown();
  console.log(res);
  $q.notify({ message: `${res}`, position: 'center' });
  setTimeout(() => {
    window.location.href = "about:blank"; window.close();
  }, 200);
};

const submitBtn = () => {
  clearTimeout(systemProperty.shutdownTimer);
  if (view.shutdownType === 'now') {
    console.log('GetShutDown now');
    GetShutDown();
  } else if (view.shutdownType === 'target') {
    systemProperty.shutdownLeftSecond =
      (view.shutdownHH || 0) * 3600 +
      (view.shutdownMM || 0) * 60 +
      (view.shutdownSS || 0);
    systemProperty.shutdownTimer = setInterval(() => {
      console.log(systemProperty.shutdownLeftSecond);
      systemProperty.shutdownLeftSecond = systemProperty.shutdownLeftSecond - 1;
      if (systemProperty.shutdownLeftSecond < 0) {
        clearTimeout(systemProperty.shutdownTimer);
        GetShutDown();
        console.log('GetShutDown timeout');
      }
    }, 1000);
  }
};

const logout = () => {
  localStorage.removeItem('isAuthenticated');
  window.location.href = '/';
};

defineExpose({
  open,
  close,
});
</script>
<style>
.timeSelect {
  width: 28px;
}
</style>
