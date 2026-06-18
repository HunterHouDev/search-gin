<template>
  <q-popup-proxy @show="refreshView">
    <div
      style="padding: 0; border-radius: 10px; max-height: 72vh; overflow: hidden"
      :style="{ width: isMobile ? '95vw' : '700px' }"
    >
      <div class="row justify-between w100">
        <IndexButton
          ref="indexButton"
          @click="refreshView"
          @refresh-done="refreshView"
          :style="{ width: '50%' }"
        />
        <q-btn
          color="primary"
          style="width: 50%"
          @click="
            () => {
              tab = 'folder';
              refreshView();
            }
          "
          label="更新数据"
        ></q-btn>
      </div>
      <q-toolbar class="bg-black text-white shadow-2 justify-between">
        <q-tabs ripple v-model="tab" color="red" class="w100">
          <q-tab
            name="tag"
            :label="isMobile ? '标签' : '标签分析'"
            class="w100"
          />
          <q-tab
            name="series"
            :label="isMobile ? '系列' : '系列分析'"
            class="w100"
          />
          <q-tab
            name="author"
            :label="isMobile ? '图鉴' : '图鉴分析'"
            class="w100"
          />
          <q-tab
            name="folder"
            :label="isMobile ? '磁盘' : '磁盘分析'"
            class="w100"
          />
        </q-tabs>
      </q-toolbar>

      <q-tab-panels v-model="tab" class="tab-ground" animated>
        <q-tab-panel
          name="tag"
          class="w100"
          :style="tagPanelStyle"
        >
          <div v-for="tag in view.tagData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              :class="isMobile ? 'btn-touch-mobile' : 'btn-fixed-width'"
              :size="isMobile ? 'sm' : 'md'"
              flat
              @click="searchKeyword(tag.Name)"
            >
              {{ `${tag.Name} (${tag.Cnt})` }}
              <q-badge color="red" floating>{{
                humanStorageSize(tag.Size)
              }}</q-badge>
            </q-btn>
          </div>
        </q-tab-panel>
        <q-tab-panel name="author" class="w100" style="max-height: 60vh">
          <div
            class="q-gutter-sm w100"
            style="
              display: flex;
              flex-direction: row;
              flex-wrap: wrap;
              justify-content: flex-start;
            "
          >
            <q-btn-toggle
              glossy
              flat
              v-model="view.sortField"
              @update:model-value="fetchAuthor"
              toggle-color="primary"
              :options="[
                { label: '容', value: 'Size' },
                { label: '数', value: 'Cnt' },
              ]"
              :size="isMobile ? 'sm' : 'md'"
            />
            <div
              v-for="item in view.resultData.Data"
              :key="item.Id"
              style="width: auto"
            >
              <q-btn
                color="primary"
                :class="isMobile ? 'btn-touch-mobile' : 'btn-fixed-width'"
                :size="isMobile ? 'sm' : 'md'"
                flat
                @click="searchKeyword(item.Name)"
              >
                {{ `${item.Name} (${item.Cnt})` }}
                <q-badge color="red" floating>{{ item.SizeStr }}</q-badge>
              </q-btn>
            </div>
          </div> </q-tab-panel
        ><q-tab-panel
          name="series"
          class="w100"
          :style="tagPanelStyle"
        >
          <div v-for="tag in view.seriesData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              :class="isMobile ? 'btn-touch-mobile' : 'btn-fixed-width'"
              :size="isMobile ? 'sm' : 'md'"
              flat
              v-if="tag.Cnt > 1"
              @click="searchKeyword(tag.Name)"
            >
              {{ `${tag.Name} (${tag.Cnt})` }}
              <q-badge color="red" floating>{{
                humanStorageSize(tag.Size)
              }}</q-badge>
            </q-btn>
          </div>
        </q-tab-panel>
        <q-tab-panel name="folder" style="padding: 8px; max-height: 60vh">
          <q-table
            class="w100 compact-table"
            id="scanTime"
            :rows="view.scanTime"
            :columns="scanTimeColumns"
            row-key="name"
            hide-bottom
            dense
            :rows-per-page-options="[0]"
            :pagination="{
              sortBy: 'desc',
              descending: false,
              page: 1,
              rowsPerPage: 0,
            }"
          >
            <template v-slot:body-cell-Name="props">
              <q-td :props="props">
                <div>
                  <q-btn
                    flat
                    color="primary"
                    :label="props.value"
                    @click="searchKeyword(props.value)"
                  ></q-btn>
                </div>
              </q-td>
            </template>
          </q-table>
        </q-tab-panel>
      </q-tab-panels>
    </div>
  </q-popup-proxy>
</template>

<script setup>
import { format } from 'quasar';
import { ScanTime, TagSizeMap, SeriesCount } from 'components/api/homeAPI';
import { QueryAuthorList } from 'components/api/authorAPI';
import { computed, onMounted, reactive, ref, inject } from 'vue';
import { useSystemProperty } from 'stores/System';
import IndexButton from 'components/IndexButton.vue';
import { useBreakpoint } from 'src/composables/useBreakpoint';
const { isMobile } = useBreakpoint();

const tagPanelStyle = computed(() => ({
  padding: isMobile.value ? '6px 4px' : '12px',
  margin: 0,
  maxHeight: '60vh',
  width: '100%',
  display: 'flex',
  flexDirection: 'row',
  flexWrap: 'wrap',
  justifyContent: 'flex-start',
  gap: isMobile.value ? '4px' : '8px',
}));

const { humanStorageSize } = format;
const systemProperty = useSystemProperty();

const indexButton = ref(null);
const tab = ref('tag');
const view = reactive({
  tagData: [],
  seriesData: [],
  scanTime: [],
  resultData: {},
  sortField: 'Cnt',
});

const searchKeyword = inject('searchKeyword', () => {
  console.log('refreshDebounceFn not found');
});

const refreshView = () => {
  loadScanTime();
  loadTagSize();
  fetchAuthor();
  loadSeriesCount();
};

const loadTagSize = async () => {
  const res = await TagSizeMap();
  if (res) {
    view.tagData = res;
    systemProperty.tagSizeMap = view.tagData;
  }
};
const loadSeriesCount = async () => {
  const res = await SeriesCount();
  if (res) {
    view.seriesData = res;
  }
};

const fetchAuthor = async () => {
  const { data } = await QueryAuthorList({
    Page: 1,
    PageSize: 400,
    SortField: view.sortField,
  });
  view.resultData = data;
};

const loadScanTime = async () => {
  let dataList = await ScanTime();
  if (dataList) {
    dataList = dataList.sort((a, b) => {
      return b.Cnt - a.Cnt;
    });
    systemProperty.SettingInfo.Dirs.forEach((item) => {
      const find = dataList.find((i) => i.Name === item);
      if (!find) {
        dataList.unshift({
          Name: item,
          Cnt: 0,
          Size: 0,
          SizeStr: '执行中',
        });
      }
    });
    view.scanTime = dataList;
  }
};

onMounted(() => {
  // 延迟加载数据，避免页面初始加载时不必要的请求
  // 数据将在用户打开 popup 时通过 @show 事件加载
});

const scanTimeColumns = [
  {
    name: 'Name',
    align: 'left',
    label: '文件夹',
    field: 'Name',
    style: { 
      width: isMobile.value ? '100px' : '180px'
    },
    sortable: true,
    headerStyle: 'white-space: nowrap;',
  },
  {
    name: 'Cnt',
    label: '耗时(ms)',
    field: 'Cnt',
    align: 'right',
    style: { width: isMobile.value ? '50px' : '70px' },
    sortable: true,
  },
  {
    name: 'FileCount',
    label: '文件数',
    field: 'Size',
    align: 'right',
    style: { width: isMobile.value ? '50px' : '70px' },
    sortable: true,
  },
  {
    name: 'SizeStr',
    label: '大小',
    field: 'SizeStr',
    align: 'right',
    style: { width: '90px' },
    sortable: true,
  },
];
</script>
<style>
.w100 {
  width: 100%;
}
.tab-ground {
  padding: 0;
  overflow: auto;
  background: rgba(250, 250, 250, 0.8);
  width: 100%;
  height: 60vh;
}

/* 磁盘表格：紧凑模式 */
.compact-table .q-table__container {
  box-shadow: none !important;
}
.compact-table .q-table__top,
.compact-table thead tr th {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 4px;
  white-space: nowrap;
}
.compact-table tbody tr td {
  padding: 0 4px;
  line-height: 1.3;
  font-size: 12px;
}
.compact-table tbody tr {
  height: 26px;
}

/* PC：固定宽度按钮 */
.btn-fixed-width {
  min-width: 120px;
  max-width: 200px;
  margin: 2px;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 移动端：大触摸区域 + 自适应宽度 */
.btn-touch-mobile {
  min-width: 80px;
  min-height: 36px;
  padding: 4px 8px;
  margin: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 移动端 tab 栏紧凑 */
@media (max-width: 600px) {
  .tab-ground {
    height: 55vh;
  }
}
</style>
