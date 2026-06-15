import { defineStore } from 'pinia'

// 主题 / 窗口 / 关机 相关的应用级状态
// 原 System.ts 中与搜索/播放无关的通用设置

export const useAppStore = defineStore({
  id: 'app',
  persist: {
    enabled: true,
    strategies: [{ key: 'appProperty', storage: localStorage }],
  },
  state: () => ({
    singleWindow: { width: 1280, height: 720 },
    showStyle: 'lg' as 'lg' | 'md' | 'sm',
    showImage: 'poster' as 'post' | 'cover',
    isFullscreen: false,
    theme: 'star' as 'star' | 'natural',
    isElectron: false,
    shutdownLeftSecond: null as number | null,
    shutdownTimer: null as ReturnType<typeof setInterval> | null,
  }),
  getters: {
    isDark: (state) => state.theme === 'star',
    windowSize: (state) => state.singleWindow,
  },
  actions: {
    setTheme(theme: 'star' | 'natural') { this.theme = theme },
    setShowStyle(style: 'lg' | 'md' | 'sm') { this.showStyle = style },
    setShowImage(mode: 'post' | 'cover') { this.showImage = mode },
    toggleFullscreen() { this.isFullscreen = !this.isFullscreen },
  },
})
