<template>
  <q-layout
    view="hHh Lpr lff"
    container
    style="height: 100vh"
    class="shadow-2 rounded-borders"
    :class="{ 'theme-natural': systemProperty.theme === 'natural' }"
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
        <q-btn dense flat icon="chat" @click="openChatRoom" class="q-ml-xs">
          <q-badge v-if="wsOnlineCount > 0" floating color="positive" transparent>
            {{ wsOnlineCount }}
          </q-badge>
        </q-btn>
      </q-toolbar>
    </q-header>

    <q-drawer v-model="drawerLeft" :width="drawerWidth" :breakpoint="700" bordered
      :dark="systemProperty.theme !== 'natural'"
      :class="{ 'drawer-natural': systemProperty.theme === 'natural' }">
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
        <keep-alive :max="5">
          <component :is="Component" :key="route.path" />
        </keep-alive>
      </router-view>
    </q-page-container>
      <ShutdownComponent ref="shutdown" />
      <ListEdit ref="listEditRef" />
      <ChatDeepseek ref="chatRef" />
      <ChatRoom ref="chatRoomRef" />
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
import ChatRoom from 'components/ChatRoom.vue';
import { useChatWs } from 'src/composables/useChatWs';

const listEditRef = ref(null);
const chatRef = ref(null);
const chatRoomRef = ref(null);

const systemProperty = useSystemProperty();
const $q = useQuasar();
const router = useRouter();

const shutdown = ref(null);
const drawerLeft = ref(false);
const view = reactive({
  fullscreen: false,
});

// 动态 header 样式 — Design System
const headerStyle = computed(() => {
  return systemProperty.theme === 'natural'
    ? 'background: #DBEAFE !important; color: #1E3A5F; border-bottom: 1px solid #93C5FD;'
    : 'background: #181B27 !important; color: #E8EAF2; border-bottom: 1px solid rgba(139,143,168,0.1);';
});

// 动态抽屉样式 — Design System: bg-shift, no borders
const drawerStyle = computed(() => {
  return systemProperty.theme === 'natural'
    ? 'background-color: #DBEAFE; color: #1E3A5F;'
    : 'background-color: #181B27; color: #E8EAF2;';
});

// 响应式抽屉宽度
const drawerWidth = computed(() => {
  return $q.screen.width > 1200 ? 240 : 200;
});

const timeLogout = ref('');
const timeLogoutShow = ref('');

let logoutTimer = null;

// 仅在用户已认证时启动定时器
if (sessionStorage.getItem('isAuthenticated')) {
  logoutTimer = setInterval(() => {
    const time = parseInt(
      (systemProperty.expireTime - new Date().getTime()) / 1000
    );
    timeLogout.value = time;
    if (time > 3600) {
      timeLogoutShow.value = `${Math.round(time / 3600)}小时`;
    } else if (time > 60) {
      timeLogoutShow.value = `${Math.round(time / 60)}分钟`;
    } else {
      timeLogoutShow.value = `${time}秒`;
    }
    if (time < 60) {
      console.log('即将退出', timeLogoutShow.value);
    }
    if (!systemProperty.expireTime || time < 0) {
      sessionStorage.removeItem('isAuthenticated');
      router.push('/');
    }
  }, 3000);
}

onUnmounted(() => {
  if (logoutTimer) {
    clearInterval(logoutTimer);
    logoutTimer = null;
  }
  wsDisconnect();
});

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

// 聊天室
const { onlineUsers: wsOnlineUsers, connect: wsConnect, disconnect: wsDisconnect } = useChatWs();
const wsOnlineCount = computed(() => wsOnlineUsers.value.length);

const openChatRoom = () => {
  chatRoomRef.value.open();
};

// 登录后自动连接 WebSocket
if (sessionStorage.getItem('isAuthenticated')) {
  wsConnect();
}

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
    caption: '数据统计与概览',
    icon: 'ti-stats-up',
    link: '/data',
  },
  {
    title: '搜索',
    caption: '多媒体文件搜索',
    icon: 'search',
    link: '/',
  },
  {
    title: '图鉴',
    caption: '图片浏览与管理',
    icon: 'image',
    link: '/picture',
  },
  {
    title: '配置',
    caption: '系统参数设置',
    icon: 'settings',
    link: '/setting',
  },
  {
    title: '系统',
    caption: '系统信息与状态',
    icon: 'info',
    link: '/system',
  },
  {
    title: '沉浸',
    caption: '沉浸式播放体验',
    icon: 'movie',
    link: '/immersive',
  },
];
</script>

<style lang="scss" scoped>
// 主 Header 样式 — Minimalism: no shadow, subtle border
.main-header {
  transition: background-color 0.4s ease;
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

// 自然模式下抽屉背景色
.drawer-natural {
  background-color: #DBEAFE !important;
}

// 自然模式下抽屉 item 文字色
.drawer-natural ::deep(.q-item) {
  color: #1E3A5F !important;
}

// 自然模式下抽屉中按钮颜色覆盖
.drawer-natural ::deep(.q-btn) {
  color: #1E3A5F !important;
}

// 自然模式下 header 中 EssentialLink 按钮文字颜色
.theme-natural .q-header ::deep(.q-btn) {
  color: #1E3A5F !important;
}

// 自然模式下 header 中当前页面的 EssentialLink 按钮保持红色
.theme-natural .q-header ::deep(.q-btn.text-red) {
  color: #EF4444 !important;
}
</style>
