<template>
  <div class="">
    <q-layout view="lHh lpr lFf" container style="height: 93vh" class="shadow-2 rounded-borders" :style="themeStyle">
      <!-- 头部 -->
      <q-header :style="themeStyle" elevated class="q-gutter-sm flex justify-center" style="
          backdrop-filter: blur(5px);
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        ">
        <div class="row justify-center q-gutter-sm w100 q-pa-xs">
          <q-btn-toggle glossy v-model="view.queryParam.SortField" @update:model-value="fetchSearch"
            toggle-color="primary" :options="[
              { label: '容', value: 'Size' },
              { label: '数', value: 'Cnt' },
            ]" />
          <q-input glossy v-model="view.queryParam.Keyword" :dense="true" placeholder="搜索" filled clearable
            @update:model-value="fetchSearch" />
          <q-select dense flat @update:model-value="
            (e) => {
              view.settingShow = false;
              currentPageSizeChange(e);
            }
          " filled bgColor="orange" style="text-align: center; " v-model="view.queryParam.PageSize"
            :options="[10, 12, 20, 30, 50, 200]">
          </q-select>
          <q-input v-model="view.queryParam.Page" filled :dense="true" type="search" style="text-align: center; width: 3rem"
            bgColor="orange" :max="view.resultData.TotalPage" :min="1" @focus="focusEvent($event)" @update:model-value="
              (e) => {
                view.settingShow = false;
                gotoPageNo(e);
              }
            " />
        </div>
      </q-header>
      <q-page-container class="scroll">
        <!-- 加载骨架屏 -->
        <div v-if="isLoading" style="display: flex; flex-direction: row; flex-wrap: wrap">
          <q-card v-for="n in 12" :key="n" class="q-ma-sm example-item">
            <q-skeleton height="232px" animation="wave" />
            <q-card-section>
              <q-skeleton type="text" width="60%" />
            </q-card-section>
          </q-card>
        </div>
        <!-- 卡片列表 -->
        <div v-else style="display: flex; flex-direction: row; flex-wrap: wrap" id="scrollTargetElement">
          <q-card class="q-ma-sm example-item" v-for="item in view.resultData.Data" :key="item.Id">
            <q-img fit="fill" :src="getAuthorImage(item.Name)" class="item-img" @click="fileEditRef.open(item)">
              <div style="
                  padding: 0;
                  margin: 0;
                  background-color: rgba(0, 0, 0, 0);
                  display: flex;
                  flex-direction: row;
                  justify-content: space-between;
                  width: 100%;
                ">
                <div @click.stop="() => undefined" style="
                    display: flex;
                    flex-direction: column;
                    justify-content: flex-start;
                    width: fit-content;
                  ">
                  <q-chip square color="red" text-color="white" style="margin-left: 0; padding: 0 4px">
                    <span>{{ item.SizeStr }}</span>
                  </q-chip>
                </div>
                <q-chip @click.stop="() => undefined" square color="green" text-color="white"
                  style="width: fit-content; margin-right: 0; padding: 0 6px">
                  <span> {{ item.Cnt }}</span>
                </q-chip>
              </div>

              <template v-slot:error>
                <div class="absolute-full flex flex-center bg-gray text-white">
                  Cannot load image
                </div>
              </template>
            </q-img>
            <div class="absolute-bottom text-body1 text-center"
              style="padding: 4px; background-color: rgba(0, 0, 0, 0.5)" @click.stop="() => undefined">
              <q-btn flat style="color: #59d89d; width: 100%" :label="item.Name?.substring(0, 10) || '未知'"
                @click="searchFiles(item.Name)" />
            </div>
          </q-card>
        </div>
      </q-page-container>
    </q-layout>
    <!-- 上一页按钮 -->
    <q-page-sticky style="z-index: 9" position="bottom-left" v-if="view.queryParam.Page > 1"
      :offset="[6, isMobile ? 200 : 300]">
      <q-btn round glossy class="page-sticky" flat text-color="blue" :label="`P${view.queryParam.Page - 1}`"
        @click="nextPage(-1)"></q-btn>
    </q-page-sticky>

    <!-- 下一页按钮 -->
    <q-page-sticky style="z-index: 9" position="bottom-right"
      :offset="[10, isMobile ? 200 : 300]"><!-- icon="keyboard_arrow_right" -->
      <q-btn round dense flat glossy class="page-sticky" text-color="blue" :label="`P${view.queryParam.Page + 1}`"
        @click="nextPage(1)"></q-btn>
    </q-page-sticky>

    <PictureInfo ref="fileEditRef" />
    <q-page-scroller position="bottom-right" :scroll-offset="150" :offset="[18, 100]">
      <q-btn fab icon="keyboard_arrow_up" color="accent" />
    </q-page-scroller>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, computed } from 'vue';
import { getAuthorImage } from '../../components/utils/images';
import { QueryAuthorList } from '../../components/api/authorAPI';
import { useSystemProperty } from '../../stores/System';
import { useRouter } from 'vue-router';
import PictureInfo from './components/PictureInfo.vue';
import { useBreakpoint } from 'src/composables/useBreakpoint';

const scrollTop = () => {
  const target = document.getElementsByClassName('scroll');
  if (target && target[2]) {
    target[2].scrollTo(0, 0);
  }
};

const isLoading = ref(false);

const { push } = useRouter();
const { isMobile } = useBreakpoint();
const fileEditRef = ref(null);

const systemProperty = useSystemProperty();

const view = reactive({
  currentData: {},
  queryParam: {
    Keyword: '',
    MovieType: '',
    OnlyRepeat: false,
    Page: 1,
    PageSize: 30,
    SortField: 'Cnt',
    SortType: 'desc',
  },
  resultData: { Data: [] },
});

const focusEvent = (e) => {
  e.target.select();
};

const searchFiles = (name) => {
  systemProperty.FileSearchParam.Keyword = name;
  push({ path: '/search', query: { Keyword: name, from: 'index' } });
};

const currentPageChange = (e) => {
  fetchSearch();
};

const currentPageSizeChange = async (size) => {
  if (size) {
    view.queryParam.PageSize = Number(size);
  }
  await fetchSearch();
};

const gotoPageNo = async (e) => {
  if (e) {
    view.queryParam.Page = Number(e);
  }
  await fetchSearch();
};

const nextPage = (n) => {
  view.queryParam.Page = view.queryParam.Page + n;
  currentPageChange();
};

const fetchSearch = async () => {
  scrollTop();
  isLoading.value = true;
  const { data } = await QueryAuthorList(view.queryParam);
  view.resultData = { ...data, Data: data.Data || [] };
  isLoading.value = false;
};

const themeStyle = computed(() => systemProperty.themeStyle);

onMounted(() => {
  document.title = '图鉴';
  fetchSearch();
});
</script>
<style lang="scss" scoped>
.example-item {
  width: fit-content;
  height: fit-content;
}

.item-img {
  width: 324px;
  height: 232px;
}

.page-sticky {
  width: 3rem;
  height: 2.8rem;
  background-color: rgba(0, 0, 0, 0.6);
}
</style>
