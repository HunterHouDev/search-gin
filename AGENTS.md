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

## Docker 支持

`Dockerfile` 使用多阶段构建（golang:1.25-alpine + Alpine 3.20 runtime），包含系统 `ffmpeg`：

| 阶段 | 镜像 | 职责 |
|------|------|------|
| frontend | node:20-alpine | `yarn build` |
| builder | golang:1.25-alpine | `go build -tags=prod -ldflags="-s -w"` |
| runtime | alpine:3.20 | `apk add ffmpeg` + 运行 binary |

⚠️ Docker 环境注意：
- Linux 容器不使用 `ffmpeg.exe`（Windows PE），依赖 Alpine 官方 `ffmpeg` 包
- `docker-compose.yml` 中媒体目录以 `:ro`（只读）挂载
- 需挂载持久化卷保存 `setting.json` 和 `search_cache.gob`
- Dockerfile 中 `COPY ffmpeg.exe /app/` 在 Linux 下无效（Windows PE），但 `2>/dev/null || true` 不阻断构建

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
| `DeepSeekApiKey` | `string` | AI API key |
| `AdminPassword | `string` | 管理员密码（无编译回退，必须配置） |

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
- HTTP 错误：`utils.NewFailByMsg(msg)` 返回 `utils.Result`（`{MessageCode, Data, EffectRows}`），序列化为 `{"Code":400,"Message":"...","Data":null,"EffectRows":0}`（PascalCase，无 json tag）；成功用 `gin.H` 或 model
- 路径分隔符：`utils.PathSeparator`（非 `os.PathSeparator`）
- 文件存在：`utils.ExistsFiles(path)`
- 搜索缓存 epoch 机制：`cacheEpoch` 在 `installIndex` 时递增，缓存读写时校验
- 文件操作后通过 `s.notifyFileChanged(oldFile, updated, action)` 统一更新索引 + SSE
- `forwardRequest` 使用 `c.GetRawData()` 而非 `io.ReadAll(c.Request.Body)`（Gin 内部缓存 body，无论中间件是否提前读取）
- SSH/WS Hub 使用 `atomic.Bool` 控制最多一个 goroutine 运行 Run() 主循环，panic 后不再递归重启

## 关键文件

`internal/service/`：
- `service.go` — 构造函数 + 全局 getter
- `interfaces.go` — 接口定义
- `index_engine_manager.go` — `searchEngineCore` + `atomic.Value` 索引指针
- `index_engine_executor.go` — `Page()` / `pageAsync()` / `tryCache()` / `doSearch()`
- `index_engine_builder.go` — 索引构建
- `index_engine_cache.go` — 磁盘缓存（gob）
- `index_engine_bucket.go` — `bucketFile` 单目录桶 + `searchBucket()`
- `file_operations.go` — SetMovieType / AddTag / Rename / Move / Delete
- `file_scanner.go` — ScanAll / Walk
- `file_video_processor.go` — 转码/截图/合并（包级函数）
- `file_directory_cleaner.go` — 目录清理
- `node_discovery.go` — 集群节点管理 + LAN 发现（子网扫描/单机检测）
- `remote_search.go` — 跨节点搜索 + URL 填充（streamToken 方式）
- `remote_operation.go` — 跨节点文件操作转发（`c.GetRawData()` 读取 body）
- `task_scheduler.go` — 扫描任务队列 + 转码/剪辑/合并任务调度
- `task_service.go` — 转码/剪辑/合并任务创建
- `background_launch.go` — `InitSetting`/`StartBackgroundTasks`/`InitPeerManager`/`StartPprof`（main.go 调用的启动入口）
- `auth_service.go` — 认证（`admin` + `setting.json` `adminPassword`）+ `GetOSSetting()` 配置读取

`internal/handler/` — 17 个 controller + `handler.go`（注入入口，`InitApp`/`UseApp`）

## 路由注册

集中在 `internal/router/build_router.go` 单文件，**平铺式注册，无分组/版本化/OpenAPI**。两个入口：

- `BuildAPIRouter(sigChan)`（10081，认证）— 业务路由，含 `/api/close`、`/api/restart` 关机/重启路由（需 `AdminRole`）
- `BuildFileRouter()`（10082，streamToken 校验）— 文件流，复用 `buildStreamMiddleware` 注册 `/api/stream/*` 路径
  - `BuildFileRouter` 已注册 `StreamTokenAuth` 中间件（AES-256-GCM token 校验）

⚠️ 部分路由 HTTP 方法已于近期调整：
| 路由 | 原方法 | 现方法 |
|------|--------|--------|
| `/api/delete/:id` | GET | DELETE |
| `/api/setMovieType/:id/:movieType` | GET | POST |
| `/api/addFileTag/:id/:tag` | GET | POST |
| `/api/clearFileTag/:id/:tag` | GET | POST |

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

## 错误响应格式

**无统一 `{code, msg, data}` 封装**，约定如下：

- 失败：`utils.NewFailByMsg(msg)` 返回 `utils.Result` struct（`pkg/utils/Result.go`），序列化为 `{"Code":400,"Message":"...","Data":null,"EffectRows":0}`（PascalCase 字段名，无 json tag）
- 成功：直接 `gin.H`（如 `gin.H{"list": ..., "total": ...}`）或 model 结构体，混用
- Recovery 异常：`gin.H{"error":"...","msg":"..."}`（HTTP 500，`middleware/recovery.go`）
- streamToken 校验失败：`gin.H{"fail":true,"msg":"..."}`（`middleware/sign_auth.go`）

⚠️ 注意 `utils.Result` 与 `gin.H` 的字段命名风格不一致：前者 PascalCase（Go 默认），后者 snake_case（手写 key）。前端需分别处理。

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
- **`utils.Result` 与 `gin.H` 混用**：前者 PascalCase，后者 snake_case。历史原因，保持兼容。
- **`forwardRequest` 用 `c.GetRawData()`**：Gin 内部缓存 body，无论中间件是否提前读取都能获取。不是 bug。
- **SSH/WS Hub 用 `atomic.Bool` 控制最多一个 goroutine**：panic 后不再递归重启，避免无限递归 → OOM。是安全设计。

## 代码风格

### 一、通用约定

#### 1.1 缩进与格式化

| 文件类型 | 缩进 | 说明 |
|---------|------|------|
| Go (`.go`) | Tab（编辑器设 `indent_size=4`） | `gofmt` 格式化，`max_line_length=120` |
| Vue/TS/JS/HTML/CSS/JSON/YAML | 2 空格 | Prettier 格式化 |
| Shell (`.sh`) | Tab | `shell_variant = bash` |
| Makefile | Tab | — |
| Batch (`.bat`) | CRLF 换行 | — |
| Markdown | 2 空格 | `trim_trailing_whitespace = false` |

#### 1.2 通用规范

- 编码：UTF-8，LF 换行（`.bat` 用 CRLF）
- 文件末尾必须有空行；行尾无多余空白
- 配置结构 `Setting` 定义在 `pkg/types/setting.go`，通过 `GetOSSetting()` 全局访问

---

### 二、Go 后端规范

#### 2.1 命名

| 元素 | 规则 | 示例 |
|------|------|------|
| 包名 | 小写单词，与目录名一致 | `service`, `handler`, `utils`, `middleware` |
| 文件名 | snake_case | `auth_service.go`, `index_engine_executor.go` |
| Model 文件名 | PascalCase（历史遗留，保持原样） | `FileItem.go`, `SearchParam.go` |
| 测试文件 | `xxx_test.go` | `auth_service_test.go` |
| 导出函数/类型 | PascalCase | `NewSearchEngine()`, `GetOSSetting()` |
| 非导出函数/类型 | camelCase | `pageAsync()`, `doSearch()`, `searchEngineCore` |
| 局部变量 | camelCase | `searchParam`, `bucketCount` |
| 方法接收者 | 类型首字母缩写（1-2 字符） | `*searchEngineCore` → `se`, `*searchService` → `s` |
| 构造函数 | `New*()` 返回指针 | `NewSearchEngine() → *searchEngineCore` |
| 接口 | 定义在 `interfaces.go`，同包内定义并实现 | `IndexEngine`, `FileService`, `Settings`, `EventBus` |
| Handler 函数 | `{Method}{Resource}` | `PostMovies`, `GetDelete`, `HandleWebSocket` |
| 结构体字段 | PascalCase | `Id`, `Path`, `MovieType` |
| JSON tag | snake_case + omitempty | `json:"minSize,omitempty"` |
| 常量 | PascalCase | `PortNo`, `FilePortNo` |

#### 2.2 导入规范

三组用空行分隔：
```go
标准库

search-gin/internal/...  search-gin/pkg/...   （内部包）

github.com/...                                  （第三方）
```
使用分组导入（`import (...)`），极少单行导入。不使用空白别名。

#### 2.3 注释规范

- **战略注释（"为什么"）**大量使用，中文书写
- **战术注释（"是什么"）**尽量省略
- 文件内段落分隔用 `// ── section ──`、`// ─── sub-section ───`
- 设计决策注释（标记审计"假阳性"）：`// 设计说明：这不是 Bug——...`
- 导出类型/函数必须有 `// 名称 说明` 注释
- 无 `/* */` 块注释，全用 `//`
- API 摘要 / doc comment 用中文

```go
// IndexEngine 搜索引擎抽象，定义搜索、查找、索引管理等核心能力。
// 实现者：*searchEngineCore（通过 atomic.Value 持有 searchIndex 快照）
type IndexEngine interface { ... }

// PostMovies 电影文件搜索处理函数
// 接收搜索参数并调用搜索服务获取结果
func PostMovies(c *gin.Context) { ... }
```

#### 2.4 错误与响应

| 场景 | 格式 | 示例 |
|------|------|------|
| 业务成功 | `gin.H` / model 结构体 | `gin.H{"list": ..., "total": ...}` |
| 业务失败 | `utils.NewFailByMsg(msg)` | `{"Code":400,"Message":"...","Data":null,"EffectRows":0}` (PascalCase) |
| Panic 恢复 | `gin.H` | `{"error":"...","msg":"..."}` (HTTP 500) |
| streamToken 失败 | `gin.H{"fail":true,"msg":"..."}` | HTTP 401 |

- 大多数错误"记日志后吞掉"（返回零值），非穿透返回
- 错误检查靠 `err != nil`，不使用 `errors.Is()` / `errors.As()`

#### 2.5 并发

- `sync.Mutex` / `sync.RWMutex` 保护关键区
- `atomic.Value` 做无锁快照读（索引指针）
- `atomic.Int64` / `atomic.Int32` / `atomic.Bool` 做计数器、标志位
- channel 做信号量：`semaphore := make(chan struct{}, N)`
- buffered channel + select + context timeout 做结果收集
- 非阻塞唤醒：`select { case ch <- struct{}{}: default: }`
- errgroup 管理服务器 goroutine 生命周期（`golang.org/x/sync/errgroup`）

```go
// 每 goroutine 必须 defer RecoverPanic
go func() {
    defer utils.RecoverPanic()
    // ...
}()

// atomic.Value 快照读写
type searchEngineCore struct {
    indexDB atomic.Value     // *searchIndex
}

// atomic.Bool 控制单 goroutine（防递归启动）
if hubRunning.CompareAndSwap(false, true) {
    go func() {
        defer hubRunning.Store(false)
        // ...
    }()
}
```

#### 2.6 依赖注入模式

```go
// main.go 显式构造（禁止隐式依赖）
engine := service.NewSearchEngine()
search := service.NewSearchService(engine, settings, events, scanQueue)
handler.InitApp(engine, search, settings)

// handler 层：通过 UseApp() 获取注入的接口
func PostMovies(c *gin.Context) {
    UseApp().search.Page(searchParam)
}

// service 层内部：通过结构体字段访问
func (s *searchService) AddTag(id string, tag string) utils.Result {
    movie := s.engine.FindById(id)
}

// 包级辅助函数：通过 getter 访问
func GetEngine() *searchEngineCore { return globalEngine }
```

**禁止**在 handler 层直接引用 service 全局变量。

#### 2.7 文件组织

| 目录 | 内容 |
|------|------|
| `internal/service/` | 按领域分文件：引擎、文件操作、网络、认证、任务 |
| `internal/handler/` | 按领域命名：`auth_controller.go`, `search_controller.go` 等 |
| `internal/model/` | 数据结构定义（`FileItem.go`, `SearchParam.go` 等） |
| `internal/router/` | `build_router.go` 平铺注册，不分版本 |
| `internal/server/` | HTTP 服务启动与优雅关闭 |
| `internal/sse/` | SSE 实时推送 hub |
| `internal/ws/` | WebSocket hub |
| `internal/env/` | 环境配置（`IsProd`）、build tag 切换 |
| `middleware/` | 中间件工厂函数返回 `gin.HandlerFunc` |
| `pkg/utils/` | 每工具类型一个文件，PascalCase 文件名 |
| `pkg/types/` | 共享类型定义（`Setting`, `TransferTask`） |

#### 2.8 模板代码要点

```go
// 结构体定义
type foo struct {
    Name string
    age  int  // 非导出字段
}

// 构造函数
func NewFoo() *foo {
    return &foo{...}
}

// 方法接收者
func (f *foo) DoSomething() { ... }

// 包级变量分组声明
var (
    globalEngine *searchEngineCore
    globalSearch *searchService
)

// Getter 函数
func GetEngine() *searchEngineCore { return globalEngine }

// 锁模式
mu.Lock()
defer mu.Unlock()
```

#### 2.9 测试

- 纯标准库 `testing`，无 testify/gomock 等第三方 mock 框架
- 测试文件同包：`xxx_test.go`
- 命名：`TestXxx`（如 `TestSearchBucket`）
- 运行：`go test ./internal/service -run TestSearchBucket -v`

---

### 三、前端规范（Vue 3 + TypeScript + Quasar 2）

#### 3.1 命名

| 元素 | 规则 | 示例 |
|------|------|------|
| 组件文件名 | PascalCase `.vue` | `SearchPanel.vue`, `VideoPlayer.vue` |
| 非组件 TS/JS 文件 | camelCase | `useSSE.ts`, `torrentAPI.ts` |
| 变量、函数、ref | camelCase | `isLoading`, `fetchSearch()` |
| 模块级常量 | UPPER_SNAKE_CASE | `SSE_MAX_BACKOFF`, `FAB_DRAG_THRESHOLD` |
| 类型/接口 | PascalCase | `SearchParams`, `SSEEvent` |
| 类 | PascalCase，构造器函数 | `FileModel`, `RecordWrapper` |
| 组合式函数 | `use<Name>` | `useSSE()`, `useChatWs()`, `useCommonExec()` |
| Pinia store | `use<Name>` | `useSystemProperty`, `usePermissionStore` |
| store 状态字段 | PascalCase（因后端 API 字段名） | `FileSearchParam`, `SearchWords` |
| store 方法 | camelCase | `setPage()`, `syncSearchParam()` |
| 事件名 | kebab-case | `@refresh-done`, `@update:model-value` |

#### 3.2 代码格式化（Prettier + ESLint）

```json
// .prettierrc
{ "singleQuote": true, "semi": true }
```

- 单引号优先，必须加分号
- ESLint: `@typescript-eslint/recommended` + `vue3-essential` + prettier
- 编辑器遵循 `.editorconfig` 和 `.prettierrc`

#### 3.3 Vue SFC 结构顺序

每个 `.vue` 文件三个区块按此顺序：

```vue
<template>
  <!-- 1. 模板 — Quasar 组件优先，自闭合标签 -->
</template>

<script setup lang="ts">
// 2. 脚本 — 100% 组合式 API
// 顺序：imports → props/emits → 状态 → computed → 函数 → 生命周期 → expose
import { ref, computed, onMounted } from 'vue';
const props = defineProps({ ... });
const emit = defineEmits(['...']);
const loading = ref(false);
const total = computed(() => ...);
async function fetchData() { ... }
onMounted(() => { fetchData(); });
defineExpose({ ... });
</script>

<style scoped lang="scss">
// 3. 样式 — scoped + SCSS，hyphen-case 类名
.search-panel { ... }
</style>
```

#### 3.4 Template 风格

```html
<!-- Quasar 组件优先 -->
<q-btn flat dense round icon="close" @click="onClose" />
<q-img :src="item.PngUrl" fit="cover" />

<!-- v-for 需要 :key -->
<div v-for="item in items" :key="item.Id">{{ item.Title }}</div>

<!-- 条件渲染 -->
<div v-if="loading">加载中...</div>
<div v-else-if="items.length > 0">...</div>
<div v-else>空状态</div>

<!-- 计算属性绑定动态 class -->
<div :class="{ active: isActive, disabled: !enabled }">
```

#### 3.5 Script Setup 风格（**严禁 Options API**）

```typescript
// ref() 用于基础类型；reactive() 用于复杂视图状态对象；computed() 用于派生状态
import { ref, reactive, computed, watch } from 'vue';
import { $q } from 'quasar';

// Props 带类型和默认值
const props = defineProps({
  visible: { type: Boolean, default: false },
});

// 事件
const emit = defineEmits(['play', 'close']);

// 异步函数：try/catch/finally 处理 loading
async function fetchData() {
  loading.value = true;
  try {
    const data = await api.search(params);
    Object.assign(results, data);
  } catch (e) {
    console.error('Failed:', e);
  } finally {
    loading.value = false;
  }
}
```

#### 3.6 TypeScript 约定

- 接口优先于类型别名：`interface SearchParams { ... }`
- 类型别名用于函数签名和联合类型：`type Handler = (msg: Msg) => void`
- 类用于带方法的模型：`class FileModel { fromObject() { ... } isEmpty() { ... } }`
- 导入路径别名：`src/`（优先）、`components/`、逐层相对 `../../`
- 类型导入显式写：`import type { QVueGlobals } from 'quasar'`
- API 错误处理双重兼容（PascalCase + camelCase）：`res.data?.Code ?? res.data?.code`

#### 3.7 Pinia Store 风格

```typescript
export const useMyStore = defineStore({
  id: 'myStore',
  persist: {
    enabled: true,
    strategies: [{ storage: localStorage, paths: ['field1', 'field2'] }],
  },
  state: () => ({
    count: 0,
    items: [] as Array<string>,
    config: {} as Record<string, string>,
  }),
  getters: {
    doubled(): number { return this.count * 2; },
  },
  actions: {
    increment() { this.count++; },
  },
});
```

- 持久化用 `pinia-plugin-persist`（注意已停维，用 `paths` 白名单）
- Options API 风格（非 Setup Store）
- 状态直接访问：`systemProperty.theme`（无需 `storeToRefs`）

#### 3.8 Composables 风格

```typescript
// composables/useXxx.ts
import { ref, onMounted, onUnmounted } from 'vue';

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const isConnected = ref(false);
  // 内部状态在函数作用域，不暴露给外部
  let backoffMs = 3_000;

  const connect = () => { /* ... */ };
  const disconnect = () => { /* ... */ };

  onMounted(() => connect());
  onUnmounted(() => disconnect());

  return { isConnected, connect, disconnect };
}
```

#### 3.9 CSS 风格

- `<style scoped lang="scss">` — 组件作用域
- hyphen-case 类名：`.search-panel`, `.search-card-title`, `.advanced-filter-panel`
- scss 嵌套 `&` 引用伪类/伪元素/选择器：`&:hover`, `&::after`, `:deep(.q-item)`
- 响应式用 `@media` 在文件内部（非外部 CSS 文件）
- CSS 变量在 `:root` 或 `frontend/src/css/tokens.css` 定义

```scss
.search-panel {
  background: rgba(9, 9, 22, 0.92);
  backdrop-filter: blur(32px);
  display: flex;
  flex-direction: column;

  &::after { content: ''; position: absolute; inset: 0; }
  &:hover { border-color: rgba(139, 92, 246, 0.4); }
}

@media (max-width: 599px) {
  .search-card { width: calc(50% - 8px) !important; }
}
```

#### 3.10 错误处理

- $q.notify() 为主要 UI 反馈方式：`type: 'positive' | 'negative' | 'warning' | 'info'`
- Axios 拦截器统一处理（401 → 跳登录，403 → 无权限提示，500+ → 重试提示）
- 逐请求 `try/catch` 提取 `err.response?.data?.message ?? err.response?.data?.Message`
- `onErrorCaptured` 在 `App.vue` 阻止白屏
- 非关键操作用 `.catch(() => {})` 静默处理

#### 3.11 API 层

```typescript
// API 函数放在 components/api/（旧）或 src/api/（新）
import { api } from 'boot/axios';

export async function SearchAPI(params: any) {
  const { data } = await api.post('/api/movieList', params);
  return data;
}
```

- 基础地址 `http://localhost:10081`
- 文件流基础 URL 通过 `setFileBaseUrl()` 指向 `:10082`

#### 3.12 代码位置

| 目录 | 内容 |
|------|------|
| `src/components/` | 全局可复用组件 |
| `src/components/api/` | API 函数（向前兼容，新代码放 `src/api/`） |
| `src/pages/<feature>/` | 页面级组件 |
| `src/pages/<feature>/components/` | 页面专属子组件 |
| `src/composables/` | 封装实时通信和复杂逻辑 |
| `src/stores/` | Pinia store |
| `src/css/` | 全局样式（`app.scss`, `components.css`, `tokens.css`） |
| `src/api/` | 新 API 层目标位置 |
| `src/types/` | TypeScript 类型定义 |
| `src/utils/` | 跨工具函数 |

#### 3.13 Build / Config

- `quasar.config.js`（CommonJS，`require()`）
- ESLint: `@typescript-eslint/recommended` + `vue3-essential` + prettier
- Prettier: 单引号、分号（`frontend/.prettierrc`）
- Vite: `quasar.config.js` 内嵌
- 路径别名：`@` → `frontend/src`（已定义但较少用，优先用 `src/`）
- TypeScript 4.5+，target ES2020，moduleResolution: Node
