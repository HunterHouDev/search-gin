# AGENTS.md — search-gin

## Build tag 系统

`-tags=prod` 控制两件事：**资源嵌入**（`assets_dev.go` 无 embed vs `assets_prod.go` `//go:embed dist ffmpeg.exe ffplay.exe setting.json`）和**运行时模式**（`internal/env/config.go` `IsProd=false` 默认 vs `internal/env/prod_config.go` `init()` 设 `IsProd=true`）。生产模式：Gin ReleaseMode、禁用 pprof、日志级别 `ErrorLevel`。开发模式 `go run main.go` 默认。

## 端口

| 端口 | 路由 | 认证 |
|------|------|------|
| `:10081` | `BuildAPIRouter()` API + 前端 | 需要 |
| `:10082` | `BuildFileRouter()` 文件/图片/视频流 | 无需 |

端口定义在 `internal/service/index_param.go`，可通过 `setting.json` 的 `ControllerHost`/`FileHost` 覆盖。

## 认证 skip paths（`middleware/common.go`）

```
/api/login, /login, /, /index.html, /api/ws, /api/events,
/api/lanPeers, /api/heartBeat, /api/authorImage/,
/css/, /js/, /assets/, /icons/, /favicon.ico
```

⚠️ skip path 检查必须在 `X-Search-Gin-Remote` 校验**之前**，否则 `verifyPeer` 反向心跳会形成递归死锁。

## 前端构建 / 嵌入

```
cd frontend && yarn build    # 产物 → frontend/dist/spa/
cp -R dist/spa/* ../dist/    # 复制到根 dist/
go build -tags=prod          # embed dist/、ffmpeg.exe、ffplay.exe、setting.json
```

- 开发模式 `go run main.go` 直接从磁盘读 `./dist/`
- 生产二进制启动时解压嵌入资源到工作目录

## 认证

- 管理员硬编码 `admin`/`qwer`（`internal/service/auth_service.go`）
- Token 存内存 map，`Authorization: Bearer <token>` 或 WebSocket `?token=`
- 集群节点间用 `X-Search-Gin-Remote: true` header 跳过 Token 认证，来源 IP 必须在 peers 列表中或通过反向心跳自动加入（`middleware/common.go` + `node_discovery.go`）

## 多节点集群

- 节点管理：`background_launch.go:InitPeerManager()` 加载 `discoveryPeers`（节点发现逻辑在 `node_discovery.go`），支持运行时增删
- 反向心跳自动发现：首次收到未知 IP 的集群请求时，反向 GET `/api/heartBeat` 验证，通过则自动加入并持久化到 `setting.json` → `node_discovery.go:TryVerifyAndAddPeer()`
- 跨节点搜索：`remote_search.go` 并发请求所有在线节点（最多 5 并发），去重策略 `Code+Size` 优先，`Name+Size` 兜底
- 文件流端口统一走 `:10082`
- 配置：`{"enableLanDiscovery": true, "discoveryPeers": ["192.168.1.102:10081"]}`

## 平台假设

- **Windows 为主**：`ffmpeg.exe`/`ffplay.exe`、`cmd /C start` 打开文件夹、`-H=windowsgui` 链接器
- `fixOnWin.go` / `fixOnNotWin.go`（build tag `windows`）隐藏子进程控制台

## 无数据库

所有数据存内存（`map` + `sync.Map` + `atomic.Value`），通过文件系统扫描填充。索引快照通过 `encoding/gob` 持久化到 `search_cache.gob`，启动时优先加载。

## Go 模块

模块名：`search-gin`。所有内部导入使用 `search-gin/internal/...` 和 `search-gin/pkg/...`。Go 版本要求：1.25.0+（见 `go.mod`）。注意 `pkg/utils/` 导入 `internal/env`——这是本仓库的设计。

## 配置结构

配置 `Setting` struct 定义在 `pkg/types/setting.go`（非 `internal/service/`），通过 `GetOSSetting()` 全局访问。关键 key：`Dirs`（扫描目录）、`VideoTypes`/`ImageTypes`/`DocsTypes`/`MovieTypes`（文件类型）、`HardwareAcceleration`/`HardwareAccelMode`（硬件加速）、`ControllerHost`/`FileHost`（端口覆盖）、`NodeName`/`EnableLanDiscovery`/`DiscoveryPeers`（集群）、`Users`（多用户）、`DeepSeekApiKey`（AI）。

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
- `node_discovery.go` — 集群节点管理 + 反向心跳
- `remote_search.go` — 跨节点搜索 + URL 填充
- `remote_operation.go` — 跨节点文件操作转发
- `task_scheduler.go` — 扫描任务队列 + 转码/剪辑/合并任务调度
- `task_service.go` — 转码/剪辑/合并任务创建
- `background_launch.go` — `InitSetting`/`StartBackgroundTasks`/`InitPeerManager`/`StartPprof`（main.go 调用的启动入口）
- `auth_service.go` — 认证（`admin`/`qwer`）+ `GetOSSetting()` 配置读取

`internal/handler/` — 17 个 controller + `handler.go`（注入入口，`InitApp`/`UseApp`）

## 路由注册

集中在 `internal/router/build_router.go` 单文件，**平铺式注册，无分组/版本化/OpenAPI**。两个入口：

- `BuildAPIRouter(sigChan)`（10081，认证）— 业务路由，含 `/api/close`、`/api/restart` 关机/重启路由（需 `AdminRole`）
- `BuildFileRouter()`（10082，无认证）— 文件流，复用 `buildStreamMiddleware` 注册 `/api/stream/*` 路径

CORS 由 `buildCORSConfig()` 配置，生产环境支持 `ALLOWED_ORIGINS` 环境变量（逗号分隔），未设置时默认 `*`。

## 中间件

`middleware/` 三文件：

| 文件 | 中间件 | 说明 |
|------|--------|------|
| `common.go` | `AuthMiddleware()` | Token 认证 + 集群 `X-Search-Gin-Remote` 校验 |
| `common.go` | `SlowRequestLogger()` | 开发环境慢请求日志（>5s） |
| `recovery.go` | `CustomRecovery()` | panic 恢复，返回 500 `{"error":"...","msg":"..."}` |
| `sign_auth.go` | `SignAuthMiddleware()` | 签名 URL 校验（防未授权文件流） |

⚠️ **`SignAuthMiddleware` 当前未启用**——`BuildFileRouter` 注释说明签名对多节点集群不可用。新增文件流功能时不要误以为签名校验已生效。

通用中间件由 `buildCommonMiddleware()`（CORS + Recovery）统一装配。

## 实时通信

| 通道 | 后端 | 前端 | 用途 | 消息格式 |
|------|------|------|------|----------|
| SSE | `internal/sse/hub.go` | `composables/useSSE.ts` | 推送索引/文件变更 | `{Type, Data}`，已知事件 `index_update` |
| WebSocket | `internal/ws/hub.go` | `composables/useChatWs.ts` | 聊天/在线状态/视频会议信令 | `{type:"online"\|"chat"\|"system", ...}` |

- SSE：`EventSource` 连 `/api/events`，客户端超时 5min 自动清理，广播 channel 缓冲 100，前端指数退避重连（3s→30s 上限）
- WS：保留最近 100 条聊天历史，支持 `SendToUser(username, msg)` 定向推送，按用户名聚合在线设备

## 错误响应格式

**无统一 `{code, msg, data}` 封装**，约定如下：

- 失败：`utils.NewFailByMsg(msg)` 返回 `utils.Result` struct（`pkg/utils/Result.go`），序列化为 `{"Code":400,"Message":"...","Data":null,"EffectRows":0}`（PascalCase 字段名，无 json tag）
- 成功：直接 `gin.H`（如 `gin.H{"list": ..., "total": ...}`）或 model 结构体，混用
- Recovery 异常：`gin.H{"error":"...","msg":"..."}`（HTTP 500，`middleware/recovery.go`）
- 签名失败：`gin.H{"fail":true,"msg":"..."}`（`middleware/sign_auth.go`，注意此中间件当前未启用）

⚠️ 注意 `utils.Result` 与 `gin.H` 的字段命名风格不一致：前者 PascalCase（Go 默认），后者 snake_case（手写 key）。前端需分别处理。

## 前端约定

- Quasar v2 + Vite，Vue Router 默认 hash 模式（Electron 下切换为 history 模式，见 `frontend/src/router/index.ts`）；已启用插件：Notify、AppFullscreen、Dialog
- Pinia + `pinia-plugin-persist` 实现 localStorage 持久化
  - stores（`frontend/src/stores/`）：`app.ts`、`player.ts`、`search.ts`、`System.ts`
- 路径别名：`@` → `frontend/src`，`components` → `frontend/src/components`
- Prettier：单引号、分号；ESLint：`@typescript-eslint/recommended` + `vue3-essential` + prettier
- TypeScript 4.5+，target ES2020，moduleResolution: Node
- 前端 API 基础地址 `http://localhost:10081`（`frontend/src/boot/axios.ts`），文件流通过 `setFileBaseUrl` 指向 `:10082`
- `composables/`：Vue 组合式函数，封装实时通信与复杂逻辑——`useSSE.ts`（SSE）、`useChatWs.ts`（WebSocket 聊天）、`useTorrentDownload.ts`（磁力链下载）、`useVideoConference.ts`（WebRTC 视频会议）、`useSortOptions.ts`、`useBreakpoint.ts`、`useCommonExec.ts`
