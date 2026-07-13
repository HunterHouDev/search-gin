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
          <div v-if="view.shutdownType === 'target'" class="row items-center q-gutter-sm">
            <q-input
              v-model.number="view.shutdownValue"
              type="number"
              min="1"
              style="width: 100px"
              outlined
              dense
            />
            <q-select
              v-model="view.shutdownUnit"
              :options="shutdownUnitOptions"
              outlined
              dense
              emit-value
              map-options
              style="min-width: 90px"
            />
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
import { GetShutDown, AppShutDown, ScheduleShutdown, CancelShutdown } from '../components/api/settingAPI';
import { useSystemProperty } from '../stores/System';
import { useQuasar } from 'quasar';

const $q = useQuasar();
const card = ref(false);

const systemProperty = useSystemProperty();

const shutdownUnitOptions = [
  { label: '分钟', value: 'minute' },
  { label: '小时', value: 'hour' },
];

const view = reactive({
  shutdownValue: 1,
  shutdownUnit: 'minute',
  shutdownType: 'now',
});

const open = () => {
  card.value = true;
};

const close = () => {
  card.value = false;
};

// 清除定时 → 调后端 API
const clearTime = async () => {
  try {
    await CancelShutdown();
  } catch (e) {
    // ignore
  }
};

const closePage = () => {
  window.location.href = 'about:blank'; window.close();
}

const closeApp = async () => {
  const res = await AppShutDown();
  $q.notify({ message: `${res}`, position: 'center' });
  setTimeout(() => {
    window.location.href = 'about:blank'; window.close();
  }, 200);
};

// 提交关机 → 调后端 API
const submitBtn = async () => {
  if (view.shutdownType === 'now') {
    GetShutDown();
  } else if (view.shutdownType === 'target') {
    const multiplier = view.shutdownUnit === 'hour' ? 3600 : 60;
    const totalSec = (view.shutdownValue || 0) * multiplier;
    if (totalSec <= 0) return;
    try {
      await ScheduleShutdown(totalSec);
    } catch (e) {
      // ignore
    }
  }
};

const logout = () => {
  sessionStorage.removeItem('isAuthenticated');
  window.location.href = '/';
};

defineExpose({
  open,
  close,
});
</script>
<style scoped>
</style>
