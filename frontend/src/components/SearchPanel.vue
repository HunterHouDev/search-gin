<template>
  <div class="search-panel" v-show="visible">
    <!-- 头部 -->
    <div class="search-panel-header">
      <!-- tab 切换 -->
      <q-tabs v-model="activeTab" dense active-color="indigo-4" indicator-color="indigo-4"
        class="sp-tabs" align="left">
        <q-tab name="search" label="搜索列表" no-caps />
        <q-tab name="images" label="本地图片" no-caps />
      </q-tabs>
      <div class="sp-header-actions">
        <q-btn flat round dense size="sm" icon="refresh" @click="onRefresh" color="indigo-4" />
        <q-btn flat round dense size="sm" icon="close" @click="$emit('close')" color="grey-4" />
      </div>
    </div>

    <!-- 搜索条件 (仅搜索列表) -->
    <div class="search-conditions" v-show="activeTab === 'search'">
      <div class="filter-item">
        <div class="filter-row">
          <!-- 移动端下拉框，PC 端 q-btn-toggle -->
          <q-select  v-model="searchParams.MovieType" :options="MovieTypeSelects" dense emit-value
            map-options borderless style="min-width: 120px" @update:model-value="fetchSearch">
          </q-select>
        </div>
        <div class="filter-row">
          <q-select  v-model="currentSort" :options="sortOptions" dense emit-value
            map-options borderless style="min-width: 120px" @update:model-value="fetchSearch">
          </q-select>
        </div>
        <div class="filter-row">
          <IndexButton flat @refresh-done="fetchSearch" color="red" toggle-color="indigo-6" glossy />
        </div>
      </div>
    </div>

    <!-- 搜索列表 -->
    <div class="search-results" v-show="activeTab === 'search'" ref="searchResultsRef">
      <div v-if="searchLoading" class="search-loading">
        <q-spinner-dots size="40px" color="indigo-4" />
        <p class="text-grey-5 q-mt-sm text-caption">加载中...</p>
      </div>

      <template v-else-if="searchResults.Data && searchResults.Data.length > 0">
        <div class="search-cards">
          <div v-for="item in searchResults.Data" :key="item.Id" class="search-card" :class="{
            'search-card-playing': currentId === item.Id
          }">
            <div class="search-card-thumb">
              <q-img :src="item.PngUrl" fit="cover" class="search-card-img" :ratio="3 / 4"
                @click="$emit('play', item)">
                <template v-slot:error>
                  <div class="search-card-placeholder">
                    <q-icon name="movie" color="grey-6" size="28px" />
                  </div>
                </template>
              </q-img>
              <div class="search-card-play-overlay">
                <q-icon name="play_circle_filled" size="28px" color="white" @click="$emit('play', item)" />
              </div>
              <!-- 播放中指示器 -->
              <div class="search-card-playing-indicator" v-if="currentId === item.Id && isPlaying">
                <q-icon name="play_arrow" size="20px" color="white" />
              </div>
            </div>

            <div class="search-card-info">
              <div class="search-card-title">
                {{ formatTitle(item.Title, 24) }}
              </div>
              <div class="search-card-tags">
                <span class="tag tag-author" v-if="item.Author" @click="$emit('keyword', item.Author)">{{
                  item.Author?.substring(0, 10) }}</span>
                <span class="tag tag-code" v-if="item.Code" @click="$emit('keyword', item.Code)">{{
                  item.Code.substring(0, 10) }}</span>
                <template v-if="item.Tags">
                  <span v-for="(value, index) in item.Tags" :key="index" class="tag tag-level"
                    @click="$emit('keyword', value)" :style="{ background: getTagColor(index) }">{{
                      value }}</span>
                </template>
              </div>
              <div class="search-card-meta">
                <span class="meta-item">
                  <q-icon name="data_usage" size="10px" />
                  {{ humanStorageSize(item.Size) }}
                </span>
                <span class="meta-item">
                  <q-icon name="schedule" size="10px" />
                  {{ getTimeAgo(item.MTime) }}
                </span>
                <q-btn-dropdown dense flat size="md" color="indigo-4"
                  :label="`${item.MovieType === '无' ? '分类' : item.MovieType}`" no-caps>
                  <q-list style="min-width: 60px">
                    <q-item v-for="mt in MovieTypeOptions" :key="mt.value" clickable v-close-popup>
                      <q-item-section @click="setMovieType(item, mt.value)">{{ mt.label }}</q-item-section>
                    </q-item>
                  </q-list>
                </q-btn-dropdown>
                <q-btn flat dense color="primary" icon="edit" size="md" label="修改"
                  @click.stop="onEdit(item)">
                  <q-tooltip>修改</q-tooltip>
                </q-btn>
                <q-btn flat dense color="negative" icon="delete" size="md" label="删除"
                  @click.stop="onDelete(item)">
                  <q-tooltip>删除</q-tooltip>
                </q-btn>
              </div>
            </div>
          </div>
        </div>
      </template>

      <div v-else-if="!searchLoading" class="search-empty">
        <q-icon name="search_off" size="48px" color="grey-7" />
        <p class="text-grey-6 q-mt-sm">暂无搜索结果</p>
      </div>

      <!-- 分页 -->
      <div class="search-pagination" v-if="searchResults.TotalPage > 0 && activeTab === 'search'">
        <q-pagination v-model="searchParams.Page" @update:model-value="fetchSearch" color="deep-orange"
          :ellipses="true" :max="searchResults.TotalPage || 0" :max-pages="isSmall ? 5 : 8" boundary-numbers
          direction-links></q-pagination>
        <span class="page-count">共 {{ searchResults.TotalCnt }} 条</span>
        <q-select size="xs" dense flat @update:model-value="currentPageSizeChange" filled bgColor="orange"
          style="text-align: center; width: 70px" v-model="searchParams.PageSize" :options="pageOptions">
        </q-select>
        <q-input v-model.number="gotoPage" :dense="true" style="text-align: center; width: 60px" bgColor="orange"
          :max="searchResults.TotalPage" :min="1" @change="pageNoGoto" />
      </div>
    </div>

    <!-- 本地图片 -->
    <div class="search-results" v-show="activeTab === 'images'">
      <div v-if="imagesLoading" class="search-loading">
        <q-spinner-dots size="40px" color="indigo-4" />
        <p class="text-grey-5 q-mt-sm text-caption">加载中...</p>
      </div>
      <template v-else-if="localImages && localImages.length > 0">
        <div class="local-images-grid">
          <div v-for="item in localImages" :key="item.Id" class="local-image-item">
            <q-img fit="fill" :src="GetFileByPathUseEncode(item.Path)" style="border-radius: 6px; overflow: hidden;">
              <template v-slot:error>
                <div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,1)">
                  <q-icon name="image_not_supported" size="2em" color="grey-6" />
                </div>
              </template>
              <div style="position:absolute;top:4px;right:4px">
                <q-btn dense color="rgba(0,0,0,0.5)" icon="ti-trash" size="xs"
                  @click.stop="deleteLocalImage(item.Path)">
                  <q-tooltip class="bg-white text-primary">删除</q-tooltip>
                </q-btn>
              </div>
            </q-img>
          </div>
        </div>
      </template>
      <div v-else class="search-empty">
        <q-icon name="image_not_supported" size="48px" color="grey-7" />
        <p class="text-grey-6 q-mt-sm">暂无本地图片</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue';
import { format } from 'quasar';
import { useQuasar } from 'quasar';
import { useBreakpoint } from 'src/composables/useBreakpoint';
import { SearchAPI, ResetMovieType } from 'components/api/searchAPI';
import { QueryDirImages, DeleteFileByPathUseEncode } from 'components/api/searchAPI';
import { GetFileByPathUseEncode } from 'components/utils/images';

import {
  MovieTypeSelects,
  MovieTypeOptions,
  FieldEnum,
  DescEnum,
  formatTitle,
} from 'components/utils';
import IndexButton from 'components/IndexButton.vue';

const props = defineProps({
  visible: { type: Boolean, default: false },
  currentId: { type: String, default: '' },
  currentTime: { type: String, default: '00:00:00' },
  isPlaying: { type: Boolean, default: false },
});

const emit = defineEmits(['play', 'close', 'keyword', 'edit', 'delete']);

// ── 状态 ───────────────────────────────────────────────────────────────
const $q = useQuasar();
const { isSmall } = useBreakpoint();

const activeTab = ref('search');
const searchLoading = ref(false);
const imagesLoading = ref(false);
const searchResults = reactive({ Data: [] , TotalPage: 0, TotalCnt: 0 });
const localImages = ref<[]>([]);
const gotoPage = ref(1);
const pageOptions = ref([10, 20, 40, 60]);

const searchParams = reactive({
  Keyword: '',
  MovieType: '',
  SortField: 'MTime',
  SortType: 'desc',
  Page: 1,
  PageSize: 20,
});

const sortOptions = computed(() => {
  const options = [];
  for (const field of FieldEnum) {
    for (const desc of DescEnum) {
      options.push({
        label: `${field.label}${desc.label}`,
        value: `${field.value}_${desc.value}`
      });
    }
  }
  return options;
});

const currentSort = computed({
  get: () => `${searchParams.SortField}_${searchParams.SortType}`,
  set: (val) => {
    const [field, type] = val.split('_');
    searchParams.SortField = field;
    searchParams.SortType = type;
  }
});

// ── 工具函数 ──────────────────────────────────────────────────────────
const { humanStorageSize } = format ;

function getTagColor(tag) {
  const colors = [
    'rgba(16, 185, 129, 0.25)', 'rgba(99, 102, 241, 0.25)',
    'rgba(245, 158, 11, 0.25)', 'rgba(239, 68, 68, 0.25)',
    'rgba(14, 165, 233, 0.25)', 'rgba(168, 85, 247, 0.25)',
    'rgba(236, 72, 153, 0.25)', 'rgba(34, 197, 94, 0.25)',
  ];
  return colors[tag % colors.length];
}

function getTimeAgo(MTime) {
  const diff = Date.now() - new Date(MTime).getTime();
  const days = Math.floor(diff / 86400000);
  if (days < 1) return '今日';
  if (days < 30) return `${days}天前`;
  if (days < 365) return `${Math.floor(days / 30)}月前`;
  return `${Math.floor(days / 365)}年前`;
}

// ── 搜索 ──────────────────────────────────────────────────────────────
async function fetchSearch() {
  searchLoading.value = true;
  try {
    const data = await SearchAPI({ ...searchParams });
    Object.assign(searchResults, data);
  } catch (e) {
    console.error('Search failed:', e);
  } finally {
    searchLoading.value = false;
  }
}

function currentPageSizeChange(val) {
  searchParams.PageSize = val;
  searchParams.Page = 1;
  fetchSearch();
}

function pageNoGoto() {
  const page = parseInt(gotoPage.value);
  if (!isNaN(page) && page >= 1 && page <= (searchResults.TotalPage || 1)) {
    searchParams.Page = page;
    fetchSearch();
  }
}

async function setMovieType(item, type) {
  try {
    await ResetMovieType( item.Id,  type );
    item.MovieType = type;
    // 后端已直接更新索引，无需额外刷新
  } catch (e) {
    console.error('ResetMovieType failed:', e);
  }
}

// ── 本地图片 ──────────────────────────────────────────────────────────
async function loadLocalImages() {
  if (!props.currentId) return;
  imagesLoading.value = true;
  try {
    const res = await QueryDirImages(props.currentId, 'desc');
    localImages.value = res?.data || [];
  } catch (e) {
    console.error('Load local images failed:', e);
  } finally {
    imagesLoading.value = false;
  }
}

async function deleteLocalImage(path) {
  try {
    await DeleteFileByPathUseEncode(path);
    localImages.value = localImages.value.filter(i => i.Path !== path);
  } catch (e) {
    console.error('Delete local image failed:', e);
  }
}

async function onDelete(item) {
  $q.dialog({
    title: '确认删除',
    message: `确定要删除「${formatTitle(item.Title, 20)}」吗？`,
    cancel: { flat: true, label: '取消', color: 'grey' },
    ok: { flat: true, label: '删除', color: 'negative' },
  }).onOk(async () => {
    emit('delete', item);
    searchResults.Data = searchResults.Data.filter(i => i.Id !== item.Id);
  });
}

function onEdit(item) {
  emit('edit', item);
}

// ── 刷新 ──────────────────────────────────────────────────────────────
function onRefresh() {
  if (activeTab.value === 'search') {
    fetchSearch();
  } else {
    loadLocalImages();
  }
}

// ── 监听 tab ──────────────────────────────────────────────────────────
watch(activeTab, (tab) => {
  if (tab === 'images') {
    loadLocalImages();
  }
});

// 初始加载搜索
fetchSearch();

// 暴露刷新方法
defineExpose({ fetchSearch });
</script>

<style scoped>
@media (max-width: 599px) {
  .search-card {
    width: calc(50% - 8px) !important;
    max-width: none !important;
  }
  .search-card-info {
    padding: 4px 6px !important;
  }
  .search-card-title {
    font-size: 12px !important;
  }
  .search-card-tags .tag {
    font-size: 10px !important;
    padding: 1px 4px !important;
  }
  .search-card-meta {
    gap: 4px !important;
  }
  .search-card-meta .meta-item {
    font-size: 10px !important;
  }
}

.sp-tabs {
  flex: 1;
  min-width: 0;
}

.sp-header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ── 搜索面板 ─────────────────────────────────────────────────────── */
.search-panel {
  position: relative;
  height: 88vh;
  margin: 20px auto;
  width: 88%;
  border-radius: 20px;
  border: #10b981 1px solid;
  background: rgba(9, 9, 22, 0.92);
  backdrop-filter: blur(32px); 
  -webkit-backdrop-filter: blur(32px);
  border-left: 1px solid rgba(99, 102, 241, 0.25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -10px 0 40px rgba(0, 0, 0, 0.4);
  z-index: 1000;
}

.search-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(99, 102, 241, 0.06);
}

.search-panel-title {
  font-size: 1rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  letter-spacing: 0.03em;
  width: 100%;
}

.search-conditions {
  padding: 10px 16px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.12);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  font-size: 11px;
  color: #818cf8;
  min-width: 28px;
  letter-spacing: 0.03em;
}

.search-results {
  flex: 1;
  overflow-y: auto;
  padding: 10px 12px;
  scrollbar-width: thin;
  scrollbar-color: rgba(99, 102, 241, 0.25) transparent;
}

.search-results::-webkit-scrollbar { width: 4px; }
.search-results::-webkit-scrollbar-thumb { background: rgba(99, 102, 241, 0.25); border-radius: 2px; }

.search-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 0;
}

.search-cards {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  justify-content: space-evenly;
}

.search-card {
  margin-bottom: 8px;
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 8px;
  border-radius: 12px;
  background: rgba(22, 22, 45, 0.75);
  border: 1px solid rgba(99, 102, 241, 0.12);
  cursor: pointer;
  position: relative;
  overflow: hidden;
  /* width: calc(100% - 20px); */
  min-width: 300px;
  max-width: 350px;
}

.search-card::after {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.07) 0%, transparent 60%);
  opacity: 0;
  transition: opacity 0.3s;
}

.search-card:hover::after { opacity: 1; }
.search-card:hover {
  border-color: rgba(139, 92, 246, 0.4);
  background: rgba(32, 32, 60, 0.65);
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.15);
  transform: translateX(3px);
}

.search-card-playing {
  border-color: rgba(139, 92, 246, 0.9) !important;
  background: rgba(45, 35, 80, 0.75) !important;
  box-shadow: 0 0 16px rgba(139, 92, 246, 0.45), 0 0 32px rgba(99, 102, 241, 0.2);
}

.search-card-playing-indicator {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(139, 92, 246, 0.55);
  animation: playing-pulse 1.5s ease-in-out infinite;
}

@keyframes playing-pulse {
  0%, 100% { background: rgba(139, 92, 246, 0.5); }
  50% { background: rgba(139, 92, 246, 0.3); }
}

.search-card-thumb {
  position: relative;
  flex-shrink: 0;
  width: 72px;
  height: 100px;
  border-radius: 8px;
  overflow: hidden;
}

.search-card-img { width: 100%; height: 100%; }

.search-card-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(25, 25, 48, 0.8);
}

.search-card-play-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  opacity: 0;
  transition: opacity 0.25s;
}

.search-card:hover .search-card-play-overlay { opacity: 1; }

.search-card-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow: hidden;
  min-width: 0;
}

.search-card-title {
  font-size: 12.5px;
  color: #dde5ff;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.search-card-tags { display: flex; flex-wrap: wrap; gap: 4px; }

.tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  line-height: 1.5;
  cursor: pointer;
}

.tag-author {
  color: #a78bfa;
  background: rgba(139, 92, 246, 0.14);
  border: 1px solid rgba(139, 92, 246, 0.25);
}

.tag-code {
  color: #f472b6;
  background: rgba(244, 114, 182, 0.12);
  border: 1px solid rgba(244, 114, 182, 0.22);
}

.tag-level { color: #fff; font-weight: 600; }

.search-card-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: auto;
  align-items: center;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 3px;
  color: rgba(165, 180, 252, 0.55);
}

.search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
}

.search-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 20px;
  border-top: 1px solid rgba(99, 102, 241, 0.15);
  background: rgba(8, 8, 20, 0.9);
  flex-wrap: wrap;
}

.page-count { color: rgba(255, 255, 255, 0.5); margin-left: 4px; }

/* ── 本地图片 ─────────────────────────────────────────────────────── */
.local-images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
}

.local-image-item {
  position: relative;
  border-radius: 6px;
  overflow: hidden;
}

/* ── 响应式 ─────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .search-panel { width: 100vw; max-width: 100vw; }
}
</style>
