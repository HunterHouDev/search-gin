# AGENTS.md — search-gin

## Build tag 系统

`-tags=prod` 控制两件事：**资源嵌入**（`assets_dev.go` 无 embed vs `assets_prod.go` `//go:embed dist ffmpeg.exe ffplay.exe setting.json`）和**运行时模式**（`internal/env/config.go` `IsProd=false` 默认 vs `internal/env/prod_config.go` `init()` 设 `IsProd=true`）。生产模式：Gin ReleaseMode、禁用 pprof、日志级别 `ErrorLevel`。开发模式 `go run main.go` 默认。

## 端口

| 端口 | 路由 | 认证 |
|------|------|------|
| `:10081` | `BuildAPIRouter()` API + 前端 | Token 认证（AuthMiddleware） |
| `:10082` | `BuildFileRouter()` 文件/图片/视频流 | streamToken 校验（StreamTokenAuth） |

端口定义在 `internal/service/index_param.go`（`PortNo`/`FilePortNo` 常量），可通过 `setting.json` 的 `ControllerHost`/`FileHost` 覆盖。

### Server 层（`internal/server/server.go`）

`CreateServer(addr, handler)` 创建 HTTP 服务，注意事项：
- 不设 `ReadTimeout` / `WriteTimeout`（WebSocket hijack 后超时残留导致断连）
- 只设 `ReadHeaderTimeout: 10s` 防范慢连接攻击
- `ResolvePort(portNo, controllerHost)` 从 `ControllerHost`（如 `"127.0.0.1:10081"`）提取端口号 `":10081"`
- `GracefulShutdown(sigChan, servers)` 收到信号后先 `FlushCache()` 强制写入索引缓存，再 5s timeout 关闭各服务

## 认证 skip paths（`middleware/common.go`）

`/api/login, /login, /, /index.html, /api/ws, /api/events, /api/lanPeers, /api/heartBeat, /api/authorImage/, /css/, /js/, /assets/, /icons/, /favicon.ico, /api/stream/`

⚠️ skip path 检查必须在 `X-Search-Gin-Remote` 校验**之前**，否则跨节点验证会形成递归死锁。

⚠️ `/` 路径须精确匹配（`path == "/"`），不能用前缀匹配，否则所有 `/api/*` 会被错误跳过。

## 前端构建 / 嵌入

```
cd frontend && yarn build && cp -R dist/spa/* ../dist/ && go build -tags=prod
```

- 开发模式 `go run main.go` 从磁盘读 `./dist/`
- 生产二进制启动时解压嵌入资源到工作目录

## 认证

- 管理员用户 `admin`，密码必须配 `setting.json` 的 `adminPassword`（无编译回退）
- Token 存内存 map `tokenStore map[string]TokenInfo` 受 `sync.RWMutex` 保护（`auth_service.go`），每个 token 创建时注册 `time.AfterFunc` 到期自动删除，无定时轮询
- `Authorization: Bearer <token>` 或 WebSocket `?token=`
- 集群节点间用 `X-Search-Gin-Remote: true` header 跳过 Token 认证，来源 IP 必须在 peers 列表中（`middleware/common.go` + `node_discovery.go`）
- `requireAdmin()` 检查 role 是否为 `AdminRole`，兼容旧 token（role 为空时放行）
- **注意事项**：
  - WebSocket 连接使用 query token（`/api/ws?token=xxx`），skip path 已跳过 AuthMiddleware 故需手动调用 `ValidateTokenWithInfo`
  - `:10082` 文件流端口使用 `StreamTokenAuth` 中间件（AES-256-GCM 加密 streamToken），定义在 `internal/router/build_router.go`，不依赖 session map，跨节点可互相解密

## 文件流安全（`:10082` 端口）

文件流端口（10082）不再使用 HMAC 签名（`SignAuthMiddleware` 未注册），改用 **streamToken**（AES-256-GCM）机制：

- 后端 `FillURLs()` 为每个文件生成加密 token（内含过期时间戳）
- 图片预览 token 有效期 **5 分钟**，视频流 token 有效期 **4 小时**
- `:10082` 端口的 `StreamTokenAuth` 中间件解密验证 token
- 每个节点启动时随机生成独立密钥（`pkg/utils/stream_crypto.go`），不持久化到 setting.json
- `SignAuthMiddleware`（HMAC 签名）仍保留在代码中但不注册——签名对多节点集群不可用

## 多节点集群

- 节点管理：`background_launch.go:InitPeerManager()` 加载 `discoveryPeers`（节点发现逻辑在 `node_discovery.go`），支持运行时增删
- 发现方式：
  - **子网扫描**：前端输入三段 IP 前缀（如 `192.168.1`）扫 /24，后端 `DiscoverLanPeers` 用共享 HTTP client + 20 并发扫描
  - **单机检测**：输入完整 IP（如 `192.168.1.50`）直接心跳验证
  - **手动添加**：发现面板中候选节点一键添加，自动持久化到 `setting.json`
  - **反向心跳自动发现**：未知 IP 发 `X-Search-Gin-Remote: true` 时，middleware 自动反向心跳验证，通过则加入集群。这是有意设计，不是漏洞——局域网是信任域，与 Redis/Cassandra 同理。
- 跨节点搜索：`remote_search.go` 并发请求所有在线节点（最多 5 并发），去重策略 `Code+Size` 优先，`Name+Size` 兜底
- 文件流端口统一走 `:10082`
- 配置：`{"enableLanDiscovery": true, "discoveryPeers": ["192.168.1.102:10081"]}`

## 平台假设

- **Windows 为主**：`ffmpeg.exe`/`ffplay.exe`、`cmd /C start` 打开文件夹、`-H=windowsgui` 链接器
- `pkg/utils/fixOnWin.go` / `pkg/utils/fixOnNotWin.go`（build tag `windows`）隐藏子进程控制台
- `cmd /C ping  -n 1 -w 2000 <ip>` 检测主机连通性（`PingHost` handler）

## Go 模块

模块名：`search-gin`。所有内部导入使用 `search-gin/internal/...` 和 `search-gin/pkg/...`。Go 版本要求：1.25.0+（见 `go.mod`）。注意 `pkg/utils/` 导入 `internal/env`——这是本仓库的设计。

## 配置结构

配置 `Setting` struct 定义在 `pkg/types/setting.go`（非 `internal/service/`），通过 `GetOSSetting()` 全局访问。关键 key：

| 字段 | 类型 | 说明 |
|------|------|------|
| `Dirs` | `[]string` | 扫描目录 |
| `VideoTypes`/`ImageTypes`/`DocsTypes`/`MovieTypes` | `[]string` | 文件类型过滤 |
| `HardwareAcceleration`/`HardwareAccelMode` | `bool`/`string` | 硬件加速 |
| `ControllerHost`/`FileHost` | `string` | 端口覆盖 |
| `NodeName`/`EnableLanDiscovery`/`DiscoveryPeers` | `string`/`*bool`/`[]string` | 集群配置 |
| `Users` | `[]User` | 多用户 |
| `AdminPassword` | `string` | 管理员密码（无编译回退，必须配置） |

## 开发命令

```bash
go run main.go                          # 后端（开发模式）
cd frontend && quasar dev               # 前端开发服务器（代理 /api → :10081）
go test ./...                            # Go 测试
cd frontend && yarn lint && yarn format  # 前端 lint + format
```

构建脚本：`ball_build.sh`（全量→`qapp/appQuaser.exe`）、`bfront_build.sh`（仅前端→`qapp/dist/`）、`bpc_build.sh`（Electron 打包）。

### 测试

```bash
go test ./...                                       # 全部 Go 测试
go test ./internal/service -run TestSearchBucket -v # 单个 Go 测试
cd frontend && npx vitest                            # 前端测试（vitest）
cd frontend && npx vitest run useTorrentDownload     # 单个前端测试
```

- Go 测试使用标准库 `testing` + `github.com/stretchr/testify/assert` 断言库（无 gomock 等 mock 框架）
- 前端 `package.json` 的 `test` 脚本为空占位，vitest 未在 scripts 注册，需手动 `npx vitest` 调用

## 后端架构

### 包结构

| 包 | 职责 |
|------|------|
| `internal/service/` | 核心业务：搜索引擎、文件扫描/操作、认证、集群、远程搜索、任务调度 |
| `internal/handler/` | HTTP handler 函数（每个 controller 文件一组相关路由处理） |
| `internal/router/` | 路由注册（单文件 `build_router.go`，平铺式） |
| `internal/middleware/` | Gin 中间件：认证、recovery、流 token 校验 |
| `internal/model/` | 数据模型：`FileItem`、`SearchParam`、`Setting`、`Author` 等 |
| `internal/sse/` | SSE 实时推送服务 |
| `internal/ws/` | WebSocket 聊天/在线状态/视频会议信令 |
| `internal/server/` | HTTP 服务创建与优雅关闭 |
| `internal/env/` | 运行时模式（prod/dev），通过 build tag 控制 |
| `pkg/utils/` | 工具函数：日志、加密、LRU 缓存、分页、路径、文件处理 |
| `pkg/types/` | 类型定义：`setting.go`、`transfer_task.go` |
| `pkg/consts/` | 常量 |

### 依赖注入

```
main.go
  ├─ NewSearchEngine()         → *searchEngineCore
  ├─ NewScanQueue(engine)      → *taskQueue
  ├─ NewSearchService(engine, settings, events, scanQueue) → *searchService
  ├─ InitService(engine, search)   → 包级 getter（内部用）
  └─ handler.InitApp(engine, search, settings) → handler 层 DI
```

**访问规则**：
- handler 层：`UseApp().search.FindById(id)`
- service 层内部：结构体字段 `s.engine.FindById(id)`
- 包级辅助函数：getter `GetEngine().FindById(id)`
- **禁止**直接引用全局单例

### 搜索流程

`UseApp().search.Page(param)` → `pageAsync` → `tryCache`（LRU 缓存，epoch 校验）→ `doSearch`（`atomic.Value` 无锁读索引快照 → goroutine pool 分发 bucket 并发搜索 → channel 收集合并 → 排序 → 聚合计算 → LRU 缓存 → 分页）。

**epoch 缓存失效机制**：
- `cacheEpoch`（`atomic.Int64`）在 `installIndex`/`syncIndex`/`installIndexSkipDisk`/`ClearCache` 时递增
- LRU 缓存的 `cachedResult` 结构体包含创建时的 epoch 值
- `tryCache` 校验 epoch：不匹配则删除缓存项并重新搜索
- `syncIndex`（全量扫描）会清空 LRU 缓存；`installIndexSkipDisk`（单文件操作）仅递增 epoch，但不清空 LRU（大多数缓存仍然有效）

### 全量扫描生命周期

`ScanAll()` → `FullScanInProgress`（`atomic.Bool` CAS 防止并发）→ `ClearCache` → 并发 `ScanDirs(dirList, types)` 收集 `bucketFile` → `rebuildWithBuckets(buckets)` → `buildIndexFromBuckets` → `installIndex`（`syncIndex`→递增 epoch+清 LRU→`saveIndexToCache` gob 持久化）→ SSE broadcast `scan_start` / `scan_complete` / `index_health`

单文件操作：`DeleteOnIndex`/`ReplaceFileOnIndex` → `installIndexSkipDisk`（只递增 epoch，不清 LRU，不写磁盘）→ `notifyFileChanged`（SSE 推送）

### Go 约定

- 日志：`utils.InfoFormat` / `utils.ErrorFormat`（封装 logrus，同时写 stdout + `gin.log`，5MB 超限时自动裁剪保留尾部 3MB）
- `main.go` 中 goroutine 必须 `defer utils.RecoverPanic()`
- 路径分隔符：`utils.PathSeparator`（非 `os.PathSeparator`）
- 文件存在：`utils.ExistsFiles(path)`
- 搜索缓存 epoch 机制：`cacheEpoch` 在 `installIndex` 时递增，缓存读写时校验
- 文件操作后通过 `s.notifyFileChanged(oldFile, updated, action)` 统一更新索引 + SSE
- 消息码使用中文文本：`"執行成功"` / `"執行失败"`（`pkg/utils/MessageCode.go`）
- 响应结构统一 `utils.Result`：`{"Code":200|400, "Message":"...", "Data":..., "EffectRows":0}`
- 支持 `allow` 打包：`golang.org/x/tools` 的 `allow` 指令可用于升级依赖
- 多处 `interface{}` 可替换为 `any`（Go 1.25），但代码中仍混用两种形式——遵循既有风格

## 路由注册

集中在 `internal/router/build_router.go` 单文件，**平铺式注册，无分组/版本化/OpenAPI**。两个入口：

- `BuildAPIRouter(sigChan)`（10081，认证）— 业务路由，含 `/api/close`、`/api/restart` 关机/重启路由（需 `AdminRole`）
- `BuildFileRouter()`（10082，streamToken 校验）— 文件流，复用 `buildStreamMiddleware` 注册 `/api/stream/*` 路径
  - `BuildFileRouter` 已注册 `StreamTokenAuth` 中间件（AES-256-GCM token 校验）

**两个 StreamTokenAuth 路径**：
- `:10082/api/stream/*`（`BuildFileRouter` 全局应用 `StreamTokenAuth`）
- `:10081/api/stream/*`（`BuildAPIRouter` 中 `AuthMiddleware` 跳过 `/api/stream/` 后，`StreamTokenAuth` 单独保护）

CORS 由 `buildCORSConfig()` 配置，生产环境支持 `ALLOWED_ORIGINS` 环境变量（逗号分隔），未设置时默认 `*`。`AllowCredentials` 未启用，不存在与 `AllowOrigins[*]` 的冲突。

## 中间件

`middleware/` 两文件：

| 文件 | 中间件 | 说明 |
|------|--------|------|
| `common.go` | `AuthMiddleware()` | Token 认证 + 集群 `X-Search-Gin-Remote` 校验 |
| `common.go` | `SlowRequestLogger()` | 开发环境慢请求日志（>5s） |
| `recovery.go` | `CustomRecovery()` | panic 恢复，返回 500 `{"error":"...","msg":"..."}` |

文件流 token 校验使用 `pkg/utils/stream_crypto.go` 的 AES-256-GCM 机制，由 `BuildFileRouter` 直接注册 `StreamTokenAuth()` 中间件（定义在已删除的 `sign_auth.go` 中，已内嵌到路由注册代码）。

通用中间件由 `buildCommonMiddleware()`（CORS + Recovery）统一装配。

## 实时通信

| 通道 | 后端 | 前端 | 用途 | 消息格式 |
|------|------|------|------|----------|
| SSE | `internal/sse/hub.go` | `composables/useSSE.ts` | 推送索引/文件变更 | `{Type, Data}`，已知事件 `index_update` |
| WebSocket | `internal/ws/hub.go` | `composables/useChatWs.ts` | 聊天/在线状态/视频会议信令 | `{type:"online"\|"chat"\|"system", ...}` |

- SSE：`EventSource` 连 `/api/events`，客户端超时 5min 自动清理，广播 channel 缓冲 100，前端指数退避重连（3s→30s 上限）
- WS：保留最近 100 条聊天历史，支持 `SendToUser(username, msg)` 定向推送。admin/super_admin 按用户名+IP 拆分为独立在线条目（区分不同设备）

## 设计决策（不可修改）

⚠️ 以下均为**有意设计**，审计报告可能将其误标为 P0/P1，但它们不是 bug。修改前必须向用户确认。

### 局域网信任模型

本项目是 **LAN 应用**，安全标准与公有云不同。局域网是信任域，与 Redis Cluster、Cassandra、Kafka 同理——这些主流集群系统同样不做节点间认证。

| 设计决策 | 原因 | 审计误报示例 |
|---------|------|------------|
| **反向心跳自动发现**（`TryVerifyAndAddPeer`） | 未知 IP 发 `X-Search-Gin-Remote: true` → 自动反向心跳验证 → 通过则加入集群。这是自举机制，不是漏洞。 | 曾被标 P0 "自动提权"，实际是 Redis Gossip 协议的简化版 |
| **`/api/lanPeers` 无认证** | 节点发现需要无认证访问——扫描方此时不知道目标节点，无法携带 token | 曾被标 P1 "暴露拓扑"，实际暴露的只是局域网内本就公开的 IP |
| **`/api/heartBeat` 无认证** | LAN 扫描探测存活用，返回的只是文件数量数字 | 同上 |
| **WebSocket `CheckOrigin` 返回 true** | 局域网场景，安全靠 token 校验（`?token=xxx`），不靠 origin。限制 origin 会阻止其他节点的 WebSocket 连接 | 曾被标 MEDIUM "生产环境绕过" |
| **SSE `/api/events` 无认证** | SSE 是只读推送，安全靠前端 token 校验 | 曾被标 MEDIUM "DoS" |
| **LRU Cache Get 不移到头部** | 注释明确说明：*"Get 不移到链表头部，读并发性能优于标准 LRU 实现"* | 曾被标"bug"，实际是性能优化 |

### 其他设计决策

- **无数据库**：所有数据存内存（`map` + `sync.Map` + `atomic.Value`），通过文件系统扫描填充。索引快照通过 `encoding/gob` 持久化到 `search_cache.gob`，启动时优先加载。这是简化设计，不是遗漏。
- **`forwardRequest` 用 `c.GetRawData()`**：Gin 内部缓存 body，无论中间件是否提前读取都能获取。不是 bug。
- **SSH/WS Hub 用 `atomic.Bool` 控制最多一个 goroutine**：panic 后不再递归重启，避免无限递归 → OOM。是安全设计。

## 代码风格

### 缩进

| 文件类型 | 缩进 | 工具 |
|---------|------|------|
| Go | Tab (`indent_size=4`) | `gofmt` |
| Vue/TS/JS/HTML/CSS/JSON/YAML | 2 空格 | Prettier |
| Shell/Bat/Makefile | Tab / CRLF | — |
| Markdown | 2 空格 | — |

### Go 后端

| 元素 | 规则 | 示例 |
|------|------|------|
| 包名 | 小写单字，同目录名 | `service`, `handler`, `utils` |
| 文件名 | snake_case | `auth_service.go` |
| 测试文件 | `xxx_test.go` | `auth_service_test.go` |
| 导出函数/类型 | PascalCase | `NewSearchEngine()` |
| 非导出函数/类型/变量 | camelCase | `pageAsync()`, `bucketCount` |
| 方法接收者 | 类型缩写 1-2 字符 | `se`, `s`, `m`, `q` |
| 构造函数 | `New*()` 返回指针 | `NewSearchEngine() → *searchEngineCore` |
| Handler 函数 | `{Method}{Resource}` | `PostMovies`, `HandleWebSocket` |
| 结构体字段 / 常量 | PascalCase | `PortNo`, `MovieType` |
| JSON tag | snake_case + omitempty | `json:"minSize,omitempty"` |

**导入**：三组空行分隔（标准库 → `search-gin/...` → 第三方），`import (...)` 分组。

**注释**：中文战略注释为主，导出类型/函数必须 `// 名称 说明`。段落分隔用 `// ── section ──`。

**错误与响应**：**禁止直接 `gin.H`**（仅 `middleware/recovery.go` 例外）。统一 `utils.Result`，序列化 `{"Code":200,"Message":"...","Data":...,"EffectRows":0}`。

**并发**：`atomic.Value` 快照读，`atomic.Int64/Bool` 计数器/标志，channel 信号量 + select 收集结果，errgroup 管理生命周期。每 goroutine 首行 `defer utils.RecoverPanic()`。

**DI**：`main.go` 显式构造 → `handler.InitApp()`，handler 用 `UseApp().xxx` 访问，**禁止直接引用 service 全局变量**。

**测试**：标准库 `testing` + `github.com/stretchr/testify/assert`。同包 `_test.go`，命名 `TestXxx`。

### 前端（Vue 3 + TS + Quasar 2）

技术栈：Quasar v2 + Vite，Vue Router hash 模式，Pinia + `pinia-plugin-persist`，路径别名 `@` → `frontend/src`，ESLint + Prettier。

| 元素 | 规则 | 示例 |
|------|------|------|
| 组件文件 | PascalCase `.vue` | `SearchPanel.vue` |
| TS 文件（非组件） | camelCase | `useSSE.ts`, `torrentAPI.ts` |
| 变量/函数/ref | camelCase | `isLoading`, `fetchSearch()` |
| 模块常量 | UPPER_SNAKE_CASE | `SSE_MAX_BACKOFF` |
| 类型/接口/类 | PascalCase | `SearchParams`, `FileModel` |
| 组合式函数 | `use<Name>` | `useSSE()`, `useCommonExec()` |
| Pinia | `use<Name>` / Options API | `useSystemProperty` |
| store 字段/方法 | PascalCase(状态) / camelCase(方法) | `FileSearchParam` / `setPage()` |
| event/hook | kebab-case | `@update:model-value` |

**Vue 约定**：100% `<script setup lang="ts">`（严禁 Options API），`defineProps`/`emit`/`expose` 免 import。`ref` 原始值、`reactive` 复杂状态、`computed` 派生、`watch` 副作用。`$q.notify()` 主 UI 反馈。Quasar 组件优先，`@event` 语法。`<style scoped lang="scss">`，hyphen-case 类名。

**SFC 顺序**：`<template>` → `<script setup>`（imports→props→state→computed→fn→lifecycle→expose）→ `<style scoped lang="scss">`。

**TypeScript 规范**：项目允许混合使用 JS 和 TS。**如果选择使用 TypeScript，则必须通过 vue-tsc 类型检查。**

- `interface` 用于数据结构，`type` 用于函数签名/联合类型，`class` 用于带方法的模型
- 导入 `src/` 优先；类型导入用 `import type`

| 禁止项 | 说明 |
|--------|------|
| `any` 类型 | 禁止 `as any`、`: any`、`ref<any>`、`<any[]>` |
| `@ts-ignore` / `@ts-nocheck` | 禁止绕过类型检查 |
| 未声明 interface | 复杂对象必须声明类型 |

| 必须项 | 说明 |
|--------|------|
| API 返回值 | 必须声明类型，不能用 `any` 接收 |
| 新增数据结构 | 必须在 `types/index.ts` 中声明 `interface` |
| 回调函数 | 必须声明参数和返回值类型 |
| 组件 props/emits | `defineProps` / `defineEmits` 必须声明类型 |
| 定时器 | 使用 `ReturnType<typeof setInterval> \| null` 而非 `any` |

**提交检查**：`cd frontend && npx vue-tsc --noEmit`，类型检查失败不允许合并。

**错误处理**：Axios 拦截器统一 401/403/500+。`try/catch` 逐请求提取 `err.response?.data?.message ?? err.response?.data?.Message`。`onErrorCaptured` 防白屏。

**Pinia**：Options API `defineStore('id', {state,getters,actions})`。`pinia-plugin-persist`（已停维）`paths` 白名单。直接访问 `systemProperty.theme` 无需 `storeToRefs`。

**代码位置**：`src/components/` 复用、`src/pages/<feature>/components/` 页面专属、`src/composables/` 组合式函数、`src/stores/`、`src/css/` 全局样式、`src/api/` 新 API 层。

## Docker 部署

- `Dockerfile`：三段式构建（node:20-alpine 前端 → golang:1.25-alpine 编译 → alpine:3.20 运行时）
- `docker-compose.yml`：暴露 10081/10082，挂载 `/media:ro` 和 `search-gin-data` 卷，默认 2GB 内存限制
- Linux 构建：`CGO_ENABLED=0 go build -tags=prod -ldflags="-s -w"`（不依赖 ffmpeg.exe 等 Windows 二进制，使用 alpine 包管理器的 `ffmpeg`）
- 容器运行无需 `-H=windowsgui` 链接器标志

## CI / 代码质量

GitHub Actions CI（`.github/workflows/ci.yml`）：
- `golangci-lint` 检查：启用 `govet`、`staticcheck`、`gosimple`、`errcheck`、`ineffassign`、`unused`、`unconvert`、`gosec`、`misspell`
- **禁用**（有意）：`gocritic`（过于主观）、`revive`（命名/风格规则）、`prealloc`（性能提示非 bug）
- **gosec 排除**（有意）：G115（整数转换安全）、G404（math/rand 用户代理旋转非加密场景安全）、G204（子进程由 `net.ParseIP` 校验输入）、G114（pprof 监听 dev 环境）

## Pre-commit Hooks

Husky + lint-staged（`.husky/pre-commit`）：
- `*.{js,ts,vue}`：`eslint --fix` + `prettier --write`
- `*.{scss,css}`：`prettier --write`

## VSCode LSP / gopls 注意事项

- `internal/env/prod_config.go` 和 `assets_prod.go` 有 `//go:build prod` tag，gopls 默认不加载
- 要检查这些文件需要配置 `"buildFlags": ["-tags=prod"]`
- 当前项目无 `.vscode/settings.json`
