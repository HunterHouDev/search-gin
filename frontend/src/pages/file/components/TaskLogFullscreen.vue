<template>
  <q-dialog v-model="show" full-height full-width>
    <q-card class="column full-height">
      <q-bar class="bg-dark text-white">
        <span class="text-weight-medium" style="max-width: 70vw">{{ title }}</span>
        <q-space />
        <q-badge v-if="isRunning" color="orange" align="middle">
          <q-spinner size="14px" color="white" class="q-mr-xs" />实时
        </q-badge>
        <q-btn dense flat size="sm" icon="refresh" color="grey-7" @click.stop="refreshLog">
          <q-tooltip>刷新日志</q-tooltip>
        </q-btn>
        <q-btn dense flat size="sm" :icon="autoScrollOn ? 'vertical_align_bottom' : 'sync_disabled'"
          :color="autoScrollOn ? 'orange' : 'grey'" @click="autoScrollOn = !autoScrollOn" />
        <q-btn dense flat icon="close" v-close-popup />
      </q-bar>
      <q-card-section class="col q-pa-none">
        <pre ref="logRef" class="bg-dark text-light-green q-pa-md" style="
            height: 100%;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            font-size: 13px;
            font-family: 'Courier New', monospace;
            line-height: 1.6;
            margin: 0;
          ">{{ logContent || '暂无日志' }}</pre>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue';
import { GetTaskLogAPI } from 'components/api/searchAPI';
import { logVersion } from 'src/stores/taskLog';

const show = ref(false);
const taskId = ref('');
const title = ref('');
const isRunning = ref(false);

const logContent = ref('');
const logRef = ref(null);
const autoScrollOn = ref(true);

const scrollToBottom = () => {
  nextTick(() => {
    if (!autoScrollOn.value || !logRef.value) return;
    logRef.value.scrollTop = logRef.value.scrollHeight;
  });
};

const refreshLog = async () => {
  if (!taskId.value) return;
  const res = await GetTaskLogAPI(taskId.value);
  if (res?.Code === 200 && res.Data?.log !== undefined) {
    logContent.value = res.Data.log;
    scrollToBottom();
  }
};

// 弹窗打开时立即加载一次
watch(show, (val) => {
  if (val && taskId.value) refreshLog();
});

// SSE 通知自动刷新（仅执行中的任务需要）
watch(
  () => `${taskId.value}:${logVersion.value}`,
  () => {
    if (show.value && isRunning.value) refreshLog();
  }
);

const open = (task) => {
  taskId.value = task.ID;
  title.value = task.Name || task.Files || task.Command || '';
  isRunning.value = task.Status === '执行中';
  show.value = true;
};

defineExpose({ open });
</script>
