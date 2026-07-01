<template>
  <q-dialog ref="dialogRef" v-model="show" @hide="dialogHide" @before-show="beforeShow">
    <q-card flat bordered style="width: 520px; max-width: 94vw; border-radius: 12px; overflow: hidden">
      <q-bar class="bg-dark text-white">
        <span class="text-weight-medium">任务执行</span>
        <q-space />
        <q-toggle v-model="autoRefresh" color="green" size="sm" label="自动" dense dark />
        <q-btn dense flat icon="close" v-close-popup />
      </q-bar>

      <q-card-section class="q-pa-sm">
        <!-- task tabs -->
        <q-tabs v-model="tab" dense no-caps class="text-grey-7" active-color="primary" indicator-color="primary">
          <q-tab name="等待" label="等待">
            <q-badge color="orange" floating>{{ totalCount[3] + totalCount[4] }}</q-badge>
          </q-tab>
          <q-tab name="完成" label="成功">
            <q-badge color="green" floating>{{ totalCount[1] }}</q-badge>
          </q-tab>
          <q-tab name="失败" label="失败">
            <q-badge color="red" floating>{{ totalCount[2] }}</q-badge>
          </q-tab>
          <q-tab name="全部" label="全部">
            <q-badge color="grey" floating>{{ totalCount[0] }}</q-badge>
          </q-tab>
        </q-tabs>

        <!-- clear bar -->
        <div class="row items-center justify-end q-py-xs q-gutter-xs" style="min-height: 28px">
          <q-btn v-if="tab === '完成'" flat dense size="sm" color="orange" icon="delete_sweep"
            label="清除已完成" @click="clearCompleted" />
          <q-btn v-if="tab === '失败'" flat dense size="sm" color="red" icon="delete_sweep"
            label="清除失败" @click="clearFailed" />
          <q-btn v-if="tab === '全部'" flat dense size="sm" color="negative" icon="delete_sweep"
            label="清除所有" @click="clearAll" />
        </div>

        <!-- running -->
        <q-list v-if="runningTasks.length" dense separator class="rounded-borders q-mb-sm"
          style="border: 1px solid rgba(255,152,0,0.2)">
          <q-item v-for="v in runningTasks" :key="v.CreateTime" class="q-py-xs">
            <q-item-section avatar>
              <q-spinner color="orange" size="18px" />
            </q-item-section>
            <q-item-section>
              <q-item-label class="text-caption text-weight-medium" style="line-clamp: 1">{{ v.Name || v.Files }}</q-item-label>
              <q-item-label caption>{{ v.Type }} &middot; {{ fmtTime(v.CreateTime) }}</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-btn dense flat size="sm" icon="fullscreen" color="orange" @click="taskLogFullscreenRef?.open(v)" />
            </q-item-section>
          </q-item>
        </q-list>

        <!-- done / failed / all -->
        <q-list dense separator class="rounded-borders" style="max-height: 55vh; overflow-y: auto">
          <q-item v-for="v in filteredTasks" :key="v.CreateTime" class="q-py-xs">
            <q-item-section avatar>
              <q-badge :color="statusColor(v.Status)" :label="v.Type" />
            </q-item-section>
            <q-item-section>
              <q-item-label class="text-caption text-weight-medium" style="line-clamp: 1">{{ v.Name || v.Files }}</q-item-label>
              <q-item-label caption>
                <span :class="'text-' + statusColor(v.Status)">{{ v.Status === '执行失败' ? '失败' : v.Status }}</span>
                <span v-if="v.FinishTime"> &middot; {{ showTimeUse(v.FinishTime, v.CreateTime) }}</span>
                <span> &middot; {{ fmtTime(v.CreateTime) }}</span>
              </q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-btn v-if="v.Log" dense flat size="sm" icon="fullscreen" color="grey" @click="taskLogFullscreenRef?.open(v)" />
              <q-btn dense flat size="sm" icon="close" color="red" @click="removeTask(v.ID)" />
            </q-item-section>
          </q-item>
          <q-item v-if="!tasks.length" class="text-grey text-center q-py-md">
            <q-item-section>暂无任务</q-item-section>
          </q-item>
        </q-list>
      </q-card-section>
    </q-card>
  </q-dialog>

  <TaskLogFullscreen ref="taskLogFullscreenRef" />
</template>

<script setup lang="ts">
import { date } from 'quasar';
import { ref, watch, computed } from 'vue';
import { useCommonExec } from 'src/composables/useCommonExec';
import { useDialogShell } from 'src/composables/useDialogShell';
import { parseTimeZH } from 'components/utils';
import {
  TransferTasksInfo, DelTransferTasksInfo,
  ClearCompletedTasks, ClearFailedTasks, ClearAllTasks,
} from 'components/api/searchAPI';
import TaskLogFullscreen from './TaskLogFullscreen.vue';

const { exec: commonExec } = useCommonExec({ notifyOnSuccess: true });

const taskLogFullscreenRef = ref<InstanceType<typeof TaskLogFullscreen> | null>(null);

let timer: any = null;
const autoRefresh = ref(true);
const tab = ref('等待');
const tasks = ref<any[]>([]);
const totalCount = ref([0, 0, 0, 0, 0]);

const runningTasks = computed(() => tasks.value.filter((t: any) => t.Status === '执行中'));
const filteredTasks = computed(() => {
  if (tab.value === '全部') return tasks.value.filter((t: any) => t.Status !== '执行中');
  return tasks.value.filter((t: any) => t.Status === tab.value && t.Status !== '执行中');
});

const statusColor = (s: string) => s === '完成' ? 'green' : s === '失败' ? 'red' : s === '执行中' ? 'orange' : 'black';
const fmtTime = (t: string) => date.formatDate(new Date(t), 'MM/DD HH:mm');

const showTimeUse = (end: string, start: string) => {
  const sec = ((new Date(end).getFullYear() > 1000 ? new Date(end).getTime() : Date.now()) - new Date(start).getTime()) / 1000;
  return parseTimeZH(Number(sec.toFixed(0)));
};

const fetchTasking = async () => {
  const res = await TransferTasksInfo();
  tasks.value = (res.Data?.tasks || []).sort((a: any, b: any) => (b.CreateTime || '').localeCompare(a.CreateTime || ''));
  totalCount.value = res.Data?.counts || [0, 0, 0, 0, 0];
};

const removeTask = async (taskId: string) => commonExec(() => DelTransferTasksInfo(taskId));
const clearCompleted = async () => { await commonExec(() => ClearCompletedTasks()); fetchTasking(); };
const clearFailed = async () => { await commonExec(() => ClearFailedTasks()); fetchTasking(); };
const clearAll = async () => { await commonExec(() => ClearAllTasks()); fetchTasking(); };

watch(autoRefresh, (v) => {
  if (v && show.value) { timer = setInterval(fetchTasking, 2000); }
  else { clearInterval(timer); }
});

const { show, dialogRef, dialogHide, beforeShow } = useDialogShell(() => {
  clearInterval(timer);
});

const open = () => {
  show.value = true;
  fetchTasking();
  if (autoRefresh.value) timer = setInterval(fetchTasking, 2000);
};

defineExpose({ open });
</script>
