// API 统一入口 — 从原有 locations 再导出，后续逐步迁移到 src/api/ 下
export * from 'components/api/searchAPI'
export * from 'components/api/settingAPI'
export * from 'components/api/homeAPI'
export * from 'components/api/authorAPI'

// Torrent API（新增，原来散落在 ImmersivePlayer.vue 中直接调 axios）
export * from './torrentAPI'
