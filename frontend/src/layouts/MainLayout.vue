<template>
  <q-layout
    view="hHh Lpr lff"
    container
    style="height: 100vh"
    class="shadow-2 rounded-borders"
  >
    <q-header reveal class="bg-black glossy">
      <q-toolbar class="q-electron-drag">
        <q-btn flat @click="drawerLeft = !drawerLeft" round dense icon="menu" />
        <q-toolbar-title style="-webkit-app-region: drag">
          <a href="/#/search" custom color="red" v-show="!isWideScreen">
            <q-btn flat color="white" dense size="lg" align="left">搜 索</q-btn>
          </a>
        </q-toolbar-title>
        <EssentialLink
          v-for="link in essentialLinks"
          :key="link.title"
          v-bind="link"
          v-show="isWideScreen"
        />
        <q-space v-if="isWideScreen" />
        <q-btn dense flat size="lg" icon="refresh" @click="refreshThis"></q-btn>
        <q-btn
          @click="changeTheme"
          v-if="isDesktop"
          dense
          icon="ti-reload"
          flat
          :color="$q.dark.mode ? 'white' : 'grey'"
        ></q-btn>
        <q-btn
          flat
          dense
          size="lg"
          :icon="view.fullscreen ? 'fullscreen_exit' : 'fullscreen'"
          v-model="view.fullscreen"
          @click="clickFullscreen"
        />
        <q-btn dense flat color="red" v-if="shutdownLeftSecond"
          >关机倒计时：{{ shutdownLeftSecond }}
        </q-btn>

        <q-btn dense flat icon="ti-na" @click="confirmShutDown" />
        <q-btn
          v-if="isDesktop"
          dense
          flat
          icon="ti-minus"
          @click="minusScreen"
        />
        <q-btn
          v-if="isDesktop"
          flat
          size="lg"
          icon="ti-close"
          @click="closeWindow"
        />
        <q-btn
          flat
          dense
          color="red"
          @click="openChatDialogRef"
          v-if="isDesktop"
          label="a"
          style="position: hidden"
        ></q-btn>
        <q-btn dense flat color="red" v-if="timeLogout < 60 * 30"
          >时长:{{ timeLogoutShow }}
        </q-btn>
      </q-toolbar>
    </q-header>

    <q-drawer v-model="drawerLeft" :width="200" :breakpoint="700" bordered>
      <q-scroll-area
        class="fit"
        style="background-color: rgba(0, 0, 0, 0.8); color: aliceblue"
      >
        <q-list>
          <q-item-label header> 你的搜索工具</q-item-label>
          <EssentialLink
            v-for="link in essentialLinks"
            :key="link.title"
            v-bind="link"
          />
        </q-list>
      </q-scroll-area>
    </q-drawer>
    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
  <ShutdownComponent ref="shutdown" />
  <ListEdit ref="listEditRef" />
  <ChatDeepseek ref="chatRef" />
</template>

<script setup>
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useSystemProperty } from 'stores/System';
import { useQuasar } from 'quasar';
import EssentialLink from 'components/EssentialLink.vue';
import ListEdit from 'pages/file/components/ListEditDialog.vue';
import ShutdownComponent from 'components/ShutdownComponent.vue';
import ChatDeepseek from 'pages/file/components/ChatDeepseek.vue';

const listEditRef = ref(null);
const chatRef = ref(null);

const systemProperty = useSystemProperty();
const $q = useQuasar();
const router = useRouter();

const shutdown = ref(null);
const drawerLeft = ref(false);
const view = reactive({
  fullscreen: false,
});

let logoutTimer = null;

logoutTimer = setInterval(() => {
  const time = parseInt(
    (systemProperty.expireTime - new Date().getTime()) / 1000
  );
  timeLogout.value = time;
  if (time > 3600) {
    timeLogoutShow.value = `${Math.round(time / 3600, 1)}小时`;
  } else if (time > 60) {
    timeLogoutShow.value = `${Math.round(time / 60, 1)}分钟`;
  } else {
    timeLogoutShow.value = `${time}秒`;
  }
  if (time < 60) {
    console.log('即将退出', timeLogoutShow.value);
  }
  if (!systemProperty.expireTime || time < 0) {
    localStorage.removeItem('isAuthenticated');
    router.push('/');
  }
}, 1000);

onUnmounted(() => {
  if (logoutTimer) {
    clearInterval(logoutTimer);
    logoutTimer = null;
  }
});

const timeLogout = ref('');
const timeLogoutShow = ref('');

const isWideScreen = computed(() => {
  return $q.screen.width > 750;
});

systemProperty.isElectron = $q.platform.is.electron;

const isDesktop = computed(() => {
  return $q.platform.is.electron;
});

const closeWindow = () => {
  window.close();
};

const minusScreen = () => {
  window.electron.hideMainWindow();
};

const changeTheme = () => {
  systemProperty.isDark = !systemProperty.isDark;
};

const openChatDialogRef = () => {
  chatRef.value.open();
};

const clickFullscreen = () => {
  if (isDesktop.value) {
    window.electron.maxMainWindow();
  } else {
    if (!view.fullscreen) {
      $q.fullscreen.request();
    } else {
      $q.fullscreen.exit();
    }
  }
  view.fullscreen = !view.fullscreen;
};

const shutdownLeftSecond = computed(() => {
  let left = systemProperty.shutdownLeftSecond;
  if (!left) {
    return null;
  }
  return `${Math.floor(
    systemProperty.shutdownLeftSecond / 3600
  )} 时 ${Math.floor(
    (systemProperty.shutdownLeftSecond / 60) % 60
  )} 分 ${Math.floor(systemProperty.shutdownLeftSecond % 60)} 秒`;
});

$q.dark.set(systemProperty.isDark);
watch(
  () => systemProperty.isDark,
  (v) => {
    $q.dark.set(v);
  }
);

const refreshThis = () => {
  window.location.reload();
};

const confirmShutDown = () => {
  shutdown.value.open();
};

const essentialLinks = [
  {
    title: '首页',
    caption: 'github.com/quasarframework',
    icon: 'ti-stats-up',
    link: '/data',
  },
  {
    title: '搜索',
    caption: 'quasar.dev',
    icon: 'search',
    link: '/',
  },
  {
    title: '图鉴',
    caption: 'chat.quasar.dev',
    icon: 'image',
    link: '/picture',
  },

  {
    title: '配置',
    caption: 'chat.quasar.dev',
    icon: 'settings',
    link: '/setting',
  },
  {
    title: '系统',
    caption: 'forum.quasar.dev',
    icon: 'chat',
    link: '/system',
  },
  {
    title: '日志',
    caption: 'forum.quasar.dev',
    icon: 'chat',
    link: '/systemLog',
  },
  {
    title: '沉浸体验',
    caption: 'immersive player',
    icon: 'movie',
    link: '/immersive',
  },
];
</script>

<style lang="scss" scoped>
// 隐藏滚动条
.scroll::-webkit-scrollbar {
  display: none;
}

// 兼容 Firefox
.scroll {
  scrollbar-width: none;
}

// 兼容 IE 和 Edge
.scroll {
  -ms-overflow-style: none;
}
</style>
