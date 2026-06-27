<template>
  <q-layout
    view="hhh Lpr Lfr"
    class="shadow-2 rounded-borders bg-black"
    :style="{ height: '100vh' }"
  >
    <VideoPlayer ref="videoPlayerRef" id="videoPlayerRef" closeBtn fullscreen @close="closeThis" />
    <q-drawer
      elevated
      overlay
      bordered
      persistent
      side="right"
      v-model="view.showDrawer"
      :width="widthRight"
      style="background-color: rgba(0, 0, 0, 0.1)"
    >
      <VideoListSide
        @open-video="videoSideOpen"
        @closebtn="
          () => {
            view.showDrawer = false;
          }
        "
        @forward-time="forwardTime"
        ref="playerlist"
      />
    </q-drawer>
  </q-layout>
</template>

<script setup>
import { onMounted, onUnmounted, ref, reactive } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { FindFileInfo } from 'components/api/searchAPI';
import VideoPlayer from 'components/VideoPlayer.vue';
import VideoListSide from 'components/VideoListSide.vue';
import { useSystemProperty } from 'src/stores/System';

const systemProperty = useSystemProperty();
const router = useRouter();

const params = useRoute().params;

const videoPlayerRef = ref(null);

const view = reactive({
  showDrawer: false,
});

const widthRight = ref(400);

const forwardTime = (time) => {
  videoPlayerRef.value.forwardTime(time);
};

const videoSideOpen = (params) => {
  const { item } = params;
  openVideo(item);
  view.showDrawer = false;
};

const beforeUnloadHandler = () => {
  closeThis();
};

const popStateHandler = () => {
  router.go(0);
};

const resizeHandler = (e) => {
  const { innerWidth, innerHeight } = e.currentTarget;
  systemProperty.singleWindow.height = innerHeight;
  systemProperty.singleWindow.width = innerWidth;
};

window.addEventListener('beforeunload', beforeUnloadHandler);
window.addEventListener('popstate', popStateHandler);
window.addEventListener('resize', resizeHandler);

onUnmounted(() => {
  window.removeEventListener('beforeunload', beforeUnloadHandler);
  window.removeEventListener('popstate', popStateHandler);
  window.removeEventListener('resize', resizeHandler);
});

const closeThis = () => {
  window.close();
};

onMounted(async () => {
  const { id } = params;
  const data = await FindFileInfo(id);
  document.title = data.Name;
  videoPlayerRef.value.openVideo(data);
});
</script>

<style lang="scss" scoped>
.example-item {
  width: 140px;
  height: auto;
  max-height: 320px;
  overflow: hidden;
}

.item-img {
  width: 140px;
  height: auto;
  max-height: 220px;
}
</style>
