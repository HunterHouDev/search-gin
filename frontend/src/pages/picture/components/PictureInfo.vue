<template>
  <q-dialog ref="dialogRef" @escape-key="onDialogClose" @hide="onDialogHide" maximized>
    <q-layout view="hHh Lpr lff" class="bg-dark">
      <q-header class="bg-primary text-white">
        <q-toolbar>
          <span class="text-title text-weight-bold">{{ view.authorName }} </span>
          <span class="q-ml-sm text-blue">({{ totalCount }})</span>
          <q-space />
          <div class="row items-center q-gutter-xs">
            <!-- 排序 -->
            <q-select v-model="sortValue" :options="sortOptions" dense borderless dark options-dark emit-value
              map-options class="sort-select" @update:model-value="pageNo = 1; fetchMovieList()" />
            <!-- 每页数量 -->
            <q-select v-model="pageSize" :options="[12, 24, 50, 100]" dense borderless dark options-dark
              style="min-width: 60px" @update:model-value="pageNo = 1; fetchMovieList()" />
            <!-- 分页 -->
            <q-pagination v-if="totalPages > 1" v-model="pageNo" :max="totalPages" size="sm" color="white" input
              boundary-links direction-links @update:model-value="fetchMovieList()" />
            <q-btn dense flat icon="close" @click="onDialogClose">
              <q-tooltip>关闭</q-tooltip>
            </q-btn>
          </div>
        </q-toolbar>
      </q-header>

      <q-page-container>
        <!-- 加载骨架屏 -->
        <div v-if="loading" class="row q-gutter-sm q-pa-sm justify-center">
          <q-card v-for="n in 12" :key="n" class="movie-card">
            <q-skeleton height="220px" square />
            <q-skeleton height="40px" square />
          </q-card>
        </div>

        <!-- 空状态 -->
        <div v-else-if="movieList.length === 0" class="column flex-center q-pa-xl">
          <q-icon name="video_library" size="80px" color="grey-6" />
          <span class="text-grey-6 q-mt-md">暂无影片</span>
        </div>

        <!-- 影片卡片网格 -->
        <div v-else class="movie-grid q-pa-sm">
          <div v-for="item in movieList" :key="item.Id" class="movie-card-wrapper" @mouseenter="hoverId = item.Id"
            @mouseleave="hoverId = null" @contextmenu.prevent="playMovie(item)">
            <q-card class="movie-card">
              <div class="card-img-wrap">
                <q-img fit="fill" :lazy="true" class="card-img" :src="item.JpgUrl" style="width: 100%; height: 232px">
                  <template v-slot:loading>
                    <q-spinner-ios color="white" size="2em" />
                  </template>
                  <template v-slot:error>
                    <div class="text-subtitle1 text-white column flex-center full-height">
                      <q-icon name="image_not_supported" size="2em" />
                    </div>
                  </template>
                </q-img>

                <!-- Hover 覆盖层 -->
                <div v-if="hoverId === item.Id" class="card-hover-overlay">
                  <!-- 上半部分：按钮 -->
                  <div class="hover-top">
                    <q-btn flat round icon="play_circle_filled" color="white" size="lg" @click.stop="playMovie(item)"
                      class="hover-btn">
                      <q-tooltip>播放</q-tooltip>
                    </q-btn>
                    <q-btn flat round icon="delete" color="negative" size="md" @click.stop="confirmDelete(item)"
                      class="hover-btn">
                      <q-tooltip>删除</q-tooltip>
                    </q-btn>
                    <q-btn flat round icon="open_in_new" color="white" size="md" @click.stop="jumpToList(item)"
                      class="hover-btn">
                      <q-tooltip>跳转列表</q-tooltip>
                    </q-btn>
                  </div>
                  <!-- 下半部分：描述 -->
                  <div class="hover-bottom">
                    <div class="hover-code">{{ item.Code }}</div>
                    <div class="hover-title">{{ formatTitle(item.Title || item.Name) }}</div>
                  </div>
                </div>

                <!-- 顶部：大小 → 时间 → Tags -->
                <div class="absolute-top-left q-ma-xs top-row">
                  <span v-if="item.SizeStr" class="chip-tag overlay-chip size-chip">{{ item.SizeStr }}</span>
                  <span v-if="item.MTime" class="chip-tag overlay-chip time-chip">{{ timeAgo(item.MTime) }}</span>
                  <span v-for="tag in (item.Tags || []).slice(0, 10)" :key="tag" class="chip-tag overlay-chip">#{{ tag
                    }}</span>
                  <span v-if="item.Tags && item.Tags.length > 10" class="chip-tag overlay-chip">+{{ item.Tags.length -
                    10 }}</span>
                </div>
              </div>

            </q-card>
          </div>
        </div>
      </q-page-container>

      <!-- 浮动画中画播放器 -->
      <VideoPlayerInPicture ref="videoPlayerRef" minimal @prev-one="prevOne" @next-one="nextOne" />
    </q-layout>
  </q-dialog>
</template>

<script setup>
import { useDialogPluginComponent, useQuasar } from 'quasar';
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useSystemProperty } from 'stores/System';
import { formatTitle } from 'components/utils';
import { SearchAPI, DeleteFileByPathUseEncode } from 'components/api/searchAPI';
import VideoPlayerInPicture from 'pages/file/components/VideoPlayerInPicture.vue';

const $q = useQuasar();
const systemProperty = useSystemProperty();
const { push } = useRouter();

defineEmits([...useDialogPluginComponent.emits]);
const { dialogRef, onDialogHide } = useDialogPluginComponent();

const view = reactive({
  authorName: '',
});

const movieList = ref([]);
const loading = ref(false);
const pageNo = ref(1);
const pageSize = ref(24);
const totalPages = ref(1);
const totalCount = ref(1);
const hoverId = ref(null);
const currentPlayingId = ref(null);

const prevOne = async () => {
  if (!currentPlayingId.value || movieList.value.length === 0) return;
  const idx = movieList.value.findIndex((i) => i.Id === currentPlayingId.value);
  if (idx > 0) {
    playMovie(movieList.value[idx - 1]);
    return;
  }
  // 翻到上一页最后一个
  if (pageNo.value <= 1) return;
  pageNo.value--;
  await fetchMovieList();
  const last = movieList.value[movieList.value.length - 1];
  if (last) playMovie(last);
};

const nextOne = async () => {
  if (!currentPlayingId.value || movieList.value.length === 0) return;
  const idx = movieList.value.findIndex((i) => i.Id === currentPlayingId.value);
  if (idx >= 0 && idx < movieList.value.length - 1) {
    playMovie(movieList.value[idx + 1]);
    return;
  }
  // 翻到下一页第一个
  if (pageNo.value >= totalPages.value) return;
  pageNo.value++;
  await fetchMovieList();
  const first = movieList.value[0];
  if (first) playMovie(first);
};
const videoPlayerRef = ref(null);
const sortValue = ref('MTime_desc');
const sortOptions = [
  { label: '时间 ↑', value: 'MTime_asc' },
  { label: '时间 ↓', value: 'MTime_desc' },
  { label: '名称 ↑', value: 'Code_asc' },
  { label: '名称 ↓', value: 'Code_desc' },
  { label: '大小 ↑', value: 'Size_asc' },
  { label: '大小 ↓', value: 'Size_desc' },
];

const timeAgo = (timeStr) => {
  if (!timeStr) return '';
  const t = new Date(timeStr).getTime();
  if (!t) return timeStr.substring(0, 10);
  const diff = Date.now() - t;
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return '刚刚';
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}分钟前`;
  const hour = Math.floor(min / 60);
  if (hour < 24) return `${hour}小时前`;
  const day = Math.floor(hour / 24);
  if (day < 30) return `${day}天前`;
  const month = Math.floor(day / 30);
  if (month < 12) return `${month}个月前`;
  return `${Math.floor(month / 12)}年前`;
};

const parseSort = (val) => {
  const parts = val.split('_');
  return { SortField: parts[0], SortType: parts[1] };
};

const fetchMovieList = async () => {
  if (!view.authorName) return;
  loading.value = true;
  try {
    const { SortField, SortType } = parseSort(sortValue.value);
    const params = {
      Keyword: view.authorName,
      Page: pageNo.value,
      PageSize: pageSize.value,
      SortField,
      SortType,
      MovieType: '',
    };
    const data = await SearchAPI(params);
    movieList.value = data.Data || [];
    totalPages.value = data.TotalPage || 1;
    totalCount.value = data.TotalCnt || 0;
  } catch (e) {
    console.error('获取影片列表失败', e);
    movieList.value = [];
  } finally {
    loading.value = false;
  }
};

const playMovie = (item) => {
  currentPlayingId.value = item.Id;
  if (videoPlayerRef.value) {
    videoPlayerRef.value.openVideo({ item, queryParam: {}, webFullScreen: false });
  }
};

const confirmDelete = (item) => {
  $q.dialog({
    title: '确认删除',
    message: `确定删除「${item.Name || item.Code}」？`,
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    try {
      await DeleteFileByPathUseEncode(item.Path);
      $q.notify({ type: 'positive', message: '删除成功' });
      fetchMovieList();
    } catch (e) {
      $q.notify({ type: 'negative', message: '删除失败' });
    }
  });
};

const jumpToList = (item) => {
  systemProperty.setPage(1);
  systemProperty.FileSearchParam.Keyword = item.Code || '';
  systemProperty.setMovieType('');
  push('/search?from=index');
  onDialogClose();
};

const onDialogClose = () => {
  dialogRef.value.hide();
  onDialogHide();
};

const open = (data) => {
  movieList.value = [];
  view.authorName = data.Name || '';
  pageNo.value = 1;
  setTimeout(() => {
    dialogRef.value.show();
    fetchMovieList();
  }, 0);
};

defineExpose({ open });
</script>

<style lang="scss" scoped>
.sort-select {
  min-width: 90px;
}

.movie-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 6px;
  justify-items: center;
}

.movie-card-wrapper {
  position: relative;
  width: 100%;
  max-width: 260px;
}

.movie-card {
  width: 100%;
  background: var(--q-bg-card);
  color: var(--q-text-primary);
  border: 1px solid var(--q-border);
  border-radius: 4px;
  overflow: hidden;
  transition: transform 0.2s ease;

  &:hover {
    transform: translateY(-2px);
  }
}

.card-img-wrap {
  position: relative;
  overflow: hidden;
  width: 100%;
}

.card-img {
  display: block;
  transition: filter 0.3s ease;
}

.card-hover-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  z-index: 2;
}

.hover-top {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.hover-bottom {
  padding: 6px 8px;
  background: rgba(0, 0, 0, 0.5);
}

.hover-code {
  font-size: 12px;
  font-weight: 600;
  color: #f1f5f9;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hover-title {
  font-size: 11px;
  color: #cbd5e1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hover-btn {
  transition: transform 0.15s ease;

  &:hover {
    transform: scale(1.2);
  }
}

.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-info {
  min-height: 72px;
}

.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-clamp: 2;
}

.chip-tag {
  padding: 0 4px;
  font-size: 10px;
  font-weight: 500;
  background: rgba(0, 0, 0, 0.15);
  border-radius: 3px;
  white-space: nowrap;
}

.top-row {
  display: flex;
  align-items: center;
  gap: 3px;
  flex-wrap: wrap;
  max-width: calc(100% - 8px);
}

.overlay-chip {
  background: rgba(0, 0, 0, 0.55);
  color: #e2e8f0;
  font-size: 9px;
  padding: 1px 5px;
  border-radius: 3px;
  white-space: nowrap;
}

.size-chip {
  background: rgba(239, 68, 68, 0.7) !important;
}

.time-chip {
  background: rgba(0, 0, 0, 0.55);
}



.tag-top-left {
  background: rgba(0, 0, 0, 0.55) !important;
  color: #e2e8f0;
  font-size: 9px;
  padding: 1px 5px;
  white-space: nowrap;
}



@media (max-width: 1400px) {
  .movie-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (max-width: 1100px) {
  .movie-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 800px) {
  .movie-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
