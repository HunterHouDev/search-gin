<template>
  <q-popup-proxy>
    <div
      style="padding: 0; border-radius: 10px; max-height: 72vh"
      :style="{ width: isMobile ? '90vw' : '700px' }"
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
            name="actress"
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
          style="
            padding: 12px;
            margin: 0;
            max-height: 60vh;
            width: 100%;
            display: flex;
            flex-direction: row;
            flex-wrap: wrap;
            justify-content: flex-start;
          "
        >
          <div v-for="tag in view.tagData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              class="btn-fixed-width"
              size="md"
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
        <q-tab-panel name="actress" class="w100" style="max-height: 60vh">
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
              @update:model-value="fetchActress"
              toggle-color="primary"
              :options="[
                { label: '容', value: 'Size' },
                { label: '数', value: 'Cnt' },
              ]"
            />
            <div
              v-for="item in view.resultData.Data"
              :key="item.Id"
              style="width: auto"
            >
              <q-btn
                color="primary"
                class="btn-fixed-width"
                size="md"
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
          style="
            padding: 12px;
            margin: 0;
            max-height: 60vh;
            width: 100%;
            display: flex;
            flex-direction: row;
            flex-wrap: wrap;
            justify-content: flex-start;
          "
        >
          <div v-for="tag in view.seriesData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              class="btn-fixed-width"
              size="md"
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
        <q-tab-panel name="actress" class="w100" style="max-height: 60vh">
          <div class="row w100">
            <div class="col row justify-evenly w100">
              <q-radio
                v-model="view.sortField"
                checked-icon="task_alt"
                unchecked-icon="panorama_fish_eye"
                :val="'Size'"
                label="空间排序"
                style="color: red"
                @click="fetchActress"
                @update:model-value="fetchActress"
              />
              <q-radio
                v-model="view.sortField"
                checked-icon="task_alt"
                unchecked-icon="panorama_fish_eye"
                :val="'Cnt'"
                style="color: red"
                label="数量排序"
                @click="fetchActress"
                @update:model-value="fetchActress"
              />
            </div>
          </div>
          <div
            class="q-gutter-sm w100"
            style="
              display: flex;
              flex-direction: row;
              flex-wrap: wrap;
              justify-content: flex-start;
            "
          >
            <div
              v-for="item in view.resultData.Data"
              :key="item.Id"
              style="width: auto"
            >
              <q-btn
                color="primary"
                class="btn-fixed-width"
                size="md"
                flat
                v-if="item.Cnt > 1"
                @click="searchKeyword(item.Name)"
              >
                {{ `${item.Name} (${item.Cnt})` }}
                <q-badge color="red" floating>{{ item.SizeStr }}</q-badge>
              </q-btn>
            </div>
          </div>
        </q-tab-panel>
        <q-tab-panel name="folder" style="padding: 8px; max-height: 60vh">
          <q-table
            class="w100"
            id="scanTime"
            :rows="view.scanTime"
            :columns="scanTimeColumns"
            row-key="name"
            hide-bottom
            :pagination="{
              sortBy: 'desc',
              descending: false,
              page: 1,
              rowsPerPage: 99,
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
import { useQuasar } from 'quasar';
import { ScanTime, TagSizeMap, SeriesCount } from 'components/api/homeAPI';
import { QueryActressList } from 'components/api/actressAPI';
import { computed, onMounted, reactive, ref, inject } from 'vue';
import { useSystemProperty } from 'stores/System';
import IndexButton from 'components/IndexButton.vue';
const $q = useQuasar();

const isMobile = computed(() => {
  return $q?.platform.is.mobile;
});

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
  fetchActress();
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

const fetchActress = async () => {
  const { data } = await QueryActressList({
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
  console.log('加载数据 refreshView');
  refreshView();
});

const scanTimeColumns = [
  {
    name: 'Name',
    align: 'left',
    label: '文件夹',
    field: 'Name',
    style: { width: '350px', height: 'auto', 'text-wrap': 'balance' },
    sortable: true,
  },
  {
    name: 'Cnt',
    label: '时间(ms)',
    field: 'Cnt',
    align: 'right',
    style: { maxWidth: '50px' },
    sortable: true,
  },
  {
    name: 'Size',
    label: '文件数',
    field: 'Size',
    align: 'right',
    sortable: true,
  },
  {
    name: 'Size',
    label: '大小',
    field: 'Size',
    align: 'right',
    sortable: true,
    format: (val, row) => {
      return row.SizeStr;
    },
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
</style>
