<template>
  <div class="q-pa-md">
    <q-card class="theme-card">
      <q-card-section>
        <div class="row items-center q-gutter-sm q-mb-md">
          <h6 class="text-subtitle1 col">系统日志</h6>

          <q-btn
            :icon="sortAsc ? 'arrow_upward' : 'arrow_downward'"
            flat
            dense
            @click="sortAsc = !sortAsc"
          >
            <q-tooltip>{{ sortAsc ? '时间正序' : '时间倒序' }}</q-tooltip>
          </q-btn>

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
          <div v-for="item in displayLogs" :key="item" class="log-item q-py-xs">
            <span class="log-time">{{ item.time.substring(0,19) }}</span>
            <span class="log-separator"> - </span>
            <span class="log-msg">{{ item.msg }}</span>
          </div>
          <div v-if="displayLogs.length === 0" class="text-center text-grey q-py-md">
            暂无匹配的日志
          </div>
        </div>
      </q-card-section>
    </q-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, onUnmounted } from 'vue';
import { GeMemeryLog } from '../../components/api/settingAPI';

const view = reactive({
  logs: [],
});

const keyword = ref('');
const sortAsc = ref(true);
const typeFilter = ref(null);

// 从 msg 提取类型前缀（冒号/空格前的第一个词）
function extractType(msg) {
  if (!msg) return '';
  const m = msg.match(/^[^：:　\s]+/);
  return m ? m[0] : '';
}

// 动态类型选项
const typeOptions = computed(() => {
  const seen = new Set();
  const types = [];
  for (const item of view.logs) {
    const t = extractType(item.msg);
    if (t && !seen.has(t)) {
      seen.add(t);
      types.push(t);
    }
  }
  return types.sort();
});

// 过滤 + 排序后的日志
const displayLogs = computed(() => {
  let list = view.logs;

  // 关键词过滤
  const kw = keyword.value?.trim().toLowerCase();
  if (kw) {
    list = list.filter(
      (item) => item.msg?.toLowerCase().includes(kw) || item.time?.includes(kw)
    );
  }

  // 类型过滤
  if (typeFilter.value) {
    list = list.filter((item) => extractType(item.msg) === typeFilter.value);
  }

  // 时间排序
  const sorted = [...list];
  sorted.sort((a, b) => {
    if (sortAsc.value) return a.time.localeCompare(b.time);
    return b.time.localeCompare(a.time);
  });

  return sorted;
});

const fetchSearch = async () => {
  const { data } = await GeMemeryLog();
  view.logs = Array.isArray(data) ? data : [];
};

let intervalId;

onMounted(() => {
  document.title = '系统日志';
  fetchSearch();
  intervalId = setInterval(fetchSearch, 5000);
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
</style>
