<template>
  <q-dialog ref="dialogRef" v-model="show" :fullscreen="isMobile" @hide="dialogHide" @before-show="beforeShow"
    :style="isMobile ? '' : 'min-width: 360px'">
    <div :style="isMobile ? '' : 'width: 80vw; max-width: 1100px; min-width: 600px; align-content: center;'">
      <q-layout container view="hHh Lpr lff"
        :style="'background: #0F1117; border-radius: 8px; ' + (isMobile ? 'height: 100vh' : 'height: 88vh')">
        <q-header class="shadow-2">

          <q-tabs v-model="dialogTab" class="bg-dark" active-color="white" indicator-color="grey-5"
            narrow-indicator>
            <q-tab name="batch" label="批量编辑" style="min-width: 100px" />
            <q-tab name="tasks" label="任务列表" style="min-width: 100px">
              <q-badge v-if="taskRunningCount > 0" color="orange" floating>{{ taskRunningCount }}</q-badge>
            </q-tab>
            <q-space />
            <q-btn flat dense icon="close" @click="dialogHide" />
          </q-tabs>

        </q-header>

        <q-page-container>
          <q-page class="q-pa-sm">
            <!-- 批量操作 -->
            <template v-if="dialogTab === 'batch'">
              <div class="row q-gutter-xs q-mb-sm items-center">
                <q-btn glossy color="primary" text-color="white"  @click="selectAll">
                  {{ state.selectAll ? '取消' : '全选' }}
                  <q-badge v-if="selectedCount > 0" color="red" floating>{{ selectedCount }}</q-badge>
                </q-btn>
                <q-btn-dropdown label="类型" glossy dense color="primary" >
                  <q-list dense>
                    <q-item v-for="mt in MovieTypeOptions" :key="mt.value" v-close-popup clickable
                      @click="setTypeBySelector(mt.value)">
                      <q-item-section>{{ mt.label }}</q-item-section>
                    </q-item>
                  </q-list>
                </q-btn-dropdown>
                <q-btn-dropdown v-permission="'op:tag'" label="标签" dense glossy color="primary" >
                  <div class="q-pa-sm" style="min-width: 220px">
                    <div class="row items-center q-mb-sm">
                      <q-radio v-model="state.chooseInput" :val="false" label="常用" dense />
                      <q-checkbox v-model="state.chooseInput" :val="false" label="输入" dense class="q-ml-md" />
                    </div>
                    <div v-if="state.chooseInput">
                      <q-input v-model="state.input" dense label="输入标签" class="q-mb-sm" />
                      <q-btn color="orange" label="提交" class="full-width" v-close-popup @click="submitInput" />
                    </div>
                    <div v-else>
                      <q-btn color="orange" label="提交" class="full-width q-mb-sm" v-close-popup
                        @click="addPlayingMutiTag" />
                      <q-checkbox v-model="state.submitMutiTag" v-for="tag in state.settingInfo.Tags" :key="tag"
                        :val="tag" dense keep-color :label="tag.substring(0, 6)" color="red" class="q-pr-md" />
                    </div>
                  </div>
                </q-btn-dropdown>
                <q-btn v-permission="'op:merge'" glossy color="orange" 
                  :disable="selectedCount === 0 || isBatchProcessing" @click="mergeFiles">合并</q-btn>
              </div>
              <div class="row q-gutter-sm q-mb-sm items-center">
                <q-input v-model="state.queryParam.Keyword" dense filled outlined color="primary" placeholder="搜索..."
                  style="width: 10rem" @update:model-value="debouncedSearch">
                  <template v-slot:append>
                    <q-icon name="search" class="cursor-pointer" @click="fetchSearch" />
                  </template>
                  <q-popup-proxy>
                    <div style="width: 200px; max-height: 50vh">
                      <q-list dense>
                        <q-item clickable v-close-popup v-ripple v-for="word in suggestions" :key="word"
                          @click="state.queryParam.Keyword = word; fetchSearch()">
                          <q-item-section>{{ word }}</q-item-section>
                        </q-item>
                      </q-list>
                    </div>
                  </q-popup-proxy>
                </q-input>
                <q-btn glossy color="black" icon="refresh" @click="refreshIndex" />
                <q-btn glossy color="black" icon="chevron_left" @click="nextPage(-1)" />
                <q-btn glossy color="black" icon="chevron_right" @click="nextPage(1)" />
              </div>

              <div v-if="state.settingInfo.MovieTypes?.length" class="q-mb-sm q-gutter-xs row">
                <q-chip v-for="mt in state.settingInfo.MovieTypes" :key="mt"
                  :color="state.queryParam.MovieType === mt ? 'primary' : 'grey-6'" text-color="white" clickable
                  @click="state.queryParam.MovieType = state.queryParam.MovieType === mt ? '' : mt; fetchSearch()">
                  {{ mt }}
                </q-chip>
                <q-chip v-if="state.queryParam.MovieType" color="red" text-color="white" clickable icon="close"
                  @click="state.queryParam.MovieType = ''; fetchSearch()">清除</q-chip>
              </div>

              <div id="batchListRef" style="height: calc(82vh - 160px); overflow: auto">
                <div v-if="!state.resultData.Data?.length" class="column items-center q-pa-xl">
                  <q-icon name="search_off" size="3rem" class="q-mb-md" />
                  <div class="text-h6">没有找到匹配的文件</div>
                  <q-btn color="primary" flat class="q-mt-md" @click="refreshIndex">刷新索引</q-btn>
                </div>
                <div v-for="item in state.resultData.Data" :key="item.Id" class="q-mb-xs batch-item"
                  style="border: 1px solid rgba(128,0,128,0.15); border-radius: 6px">
                  <div class="row items-center q-px-sm q-py-xs" style="gap: 4px">
                    <q-checkbox v-model="state.selector" :val="item.Id" color="red" dense size="xs" class="q-mr-xs" />
                    <q-img v-if="item.PngUrl" :src="item.PngUrl" style="width: 48px; height: 36px; border-radius: 4px"
                      fit="cover" @click="checkThis(item)" />
                    <div class="col" style="min-width: 0; line-height: 1.3">
                      <div class="row items-center" style="gap: 3px; flex-wrap: wrap">
                        <span v-if="state.cutListIds.includes(item.Id)" style="color: red; font-size: 11px">剪切中</span>
                        <q-btn v-permission="'op:movie:type'" flat dense icon="label" color="blue-6" 
                          :label="item.MovieType">
                          <q-tooltip>修改类型</q-tooltip>
                          <q-menu>
                            <q-list dense>
                              <q-item v-for="mt in MovieTypeOptions" :key="mt.value" v-close-popup clickable
                                @click="doSetMovieType(item, mt.value)">
                                <q-item-section>{{ mt.label }}</q-item-section>
                              </q-item>
                            </q-list>
                          </q-menu>
                        </q-btn>
                        <span class="dim" style="font-size: 12px">【{{ item.SizeStr }}】</span>
                        <q-btn flat dense icon="open_in_new" 
                          @click="commonExec(() => OpenFileFolder(item.Id))">
                          <q-tooltip>打开文件夹</q-tooltip>
                        </q-btn>
                        <q-btn flat dense icon="play_circle"  @click="playNewWindow(item)">
                          <q-tooltip>播放</q-tooltip>
                        </q-btn>
                        <q-btn flat dense icon="content_copy" color="grey"  @click.stop="copyPath(item)">
                          <q-tooltip>复制路径</q-tooltip>
                        </q-btn>
                        <q-btn-dropdown v-permission="'op:transcode'" flat dense icon="transform" color="teal"
                          >
                          <q-tooltip>转码</q-tooltip>
                          <q-list dense>
                            <q-item v-close-popup clickable
                              @click="toMp4(item)"><q-item-section>MP4</q-item-section></q-item>
                            <q-item v-close-popup clickable
                              @click="toVcode(item, 'h264')"><q-item-section>H264</q-item-section></q-item>
                            <q-item v-close-popup clickable
                              @click="toVcode(item, 'h265')"><q-item-section>H265</q-item-section></q-item>
                          </q-list>
                        </q-btn-dropdown>
                        <span class="text-weight-medium" style="
                          flex: 1; min-width: 60px; font-size: 13px;
                          display: -webkit-box; -webkit-box-orient: vertical; line-clamp: 1;
                          overflow: hidden; text-overflow: ellipsis;
                        ">{{ item.Title }}</span>
                        <span class="dim cursor-pointer" style="font-size: 10px"
                          @click="searchCode(item)">{{
                            item.Code?.substring(0, 10) }}</span>
                      </div>
                      <div class="row items-center" style="gap: 3px; flex-wrap: wrap">
                        <span class="dim cursor-pointer" @click="state.queryParam.Keyword = item.Author; fetchSearch()">
                          {{ item.Author?.substring(0, 8) }}
                        </span>
                        <span class="dim">{{ item.FileType }}</span>
                        <q-chip v-for="ta in (item.Tags || [])" :key="ta" dense color="orange-2" text-color="orange-9"
                          removable @remove="doCloseTag(item, ta)">
                          {{ ta }}
                        </q-chip>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </template>

            <!-- 任务列表 -->
            <template v-else>
              <div class="row items-center no-wrap" style="gap: 4px">
                <q-tabs v-model="taskTab" class="col" active-color="primary" indicator-color="primary">
                  <q-tab name="等待" label="等待">
                    <q-badge color="orange" floating>{{ taskTotalCount[3] + taskTotalCount[4] }}</q-badge>
                  </q-tab>
                  <q-tab name="完成" label="成功">
                    <q-badge color="green" floating>{{ taskTotalCount[1] }}</q-badge>
                  </q-tab>
                  <q-tab name="失败" label="失败">
                    <q-badge color="red" floating>{{ taskTotalCount[2] }}</q-badge>
                  </q-tab>
                  <q-tab name="全部" label="全部">
                    <q-badge color="grey" floating>{{ taskTotalCount[0] }}</q-badge>
                  </q-tab>
                </q-tabs>
                <q-toggle v-model="taskAutoRefresh" color="green"  label="自动" dense dark />
                <q-btn v-if="taskTab === '完成'" flat dense  color="orange" icon="delete_sweep" label="清除已完成"
                  @click="clearCompleted" />
                <q-btn v-if="taskTab === '失败'" flat dense  color="red" icon="delete_sweep" label="清除失败"
                  @click="clearFailed" />
                <q-btn v-if="taskTab === '全部'" flat dense  color="negative" icon="delete_sweep" label="清除所有"
                  @click="clearAll" />
              </div>

              <q-list v-if="taskRunningList.length" dense separator class="rounded-borders q-mb-sm"
                style="border: 1px solid rgba(255,152,0,0.2)">
                <q-item v-for="v in taskRunningList" :key="v.CreateTime" class="q-py-xs">
                  <q-item-section avatar>
                    <q-spinner color="orange" size="18px" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label class="text-caption text-weight-medium" style="line-clamp: 1">{{ v.Name || v.Files
                      }}</q-item-label>
                    <q-item-label caption>{{ v.Type }} &middot; {{ taskFmtTime(v.CreateTime) }}</q-item-label>
                  </q-item-section>
                  <q-item-section side>
                    <q-btn dense flat  icon="fullscreen" color="orange"
                      @click="taskLogFullscreenRef?.open(v)" />
                  </q-item-section>
                </q-item>
              </q-list>

              <q-list dense separator class="rounded-borders" style="max-height: 55vh; overflow-y: auto">
                <q-item v-for="v in taskFilteredList" :key="v.CreateTime" class="q-py-xs">
                  <q-item-section avatar>
                    <q-badge :color="taskStatusColor(v.Status)" :label="v.Type" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label class="text-caption text-weight-medium" style="line-clamp: 1">{{ v.Name || v.Files
                      }}</q-item-label>
                    <q-item-label caption>
                      <span :class="'text-' + taskStatusColor(v.Status)">{{ v.Status === '执行失败' ? '失败' : v.Status
                        }}</span>
                      <span v-if="v.FinishTime"> &middot; {{ taskShowTimeUse(v.FinishTime, v.CreateTime) }}</span>
                      <span> &middot; {{ taskFmtTime(v.CreateTime) }}</span>
                    </q-item-label>
                  </q-item-section>
                  <q-item-section side>
                    <q-btn v-if="v.Log" dense flat  icon="fullscreen" color="grey"
                      @click="taskLogFullscreenRef?.open(v)" />
                    <q-btn dense flat  icon="close" color="red" @click="taskRemove(v.ID)" />
                  </q-item-section>
                </q-item>
                <q-item v-if="!taskList.length" class="text-center q-py-md">
                  <q-item-section>暂无任务</q-item-section>
                </q-item>
              </q-list>
            </template>
          </q-page>
        </q-page-container>

        <!-- 底部状态（仅批量） -->
        <q-footer v-if="dialogTab === 'batch'" class="text-white row items-center q-px-sm"
          style="min-height: 28px; border-radius: 0 0 8px 8px">
          <span>第 {{ state.queryParam.Page }} 页，{{ state.queryParam.PageSize }} 条/页，共 {{ state.resultData.TotalCnt || 0
            }}
            条</span>
          <q-space />
          <span v-if="isBatchProcessing" class="text-orange text-bold q-mr-sm">
            <q-spinner-dots size="xs" color="orange" /> {{ batchProgress }}/{{ selectedCount }}
          </span>
        </q-footer>
      </q-layout>
    </div>
  </q-dialog>
  <TaskLogFullscreen ref="taskLogFullscreenRef" />
</template>

<script setup>
import { useQuasar } from 'quasar';
import { reactive, ref, computed, watch } from 'vue';
import { useSystemProperty } from 'stores/System';
import { useCommonExec } from 'src/composables/useCommonExec';
import { useBreakpoint } from 'src/composables/useBreakpoint';
import { useDialogShell } from 'src/composables/useDialogShell';
import { MovieTypeOptions, parseTimeZH } from 'components/utils';
import {
  ResetMovieType, SearchAPI, RefreshAPI, FilesMerge,
  TansferFileVcode, CloseTag, AddTag, OpenFileFolder,
  TransferTasksInfo, DelTransferTasksInfo,
  ClearCompletedTasks, ClearFailedTasks, ClearAllTasks,
} from 'components/api/searchAPI';
import { date } from 'quasar';
import Sortable from 'sortablejs';
import TaskLogFullscreen from './TaskLogFullscreen.vue';

const $q = useQuasar();
const systemProperty = useSystemProperty();
const { isMobile } = useBreakpoint();
const { exec: commonExec } = useCommonExec({ notifyOnSuccess: true });

let sortableInstance = null;
let debounceTimer;
let taskTimer = null;

const { show, dialogRef, dialogHide, beforeShow } = useDialogShell(() => {
  clearTimeout(debounceTimer);
  clearInterval(taskTimer);
  if (sortableInstance) { sortableInstance.destroy(); sortableInstance = null; }
  if (state.callback) state.callback({ settingInfo: state.settingInfo });
});

// ── Tab ──────────────────────────────────────────────────────────
const dialogTab = ref('batch');
const taskTab = ref('等待');

watch(dialogTab, (tab) => {
  if (tab === 'tasks') {
    taskFetch();
    if (taskAutoRefresh.value) {
      clearInterval(taskTimer);
      taskTimer = setInterval(taskFetch, 2000);
    }
  } else {
    clearInterval(taskTimer);
  }
});

// ── 批量状态 ──────────────────────────────────────────────────────
const state = reactive({
  selectAll: false,
  selector: [],
  cutListIds: [],
  resultData: {},
  queryParam: { Keyword: '', MovieType: '', Page: 1, PageSize: 20 },
  settingInfo: {},
  callback: null,
  submitMutiTag: [],
  chooseInput: false,
  input: '',
});

const isBatchProcessing = ref(false);
const batchProgress = ref(0);
const selectedCount = computed(() => state.selector.length);
const suggestions = computed(() => systemProperty.getSuggestions);

// ── 任务状态 ──────────────────────────────────────────────────────
const taskAutoRefresh = ref(true);
const taskList = ref([]);
const taskTotalCount = ref([0, 0, 0, 0, 0]);
const taskLogFullscreenRef = ref(null);

const taskRunningCount = computed(() => taskRunningList.value.length);
const taskRunningList = computed(() => taskList.value.filter((t) => t.Status === '执行中'));
const taskFilteredList = computed(() => {
  if (taskTab.value === '全部') return taskList.value.filter((t) => t.Status !== '执行中');
  return taskList.value.filter((t) => t.Status === taskTab.value && t.Status !== '执行中');
});

const taskStatusColor = (s) => s === '完成' ? 'green' : s === '失败' ? 'red' : s === '执行中' ? 'orange' : 'black';
const taskFmtTime = (t) => date.formatDate(new Date(t), 'MM/DD HH:mm');
const taskShowTimeUse = (end, start) => {
  const sec = ((new Date(end).getFullYear() > 1000 ? new Date(end).getTime() : Date.now()) - new Date(start).getTime()) / 1000;
  return parseTimeZH(Number(sec.toFixed(0)));
};
const taskFetch = async () => {
  const res = await TransferTasksInfo();
  taskList.value = res.Data?.tasks || [];
  taskTotalCount.value = res.Data?.counts || [0, 0, 0, 0, 0];
};
const taskRemove = async (id) => commonExec(() => DelTransferTasksInfo(id));
const clearCompleted = async () => { await commonExec(() => ClearCompletedTasks()); taskFetch(); };
const clearFailed = async () => { await commonExec(() => ClearFailedTasks()); taskFetch(); };
const clearAll = async () => { await commonExec(() => ClearAllTasks()); taskFetch(); };

watch(taskAutoRefresh, (v) => {
  if (v && show.value) { taskTimer = setInterval(taskFetch, 2000); }
  else { clearInterval(taskTimer); }
});

// ── 搜索 ──────────────────────────────────────────────────────────
const fetchSearch = async () => {
  const data = await SearchAPI(state.queryParam);
  state.resultData = { ...data };
};
const debouncedSearch = () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(fetchSearch, 300);
};
const refreshIndex = async () => {
  await RefreshAPI();
  await fetchSearch();
};
const nextPage = (n) => {
  state.queryParam.Page += n;
  fetchSearch();
};
const searchCode = (item) => {
  let c = item.Code;
  if (c.indexOf('-C') > 1) c = c.substring(0, c.indexOf('-C'));
  window.open(`${state.settingInfo.BaseUrl}${c}`, '_blank');
};

// ── 选择 ──────────────────────────────────────────────────────────
const checkThis = (item) => {
  const idx = state.selector.indexOf(item.Id);
  idx < 0 ? state.selector.push(item.Id) : state.selector.splice(idx, 1);
};
const resetSelector = () => { state.selector = []; state.selectAll = false; };
const selectAll = () => {
  state.selectAll = !state.selectAll;
  state.selectAll
    ? (state.selector = state.resultData.Data.map((f) => f.Id))
    : resetSelector();
};

// ── 批量操作 ──────────────────────────────────────────────────────
const setTypeBySelector = async (value) => {
  if (!state.selector.length) return;
  for (const id of state.selector) {
    const item = state.resultData.Data?.find((f) => f.Id === id);
    const updated = await commonExec(() => ResetMovieType(id, value));
    if (item && updated) Object.assign(item, updated);
  }
  resetSelector();
};
const addTagBySelector = async (value) => {
  if (!state.selector.length) return;
  for (const id of state.selector) {
    const item = state.resultData.Data?.find((f) => f.Id === id);
    const updated = await commonExec(() => AddTag(id, value));
    if (item && updated) Object.assign(item, updated);
  }
  resetSelector();
};
const submitInput = async () => {
  if (state.input) { await addTagBySelector(state.input); state.input = ''; }
};
const addPlayingMutiTag = async () => {
  if (state.submitMutiTag.length > 0) {
    await addTagBySelector(state.submitMutiTag.join(','));
    state.submitMutiTag = [];
  }
};
const mergeFiles = () => {
  if (state.selector.length) commonExec(() => FilesMerge({ files: state.selector, DeleteFlag: false }));
};

// ── 单项操作 ──────────────────────────────────────────────────────
const doSetMovieType = async (item, type) => {
  const u = await commonExec(() => ResetMovieType(item.Id, type));
  if (u) Object.assign(item, u);
};
const doCloseTag = async (item, tag) => {
  const u = await commonExec(() => CloseTag(item.Id, tag));
  if (u) Object.assign(item, u);
};
const toMp4 = (item) => {
  if (!state.cutListIds.includes(item.Id)) state.cutListIds.push(item.Id);
  commonExec(() => TansferFileVcode(item.Id, 'copy'));
};
const toVcode = (item, vcode) => {
  if (!state.cutListIds.includes(item.Id)) state.cutListIds.push(item.Id);
  commonExec(() => TansferFileVcode(item.Id, vcode));
};
const playNewWindow = (item) => {
  const w = systemProperty.singleWindow;
  window.open(item.Path, 'player', `width=${w.width},height=${w.height},titleBarStyle=`);
};
const copyPath = async (item) => {
  try {
    await navigator.clipboard.writeText(item.Path || item.Name || '');
    $q.notify({ type: 'positive', message: '路径已复制', position: 'bottom-left', timeout: 1500 });
  } catch {
    $q.notify({ type: 'negative', message: '复制失败', position: 'bottom-left' });
  }
};

// ── 排序初始化 ────────────────────────────────────────────────────
const initSortable = () => {
  setTimeout(() => {
    const el = document.getElementById('batchListRef');
    if (!el) return;
    if (sortableInstance) sortableInstance.destroy();
    sortableInstance = new Sortable(el, {
      animation: 150,
      handle: '.sortable-handle',
      onEnd(evt) {
        if (evt.oldIndex !== evt.newIndex) {
          state.resultData.Data.splice(evt.newIndex, 0, state.resultData.Data.splice(evt.oldIndex, 1)[0]);
        }
      },
    });
  }, 800);
};

// ── 对外 open ─────────────────────────────────────────────────────
const open = (data) => {
  const { queryParam, settingInfo, cb } = data || {};
  if (queryParam) state.queryParam = { ...queryParam };
  else state.queryParam = { ...systemProperty.getSearchParam, Page: 1, PageSize: 20 };
  if (settingInfo) state.settingInfo = settingInfo;
  else state.settingInfo = systemProperty.getSettingInfo;
  state.callback = cb || null;
  dialogTab.value = 'batch';
  show.value = true;
  fetchSearch();
  initSortable();
};

// ── 打开任务面板（供 SearchPage 调用） ──────────────────────────
const openTaskPanel = () => {
  dialogTab.value = 'tasks';
  show.value = true;
  taskFetch();
  if (taskAutoRefresh.value) taskTimer = setInterval(taskFetch, 2000);
};

defineExpose({ open, openTaskPanel });
</script>

<style scoped>
.q-page {
  color: var(--q-text-primary);
}
.batch-item {
  color: var(--q-text-primary);
}
.batch-item .dim {
  color: var(--q-text-secondary);
}
.q-item-label--caption {
  color: var(--q-text-secondary) !important;
}
</style>
<style>
.q-dialog__backdrop {
  background: rgba(0, 0, 0, 0.65) !important;
}
</style>
