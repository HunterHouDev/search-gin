<template>
  <div v-cloak class="app-root">
    <ParticleBackground />
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue';
import ParticleBackground from 'components/ParticleBackground.vue';
import { useSystemProperty } from './stores/System';

const systemProperty = useSystemProperty();

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

onMounted(() => {
  applyTheme(systemProperty.theme);
  // 移除加载完成标识
  document.body.classList.add('app-ready');
});
</script>

<style scoped>
.app-root {
  min-height: 100vh;
  background-color: var(--q-bg-darker);
}
</style>
