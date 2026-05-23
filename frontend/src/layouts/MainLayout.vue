<template>
  <q-layout
    view="hHh Lpr lff"
    container
    style="height: 100vh"
    class="shadow-2 rounded-borders"
  >
    <q-header 
      reveal 
      class="main-header"
      :style="headerStyle"
    >
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

    <q-drawer v-model="drawerLeft" :width="drawerWidth" :breakpoint="700" bordered>
      <q-scroll-area
        class="fit"
        :style="drawerStyle"
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
      <router-view v-slot="{ Component, route }">
        <transition name="page-fade" mode="out-in">
          <component :is="Component" :key="route.path" />
        </transition>
      </router-view>
    </q-page-container>
      <ShutdownComponent ref="shutdown" />
      <ListEdit ref="listEditRef" />
      <ChatDeepseek ref="chatRef" />
  </q-layout>

</template>

<script setup>
import { computed, onUnmounted, reactive, ref } from 'vue';
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

// 动态 header 样式
const headerStyle = computed(() => {
  return systemProperty.theme === 'natural'
    ? 'background: linear-gradient(90deg, #94a3b8 0%, #cbd5e1 100%) !important; color: white;'
    : 'background: rgba(0, 0, 0, 0.85) !important; color: white;';
});

// 动态抽屉样式 - 与主题保持一致
const drawerStyle = computed(() => {
  return systemProperty.theme === 'natural'
    ? 'background-color: rgba(245, 243, 255, 0.95); color: #5b21b6;'
    : 'background-color: rgba(0, 0, 0, 0.85); color: aliceblue;';
});

// 响应式抽屉宽度
const drawerWidth = computed(() => {
  return $q.screen.width > 1200 ? 240 : 200;
});

let logoutTimer = null;

// 仅在用户已认证时启动定时器
if (localStorage.getItem('isAuthenticated')) {
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
  }, 3000);
}

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
    icon: 'info',
    link: '/system',
  },
  {
    title: '沉浸',
    caption: 'immersive player',
    icon: 'movie',
    link: '/immersive',
  },
];
</script>

<style lang="scss" scoped>
// 主 Header 样式
.main-header {
  /* backdrop-filter: blur(16px); - 移除以提升性能 */
  border-bottom: 1px solid var(--q-border);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  transition: background-color 0.4s ease;
}

// 页面切换过渡动画
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

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

// 抽屉样式优化
:deep(.q-drawer) {
  /* backdrop-filter: blur(16px); - 移除以提升性能 */
  transition: transform 0.3s ease;
}
</style>
