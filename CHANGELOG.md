# Changelog

格式基于 [Keep a Changelog](https://keepachangelog.com/)，版本遵循 [SemVer](https://semver.org/)。

## [Unreleased]

### Added

- **工程化基础设施**
  - CI/CD：GitHub Actions 自动化（Go lint/test/build/vet + 前端 lint/build）(#2)
  - Linter 配置：`.golangci.yml`（govet/staticcheck/gosec/errcheck 等）
  - Makefile：统一 dev/build/test/lint/clean 工作流
  - CHANGELOG.md 版本发布日志
  - CONTRIBUTING.md 开发者指南
  - Docker 化：Dockerfile（multistage build）+ docker-compose.yml
  - pre-commit hooks：husky + lint-staged 自动 lint

- **安全**
  - 密码 bcrypt 哈希化：启动时自动迁移 `setting.json` 中的明文密码 (#3)
  - `os.WriteFile` 权限收紧：日志文件 0600，配置文件 0600

- **UI/UX 设计系统**
  - 登录页适配双主题（星空/自然），替换硬编码渐变 (#4)
  - 空状态组件 `EmptyState.vue`（4 种插画）(#5)
  - 全局 axios 错误拦截器：统一 4xx/5xx/超时/断网提示 (#5)
  - 全局 Vue 错误边界：`onErrorCaptured` 防止白屏 (#5)
  - 微交互动效：卡片 hover、列表入场/出场动画、展开折叠 (#7)
  - 响应式工具类：`.mobile-only` / `.desktop-only` / 触摸优化 (#8)
  - 品牌 Logo 系统：SVG 品牌标识 + SVG favicon + loading splash
  - 路由过渡动画：页面切换淡入淡出

- **架构**
  - Handler 层测试覆盖：11 个测试用例（health/auth/search）(#9)
  - 增强 `useBreakpoint` composable（isNarrow/cardColumns/pageSize 等）(#8)

### Changed

- `.gitignore` 精简分组，移除历史遗留目录 (#1)
- 全局 CSS 变量一致性：SearchPanel 硬编码 `indigo-4` → `primary` (#6)
- MainLayout 响应式：导入 useBreakpoint composable，移动端导航后自动关闭抽屉 (#8)

### Security

- `setting.json` 用户密码存储方式从明文改为 bcrypt hash（启动时自动迁移，无感）(#2)
- 移除 `os.ModePerm` 文件写入权限，统一为 0600

---

## [0.1.0] - 2026-06-25

### Added

- 基于 Golang + Vue 3 的本地文件搜索、管理与媒体播放系统
- 内存全文索引 + LRU 缓存，atomic.Value 无锁并发读
- 视频播放（内嵌/画中画/沉浸式全屏）
- 磁力链接解析与边下边播
- FFmpeg 视频剪辑（时间范围剪切/截图/转码）
- 图片缩略图网格浏览
- 文件管理（重命名/移动/删除/标签）
- 多用户认证（管理员 + 普通用户）
- 多节点集群（HTTP 信令发现 + 跨节点搜索）
- WebRTC 点对点视频通话
- WebSocket 实时聊天 + AI 集成
- 双主题设计系统（星空深色 / 自然浅色）
- Electron 桌面应用打包
- Quasar 前端框架 + Pinia 状态管理
