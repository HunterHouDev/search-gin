<template>
  <q-btn flat dense color="orange" class="fts12" v-if="!props.nosize">
    {{
      systemProperty.PlayingMovie?.Size
        ? humanStorageSize(systemProperty.PlayingMovie.Size)
        : ''
    }}
  </q-btn>
  <q-btn
    flat
    color="orange"
    label="PNG"
    class="fts12"
    @click="previewPng"
    v-if="systemProperty.PlayingMovie.MovieType != '骑兵'"
  />
  <q-btn
    flat
    v-for="item in fowartBtn"
    :color="
      item == systemProperty.videoOptions.arrowForwardTime ? 'orange' : 'red'
    "
    :key="item"
    :label="item"
    @click="
      systemProperty.videoOptions.arrowForwardTime = item;
      forwardTime(item);
    "
    @contextmenu="
      (e) => {
        forwardTime(-item);
        e.returnValue = false;
      }
    "
    class="q-pa-sm fts12"
    :style="{
      fontSize:
        item == systemProperty.videoOptions.arrowForwardTime
          ? '1.4rem'
          : '1.2rem',
    }"
  >
  </q-btn>

  <q-btn
    dense
    flat
    icon="ti-control-play"
    color="red"
    v-if="!systemProperty.playerRunning"
    @click="playVideo"
  ></q-btn>
  <q-btn
    dense
    flat
    icon="ti-control-pause"
    v-if="systemProperty.playerRunning"
    color="red"
    @click="stopVideo"
  ></q-btn>
</template>

<script setup>
import { inject } from 'vue';
import { CutImage } from 'components/api/searchAPI';
import { onKeyStroke } from '@vueuse/core';
import { useSystemProperty } from 'src/stores/System';
import { format } from 'quasar';
import { DeleteFile } from 'components/api/searchAPI';
import { useQuasar } from 'quasar';

const $q = useQuasar();
const { humanStorageSize } = format;

const systemProperty = useSystemProperty();

const props = defineProps({
  currentTime: {
    type: String,
    default: '00:00:00',
  },
  nosize: {
    type: Boolean,
    default: false,
  },
});

const emmits = defineEmits([
  'forwardTime',
  'timeRate',
  'stopVideo',
  'playVideo',
  'prevOne',
  'nextOne',
  'delete',
]);

const fowartBtn = [-30, -15, 60, 120, 240, 300];

const forwardTime = (time) => {
  emmits('forwardTime', time);
};

const previewPng = async () => {
  setTimeout(async () => {
    await CutImage(systemProperty.PlayingMovie.Id, 'Png', props.currentTime, false);
    $q.notify({ message: `已执行`, position: 'bottom-left' });
  }, 1);
};

onKeyStroke(['ArrowRight'], (e) => {
  if (systemProperty.PlayingMovie.Id) {
    if (systemProperty.videoOptions.arrowForwardTime) {
      forwardTime(systemProperty.videoOptions.arrowForwardTime);
    } else {
      forwardTime(60);
    }
    e.preventDefault();
  }
});
onKeyStroke(['ArrowLeft'], (e) => {
  if (systemProperty.PlayingMovie.Id) {
    if (systemProperty.videoOptions.arrowForwardTime) {
      forwardTime(-systemProperty.videoOptions.arrowForwardTime);
    } else {
      forwardTime(-60);
    }
    e.preventDefault();
  }
});

const nextPage = inject('gotoNextPage', () => {
  console.log('gotoNextPage not found');
});

const prevPage = inject('gotoPrevPage', () => {
  console.log('gotoPrevPage not found');
});
onKeyStroke(true, (e) => {
  console.log('onKeyStroke', e.code);
  if (!isNaN(e.key) && systemProperty.PlayingMovie.Id) {
    const r = e.key / 10;
    emmits('timeRate', r);
  } else if (e.code === 'NumpadSubtract') {
    console.log('prevPage', prevPage);

    emmits('prevOne');
  } else if (e.code === 'NumpadAdd') {
    console.log('nextPage', nextPage);

    emmits('nextOne');
  }
});

onKeyStroke(['PageUp'], (e) => {
  if (systemProperty.PlayingMovie.Id) {
    prevPage();
    e.preventDefault();
  }
});
onKeyStroke(['PageDown'], (e) => {
  if (systemProperty.PlayingMovie.Id) {
    nextPage();
    e.preventDefault();
  }
});
onKeyStroke(['Delete'], () => {
  if (systemProperty.PlayingMovie && systemProperty.PlayingMovie.Id) {
    confirmDelete();
  }
});

const stopVideo = () => {
  emmits('stopVideo');
};

const playVideo = () => {
  emmits('playVideo');
};

const confirmDelete = () => {
  const item = systemProperty.PlayingMovie; // 从 props 中获取 currentData 的值
  if (!item) return; // 如果 item 不存在，直接返回，不执行后续代码
  $q.dialog({
    title: item.Name,
    message: '确定删除吗?',
    cancel: true,
    persistent: true,
  }).onOk(() => {
    if (systemProperty.addPlayingTagGoNext) {
      emmits('nextOne');
    } else {
      emmits('prevOne');
    }

    const time1 = setTimeout(async () => {
      const { Code, Message } = (await DeleteFile(item.Id)) || {};
      if (Code !== 200) {
        $q.notify({ message: `${Message}`, position: 'bottom-left' });
      }
      clearTimeout(time1);
    }, 2000);
  });
};
</script>

<style scoped>
/* 视频播放器样式 */
.fts12 {
  font-size: 1.2rem;
  padding: 0px;
  margin-right: 6px;
}
.w100 {
  width: 100%;
}
</style>
