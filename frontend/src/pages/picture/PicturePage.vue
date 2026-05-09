<template>
  <div class="">
    <q-layout
      view="lHh lpr lFf"
      container
      style="height: 93vh"
      class="shadow-2 rounded-borders"
      :style="themeStyle"
    >
      <!-- 头部 -->
      <q-header
        :style="themeStyle"
        elevated
        class="q-gutter-sm flex justify-center"
        style="
          backdrop-filter: blur(5px);
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        "
      >
        <div class="row justify-center q-gutter-sm">
          <q-btn-toggle
            glossy
            v-model="view.queryParam.SortField"
            @update:model-value="fetchSearch"
            toggle-color="primary"
            :options="[
              { label: '容', value: 'Size' },
              { label: '数', value: 'Cnt' },
            ]"
          />
          <q-input
            glossy
            v-model="view.queryParam.Keyword"
            :dense="true"
            placeholder="搜索"
            filled
            clearable
            @update:model-value="fetchSearch"
          />
          <q-btn icon="settings" color="orange" flat glossy>
            <q-popup-proxy
              style="background: rgba(250, 250, 250, 0.8)"
              v-model:model-value="view.settingShow"
            >
              <div
                class="q-gutter-md"
                style="
                  width: 12rem;
                  height: 8rem;
                  display: flex;
                  flex-direction: column;
                  justify-content: space-evenly;
                "
              >
                <div class="row justify-between">
                  <q-btn flat dense> 每页大小 </q-btn>
                  <q-select
                    size="sm"
                    dense
                    flat
                    @update:model-value="
                      (e) => {
                        view.settingShow = false;
                        currentPageSizeChange(e);
                      }
                    "
                    filled
                    bgColor="orange"
                    style="text-align: center; width: 40%"
                    v-model="view.queryParam.PageSize"
                    :options="[10, 12, 20, 30, 50, 200]"
                  >
                  </q-select>
                </div>
                <div class="row justify-between">
                  <q-btn flat dense>页码 </q-btn>
                  <q-input
                    v-model="view.queryParam.Page"
                    :dense="true"
                    type="search"
                    style="text-align: center; width: 40%"
                    bgColor="orange"
                    :max="view.resultData.TotalPage"
                    :min="1"
                    @focus="focusEvent($event)"
                    @update:model-value="
                      (e) => {
                        view.settingShow = false;
                        gotoPageNo(e);
                      }
                    "
                  />
                </div>
              </div>
              <!-- 每页数量选择 -->
            </q-popup-proxy>
          </q-btn>

          <q-pagination
            v-model="view.queryParam.Page"
            @update:model-value="currentPageChange"
            color="purple"
            :ellipses="false"
            :max="view.resultData.TotalCnt / view.queryParam.PageSize + 1 || 0"
            :max-pages="6"
            boundary-numbers
          />
        </div>
      </q-header>
      <q-page-container class="scroll">
        <div
          style="display: flex; flex-direction: row; flex-wrap: wrap"
          id="scrollTargetElement"
        >
          <q-card
            class="q-ma-sm example-item"
            v-for="item in view.resultData.Data"
            :key="item.Id"
          >
            <q-img
              fit="fill"
              :src="getActressImage(item.Name)"
              class="item-img"
              @click="fileEditRef.open(item)"
            >
              <div
                style="
                  padding: 0;
                  margin: 0;
                  background-color: rgba(0, 0, 0, 0);
                  display: flex;
                  flex-direction: row;
                  justify-content: space-between;
                  width: 100%;
                "
              >
                <div
                  @click.stop="() => {}"
                  style="
                    display: flex;
                    flex-direction: column;
                    justify-content: flex-start;
                    width: fit-content;
                  "
                >
                  <q-chip
                    square
                    color="red"
                    text-color="white"
                    style="margin-left: 0; padding: 0 4px"
                  >
                    <span>{{ item.SizeStr }}</span>
                  </q-chip>
                </div>
                <q-chip
                  @click.stop="() => {}"
                  square
                  color="green"
                  text-color="white"
                  style="width: fit-content; margin-right: 0; padding: 0 6px"
                >
                  <span> {{ item.Cnt }}</span>
                </q-chip>
              </div>

              <template v-slot:error>
                <div class="absolute-full flex flex-center bg-gray text-white">
                  Cannot load image
                </div>
              </template>
            </q-img>
            <div
              class="absolute-bottom text-body1 text-center"
              style="padding: 4px; background-color: rgba(0, 0, 0, 0.5)"
              @click.stop="() => {}"
            >
              <q-btn
                flat
                style="color: #59d89d; width: 100%"
                :label="item.Name?.substring(0, 10) || '未知'"
                @click="searchFiles(item.Name)"
              />
            </div>
          </q-card>
        </div>
      </q-page-container>
    </q-layout>
    <!-- 上一页按钮 -->
    <q-page-sticky
      style="z-index: 9"
      position="bottom-left"
      v-if="view.queryParam.Page > 1"
      :offset="[6, isMobile ? 200 : 300]"
    >
      <q-btn
        round
        glossy
        class="page-sticky"
        flat
        text-color="blue"
        :label="`P${view.queryParam.Page - 1}`"
        @click="nextPage(-1)"
      ></q-btn>
    </q-page-sticky>

    <!-- 下一页按钮 -->
    <q-page-sticky
      style="z-index: 9"
      position="bottom-right"
      :offset="[10, isMobile ? 200 : 300]"
      ><!-- icon="keyboard_arrow_right" -->
      <q-btn
        round
        dense
        flat
        glossy
        class="page-sticky"
        text-color="blue"
        :label="`P${view.queryParam.Page + 1}`"
        @click="nextPage(1)"
      ></q-btn>
    </q-page-sticky>

    <PictureInfo ref="fileEditRef" />
    <q-page-scroller
      position="bottom-right"
      :scroll-offset="150"
      :offset="[18, 100]"
    >
      <q-btn fab icon="keyboard_arrow_up" color="accent" />
    </q-page-scroller>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, computed } from 'vue';
import { getActressImage } from '../../components/utils/images';
import { QueryActressList } from '../../components/api/actressAPI';
import { useSystemProperty } from '../../stores/System';
import { useRouter } from 'vue-router';
import PictureInfo from './components/PictureInfo.vue';

const scrollTop = () => {
  const target = document.getElementsByClassName('scroll');
  if (target && target[2]) {
    target[2].scrollTo(0, 0);
  }
};

const { push } = useRouter();
const fileEditRef = ref(null);

const isMobile = computed(() => {
  return window.innerWidth < 768;
});

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
  resultData: {},
});

const focusEvent = (e) => {
  e.target.select();
};

const searchFiles = (name) => {
  systemProperty.FileSearchParam.Keyword = name;
  console.log(name);
  push({ path: '/search', query: { Keyword: name, from: 'index' } });
};

const currentPageChange = (e) => {
  console.log(e);
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
  const { data } = await QueryActressList(view.queryParam);
  view.resultData = data;
};

const themeStyle = computed(() => {
  return {
    color: '#e0e7ff',
    backgroundColor: 'rgba(9, 9, 18, 0.95)',
  };
});

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
