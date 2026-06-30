<template>
  <div v-cloak class="app-root">
    <ParticleBackground />
    <router-view v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch, onErrorCaptured } from 'vue';
import { useQuasar } from 'quasar';
import ParticleBackground from 'components/ParticleBackground.vue';
import { useSystemProperty } from './stores/System';
const systemProperty = useSystemProperty();
const $q = useQuasar();

// 设置主题类
const applyTheme = (theme: string) => {
  const html = document.documentElement;
  if (theme === 'natural') {
    html.classList.add('theme-natural');
  } else {
    html.classList.remove('theme-natural');
  }
};

// 监听主题变化
watch(
  () => systemProperty.theme,
  (newTheme) => applyTheme(newTheme),
  { immediate: true }
);

// 全局 Vue 错误捕获 — 防止白屏，显示友好提示
onErrorCaptured((err, instance, info) => {
  console.error('[Vue Error]', err, info);
  $q.notify({
    type: 'negative',
    message: '页面渲染异常，请刷新重试',
    position: 'top',
    timeout: 5000,
    icon: 'bug_report',
  });
  // 阻止错误继续传播（防止白屏）
  return false;
});

// 全局未捕获 Promise 异常
onMounted(() => {
  applyTheme(systemProperty.theme);
  document.body.classList.add('app-ready');
  // 全局 SSE 已移除，任务日志 SSE 在 ListEditDialog 弹窗内独立管理

  window.addEventListener('unhandledrejection', (event) => {
    console.error('[Unhandled Promise]', event.reason);
    // 网络错误已在 axios 拦截器中处理，这里只打日志不做通知
  });
});

  // onUnmounted 无需额外清理，任务日志 SSE 在弹窗内管理
</script>

<style>
[v-cloak] { display: none; }

/* 路由过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
