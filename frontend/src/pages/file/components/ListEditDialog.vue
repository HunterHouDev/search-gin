<template>
  <q-dialog ref="dialogRef" v-model:model-value="view.showDiaolg" @hide="dialogHide" @before-show="beforeShow">
    <q-layout container view="hHh Lpr lff" style="height: 80vh; margin: 0" :style="{
      'padding-top': '0px',
      'max-width': isMobile ? '94vw' : '800px',
    }">
      <q-header class="bg-black text-white shadow-2 rounded-borders justify-between row items-center">
        <q-tabs alert ripple v-model="tab" align="justify" mobile-arrows style="width: 90%"
          :active-color="systemProperty.theme === 'natural' ? 'primary' : 'white'"
          :indicator-color="systemProperty.theme === 'natural' ? 'green' : 'white'">
          <q-tab name="filelist" :label="isMobile ? '操作' : '批量操作'" />
          <q-tab name="tasking" :label="isMobile ? '任务' : '任务执行'" />
        </q-tabs>
        <q-btn dense flat icon="close" @click="dialogHide">
          <q-tooltip class="bg-white text-primary">关闭</q-tooltip>
        </q-btn>
      </q-header>
      <q-page-container style="padding-top: 2.8rem">
        <q-page-sticky v-if="tab == 'filelist'" style="z-index: 9" position="top-right"
          :offset="isMobile ? [2, 50] : [10, 100]">
          <div class="row column justify-end q-gutter-sm items-end">
            <q-btn glossy color="black" @click="selectAll">
              {{ view.selectAll ? '取消' : '全选' }}
              <q-badge v-if="selectedCount > 0" color="red" floating>
                {{ selectedCount }}
              </q-badge>
            </q-btn>
            <q-btn-dropdown label="类型" glossy dense color="primary">
              <q-list>
                <q-item v-for="mt in MovieTypeOptions" :key="mt.value" v-close-popup class="movieTypeSelectItem">
                  <q-item-section @click="setTypeBySelector(mt.value)">
                    <q-item-label>{{ mt.label }}</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-btn-dropdown>
            <q-btn-dropdown v-permission="'op:tag'" label="标签" dense glossy color="primary">
              <div class="col">
                <q-btn flat dense> 种草来源 </q-btn>
                <q-radio v-model="view.chooseInput" checked-icon="task_alt" unchecked-icon="panorama_fish_eye"
                  :val="false" label="常用" />
                <q-checkbox v-model="view.chooseInput" checked-icon="task_alt" unchecked-icon="panorama_fish_eye"
                  :val="false" label="输入" @click="chooseInput" />
              </div>
              <div v-show="view.chooseInput" style="padding: 10px">
                <q-input v-model="view.input" style="width: 100%" label="输入" class="inputWords" />
                <q-btn color="orange" style="width: 100%" label="提交" v-close-popup class="tag-item glossy"
                  @click="submitInput"></q-btn>
              </div>
              <div v-show="!view.chooseInput" style="
                  max-width: 400px;
                  max-height: 880px;
                  padding: 10px 4px;
                  height: auto;
                  display: flex;
                  flex-direction: column;
                  justify-content: flex-start;
                ">
                <div class="row w100" v-show="!view.chooseInput">
                  <q-btn color="orange" style="width: 100%" label="提交" class="tag-item glossy" v-close-popup
                    @click="addPlayingMutiTag"></q-btn>
                </div>
                <div v-show="!view.chooseInput" class="row w100"
                  style="max-width: 400px; max-height: 400px; overflow: auto">
                  <q-checkbox v-model="view.submitMutiTag" v-for="tag in view.settingInfo.Tags" :key="tag" :val="tag"
                    dense keep-color :label="tag.substring(0, 6)" color="red" class="q-pr-md glossy" />
                </div>
              </div>
            </q-btn-dropdown>
            <q-btn v-permission="'op:edit'" glossy color="teal" :disable="selectedCount === 0 || isBatchProcessing"
              @click="batchRename">改名
            </q-btn>
            <q-btn v-permission="'op:edit'" glossy color="red" :disable="selectedCount === 0 || isBatchProcessing"
              @click="confirmDelete" :loading="isBatchProcessing">删除 </q-btn>
            <q-btn v-permission="'op:merge'" glossy color="red" :disable="selectedCount === 0 || isBatchProcessing"
              @click="mergeFiles">合并 </q-btn>
          </div>
        </q-page-sticky>
        <q-page-sticky v-if="tab == 'filelist'" style="z-index: 9" position="bottom-right" :offset="[2, 0]">
          <div class="row justify-start q-pa-xs">
            <div>
              当前{{ view.queryParam.Page }}页， 每页{{
                view.queryParam.PageSize
              }}条，共{{ view.resultData.TotalCnt }}条记录
            </div>
            <div v-if="isBatchProcessing" class="q-ml-md text-orange text-bold">
              <q-spinner-dots size="sm" color="orange" />
              批量处理中：{{ batchProgress }}/{{ selectedCount }}
            </div>
          </div>
        </q-page-sticky>
        <q-page class="shadow-2 rounded-borders">
          <q-tab-panels v-model="tab" animated>
            <q-tab-panel name="filelist" style="padding: 6px; height: 100%">
              <div class="q-gutter-sm row justify-left search-toolbar">
                <q-input v-model="view.queryParam.Keyword" :dense="true" filled outlined color="primary"
                  placeholder="搜索..." style="width: 10rem" @update:model-value="debouncedSearch">
                  <template v-slot:append>
                    <q-icon name="ti-search" title="搜" glossy class="cursor-pointer" @click="fetchSearch">
                    </q-icon>
                  </template>
                  <q-popup-proxy>
                    <div style="width: 200px; max-height: 50vh">
                      <q-list>
                        <q-item clickable v-close-popup v-ripple v-for="word in suggestions" :key="word" @click="
                          view.queryParam.Keyword = word;
                        fetchSearch();
                        ">
                          <q-item-section>
                            <q-item-label>{{ word }}</q-item-label>
                          </q-item-section>
                        </q-item>
                      </q-list>
                    </div>
                  </q-popup-proxy>
                </q-input>
                <q-btn glossy size="sm" color="black" icon="refresh" @click="refreshIndex">
                </q-btn>
                <q-btn glossy color="black" @click="nextPage(-1)">上 </q-btn>
                <q-btn glossy color="black" @click="nextPage(1)">下 </q-btn>
              </div>

              <div v-if="view.settingInfo.MovieTypes?.length" class="q-mb-sm q-gutter-xs row">
                <q-chip v-for="mt in view.settingInfo.MovieTypes" :key="mt"
                  :color="view.queryParam.MovieType === mt ? 'primary' : 'grey-6'" text-color="white" size="sm"
                  clickable
                  @click="view.queryParam.MovieType = view.queryParam.MovieType === mt ? '' : mt; fetchSearch()">{{ mt
                  }}</q-chip>
                <q-chip v-if="view.queryParam.MovieType" color="red" text-color="white" size="sm" clickable icon="close"
                  @click="view.queryParam.MovieType = ''; fetchSearch()">清除</q-chip>
              </div>

              <div id="listRef" style="height: 67vh; width: 100%; overflow: auto; padding: 4px">
                <div v-if="!view.resultData.Data || view.resultData.Data.length === 0"
                  class="column items-center q-pa-xl text-grey-7">
                  <q-icon name="search_off" size="3rem" class="q-mb-md" />
                  <div class="text-h6">没有找到匹配的文件</div>
                  <div class="text-caption q-mt-sm">换个关键词试试，或者刷新索引</div>
                  <q-btn color="primary" flat class="q-mt-md" @click="refreshIndex">刷新索引</q-btn>
                </div>
                <div v-for="item in view.resultData.Data" :key="item.Id" style="
                    border: 1px dotted purple;
                    border-radius: 4px;
                    padding: 0;
                    align-items: center;
                  ">
                  <q-expansion-item dense hideExpandIcon>
                    <template v-slot:header>
                      <q-item-section avatar>
                        <q-img fit="fill" height="auto" :src="item.PngUrl"
                          style="width: 80px; height: auto; max-height: 80px" @click="checkThis(item)">
                          <q-checkbox v-model="view.selector" color="red" leftLabel glossy :val="item.Id" style="
                              background-color: rgba(250, 250, 250, 0.1);
                              width: 3rem;
                              height: 2rem;
                            ">
                          </q-checkbox>
                        </q-img>
                      </q-item-section>

                      <q-item-section style="
                          margin: 0;
                          padding: 0;
                          line-height: 12px;
                          display: flex;
                          justify-content: start;
                          align-content: flex-start;
                          flex-direction: row;
                          flex-wrap: wrap;
                        ">
                        <span v-if="view.cutListIds.indexOf(item.Id) >= 0" style="color: red">剪切中：：</span>
                        <q-btn-dropdown v-permission="'op:movie:type'" dense glossy :label="item.MovieType"
                          type="primary" color="blue-6" size="sm">
                          <q-list>
                            <q-item v-for="mt in MovieTypeOptions" :key="mt.value" v-close-popup
                              class="movieTypeSelectItem">
                              <q-item-section>
                                <q-item-label @click="doSetMovieType(item, mt.value)">{{ mt.label }}
                                </q-item-label>
                              </q-item-section>
                            </q-item>
                          </q-list>
                        </q-btn-dropdown>

                        <q-btn outline dense glossy icon="open_in_new"
                          @click="commonExec(() => OpenFileFolder(item.Id))" />
                        <q-btn outline dense glossy icon="player" @click="playNewWindow(item)" />

                        <q-btn-dropdown v-permission="'op:transcode'" label="转码" outline dense glossy type="primary"
                          color="teal">
                          <q-list>
                            <q-item glossy>
                              <q-item-section v-close-popup @click="toMp4(item)">
                                <q-item-label>MP4</q-item-label>
                              </q-item-section>
                            </q-item>
                            <q-item>
                              <q-item-section v-close-popup @click="toVcode(item, 'h264')">
                                <q-item-label>H264</q-item-label>
                              </q-item-section>
                            </q-item>
                            <q-item>
                              <q-item-section v-close-popup @click="toVcode(item, 'h265')">
                                <q-item-label>H265</q-item-label>
                              </q-item-section>
                            </q-item>
                          </q-list>
                        </q-btn-dropdown>

                        <q-btn class="mr10 cursor-pointer" target="_blank" @click="searchCode(item)">{{
                          item.Code?.substring(0, 10) }}</q-btn>
                        <q-btn style="color: #9e089e" dense flat glossy class="mr10 cursor-pointer" @click="
                          view.queryParam.Keyword = item.Author;
                        fetchSearch();
                        ">{{ item.Author?.substring(0, 8) }}</q-btn>
                        <q-chip color="orange" text-color="white" size="sm">
                          {{ `${item.FileType}` }}
                        </q-chip>
                        <div v-if="item.Tags">
                          <q-chip color="orange" text-color="white" size="sm" v-for="ta in item.Tags" :key="ta"
                            removable @remove="doCloseTag(item, ta)">
                            {{ `${ta}` }}
                          </q-chip>
                        </div>
                        <q-btn dense flat size="xs" icon="content_copy" color="grey" class="q-mr-xs"
                          @click.stop="copyPath(item)">
                          <q-tooltip>复制路径</q-tooltip>
                        </q-btn>
                        <p style="
                            display: -webkit-box; /* 将对象作为弹性伸缩盒子模型显示 */
                            -webkit-box-orient: vertical; /* 设置子元素的排列方式为垂直方向 */
                            line-clamp: 2; /* 设置显示的行数 */
                            overflow: hidden; /* 隐藏溢出文本 */
                            text-overflow: ellipsis; /* 显示省略号 */
                          ">
                          【{{ item.SizeStr }}】{{ item.Title }}
                          <q-tooltip anchor="bottom middle" self="top middle" max-width="600px">
                            <div style="word-break: break-all; font-size: 0.8rem">{{ item.Path }}</div>
                          </q-tooltip>
                        </p>
                      </q-item-section>
                    </template>
                  </q-expansion-item>
                </div>
              </div>
            </q-tab-panel>

            <q-tab-panel name="tasking" style="padding: 6px; height: 100%">
              <q-tabs alert ripple v-model="tabTask" align="justify" class="shadow-2 w100" dense no-caps>
                <q-tab name="等待" label="等待">
                  <q-badge color="orange" floating>{{
                    view.totalCount[3] + view.totalCount[4]
                    }}</q-badge>
                </q-tab>
                <q-tab name="完成" label="成功">
                  <q-badge color="green" floating>{{
                    view.totalCount[1]
                    }}</q-badge>
                </q-tab>
                <q-tab name="失败" label="失败">
                  <q-badge color="red" floating>{{
                    view.totalCount[2]
                    }}</q-badge>
                </q-tab>
                <q-tab name="全部" label="全部">
                  <q-badge color="grey" floating>{{
                    view.totalCount[0]
                    }}</q-badge>
                </q-tab>
                <q-separator vertical />
                <q-tab name="all" label="" class="justify-center">
                  <q-toggle color="green" v-model="view.autoRefresh" label="自动" dense />
                </q-tab>
              </q-tabs>

              <div class="row items-center q-pa-xs q-gutter-xs" style="min-height: 36px">
                <q-space />
                <q-btn v-if="tabTask === '完成'" outline color="orange" icon="delete_sweep" label="清除已完成" size="sm" dense
                  @click="clearCompletedTasks" />
                <q-btn v-if="tabTask === '失败'" outline color="red" icon="delete_sweep" label="清除失败" size="sm" dense
                  @click="clearFailedTasks" />
                <q-btn v-if="tabTask === '全部'" outline color="negative" icon="delete_sweep" label="清除所有" size="sm" dense
                  @click="clearAllTasks" />
              </div>

              <!-- 执行中的任务 -->
              <q-list dense bordered separator class="rounded-borders">
                <template v-for="v in view.tasking" :key="v.CreateTime">
                  <q-item v-if="v.Status === '执行中'" class="q-py-xs">
                    <q-item-section avatar>
                      <q-spinner color="orange" size="20px" />
                    </q-item-section>
                    <q-item-section>
                      <q-item-label class="text-weight-medium" style="
                          display: -webkit-box;
                          -webkit-box-orient: vertical;
                          line-clamp: 1;
                          overflow: hidden;
                          text-overflow: ellipsis;
                        ">
                        {{ v.Name || v.Files }}
                      </q-item-label>
                      <q-item-label caption class="row q-gutter-sm">
                        <span class="text-orange">{{ v.Type }}</span>
                        <span v-if="v.Start">开始 {{ v.Start }}</span>
                        <span v-if="v.End">结束 {{ v.End }}</span>
                        <span>{{
                          date.formatDate(new Date(v.CreateTime), 'MM/DD HH:mm')
                          }}</span>
                      </q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-btn dense flat size="sm" icon="fullscreen" color="orange" @click="openFullscreenLog(v)">
                        <q-tooltip>全屏查看日志</q-tooltip>
                      </q-btn>
                    </q-item-section>
                  </q-item>
                </template>
              </q-list>

              <!-- 等待/完成/失败/全部 任务列表 -->
              <q-list dense bordered separator class="rounded-borders"
                style="max-height: calc(100% - 120px); overflow-y: auto">
                <template v-for="v in view.tasking" :key="v.CreateTime">
                  <q-item v-if="
                    v.Status !== '执行中' &&
                    (tabTask === '全部' || v.Status === tabTask)
                  " class="q-py-xs">
                    <q-item-section avatar>
                      <q-badge :color="getColor(v.Status)" :label="v.Type" />
                    </q-item-section>
                    <q-item-section>
                      <q-item-label class="text-weight-medium" style="
                          display: -webkit-box;
                          -webkit-box-orient: vertical;
                          line-clamp: 1;
                          overflow: hidden;
                          text-overflow: ellipsis;
                        ">
                        {{ v.Name || v.Files }}
                      </q-item-label>
                      <q-item-label caption class="row q-gutter-sm">
                        <span :class="'text-' + getColor(v.Status)">
                          {{ v.Status === '执行失败' ? '失败' : v.Status }}
                        </span>
                        <span v-if="v.FinishTime">
                          耗时 {{ parseTimeZH(Number(showTimeUse(v.FinishTime, v.CreateTime)).toFixed(0)) }}
                        </span>
                        <span>
                          {{ date.formatDate(new Date(v.CreateTime), 'MM/DD HH:mm') }}
                        </span>
                      </q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-btn v-if="v.Log" dense flat size="sm" icon="fullscreen" color="grey" @click="openFullscreenLog(v)" />
                      <q-btn dense flat size="sm" icon="close" color="red" @click="removeTask(v.ID)" />
                    </q-item-section>
                  </q-item>
                </template>
                <q-item v-if="view.tasking.length === 0">
                  <q-item-section class="text-grey text-center">
                    暂无任务
                  </q-item-section>
                </q-item>
              </q-list>

            </q-tab-panel>
          </q-tab-panels>
        </q-page>
      </q-page-container>
    </q-layout>
  </q-dialog>

  <TaskLogFullscreen ref="taskLogFullscreenRef" />
</template>

<script setup>
import { useQuasar, date } from 'quasar';
import { useDialogPluginComponent } from 'quasar';
import { reactive, ref, watch, computed } from 'vue';
import { useSystemProperty } from 'stores/System';
import { useCommonExec } from 'src/composables/useCommonExec';
import { useBreakpoint } from 'src/composables/useBreakpoint';
import { notifyTaskLog } from 'src/stores/taskLog';
import TaskLogFullscreen from './TaskLogFullscreen.vue';

import {
  MovieTypeOptions,
} from 'components/utils';
import {
  parseTimeZH,
} from 'components/utils';
import {
  ResetMovieType,
  SearchAPI,
  RefreshAPI,
  DeleteFile,
  FilesMerge,
  TransferTasksInfo,
  TansferFileVcode,
  CloseTag,
  DelTransferTasksInfo,
  AddTag,
  FileRename,
  ClearCompletedTasks,
  ClearFailedTasks,
  ClearAllTasks,
} from 'components/api/searchAPI';

import Sortable from 'sortablejs';
const $q = useQuasar();

const tab = ref('filelist');
const tabTask = ref('等待');
const view = reactive({
  autoRefresh: true,
  selectAll: false,
  showDiaolg: false,
  settingInfo: {},
  resultData: {},
  queryParam: {},
  selector: [],
  callback: null,
  cutListIds: [],
  tasking: [],
  submitMutiTag: [],
  editItem: {},
  totalCount: [0, 0, 0, 0, 0],
  chooseInput: false,
  input: '',
});

const isBatchProcessing = ref(false);
const batchProgress = ref(0);

const selectedCount = computed(() => view.selector.length);

const checkThis = (item) => {
  if (view.selector.indexOf(item.Id) < 0) {
    view.selector.push(item.Id);
  } else {
    view.selector.splice(view.selector.indexOf(item.Id), 1);
  }
};
const simgleWindow = computed(() => {
  return systemProperty.singleWindow;
});

const playNewWindow = (item) => {
  const options = `width=${simgleWindow.value.width},height=${simgleWindow.value.height},titleBarStyle=`;
  window.open(item.Path, 'player', options);
}

let timeFunc;
let sortableInstance = null;
watch(
  () => tab.value,
  (v) => {
    if (view.showDiaolg) {
      if (v === 'tasking' && view.autoRefresh) {
        fetchTasking();
      }
      if (v === 'tasking' && view.autoRefresh && view.showDiaolg) {
        timeFunc = setInterval(fetchTasking, 2000);
      } else {
        clearInterval(timeFunc);
      }
      if (v === 'filelist') {
      }
    }
  }
);

watch(
  () => view.autoRefresh,
  (v) => {
    if (view.showDiaolg) {
      if (v && tab.value === 'tasking') {
        timeFunc = setInterval(fetchTasking, 2000);
      } else {
        clearInterval(timeFunc);
      }
    } else {
      clearInterval(timeFunc);
    }
  }
);

const systemProperty = useSystemProperty();
const { isMobile } = useBreakpoint();
const { exec: commonExec } = useCommonExec({ notifyOnSuccess: true });

// ─── 全屏日志 ─────────────────────────
const taskLogFullscreenRef = ref(null);

const getColor = (status) => {
  return status == '完成'
    ? 'green'
    : status == '失败'
      ? 'red'
      : status == '执行中'
        ? 'orange'
        : 'black';
};

// 打开全屏日志
const openFullscreenLog = (v) => taskLogFullscreenRef.value?.open(v);

const removeTask = async (taskId) => {
  commonExec(() => DelTransferTasksInfo(taskId));
};

const clearCompletedTasks = async () => {
  await commonExec(() => ClearCompletedTasks());
  fetchTasking();
};

const clearFailedTasks = async () => {
  await commonExec(() => ClearFailedTasks());
  fetchTasking();
};

const clearAllTasks = async () => {
  await commonExec(() => ClearAllTasks());
  fetchTasking();
};

const doSetMovieType = async (item, type) => {
  const updated = await commonExec(() => ResetMovieType(item.Id, type));
  if (updated) Object.assign(item, updated);
};

const doCloseTag = async (item, tag) => {
  const updated = await commonExec(() => CloseTag(item.Id, tag));
  if (updated) Object.assign(item, updated);
};

const fetchTasking = async () => {
  const res = await TransferTasksInfo();
  view.tasking = (res.Data?.tasks || []).sort((a, b) => (b.CreateTime || '').localeCompare(a.CreateTime || ''));
  view.totalCount = res.Data?.counts || [0, 0, 0, 0, 0];
};

const searchCode = (item) => {
  let { Code } = item;
  if (Code.indexOf('-C') > 1) {
    Code = Code.substring(0, Code.indexOf('-C'));
  }
  const url = `${view.settingInfo.BaseUrl}${Code}`;
  window.open(url, '_blank');
};

const showTimeUse = (end, start) => {
  return `${((new Date(end).getFullYear() > 1000
      ? new Date(end)
      : new Date()
    ).getTime() -
      new Date(start).getTime()) /
    1000
    }`;
};

const toMp4 = (item) => {
  if (view.cutListIds.indexOf(item.Id) < 0) {
    view.cutListIds.push(item.Id);
  }
  commonExec(() => TansferFileVcode(item.Id, 'copy'));
};

const toVcode = (item, vcode) => {
  if (view.cutListIds.indexOf(item.Id) < 0) {
    view.cutListIds.push(item.Id);
  }
  commonExec(() => TansferFileVcode(item.Id, vcode));
};

const resetSelector = () => {
  view.selector = [];
  view.selectAll = false;
};

const selectAll = () => {
  view.selectAll = !view.selectAll;
  if (view.selectAll) {
    view.selector = view.resultData.Data.map((item) => item.Id);
  } else {
    resetSelector();
  }
};

const setTypeBySelector = async (value) => {
  if (view.selector && view.selector.length > 0) {
    for (const id of view.selector) {
      const item = view.resultData.Data?.find((f) => f.Id === id);
      const updated = await commonExec(() => ResetMovieType(id, value));
      if (item && updated) Object.assign(item, updated);
    }
  }
  resetSelector();
};

// ── 批量处理进度辅助 ──
const batchExec = async (items, itemFn) => {
  isBatchProcessing.value = true;
  batchProgress.value = 0;
  let ok = 0;
  let fail = 0;
  let idx = 0;
  for (const item of items) {
    try {
      const res = await itemFn(item, idx);
      if (res && res.Code === 200) ok++;
      else fail++;
    } catch {
      fail++;
    }
    batchProgress.value++;
    idx++;
  }
  isBatchProcessing.value = false;
  return { ok, fail, total: items.length };
};

// ── 批量删除（带确认） ──
const confirmDelete = () => {
  if (selectedCount.value === 0) return;
  $q.dialog({
    title: '确认删除',
    message: `确定要删除选中的 ${selectedCount.value} 个文件吗？此操作不可撤销。`,
    ok: { label: '删除', color: 'red', flat: true },
    cancel: { label: '取消', color: 'grey', flat: true },
  }).onOk(async () => {
    const { ok, fail, total } = await batchExec(view.selector.slice(), (id) =>
      DeleteFile(id)
    );
    $q.notify({
      type: ok === total ? 'positive' : 'warning',
      message: `删除完成：成功 ${ok}，失败 ${fail}，共 ${total}`,
      position: 'bottom-left',
    });
    resetSelector();
    fetchSearch();
  });
};

// ── 批量改名 ──
const batchRename = () => {
  if (selectedCount.value === 0) return;
  $q.dialog({
    title: '批量改名',
    message: '输入新名称（不含扩展名），多个文件将自动追加序号：',
    prompt: {
      model: '',
      type: 'text',
      label: '新名称',
    },
    ok: { label: '确认', color: 'teal', flat: true },
    cancel: { label: '取消', color: 'grey', flat: true },
  }).onOk(async (data) => {
    const baseName = (data || '').trim();
    if (!baseName) {
      $q.notify({ type: 'warning', message: '名称不能为空', position: 'bottom-left' });
      return;
    }
    const ids = view.selector.slice();
    const { ok, fail, total } = await batchExec(ids, async (id, i) => {
      const file = view.resultData.Data.find((f) => f.Id === id);
      if (!file) return { Code: -1 };
      const ext = file.FileType ? `.${file.FileType}` : '';
      const newTitle = ids.length === 1 ? baseName : `${baseName}_${String(i + 1).padStart(2, '0')}`;
      return FileRename({ Id: id, Name: newTitle + ext });
    });
    $q.notify({
      type: ok === total ? 'positive' : 'warning',
      message: `改名完成：成功 ${ok}，失败 ${fail}，共 ${total}`,
      position: 'bottom-left',
    });
    resetSelector();
    fetchSearch();
  });
};

const mergeFiles = () => {
  if (view.selector && view.selector.length > 0) {
    commonExec(() => FilesMerge({ files: view.selector, DeleteFlag: false }));
  }
};

const chooseInput = () => {
  setTimeout(() => {
    const inputElement = document.getElementsByClassName('inputWords');
    if (inputElement) {
      inputElement[0].focus();
    }
  }, 100);
};

const submitInput = async () => {
  if (view.input) {
    await addTagBySelector(view.input);
    view.input = '';
  }
};

const addPlayingMutiTag = async () => {
  if (view.submitMutiTag.length > 0) {
    const tags = view.submitMutiTag.join(',');
    await addTagBySelector(tags);
    view.submitMutiTag = [];
  }
};

const addTagBySelector = async (value) => {
  if (view.selector && view.selector.length > 0) {
    for (const id of view.selector) {
      const item = view.resultData.Data?.find((f) => f.Id === id);
      const updated = await commonExec(() => AddTag(id, value));
      if (item && updated) Object.assign(item, updated);
    }
  }
  resetSelector();
};

const refreshIndex = async () => {
  await RefreshAPI();
  await fetchSearch();
};

const nextPage = (n) => {
  view.queryParam.Page = view.queryParam.Page + n;
  fetchSearch();
};

const suggestions = computed(() => {
  return systemProperty.getSuggestions;
});

const fetchSearch = async () => {
  const data = await SearchAPI(view.queryParam);
  view.resultData = { ...data };
};

let debounceTimer;
const debouncedSearch = () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => fetchSearch(), 300);
};

const copyPath = async (item) => {
  try {
    await navigator.clipboard.writeText(item.Path || item.Name || '');
    $q.notify({ type: 'positive', message: '路径已复制', position: 'bottom-left', timeout: 1500 });
  } catch {
    $q.notify({ type: 'negative', message: '复制失败', position: 'bottom-left' });
  }
};

const open = (data) => {
  const { queryParam, settingInfo, cb, tabName } = data;
  if (tabName) {
    tab.value = tabName;
  }
  if (queryParam) {
    view.queryParam = queryParam;
    view.queryParam.PageSize = queryParam.PageSize;
  } else {
    view.queryParam = systemProperty.getSearchParam;
  }
  if (settingInfo) {
    view.settingInfo = settingInfo;
  } else {
    view.settingInfo = systemProperty.getSettingInfo;
  }
  view.callback = cb;
  dialogRef.value.show();
  fetchSearch();
  setTimeout(() => {
    const listEl = document.getElementById('listRef');
    if (!listEl) {
      console.warn('listRef element not found, skipping Sortable init');
      return;
    }
    if (sortableInstance) {
      sortableInstance.destroy();
    }
    sortableInstance = new Sortable(listEl, {
      animation: 150,
      onEnd: function (evt) {
        // 数组根据移动的位置进行重新排序
        if (evt.oldIndex != evt.newIndex) {
          view.resultData.Data.splice(
            evt.newIndex,
            0,
            view.resultData.Data.splice(evt.oldIndex, 1)[0]
          );
        }
      },
    });
  }, 1000);
};

// ── 任务日志 SSE（弹窗打开时连接，关闭时断开） ────────────
let taskLogEventSource = null;

const dialogHide = async () => {
  clearInterval(timeFunc);
  clearTimeout(debounceTimer);
  closeTaskLogSSE();
  if (sortableInstance) {
    sortableInstance.destroy();
    sortableInstance = null;
  }
  if (view.callback) {
    view.callback({ settingInfo: view.settingInfo });
  }
  onDialogCancel();
  onDialogOK();
  onDialogHide();
};

const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } =
  useDialogPluginComponent();

// 弹窗专用 SSE — 只处理 task_log 事件，弹窗关闭即断连
function openTaskLogSSE() {
  closeTaskLogSSE();
  const url = `${window.location.origin}/api/events`;
  taskLogEventSource = new EventSource(url);
  taskLogEventSource.onmessage = (e) => {
    try {
      const event = JSON.parse(e.data);
      if (event.Type === 'task_log' && event.Data?.taskKey) {
        notifyTaskLog();
      }
    } catch (err) {
      // 静默忽略
    }
  };
  taskLogEventSource.onerror = () => {
    closeTaskLogSSE();
  };
}

function closeTaskLogSSE() {
  if (taskLogEventSource) {
    taskLogEventSource.close();
    taskLogEventSource = null;
  }
}

const beforeShow = () => {
  openTaskLogSSE();
};

defineExpose({
  open,
});
</script>

<style>
.tag-item {
  margin: 2px 4px;
  padding: 1px 6px;
  border-radius: 8px;
}

.w100 {
  width: 100%;
}

/* 按钮压缩 */
.q-dialog .q-btn--glossy {
  min-height: 28px;
  padding: 2px 10px;
  font-size: 0.85rem;
}

.q-dialog .q-btn--dense {
  min-height: 24px;
  padding: 0 6px;
}

.q-dialog .q-btn-dropdown--dense {
  min-height: 24px;
}

/* 搜索栏移动端 column */
@media (max-width: 599px) {
  .search-toolbar {
    flex-direction: column !important;
    align-items: stretch !important;
  }

  .search-toolbar .q-input {
    width: 100% !important;
  }

  /* sticky 按钮组移动端压缩 */
  .q-page-sticky .column.items-end .q-btn {
    min-height: 26px;
    font-size: 0.8rem;
    padding: 0 8px;
  }

  .q-page-sticky .column.items-end .q-btn-dropdown {
    min-height: 26px;
    font-size: 0.8rem;
  }

  /* 列表项按钮压缩 */
  .q-expansion-item .q-btn--dense {
    min-height: 22px;
    font-size: 0.75rem;
    padding: 0 4px;
  }

  .q-expansion-item .q-btn-dropdown--dense {
    min-height: 22px;
    font-size: 0.75rem;
  }

  /* 底部信息区 */
  .q-page-sticky[position="bottom-left"] .row {
    font-size: 0.75rem;
  }
}
</style>
