<template>
  <q-dialog ref="dialogRef" v-model="show" @hide="dialogHide" @before-show="beforeShow">
    <q-layout container view="hHh Lpr lff" style="margin: 0"
      :style="isMobile ? { width: '90vw', height: '90vh' } : { width: '400px', height: '700px' }">
      <q-header class="bg-primary text-white shadow-2 row items-center q-px-sm"
        style="min-height: 40px; border-radius: 8px 8px 0 0">
        <q-toolbar-title class="text-subtitle2">批量操作</q-toolbar-title>
        <q-space />
        <div class="row q-gutter-xs">
          <q-btn glossy color="white" text-color="black" size="sm" @click="selectAll">
            {{ state.selectAll ? '取消' : '全选' }}
            <q-badge v-if="selectedCount > 0" color="red" floating>{{ selectedCount }}</q-badge>
          </q-btn>
          <q-btn-dropdown label="类型" glossy dense color="white" text-color="primary" size="sm">
            <q-list dense>
              <q-item v-for="mt in MovieTypeOptions" :key="mt.value" v-close-popup clickable
                @click="setTypeBySelector(mt.value)">
                <q-item-section>{{ mt.label }}</q-item-section>
              </q-item>
            </q-list>
          </q-btn-dropdown>
          <q-btn-dropdown v-permission="'op:tag'" label="标签" dense glossy color="white" text-color="primary" size="sm">
            <div class="q-pa-sm" style="min-width: 220px">
              <div class="row items-center q-mb-sm">
                <q-radio v-model="state.chooseInput" :val="false" label="常用" dense />
                <q-checkbox v-model="state.chooseInput" :val="false" label="输入" dense class="q-ml-md" />
              </div>
              <div v-if="state.chooseInput">
                <q-input v-model="state.input" dense label="输入标签" class="q-mb-sm" />
                <q-btn color="orange" size="sm" label="提交" class="full-width" v-close-popup @click="submitInput" />
              </div>
              <div v-else>
                <q-btn color="orange" size="sm" label="提交" class="full-width q-mb-sm" v-close-popup
                  @click="addPlayingMutiTag" />
                <q-checkbox v-model="state.submitMutiTag" v-for="tag in state.settingInfo.Tags" :key="tag" :val="tag"
                  dense keep-color :label="tag.substring(0, 6)" color="red" class="q-pr-md" />
              </div>
            </div>
          </q-btn-dropdown>
          <q-btn v-permission="'op:edit'" glossy size="sm" color="teal"
            :disable="selectedCount === 0 || isBatchProcessing" @click="batchRename">改名</q-btn>
          <q-btn v-permission="'op:edit'" glossy size="sm" color="red"
            :disable="selectedCount === 0 || isBatchProcessing" @click="confirmDelete">删除</q-btn>
          <q-btn v-permission="'op:merge'" glossy size="sm" color="orange"
            :disable="selectedCount === 0 || isBatchProcessing" @click="mergeFiles">合并</q-btn>
        </div>
        <q-btn flat dense icon="close" size="sm" class="q-ml-sm" @click="dialogHide" />
      </q-header>

      <q-page-container style="padding-top: 2.8rem">
        <q-page class="q-pa-sm">
          <!-- 搜索栏 -->
          <div class="row q-gutter-sm q-mb-sm items-center">
            <q-input v-model="state.queryParam.Keyword" dense filled outlined color="primary"
              placeholder="搜索..." style="width: 10rem" @update:model-value="debouncedSearch">
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
            <q-btn glossy size="sm" color="black" icon="refresh" @click="refreshIndex" />
            <q-btn glossy size="sm" color="black" icon="chevron_left" @click="nextPage(-1)" />
            <q-btn glossy size="sm" color="black" icon="chevron_right" @click="nextPage(1)" />
          </div>

          <!-- 类型 chips -->
          <div v-if="state.settingInfo.MovieTypes?.length" class="q-mb-sm q-gutter-xs row">
            <q-chip v-for="mt in state.settingInfo.MovieTypes" :key="mt"
              :color="state.queryParam.MovieType === mt ? 'primary' : 'grey-6'" text-color="white" size="sm" clickable
              @click="state.queryParam.MovieType = state.queryParam.MovieType === mt ? '' : mt; fetchSearch()">
              {{ mt }}
            </q-chip>
            <q-chip v-if="state.queryParam.MovieType" color="red" text-color="white" size="sm" clickable icon="close"
              @click="state.queryParam.MovieType = ''; fetchSearch()">清除</q-chip>
          </div>

          <!-- 文件列表 -->
          <div id="batchListRef" style="height: calc(82vh - 160px); overflow: auto">
            <div v-if="!state.resultData.Data?.length"
              class="column items-center q-pa-xl text-grey-7">
              <q-icon name="search_off" size="3rem" class="q-mb-md" />
              <div class="text-h6">没有找到匹配的文件</div>
              <q-btn color="primary" flat class="q-mt-md" @click="refreshIndex">刷新索引</q-btn>
            </div>
            <div v-for="item in state.resultData.Data" :key="item.Id"
              class="q-mb-xs" style="border: 1px solid rgba(128,0,128,0.15); border-radius: 6px">
              <div class="row items-center q-px-sm q-py-xs" style="gap: 4px">
                <!-- checkbox + thumb -->
                <q-checkbox v-model="state.selector" :val="item.Id" color="red" dense size="xs" class="q-mr-xs" />
                <q-img v-if="item.PngUrl" :src="item.PngUrl" style="width: 48px; height: 36px; border-radius: 4px"
                  fit="cover" @click="checkThis(item)" />

                <!-- 文件信息 -->
                <div class="col" style="min-width: 0; line-height: 1.3">
                  <div class="row items-center" style="gap: 3px; flex-wrap: wrap">
                    <span v-if="state.cutListIds.includes(item.Id)" style="color: red; font-size: 11px">剪切中</span>
                    <q-chip dense size="sm" :label="item.MovieType" color="blue-6" text-color="white"
                      class="q-mr-none" />
                    <span class="text-caption text-grey-7">【{{ item.SizeStr }}】</span>
                    <span class="text-caption text-weight-medium" style="
                        display: -webkit-box; -webkit-box-orient: vertical; line-clamp: 1;
                        overflow: hidden; text-overflow: ellipsis;
                      ">{{ item.Title }}</span>
                  </div>
                  <div class="row items-center" style="gap: 3px; flex-wrap: wrap">
                    <span class="text-caption text-purple-7 cursor-pointer"
                      @click="state.queryParam.Keyword = item.Author; fetchSearch()">
                      {{ item.Author?.substring(0, 8) }}
                    </span>
                    <span class="text-caption text-grey-5">{{ item.FileType }}</span>
                    <q-chip v-for="ta in (item.Tags || [])" :key="ta" dense size="sm"
                      color="orange-2" text-color="orange-9" removable @remove="doCloseTag(item, ta)">
                      {{ ta }}
                    </q-chip>
                  </div>
                </div>

                <!-- 操作按钮 -->
                <q-btn v-permission="'op:movie:type'" flat dense size="sm" icon="label" color="blue-6" class="q-ml-auto">
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
                <q-btn flat dense size="sm" icon="open_in_new" @click="commonExec(() => OpenFileFolder(item.Id))">
                  <q-tooltip>打开文件夹</q-tooltip>
                </q-btn>
                <q-btn flat dense size="sm" icon="play_circle" @click="playNewWindow(item)">
                  <q-tooltip>播放</q-tooltip>
                </q-btn>
                <q-btn flat dense size="sm" icon="content_copy" color="grey" @click.stop="copyPath(item)">
                  <q-tooltip>复制路径</q-tooltip>
                </q-btn>
                <q-btn-dropdown v-permission="'op:transcode'" flat dense size="sm" icon="transform" color="teal">
                  <q-tooltip>转码</q-tooltip>
                  <q-list dense>
                    <q-item v-close-popup clickable @click="toMp4(item)"><q-item-section>MP4</q-item-section></q-item>
                    <q-item v-close-popup clickable @click="toVcode(item, 'h264')"><q-item-section>H264</q-item-section></q-item>
                    <q-item v-close-popup clickable @click="toVcode(item, 'h265')"><q-item-section>H265</q-item-section></q-item>
                  </q-list>
                </q-btn-dropdown>
                <span class="text-caption text-grey-6 cursor-pointer"
                  style="font-size: 10px" @click="searchCode(item)">{{ item.Code?.substring(0, 10) }}</span>
              </div>
            </div>
          </div>
        </q-page>
      </q-page-container>

      <!-- 底部状态 -->
      <q-footer class="bg-grey-2 text-caption text-grey-7 row items-center q-px-sm" style="min-height: 28px; border-radius: 0 0 8px 8px">
        <span>第 {{ state.queryParam.Page }} 页，{{ state.queryParam.PageSize }} 条/页，共 {{ state.resultData.TotalCnt || 0 }} 条</span>
        <q-space />
        <span v-if="isBatchProcessing" class="text-orange text-bold q-mr-sm">
          <q-spinner-dots size="xs" color="orange" /> {{ batchProgress }}/{{ selectedCount }}
        </span>
      </q-footer>
    </q-layout>
  </q-dialog>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { reactive, ref, computed } from 'vue';
import { useSystemProperty } from 'stores/System';
import { useCommonExec } from 'src/composables/useCommonExec';
import { useBreakpoint } from 'src/composables/useBreakpoint';
import { useDialogShell } from 'src/composables/useDialogShell';
import { MovieTypeOptions } from 'components/utils';
import {
  ResetMovieType, SearchAPI, RefreshAPI, DeleteFile, FilesMerge,
  TansferFileVcode, CloseTag, AddTag, FileRename, OpenFileFolder,
} from 'components/api/searchAPI';
import Sortable from 'sortablejs';

const $q = useQuasar();
const systemProperty = useSystemProperty();
const { isMobile } = useBreakpoint();
const { exec: commonExec } = useCommonExec({ notifyOnSuccess: true });

let sortableInstance = null;
let debounceTimer;

const { show, dialogRef, dialogHide, beforeShow } = useDialogShell(() => {
  clearTimeout(debounceTimer);
  if (sortableInstance) { sortableInstance.destroy(); sortableInstance = null; }
  if (state.callback) state.callback({ settingInfo: state.settingInfo });
});

const state = reactive({
  selectAll: false,
  selector: [],
  cutListIds: [],
  resultData: {},
  queryParam: { Keyword: '', MovieType: '', Page: 1, PageSize: 20 },
  settingInfo: {} ,
  callback: null ,
  submitMutiTag: [],
  chooseInput: false,
  input: '',
});

const isBatchProcessing = ref(false);
const batchProgress = ref(0);
const selectedCount = computed(() => state.selector.length);
const suggestions = computed(() => systemProperty.getSuggestions);

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

const batchExec = async (items, itemFn) => {
  isBatchProcessing.value = true;
  batchProgress.value = 0;
  let ok = 0, fail = 0;
  for (let i = 0; i < items.length; i++) {
    try {
      const res = await itemFn(items[i], i);
      res && res.Code === 200 ? ok++ : fail++;
    } catch { fail++; }
    batchProgress.value++;
  }
  isBatchProcessing.value = false;
  return { ok, fail, total: items.length };
};

const confirmDelete = () => {
  if (!selectedCount.value) return;
  $q.dialog({
    title: '确认删除',
    message: `确定要删除 ${selectedCount.value} 个文件？此操作不可撤销。`,
    ok: { label: '删除', color: 'red', flat: true },
    cancel: { label: '取消', color: 'grey', flat: true },
  }).onOk(async () => {
    const { ok, fail, total } = await batchExec(state.selector.slice(), (id) => DeleteFile(id));
    $q.notify({ type: ok === total ? 'positive' : 'warning', message: `删除：成功 ${ok}，失败 ${fail}，共 ${total}`, position: 'bottom-left' });
    resetSelector();
    fetchSearch();
  });
};

const batchRename = () => {
  if (!selectedCount.value) return;
  $q.dialog({
    title: '批量改名',
    message: '输入新名称（不含扩展名），多文件将自动追加序号',
    prompt: { model: '', type: 'text', label: '新名称' },
    ok: { label: '确认', color: 'teal', flat: true },
    cancel: { label: '取消', color: 'grey', flat: true },
  }).onOk(async (data) => {
    const baseName = (data || '').trim();
    if (!baseName) { $q.notify({ type: 'warning', message: '名称不能为空', position: 'bottom-left' }); return; }
    const ids = state.selector.slice();
    const { ok, fail, total } = await batchExec(ids, async (id, i) => {
      const file = state.resultData.Data.find((f) => f.Id === id);
      if (!file) return { Code: -1 };
      const ext = file.FileType ? `.${file.FileType}` : '';
      return FileRename({ Id: id, Name: (ids.length === 1 ? baseName : `${baseName}_${String(i + 1).padStart(2, '0')}`) + ext });
    });
    $q.notify({ type: ok === total ? 'positive' : 'warning', message: `改名：成功 ${ok}，失败 ${fail}，共 ${total}`, position: 'bottom-left' });
    resetSelector();
    fetchSearch();
  });
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
  show.value = true;
  fetchSearch();
  initSortable();
};

defineExpose({ open });
</script>

<style scoped>
</style>
