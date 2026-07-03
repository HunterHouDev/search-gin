<template>
  <q-dialog v-model="show" maximized>
    <q-card style="display:flex;flex-direction:column;max-height:80vh;width: 88%;">
      <q-bar class="bg-primary text-white q-pa-md shadow-2">
        <span class="text-weight-medium ellipsis" style="max-width: 60%;">{{ title }}</span>
        <q-space></q-space>
        <q-btn v-if="isRunning" color="orange" align="middle" label="实时刷新" class="q-ml-md">
        </q-btn>
        <q-btn icon="refresh" color="green" @click.stop="refreshLog" align="middle" label="刷新日志">
        </q-btn>
        <q-btn :icon="autoScrollOn ? 'vertical_align_bottom' : 'sync_disabled'" label="自动滚动"
          :color="autoScrollOn ? 'orange' : 'grey'" @click="autoScrollOn = !autoScrollOn" />
        <q-btn dense size="lg" flat color="black" icon="close" v-close-popup />
      </q-bar>

      <pre ref="logRef" class="bg-dark text-light-green q-pa-md" style="
            flex:1;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            font-size: 13px;
            font-family: 'Courier New', monospace;
            line-height: 1.6;
            margin: 0;
          ">{{ logContent || '暂无日志' }}</pre>
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
  if (!autoScrollOn.value || !logRef.value) return;
  setTimeout(() => {
    logRef.value.scrollTop = logRef.value.scrollHeight;
  }, 50);
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

// SSE 通知自动刷新（弹窗打开时无论任务状态都拉取最新日志）
watch(
  () => `${taskId.value}:${logVersion.value}`,
  () => {
    if (show.value) refreshLog();
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
