import { computed } from 'vue'
import { useQuasar } from 'quasar'

// 统一响应式断点判断
// 替代各组件中各自定义的 isMobile / isSmall / isLarge / isMedium

export function useBreakpoint() {
  const $q = useQuasar()

  const isMobile = computed(() => $q.platform.is.mobile)
  const isSmall = computed(() => $q.screen.lt.sm)
  const isMedium = computed(() => $q.screen.md)
  const isLarge = computed(() => $q.screen.gt.md)

  // 兼容旧组件的 showStyle 模式
  function fromStyle(style: () => string) {
    return {
      isSmall: computed(() => style() === 'sm'),
      isMedium: computed(() => style() === 'md'),
      isLarge: computed(() => style() === 'lg'),
    }
  }

  return { isMobile, isSmall, isMedium, isLarge, fromStyle }
}
