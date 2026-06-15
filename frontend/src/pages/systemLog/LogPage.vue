<template>
  <div class="q-pa-md">
    <q-card class="theme-card">
      <q-tabs
        v-model="tab"
        dense
        class="text-grey"
        active-color="primary"
        indicator-color="primary"
        align="left"
      >
        <q-tab name="memory" label="内存日志" />
        <q-tab name="local" label="本地日志" />
      </q-tabs>

      <q-separator />

      <q-card-section>
        <!-- 内存日志 -->
        <template v-if="tab === 'memory'">
          <div class="row items-center q-gutter-sm q-mb-md">
            <h6 class="text-subtitle1 col">内存日志</h6>

            <q-btn
              :icon="sortAsc ? 'arrow_upward' : 'arrow_downward'"
              flat
              dense
              @click="sortAsc = !sortAsc"
            >
              <q-tooltip>{{ sortAsc ? '时间正序' : '时间倒序' }}</q-tooltip>
            </q-btn>

            <q-btn-toggle
              v-model="timeFilter"
              :options="timeOptions"
              dense
              flat
              no-caps
              class="q-ml-xs"
            />

            <q-select
              v-model="typeFilter"
              :options="typeOptions"
              dense
              clearable
              placeholder="类型"
              class="col-2"
              style="min-width:100px"
            />

            <q-input
              v-model="keyword"
              dense
              debounce="300"
              placeholder="过滤关键词"
              clearable
              class="col-3"
            >
              <template v-slot:prepend>
                <q-icon name="search" />
              </template>
            </q-input>
          </div>

          <div class="log-list">
            <div v-for="item in memoryPageData" :key="item" class="log-item q-py-xs">
              <span class="log-type-dot" :class="typeColor(extractType(item.msg))" />
              <span class="log-time">{{ item.time.substring(0,19) }}</span>
              <span class="log-separator"> - </span>
              <span class="log-msg">{{ item.msg }}</span>
            </div>
            <div v-if="memoryPageData.length === 0" class="text-center text-grey q-py-md">
              暂无匹配的日志
            </div>
          </div>

          <div class="row justify-center q-mt-md" v-if="memoryTotalPages > 1">
            <q-pagination
              v-model="memoryPage"
              :max="memoryTotalPages"
              :max-pages="7"
              boundary-links
              direction-links
            />
          </div>
        </template>

        <!-- 本地日志 -->
        <template v-if="tab === 'local'">
          <div class="row items-center q-mb-md">
            <h6 class="text-subtitle1 col">本地日志 (gin.log)</h6>
            <q-btn flat dense icon="refresh" @click="fetchLocalLog" />
          </div>

          <div class="log-list">
            <div v-for="(line, idx) in localPageData" :key="idx" class="log-item q-py-xs">
              <span class="log-raw">{{ line }}</span>
            </div>
            <div v-if="localPageData.length === 0" class="text-center text-grey q-py-md">
              暂无日志
            </div>
          </div>

          <div class="row justify-center q-mt-md" v-if="localTotalPages > 1">
            <q-pagination
              v-model="localPage"
              :max="localTotalPages"
              :max-pages="7"
              boundary-links
              direction-links
            />
          </div>
        </template>
      </q-card-section>
    </q-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, onUnmounted, watch } from 'vue';
import { GeMemeryLog, GetLocalLog } from '../../components/api/settingAPI';

// ── 状态 ──
const tab = ref('memory');
const keyword = ref('');
const sortAsc = ref(true);
const typeFilter = ref(null);
const timeFilter = ref('');
const timeOptions = [
  { label: '全部', value: '' },
  { label: '今天', value: 'today' },
  { label: '昨天', value: 'yesterday' },
  { label: '≥3天', value: 'older' },
];
const pageSize = 50;

// 内存日志
const allMemoryLogs = ref([]);
const memoryPage = ref(1);

// 本地日志
const allLocalLines = ref([]);
const localPage = ref(1);

// ── 内存日志：提取类型 ──
function extractType(msg) {
  if (!msg) return '';
  const m = msg.match(/^[^：:　\s]+/);
  return m ? m[0] : '';
}

// 根据类型前缀返回颜色 class
const typeColorMap = {
  '扫描': 'type-scan',
  '添加': 'type-add',
  '取消': 'type-cancel',
  '开始': 'type-scan',
  '完成': 'type-done',
  '拒绝': 'type-deny',
  '首次': 'type-join',
  '新节点': 'type-join',
  '全量': 'type-scan',
  'Plan': 'type-info',
  'ScanAll': 'type-scan',
  '索引': 'type-info',
  '搜索': 'type-search',
};
function typeColor(t) {
  return typeColorMap[t] || 'type-default';
}

const typeOptions = computed(() => {
  const seen = new Set();
  const types = [];
  for (const item of allMemoryLogs.value) {
    const t = extractType(item.msg);
    if (t && !seen.has(t)) {
      seen.add(t);
      types.push(t);
    }
  }
  return types.sort();
});

// ── 内存日志：过滤 + 排序 ──
function getDateYMD(timeStr) {
  return timeStr ? timeStr.substring(0, 10) : '';
}

function todayYMD() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function daysAgoYMD(n) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

const memoryFiltered = computed(() => {
  let list = [...allMemoryLogs.value];

  const kw = keyword.value?.trim().toLowerCase();
  if (kw) {
    list = list.filter(
      (item) => item.msg?.toLowerCase().includes(kw) || item.time?.includes(kw)
    );
  }

  if (typeFilter.value) {
    list = list.filter((item) => extractType(item.msg) === typeFilter.value);
  }

  // 时间范围过滤
  if (timeFilter.value) {
    const today = todayYMD();
    const yesterday = daysAgoYMD(1);
    const threeDaysAgo = daysAgoYMD(3);
    list = list.filter((item) => {
      const d = getDateYMD(item.time);
      if (!d) return false;
      switch (timeFilter.value) {
        case 'today': return d === today;
        case 'yesterday': return d === yesterday;
        case 'older': return d <= threeDaysAgo;
        default: return true;
      }
    });
  }

  list.sort((a, b) => {
    if (sortAsc.value) return a.time.localeCompare(b.time);
    return b.time.localeCompare(a.time);
  });

  return list;
});

const memoryTotalPages = computed(() =>
  Math.max(1, Math.ceil(memoryFiltered.value.length / pageSize))
);

const memoryPageData = computed(() => {
  const start = (memoryPage.value - 1) * pageSize;
  return memoryFiltered.value.slice(start, start + pageSize);
});

// 过滤条件变化时重置到第一页
watch([keyword, typeFilter, timeFilter, sortAsc], () => {
  memoryPage.value = 1;
});

// ── 本地日志：全量拉取 + 前端分页 ──

const localTotalPages = computed(() =>
  Math.max(1, Math.ceil(allLocalLines.value.length / pageSize))
);

const localPageData = computed(() => {
  const start = (localPage.value - 1) * pageSize;
  return allLocalLines.value.slice(start, start + pageSize);
});

async function fetchLocalLog() {
  const { data } = await GetLocalLog();
  allLocalLines.value = Array.isArray(data) ? data : [];
}

// ── 获取内存日志 ──
async function fetchMemoryLog() {
  const { data } = await GeMemeryLog();
  allMemoryLogs.value = Array.isArray(data) ? data : [];
}

// ── 生命周期 ──
let intervalId;

onMounted(() => {
  document.title = '系统日志';
  fetchMemoryLog();
  fetchLocalLog();
  intervalId = setInterval(() => {
    if (tab.value === 'memory') fetchMemoryLog();
    else fetchLocalLog();
  }, 5000);
});

onUnmounted(() => {
  clearInterval(intervalId);
});
</script>
<style lang="scss" scoped>
.theme-card {
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  color: var(--q-text-primary);
}

.text-subtitle1 {
  color: var(--q-text-secondary);
}

.log-list {
  max-height: 70vh;
  overflow-y: auto;
}

.log-item {
  border-bottom: 1px solid var(--q-border-light);
  font-family: monospace;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
}

.log-time {
  color: var(--q-text-secondary);
}

.log-separator {
  color: var(--q-border);
}

.log-msg {
  color: var(--q-text-primary);
}

/* 类型颜色圆点 */
.log-type-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
  flex-shrink: 0;
}
.type-scan   { background: #42a5f5; }  /* 蓝 */
.type-add    { background: #66bb6a; }  /* 绿 */
.type-cancel { background: #ef5350; }  /* 红 */
.type-done   { background: #26a69a; }  /* 青 */
.type-deny   { background: #ff7043; }  /* 橙 */
.type-join   { background: #ab47bc; }  /* 紫 */
.type-info   { background: #78909c; }  /* 灰蓝 */
.type-search { background: #ffca28; }  /* 黄 */
.type-default { background: #90a4ae; } /* 默认灰 */
</style>
