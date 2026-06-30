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

```
/api/login, /login, /, /index.html, /api/ws, /api/events,
/api/lanPeers, /api/heartBeat, /api/authorImage/,
/css/, /js/, /assets/, /icons/, /favicon.ico
```

⚠️ skip path 检查必须在 `X-Search-Gin-Remote` 校验**之前**，否则跨节点验证会形成递归死锁。

## 前端构建 / 嵌入

```
cd frontend && yarn build    # 产物 → frontend/dist/spa/
cp -R dist/spa/* ../dist/    # 复制到根 dist/
go build -tags=prod          # embed dist/、ffmpeg.exe、ffplay.exe、setting.json
```

- 开发模式 `go run main.go` 直接从磁盘读 `./dist/`
- 生产二进制启动时解压嵌入资源到工作目录

## 认证

- 管理员用户 `admin`，密码必须配 `setting.json` 的 `adminPassword`（无编译回退）
- Token 存内存 map `tokenStore map[string]TokenInfo` 受 `sync.RWMutex` 保护（`auth_service.go`），每个 token 创建时注册 `time.AfterFunc` 到期自动删除，无定时轮询
- `Authorization: Bearer <token>` 或 WebSocket `?token=`
- 集群节点间用 `X-Search-Gin-Remote: true` header 跳过 Token 认证，来源 IP 必须在 peers 列表中（`middleware/common.go` + `node_discovery.go`）
- `requireAdmin()` 检查 role 是否为 `AdminRole`，兼容旧 token（role 为空时放行）
- **注意事项**：
  - WebSocket 连接使用 query token（`/api/ws?token=xxx`），skip path 已跳过 AuthMiddleware 故需手动调用 `ValidateTokenWithInfo`
  - `:10082` 文件流端口使用 `StreamTokenAuth` 中间件（AES-256-GCM 加密 streamToken），不依赖 session map，跨节点可互相解密

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

## 无数据库

所有数据存内存（`map` + `sync.Map` + `atomic.Value`），通过文件系统扫描填充。索引快照通过 `encoding/gob` 持久化到 `search_cache.gob`，启动时优先加载。

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
bash ball_build.sh                       # 完整生产构建
```

### 构建脚本三件套

| 脚本 | 用途 | 产物 |
|------|------|------|
| `ball_build.sh` | 完整生产构建（前端 + Go `-tags=prod` 嵌入） | `qapp/appQuaser.exe` |
| `bfront_build.sh` | 仅前端构建（不编译 Go） | `qapp/dist/` |
| `bpc_build.sh` | Electron 桌面应用打包（前端 + Go + Electron `yarn topc`） | Electron 安装包 |



### 测试

```bash
go test ./...                                       # 全部 Go 测试
go test ./internal/service -run TestSearchBucket -v # 单个 Go 测试
cd frontend && npx vitest                            # 前端测试（vitest）
cd frontend && npx vitest run useTorrentDownload     # 单个前端测试
```

- Go 测试仅用标准库 `testing`，未引入 testify/gomock 等 mock 框架
- 前端 `package.json` 的 `test` 脚本为空占位，vitest 未在 scripts 注册，需手动 `npx vitest` 调用

## 后端架构

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

### 接口（`interfaces.go`）

| 接口 | 主要方法 | 实现者 |
|------|---------|--------|
| `IndexEngine` | `Page`, `FindById`, `DeleteOnIndex`, `ReplaceFileOnIndex`, `GetTypeMenu` | `*searchEngineCore` |
| `FileService` | `SetMovieType`, `AddTag`, `Rename`, `Move`, `Delete`, `ScanAll` | `*searchService` |
| `Settings` | `Get`, `Set`, `Flush` | `settingsAdapter` |
| `EventBus` | `Broadcast` | `sseAdapter` |

### 搜索流程

```
UseApp().search.Page(param)
  → pageAsync(param)
    → loadIndex()               // atomic.Value.Load（无锁）
    → OnlyRepeat? → returnRepeatSearch(snap)
    → tryCache(param)           // LRU + epoch 校验
    → doSearch(snap, param)     // 分发 bucket 并发搜索
      → collectResults()        // channel 合并 + 超时
      → SortFileItems() → 写入缓存 → GetPageOfFiles()
```

### Go 约定

- Go：`gofmt`（tabs，`.editorconfig` 设 `indent_size=4`）；非 Go：2 空格缩进
- 日志：`utils.InfoFormat` / `utils.ErrorFormat`（封装 logrus，同时写 stdout + `gin.log`）
- `main.go` 中 goroutine 必须 `defer utils.RecoverPanic()`
- HTTP 响应：**禁止直接使用 `gin.H`**（仅 `middleware/recovery.go` panic 恢复和 `home_controller.go` HTML 模板除外）。统一用 `utils.Result`：失败用 `utils.NewFailByMsg(msg)`，成功用 `utils.NewSuccess()` / `utils.NewSuccessByMsg(msg)` + `res.Data = ...`。序列化格式 `{"Code":200,"Message":"...","Data":...,"EffectRows":0}`（PascalCase，无 json tag）
- 路径分隔符：`utils.PathSeparator`（非 `os.PathSeparator`）
- 文件存在：`utils.ExistsFiles(path)`
- 搜索缓存 epoch 机制：`cacheEpoch` 在 `installIndex` 时递增，缓存读写时校验
- 文件操作后通过 `s.notifyFileChanged(oldFile, updated, action)` 统一更新索引 + SSE
- `forwardRequest` 使用 `c.GetRawData()` 而非 `io.ReadAll(c.Request.Body)`（Gin 内部缓存 body，无论中间件是否提前读取）
- SSH/WS Hub 使用 `atomic.Bool` 控制最多一个 goroutine 运行 Run() 主循环，panic 后不再递归重启

## 路由注册

集中在 `internal/router/build_router.go` 单文件，**平铺式注册，无分组/版本化/OpenAPI**。两个入口：

- `BuildAPIRouter(sigChan)`（10081，认证）— 业务路由，含 `/api/close`、`/api/restart` 关机/重启路由（需 `AdminRole`）
- `BuildFileRouter()`（10082，streamToken 校验）— 文件流，复用 `buildStreamMiddleware` 注册 `/api/stream/*` 路径
  - `BuildFileRouter` 已注册 `StreamTokenAuth` 中间件（AES-256-GCM token 校验）

CORS 由 `buildCORSConfig()` 配置，生产环境支持 `ALLOWED_ORIGINS` 环境变量（逗号分隔），未设置时默认 `*`。`AllowCredentials` 未启用，不存在与 `AllowOrigins[*]` 的冲突。

## 中间件

`middleware/` 三文件：

| 文件 | 中间件 | 说明 |
|------|--------|------|
| `common.go` | `AuthMiddleware()` | Token 认证 + 集群 `X-Search-Gin-Remote` 校验 |
| `common.go` | `SlowRequestLogger()` | 开发环境慢请求日志（>5s） |
| `recovery.go` | `CustomRecovery()` | panic 恢复，返回 500 `{"error":"...","msg":"..."}` |
| `sign_auth.go` | `SignAuthMiddleware()` | 签名 URL 校验（**未注册** — 集群模式下不可用） |
| `sign_auth.go` | `StreamTokenAuth()` | 文件流 streamToken 校验（**已注册** — `BuildFileRouter` 使用） |

通用中间件由 `buildCommonMiddleware()`（CORS + Recovery）统一装配。

## 实时通信

| 通道 | 后端 | 前端 | 用途 | 消息格式 |
|------|------|------|------|----------|
| SSE | `internal/sse/hub.go` | `composables/useSSE.ts` | 推送索引/文件变更 | `{Type, Data}`，已知事件 `index_update` |
| WebSocket | `internal/ws/hub.go` | `composables/useChatWs.ts` | 聊天/在线状态/视频会议信令 | `{type:"online"\|"chat"\|"system", ...}` |

- SSE：`EventSource` 连 `/api/events`，客户端超时 5min 自动清理，广播 channel 缓冲 100，前端指数退避重连（3s→30s 上限）。Hub 使用 `atomic.Bool hubRunning` 防递归启动
- WS：保留最近 100 条聊天历史，支持 `SendToUser(username, msg)` 定向推送。admin/super_admin 按用户名+IP 拆分为独立在线条目（区分不同设备）。Hub 使用 `atomic.Bool hubRunning` 防递归启动

## 前端约定

- Quasar v2 + Vite，Vue Router 默认 hash 模式（Electron 下切换为 history 模式，见 `frontend/src/router/index.ts`）；已启用插件：Notify、AppFullscreen、Dialog
- Pinia + `pinia-plugin-persist`（2022 年起停维，参见 `package.json` 注释）实现 localStorage 持久化
  - stores（`frontend/src/stores/`）：`index.ts`、`System.ts`（原名 `app.ts`/`player.ts`/`search.ts` 已合并）
- 路径别名：`@` → `frontend/src`，`components` → `frontend/src/components`
- Prettier：单引号、分号；ESLint：`@typescript-eslint/recommended` + `vue3-essential` + prettier
- TypeScript 4.5+，target ES2020，moduleResolution: Node
- 前端 API 基础地址 `http://localhost:10081`（`frontend/src/boot/axios.ts`），文件流通过 `setFileBaseUrl` 指向 `:10082`
- `composables/`：Vue 组合式函数，封装实时通信与复杂逻辑——`useSSE.ts`（SSE）、`useChatWs.ts`（WebSocket 聊天）、`useTorrentDownload.ts`（磁力链下载）、`useVideoConference.ts`（WebRTC 视频会议）、`useSortOptions.ts`、`useBreakpoint.ts`、`useCommonExec.ts`

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
| **`GET + DELETE` 双路由**（`/api/delete/:id`） | 向后兼容期故意保留，非重复注册 | 曾被标"假阳性"，实际是设计决策 |
| **LRU Cache Get 不移到头部** | 注释明确说明：*"Get 不移到链表头部，读并发性能优于标准 LRU 实现"* | 曾被标"bug"，实际是性能优化 |

### 认证与安全边界

| 边界 | 机制 | 说明 |
|------|------|------|
| 用户 → API | Bearer Token | `auth_service.go` 签发，4h 有效期 |
| 浏览器 → 文件流 | streamToken（AES-256-GCM） | `FillURLs` 生成，图片 5min / 视频 4h 过期 |
| 节点 → 节点 | `X-Search-Gin-Remote` header + IP 白名单 | 已知 peer IP 直接放行，未知 IP 反向心跳验证 |
| 管理操作 | `requireAdmin(c)` | 检查 role 是否为 `super_admin` |

### 其他设计决策

- **无数据库**：所有数据存内存，通过文件系统扫描填充。索引快照用 gob 持久化到 `search_cache.gob`。这是简化设计，不是遗漏。
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

```go
// ✅ 正确
res := utils.NewSuccess()
res.Data = gin.H{"list": items, "total": total}
c.JSON(http.StatusOK, res)

// ❌ 禁止
c.JSON(http.StatusOK, gin.H{"list": items, "total": total})
c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "参数错误"})
```

**并发**：`atomic.Value` 快照读，`atomic.Int64/Bool` 计数器/标志，channel 信号量 + select 收集结果，errgroup 管理生命周期。每 goroutine 首行 `defer utils.RecoverPanic()`。

**DI**：`main.go` 显式构造 → `handler.InitApp()`，handler 用 `UseApp().xxx` 访问，**禁止直接引用 service 全局变量**。

**测试**：仅标准库 `testing`，无 testify/gomock。同包 `_test.go`，命名 `TestXxx`。

### 前端（Vue 3 + TS + Quasar 2）

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

**TS 约定**：`interface` 数据、`type` 函数签名/联合、`class` 带方法模型。导入 `src/` 优先。类型导入 `import type`。API 双重兼容 `res.data?.Code ?? res.data?.code`。

**错误处理**：Axios 拦截器统一 401/403/500+。`try/catch` 逐请求提取 `err.response?.data?.message ?? err.response?.data?.Message`。`onErrorCaptured` 防白屏。

**Pinia**：Options API `defineStore('id', {state,getters,actions})`。`pinia-plugin-persist`（已停维）`paths` 白名单。直接访问 `systemProperty.theme` 无需 `storeToRefs`。

**代码位置**：`src/components/` 复用、`src/pages/<feature>/components/` 页面专属、`src/composables/` 组合式函数、`src/stores/`、`src/css/` 全局样式、`src/api/` 新 API 层。
