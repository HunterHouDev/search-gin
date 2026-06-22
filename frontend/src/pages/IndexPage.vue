<template>
  <q-layout
    view="lHh lpr lFf"
    container
    style="height: 93vh"
  >
    <!-- 头部 -->
    <q-header elevated class="bg-gradient-primary">
      <div class="row justify-between items-center w100 q-pa-sm">
        <div class="row items-center q-gutter-sm">
          <IndexButton
            glossy
            color="primary"
            ref="indexButton"
            @refresh-done="loadTypeSize"
          />
          <q-btn 
            color="white" 
            text-color="primary" 
            label="刷新" 
            icon="refresh"
            flat
            @click="f5" 
          />
        </div>
        <q-btn-toggle
          v-model="currentDiv"
          color="white"
          outline
          glossy
          text-color="white"
          toggle-color="primary"
          toggle-text-color="white"
          @update:model-value="toDiv"
          :options="[
            { value: 'tagDiv', label: '标签', icon: 'label' },
            { value: 'seriesDiv', label: '系列', icon: 'movie' },
            { value: 'typeDiv', label: '类型', icon: 'category' },
            { value: 'diskDiv', label: '容量', icon: 'hard_drive' },
            { value: 'scanTimeDiv', label: '耗时', icon: 'timer' },
          ]"
        />
      </div>
    </q-header>
    
    <q-page-container class="q-pa-md q-gutter-md">
      <!-- 加载状态 -->
      <SkeletonLoader v-if="isLoading" type="list" :count="8" />
      
      <!-- 标签分析卡片 -->
      <q-card class="cardcard" v-if="tagData.filter(t => t.Cnt > 1).length > 0">
        <q-toolbar class="bg-gradient-primary text-white" id="tagDiv">
          <q-icon name="label" class="q-mr-sm" />
          标签分析
          <q-space />
          <q-badge color="orange" text-color="white">
            {{ tagData.filter(t => t.Cnt > 1).length }} 个标签
          </q-badge>
        </q-toolbar>
        <div class="q-pa-md">
          <div class="row q-gutter-sm">
            <q-btn
              v-for="tag in tagData.filter(t => t.Cnt > 1)"
              :key="tag.Name"
              color="primary"
              glossy
              rounded
              @click="folderGotoMenu(tag.Name)"
            >
              {{ tag.Name }}
              <q-badge color="orange" text-color="white" class="q-ml-xs">
                {{ tag.Cnt }}
              </q-badge>
              <q-tooltip>{{ tag.SizeStr }}</q-tooltip>
            </q-btn>
          </div>
        </div>
      </q-card>

      <!-- 系列分析卡片 -->
      <q-card class="cardcard" v-if="seriesData.filter(t => t.Cnt > 1).length > 0">
        <q-toolbar class="bg-gradient-primary text-white" id="seriesDiv">
          <q-icon name="movie" class="q-mr-sm" />
          系列分析
          <q-space />
          <q-badge color="orange" text-color="white">
            {{ seriesData.filter(t => t.Cnt > 1).length }} 个系列
          </q-badge>
        </q-toolbar>
        <div class="q-pa-md">
          <div class="row q-gutter-sm">
            <q-btn
              v-for="tag in seriesData.filter(t => t.Cnt > 1)"
              :key="tag.Name"
              color="secondary"
              glossy
              rounded
              @click="folderGotoMenu(tag.Name)"
            >
              {{ tag.Name }}
              <q-badge color="orange" text-color="white" class="q-ml-xs">
                {{ tag.Cnt }}
              </q-badge>
              <q-tooltip>{{ tag.SizeStr }}</q-tooltip>
            </q-btn>
          </div>
        </div>
      </q-card>

      <!-- 类型分析卡片 -->
      <q-card class="cardcard">
        <q-toolbar class="bg-gradient-primary text-white" id="typeDiv">
          <q-icon name="category" class="q-mr-sm" />
          类型分析
          <q-space />
          <q-badge color="orange" text-color="white">
            {{ tableData.length }} 种类型
          </q-badge>
        </q-toolbar>
        <div class="q-pa-md">
          <div class="row q-gutter-md">
            <q-card
              class="type-card"
              v-for="item in tableData"
              :key="item.Name"
              flat
              bordered
            >
              <q-badge color="negative" floating>{{ item.Cnt }}</q-badge>
              <q-card-section>
                <div class="row items-center justify-between">
                  <q-btn
                    dense
                    :icon="item.IsDir ? 'folder' : 'description'"
                    :color="item.IsDir ? 'amber' : 'primary'"
                    flat
                    @click="gotoMenu(item)"
                  >
                    {{ item.IsDir ? '📁 ' + item.Name : item.Name }}
                  </q-btn>
                </div>
                <div class="text-caption q-mt-sm" style="color: var(--q-text-secondary)">
                  <q-icon name="storage" size="xs" />
                  {{ item.SizeStr }}
                  <q-icon name="description" size="xs" class="q-ml-sm" />
                  {{ item.Cnt }} 个文件
                </div>
              </q-card-section>

              <q-card-actions v-if="item.IsDir" align="center">
                <q-btn
                  color="primary"
                  flat
                  glossy
                  dense
                  icon="folder-open"
                  @click="openThis(item)"
                >打开</q-btn>
                <q-btn
                  color="negative"
                  glossy
                  dense
                  flat
                  icon="delete"
                  @click="deleteThis(item)"
                >删除</q-btn>
              </q-card-actions>
            </q-card>
          </div>
        </div>
      </q-card>

      <!-- 磁盘容量卡片 -->
      <q-card class="cardcard" v-if="diskUsage.length > 0">
        <q-toolbar class="bg-gradient-primary text-white" id="diskDiv">
          <q-icon name="hard_drive" class="q-mr-sm" />
          磁盘容量
          <q-space />
          <q-badge color="orange" text-color="white">
            {{ diskUsage.length }} 个磁盘
          </q-badge>
        </q-toolbar>
        <div class="q-pa-md">
          <div class="q-gutter-md">
            <div v-for="item in diskUsage" :key="item.Path" class="disk-item">
              <div class="row items-center justify-between q-mb-xs">
                <div class="text-subtitle2">
                  <q-icon name="folder" color="primary" class="q-mr-xs" />
                  {{ item.Path }}
                </div>
                <div class="text-caption text-grey">
                  {{ formatSize(item.Used) }} / {{ formatSize(item.All) }}
                </div>
              </div>
              <q-linear-progress
                :value="item.Percent / 100"
                :color="getProgressColor(item.Percent)"
                size="12px"
                rounded
                stripe
              >
                <div class="absolute-full flex flex-center">
                  <q-badge color="white" text-color="dark" :label="item.Percent.toFixed(1) + '%'" />
                </div>
              </q-linear-progress>
            </div>
          </div>
        </div>
      </q-card>

      <!-- 扫描耗时卡片 -->
      <q-card class="cardcard">
        <q-toolbar class="bg-gradient-primary text-white" id="scanTimeDiv">
          <q-icon name="timer" class="q-mr-sm" />
          扫描耗时
          <q-space />
          <q-badge color="orange" text-color="white">
            {{ scanTime.length }} 个目录
          </q-badge>
        </q-toolbar>
        <div class="q-pa-md">
          <div class="row q-gutter-md">
            <q-card
              v-for="item in scanTime"
              :key="item.Name"
              class="disk-card"
              flat
              bordered
            >
              <q-card-section>
                <div class="row items-center justify-between">
                  <q-btn
                    flat
                    dense
                    icon="folder"
                    :label="item.Name"
                    color="primary"
                    @click="folderGotoMenu(item.Name)"
                  />
                </div>
                <div class="text-caption q-mt-sm" style="color: var(--q-text-secondary)">
                  <q-icon name="storage" size="xs" />
                  {{ item.SizeStr }}
                  <q-icon name="timer" size="xs" class="q-ml-sm" />
                  {{ item.Cnt }}ms
                </div>
              </q-card-section>

              <q-card-actions v-if="item.IsDir" align="center">
                <q-btn
                  color="primary"
                  flat
                  glossy
                  dense
                  icon="folder-open"
                  @click="openThis(item)"
                >打开</q-btn>
                <q-btn
                  color="negative"
                  dense
                  glossy
                  flat
                  icon="delete"
                  @click="deleteThis(item)"
                >删除</q-btn>
              </q-card-actions>
            </q-card>
          </div>
        </div>
      </q-card>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { onMounted, onUnmounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import SkeletonLoader from 'components/SkeletonLoader.vue';
import {
  DeleteFolderByPath,
  OpenFolderByPath,
} from '../components/api/searchAPI';
import {
  ScanTime,
  TagSizeMap,
  TypeSizeMap,
  SeriesCount,
  DiskUsage,
} from '../components/api/homeAPI';
import { onKeyStroke } from '@vueuse/core';
import { useSystemProperty } from '../stores/System';
import IndexButton from 'components/IndexButton.vue';
const { push } = useRouter();
const systemProperty = useSystemProperty();
document.title = '分析';

const $q = useQuasar();
const tableData = ref([]);
const tagData = ref([]);
const seriesData = ref([]);
const scanTime = ref([]);
const diskUsage = ref([]);
const currentDiv = ref('tagDiv');
const isLoading = ref(true);
let inter;

onKeyStroke(['`'], () => {
  refreshIndex();
});

const folderGotoMenu = (Name) => {
  systemProperty.setPage(1);
  systemProperty.FileSearchParam.Keyword = Name;
  systemProperty.setMovieType('');
  push('/search?from=index');
};

const toDiv = (id) => {
  const element = document.getElementById(id);
  if (!element) return;
  element.scrollIntoView({ behavior: 'smooth', block: 'center' });
};

const gotoMenu = (data) => {
  const { IsDir, Name } = data;
  const movieType = !IsDir && Name !== '全部' ? Name : '';
  systemProperty.setPage(1);
  if (IsDir) {
    systemProperty.setKeyword(Name);
  }
  systemProperty.setMovieType(movieType);
  push('/search?from=index');
};
const loadTypeSize = async () => {
  isLoading.value = true;
  const res = await TypeSizeMap();
  if (res) {
    tableData.value = res;
  }
  await Promise.all([
    loadTagSize(),
    loadScanTime(),
    loadSeriesCount(),
    loadDiskUsage(),
  ]);
  isLoading.value = false;
};

const loadTagSize = async () => {
  const res = await TagSizeMap();
  if (res) {
    tagData.value = res.length > 80 ? res.slice(0, 80) : res;
  }
};

const loadSeriesCount = async () => {
  const res = await SeriesCount();
  if (res) {
    seriesData.value = res.length > 80 ? res.slice(0, 80) : res;
  }
};

const loadDiskUsage = async () => {
  const res = await DiskUsage();
  if (res) {
    diskUsage.value = res;
  }
};

const loadScanTime = async () => {
  scanTime.value = (await ScanTime()) || [];
  scanTime.value = scanTime.value.sort((a, b) => {
    return b.Cnt - a.Cnt;
  });
  systemProperty.SettingInfo.Dirs.forEach((item) => {
    if (scanTime.value) {
      const find = scanTime.value.find((i) => i.Name === item);
      if (!find) {
        scanTime.value.unshift({
          Name: item,
          Cnt: 0,
          Size: 0,
          SizeStr: '执行中',
        });
      }
    } else {
      scanTime.value.unshift({
        Name: item,
        Cnt: 0,
        Size: 0,
        SizeStr: '执行中',
      });
    }
  });
};
onMounted(() => {
  inter = setInterval(() => {
    if (!tableData.value || tableData.value.length === 0) {
      loadTypeSize();
    } else {
      clearInterval(inter);
    }
  }, 5000);
});
onUnmounted(() => {
  if (inter) clearInterval(inter);
});

const openThis = async (data) => {
  const { Name } = data;
  const res = await OpenFolderByPath({ dirpath: Name });
  if (res.Code === 200) {
    $q.notify({ type: 'positive', message: '执行成功' });
  } else {
    $q.notify({ type: 'warning', message: '执行失败' });
  }
};
const deleteThis = async (data) => {
  const { Name } = data;
  const res = await DeleteFolderByPath({ dirpath: Name });
  if (res.Code === 200) {
    $q.notify({ type: 'positive', message: '执行成功' });
    indexButton.value.refreshIndex();
    await f5();
  } else {
    $q.notify({ type: 'warning', message: '执行失败' });
  }
};

const formatSize = (bytes) => {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const getProgressColor = (percent) => {
  if (percent >= 90) return 'negative';
  if (percent >= 70) return 'warning';
  return 'positive';
};
const refreshIndex = async () => {
  indexButton.value.refreshIndex();
};

const f5 = () => {
  window.location.reload();
};
</script>
<style scoped>
.cardcard {
  border-radius: 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  transition: all 0.3s ease;
}

.cardcard:hover {
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.type-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  min-width: 200px;
  flex: 1 1 auto;
}

.type-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.disk-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  min-width: 200px;
  flex: 1 1 auto;
}

.disk-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.disk-item {
  padding: 12px;
  border-radius: 8px;
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
}

.bg-gradient-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.text_subtitle {
  color: var(--q-text-secondary);
}
</style>
