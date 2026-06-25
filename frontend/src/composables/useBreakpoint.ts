import { computed } from 'vue'
import { useQuasar } from 'quasar'

// 统一响应式断点判断
// 替代各组件中各自定义的 isMobile / isSmall / isLarge / isMedium

export function useBreakpoint() {
  const $q = useQuasar()

  // Quasar 断点: xs < 600 < sm < 1024 < md < 1440 < lg < 1920 < xl
  const isMobile = computed(() => $q.platform.is.mobile)
  const isSmall  = computed(() => $q.screen.lt.sm)     // < 600px
  const isMedium = computed(() => $q.screen.md)          // 1024-1440
  const isLarge  = computed(() => $q.screen.gt.md)       // > 1440

  /** 窄屏：手机或超小窗口 */
  const isNarrow  = computed(() => $q.screen.lt.md)     // < 1024px

  /** 宽屏：桌面标准尺寸 */
  const isWide    = computed(() => $q.screen.gt.sm)     // > 600px

  /** 极宽屏：大显示器 */
  const isExtraWide = computed(() => $q.screen.gt.lg)   // > 1920px

  // 自适应：卡片显示的列数
  const cardColumns = computed(() => {
    if ($q.screen.xl) return 8
    if ($q.screen.lg) return 6
    if ($q.screen.md) return 4
    if ($q.screen.sm) return 3
    return 2
  })

  // 自适应：侧边栏宽度
  const drawerWidth = computed(() => {
    return isNarrow.value ? 200 : 260
  })

  // 自适应：搜索卡片尺寸标识
  const cardSize = computed(() => {
    if ($q.screen.xl) return 'lg'
    if ($q.screen.lg) return 'md'
    return 'sm'
  })

  // 自适应：分页大小
  const pageSize = computed(() => {
    if ($q.screen.xl) return 48
    if ($q.screen.lg) return 36
    if ($q.screen.md) return 24
    return 14
  })

  // 兼容旧组件的 showStyle 模式
  function fromStyle(style: () => string) {
    return {
      isSmall:  computed(() => style() === 'sm'),
      isMedium: computed(() => style() === 'md'),
      isLarge:  computed(() => style() === 'lg'),
    }
  }

  return {
    isMobile, isSmall, isMedium, isLarge,
    isNarrow, isWide, isExtraWide,
    cardColumns, drawerWidth, cardSize, pageSize,
    fromStyle,
  }
}
