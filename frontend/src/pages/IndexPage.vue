<template>
  <q-layout
    view="lHh lpr lFf"
    container
    style="height: 93vh"
  >
    <!-- 头部 -->
    <q-header elevated class="bg-primary">
      <div class="row justify-between w100" style="padding: 6px 12px">
        <div class="row justify-start q-gutter-sm">
          <IndexButton
            glossy
            color="primary"
            ref="indexButton"
            @refresh-done="loadTypeSize"
          />
          <q-btn color="primary" label="刷新" width="10" glossy @click="f5" />
        </div>
        <q-btn-toggle
          v-model="currentDiv"
          color="primary"
          outlined
          glossy
          text-color="white"
          @update:model-value="toDiv"
          :options="[
            { value: 'tagDiv', label: '标签' },
            { value: 'seriesDiv', label: '系列' },
            { value: 'typeDiv', label: '类型' },
            { value: 'diskDiv', label: '磁盘' },
          ]"
        />
      </div>
    </q-header>
    <q-page-container class="q-gutter-sm">
      <q-card class="cardcard">
        <q-toolbar class="bg-primary text-white" id="tagDiv">标签分析</q-toolbar>
        <div
          class="q-gutter-sm q-pa-sm"
          style="
            display: flex;
            flex-direction: row;
            flex-wrap: wrap;
            justify-content: flex-start;
            border-radius: 10px;
            overflow: auto;
          "
        >
          <div v-for="tag in tagData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              class="btn-fixed-width p0"
              glossy
              v-if="tag.Cnt > 1"
              @click="folderGotoMenu(tag.Name)"
            >
              {{ `${tag.Name} (${tag.Cnt})` }}
              <q-badge
                size="sm"
                color="orange"
                floating
                style="font-size: 0.5rem"
                >{{ tag.SizeStr }}</q-badge
              >
            </q-btn>
          </div>
        </div>
      </q-card>

      <q-card class="cardcard">
        <q-toolbar class="bg-primary text-white" id="seriesDiv"
          >系列分析</q-toolbar
        >
        <div
          class="q-gutter-sm q-pa-sm"
          style="
            display: flex;
            flex-direction: row;
            flex-wrap: wrap;
            justify-content: flex-start;
            border-radius: 10px;
            overflow: auto;
          "
        >
          <div v-for="tag in seriesData" :key="tag" style="width: auto">
            <q-btn
              color="primary"
              class="btn-fixed-width p0"
              glossy
              v-if="tag.Cnt > 1"
              @click="folderGotoMenu(tag.Name)"
            >
              {{ `${tag.Name} (${tag.Cnt})` }}
              <q-badge
                size="sm"
                color="orange"
                floating
                style="font-size: 0.5rem"
                >{{ tag.SizeStr }}</q-badge
              >
            </q-btn>
          </div>
        </div>
      </q-card>

      <q-card class="cardcard">
        <q-toolbar class="bg-primary text-white" id="typeDiv">类型分析</q-toolbar>
        <div
          class="row q-gutter-sm q-pa-sm justfity-start shadow-2 rounded-borders"
        >
          <q-card
            class="p0"
            v-for="item in tableData"
            :key="item"
            style="height: fit-content"
          >
            <q-badge color="negative" floating>{{ item.Cnt }}</q-badge>
            <q-card-section class="justify-between m0 p0">
              <q-btn
                dense
                icon="folder"
                color="primary"
                flat
                @click="gotoMenu(item)"
              >
                {{ !item.IsDir ? item.Name : '文件夹' }}
              </q-btn>
              <q-separator inset />
              <div class="text_subtitle" style="text-align: right">
                <span> {{ item.SizeStr + ' | ' }}</span>
                <span style="color: var(--q-success)"> {{ item.Cnt }}</span>

                <div v-if="item.IsDir">{{ item.Name }}</div>
              </div>
            </q-card-section>

            <q-card-actions>
              <q-btn
                color="primary"
                flat
                glossy
                dense
                v-if="item.IsDir"
                @click="openThis(item)"
                >打开
              </q-btn>
              <q-btn
                color="negative"
                glossy
                dense
                flat
                v-if="item.IsDir"
                @click="deleteThis(item)"
                >删除
              </q-btn>
            </q-card-actions>
          </q-card>
        </div>
      </q-card>
      <q-card class="cardcard">
        <q-toolbar class="bg-primary text-white" id="diskDiv">磁盘分析</q-toolbar>
        <div
          class="row q-gutter-sm q-pa-sm justfity-start shadow-2 rounded-borders"
        >
          <q-card
            v-for="item in scanTime"
            :key="item"
            style="height: fit-content; padding: 2px"
          >
            <q-card-section class="justify-between m0 p0" style="">
              <q-btn
                flat
                dense
                icon="folder"
                :label="item.Name"
                color="primary"
                @click="folderGotoMenu(item.Name)"
              >
              </q-btn>
              <q-separator inset />
              <div class="text_subtitle" style="text-align: right">
                <span> {{ item.SizeStr + ' | ' }}</span>
                <span style="color: var(--q-success)"> {{ item.Cnt }}ms</span>
              </div>
            </q-card-section>

            <q-card-actions align="center">
              <div class="row q-gutter-sm">
                <q-btn
                  color="primary"
                  flat
                  glossy
                  dense
                  v-if="item.IsDir"
                  @click="openThis(item)"
                  >打开
                </q-btn>
                <q-btn
                  color="negative"
                  dense
                  glossy
                  flat
                  v-if="item.IsDir"
                  @click="deleteThis(item)"
                  >删除
                </q-btn>
              </div>
            </q-card-actions>
          </q-card>
        </div>
      </q-card>
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
  DeleteFolerByPath,
  OpenFolerByPath,
} from '../components/api/searchAPI';
import {
  ScanTime,
  TagSizeMap,
  TypeSizeMap,
  SeriesCount,
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
const currentDiv = ref('tagDiv');

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
  const res = await TypeSizeMap();
  if (res) {
    tableData.value = res;
  }
  loadTagSize();
  loadScanTime();
  loadSeriesCount();
};

const loadTagSize = async () => {
  const res = await TagSizeMap();
  if (res) {
    tagData.value = res.length > 80 ? res.splice(0, 80) : res;
  }
};

const loadSeriesCount = async () => {
  const res = await SeriesCount();
  if (res) {
    seriesData.value = res.length > 80 ? res.splice(0, 80) : res;
  }
};
const loadScanTime = async () => {
  scanTime.value = await ScanTime();
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
  const inter = setInterval(() => {
    if (!tableData.value || tableData.value.length === 0) {
      loadTypeSize();
    } else {
      clearInterval(inter);
    }
  }, 5000);
});

const openThis = async (data) => {
  const { Name } = data;
  const res = await OpenFolerByPath({ dirpath: Name });
  if (res.Code === 200) {
    $q.notify({ type: 'positive', message: '执行成功' });
  } else {
    $q.notify({ type: 'warning', message: '执行失败' });
  }
};
const deleteThis = async (data) => {
  const { Name } = data;
  const res = await DeleteFolerByPath({ dirpath: Name });
  if (res.Code === 200) {
    $q.notify({ type: 'positive', message: '执行成功' });
    indexButton.value.refreshIndex();
    await f5();
  } else {
    $q.notify({ type: 'warning', message: '执行失败' });
  }
};
const refreshIndex = async () => {
  indexButton.value.refreshIndex();
};

const f5 = () => {
  window.location.reload();
};
</script>
<style>
.cardcard {
  border-radius: 10px;
  box-shadow: 0 4px 16px var(--q-shadow);
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
}
.p0 {
  padding: 2px;
}
.m0 {
  margin: 2px;
}
.text_subtitle {
  color: var(--q-text-secondary);
}
</style>
