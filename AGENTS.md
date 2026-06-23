# AGENTS.md — search-gin

## Build tag 系统（双重作用）

`prod` 构建标签同时控制两件事：

- **资源嵌入**：`assets_dev.go`（默认，不嵌入） vs `assets_prod.go`（`//go:embed dist ffmpeg.exe ffplay.exe setting.json`）
- **运行时模式**：`internal/env/config.go`（默认 `IsProd=false`） vs `prod_config.go`（`IsProd=true`），控制 Gin 运行模式、pprof（仅开发环境监听 `:6060`）、CORS 配置及日志级别。

默认 `go run main.go` = 开发模式。添加 `-tags=prod` 编译生产环境二进制。

## 端口

| 端口 | 用途 | 说明 |
|------|------|------|
| `:10081` | API + 前端 | 业务路由，注册在 `BuildAPIRouter()`，需要认证 |
| `:10082` | 文件/图片/视频流 | 注册在 `BuildFileRouter()`，无需认证 |
| `:6060` | pprof（仅开发环境） | 开发调试用 |

端口在 `internal/service/index_param.go` 硬编码（`PortNo=:10081`，`FilePortNo=:10082`）。

## 前端构建 / 嵌入流程

1. `cd frontend && yarn build` → 产物输出到 `frontend/dist/spa/`
2. 构建脚本将 `frontend/dist/spa/*` 复制到 `./dist/`
3. `go build -tags=prod` 嵌入 `./dist/` 并在启动时解压到当前工作目录

`go run main.go`（开发模式）不嵌入资源，直接从磁盘读取 `./dist/`。开发时若 `dist/` 未更新，需先重新构建前端。

## Go 模块与导入路径

模块名：`search-gin`。所有内部导入使用 `search-gin/internal/...` 和 `search-gin/pkg/...`。

注意：`pkg/utils/` 导入 `internal/env`——这是本仓库的设计。

## 无数据库

所有数据存储在内存中（Go struct + `sync.Map`），通过文件系统扫描填充。`FileItem` 结构体中历史遗留的 `xorm` 标签已清理。

## 认证

- 硬编码管理员账号：`admin` / `qwer`（`internal/service/auth_service.go`）
- Token 存储在内存中（`TokenStore` map），通过 `Authorization: Bearer <token>` 发送
- WebSocket 使用 `?token=` 查询参数传递（无法设置自定义 Header）
- **集群节点间转发**使用 `X-Search-Gin-Remote: true` header 跳过 Token 认证，但来源 IP 必须为集群内已知 peer（`middleware/common.go`），详见下方多节点集群认证机制
- 中间件跳过认证的路径（API 路由 10081）：`/api/login`、`/login`、`/`、`/index.html`、`/api/ws`、`/api/lanPeers`、`/api/heartBeat`
  - ⚠️ skip path 检查必须在 `X-Search-Gin-Remote` 校验**之前**，否则 `verifyPeer` 反向心跳请求会形成递归死锁
- 文件流路由统一走端口 10082（`BuildFileRouter`，无认证），前端通过 `setFileBaseUrl` 自动指向 `:10082`
- `StreamUrl`（视频）→ `GetFileByPathUseEncode/:path`，`PngUrl`/`JpgUrl`（缩略图）→ `png/:id`/`jpg/:id`
- 前端 API 基础地址为 `http://localhost:10081`（`frontend/src/boot/axios.ts:18`）

## 多节点集群

支持局域网多节点协作：

| 功能 | 文件 | 说明 |
|------|------|------|
| **节点管理** | `internal/service/lan_discovery.go` | `InitPeerManager()` 从 `discoveryPeers` 加载手动节点，支持运行时动态增删 |
| **集群内认证** | `middleware/common.go` + `lan_discovery.go` | 跨节点请求通过 `X-Search-Gin-Remote: true` header 识别，来源 IP 必须在 peers 列表中或通过反向心跳自动加入 |
| **反向心跳自动发现** | `lan_discovery.go:TryVerifyAndAddPeer()` | 首次收到未知 IP 的集群请求时，反向 GET 该 IP 的 `/api/heartBeat` 验证，通过则自动加入集群（持久化到 `setting.json`） |
| **Peer 列表** | `internal/handler/lan_controller.go` | `GET /api/lanPeers` 返回在线节点 |
| **跨节点搜索** | `internal/service/remote_search.go` | 并发请求所有在线节点（最多 5 个并发），合并结果并去重 |
| **跨节点操作** | `internal/service/remote_operation.go` | 文件删除/重命名/移动/标签/转码等操作转发到源节点执行 |
| **URL 填充** | `internal/service/remote_search.go:FillURLs()` | 本机文件用 `pickLocalIP()` 取客户端同网段 IP，远程文件用 `resolvePeerIP()` |
| **前端集群页** | `frontend/src/pages/system/SystemPage.vue` | 系统 → 集群 标签页，显示节点列表 + 连通检测 |

**关键配置**（`setting.json`）：

```json
{
  "nodeName": "书房电脑",
  "enableLanDiscovery": true,
  "discoveryPeers": ["192.168.1.102:10081"]
}
```

**去重策略**：`Code+Size`（优先）或 `Name+Size`（兜底），不用 Id。本机文件优先。

**文件流端口**：所有节点间文件流统一走 `:10082`，与 API 端口 `:10081` 分离，避免图片/视频流占用 API 带宽。

## 平台假设

- **Windows 为主要目标**：使用 `ffmpeg.exe` / `ffplay.exe` 二进制、`cmd /C start` 打开文件夹、`-H=windowsgui` 链接器标志
- `fixOnWin.go` / `fixOnNotWin.go`（build tag `windows`）——Windows 上隐藏子进程的控制台窗口
- 交叉编译（例如在 gitlab-ci 中编译 Linux 版本）会丢失打开文件夹和媒体播放功能

## 开发命令

```bash
# Go 后端（开发模式，不嵌入资源）
go run main.go

# 前端开发服务器（代理 /api → localhost:10081）
cd frontend && quasar dev

# 运行 Go 测试套件
go test ./...

# 前端 lint / format
cd frontend && yarn lint
cd frontend && yarn format

# 完整生产构建（前端 + Go 嵌入）
bash ball_build.sh
```

## 前端约定

- Quasar v2 + Vite，Vue Router hash 模式
- Quasar 已启用的插件：Notify、AppFullscreen、Dialog
- Pinia + `pinia-plugin-persist` 实现 localStorage 持久化
- `@` 别名 → `frontend/src`，`components` 别名 → `frontend/src/components`
- Prettier：单引号、分号
- ESLint：`@typescript-eslint/recommended` + `vue3-essential` + prettier
- TypeScript 4.5+，target ES2020，moduleResolution: Node

## 后端架构

### 依赖注入架构（2024年重构）

采用**显式依赖注入**模式，消除全局单例依赖，提升可测试性。

#### 核心依赖图

```
main.go
  ├─ NewSearchEngine()         → *searchEngineCore
  ├─ NewScanQueue(engine)      → *taskQueue
  ├─ NewSearchService(engine, settings, events, scanQueue) → *searchService
  ├─ InitService(engine, search)   → 注册全局 getter（内部使用）
  └─ handler.InitApp(search, files, config) → handler 层依赖注入
```

#### 服务层结构（`internal/service/`）

| 结构体 | 字段 | 职责 |
|--------|------|------|
| `searchService` | `engine`, `settings`, `events`, `scanQueue` | 文件操作 / 扫描 / 流媒体 / 目录清理 |
| `searchEngineCore` | `KeywordHistoryCache`, `buckets`, `index` | 搜索引擎：索引加载、分页搜索、缓存、并发 searchPool |
| `taskQueue` | `tasks chan` | 扫描任务队列（容量 100 channel） |

**接口定义（`interfaces.go`）：**

| 接口 | 方法 | 说明 |
|------|------|------|
| `IndexEngine` | `Page`, `FindById`, `ReplaceFile`, `DeleteFile` 等 | 搜索引擎抽象 |
| `FileService` | `SetMovieType`, `AddTag`, `Rename`, `Move`, `Delete` 等 | 文件操作抽象 |
| `Settings` | `Get`, `Set`, `Flush` | 配置读写抽象，替代全局 `GetOSSetting()` |
| `EventBus` | `Broadcast` | 事件广播抽象，替代 `sse.BroadcastEvent()` |

#### Handler 层结构（`internal/handler/`）

```go
type AppHandle struct {
    search  service.IndexEngine   // 搜索引擎
    files   service.FileService   // 文件操作
    config  service.Settings      // 配置管理
}

func InitApp(search, files, config)  // main.go 调用初始化
func UseApp() *AppHandle             // 获取全局 handler
```

#### 初始化流程（`main.go`）

```go
// 1. 创建核心组件（显式依赖图）
engine := service.NewSearchEngine()
settings := service.DefaultSettings()
events := service.DefaultEventBus()

// 2. 创建扫描队列并关联 searchService
scanQueue := service.NewScanQueue(engine, settings)
search := service.NewSearchService(engine, settings, events, scanQueue)

// 3. 注册全局（内部函数仍需通过 getter 访问）
service.InitService(engine, search)

// 4. 加载上次扫描的索引缓存
engine.LoadCachedIndex()

// 5. 创建 Handler（注入依赖）
handler.InitApp(engine, search, settings)
```

**核心文件说明：**

| 文件 | 内容 |
|------|------|
| `service.go` | 依赖注入：`NewSearchEngine()`, `NewSearchService()`, `InitService()`, 全局 getter |
| `interfaces.go` | 接口定义：`IndexEngine`, `FileService`, `Settings`, `EventBus` |
| `search_executor.go` | `Page()`（导出）/ `pageAsync()`（内部）/ `tryCache()` / `doSearch()` / `collectResults()` / `returnRepeatSearch()` / `PageAuthor()` / `FindById()` |
| `index_engine_manager.go` | `searchEngineCore` struct 定义 + `loadIndex()` / `installIndex()` / `syncIndex()` |
| `index_engine_builder.go` | 索引构建：`buildIndexFromBuckets()` / `addFileToIndex()` |
| `index_engine_cache.go` | `saveIndexToCache` / `LoadCachedIndex` — 磁盘缓存 |
| `index_engine_bucket.go` | `bucketFile` — 单目录下的文件桶，支持 `searchBucket()` |
| `file_operations.go` | `SetMovieType` / `AddTag` / `ClearTag` / `Rename` / `Move` / `Delete` — receiver 为 `*searchService`，通过 `s.engine` 访问索引 |
| `file_scanner.go` | `ScanAll` / `ScanTarget` / `Walk` / `WalkInner` — 通过 `s.settings.Get()` 读取配置 |
| `file_video_processor.go` | `TransferFormatter` / `CutImage` / `MergeFiles` — 包级函数（无状态，不依赖 receiver） |
| `file_downloader.go` | `DownJpgMakePng` / `DownJpgAsPng` — 包级函数 |
| `directory_cleaner.go` | `DeleteOne` / `DownDeleteDir` / `UpDirClear` / `removeWalk()` — 方法为 `*searchService` |
| `hw_accel.go` | `detectHwAccel` / `getH264Encoder` / `getH265Encoder` — 包级函数（无状态） |
| `task_scheduler.go` | `TaskExecuting()` / `HeartBeat()` + 扫描任务队列（`scanTask` / `taskQueue`） |
| `background_launch.go` | `InitSetting()` / `StartBackgroundTasks()` — 包级函数，由 `main.go` 调用 |
| `torrent_service.go` | BT 下载（独立 `TorrentService` struct） |

#### 全局访问（仅限必要场景）

```go
// service 包内部使用（通过 getter）
engine := GetEngine()   // *searchEngineCore
search := GetSearch()   // *searchService
workDir := GetWorkDir() // string

// handler 层使用（通过 InitApp 注入）
app := UseApp()
app.search.FindById(id)
app.files.SetMovieType(file, movieType)
app.config.Get()
```

### 搜索流程

```
handler: UseApp().search.Page(param)
  → pageAsync(param)
    → loadIndex()               // atomic.Value 读取当前索引
    → OnlyRepeat? → returnRepeatSearch(snap)
    → tryCache(param)           // LRU + epoch 校验
    → doSearch(snap, param)     // 分发 bucket 并发搜索
      → collectResults()        // channel 合并 + 超时处理
      → SortFileItems()
      → 写入缓存 (KeywordHistoryCache)
      → GetPageOfFiles()
```

## Go 约定

- 2 空格缩进（`.editorconfig`），LF 换行
- 日志：使用 `utils.InfoFormat` / `utils.ErrorFormat`（封装 logrus；同时写入 stdout 和 `gin.log`）
  - `InfoFormat` 对应 logrus `INFO` 级别（开发环境 `InfoLevel`，生产环境 `ErrorLevel` 下被抑制）
- `main.go` 中启动的 goroutine 必须使用 `defer utils.RecoverPanic()`
- HTTP 错误响应使用 `utils.NewFailByMsg(msg)` 返回 JSON `{fail: true, msg: "..."}`，成功响应使用 `gin.H` 或 model 结构体
- `pkg/utils/Os.go` 导出 `PathSeparator`——请使用此常量而非直接使用 `os.PathSeparator`
- 文件存在判断：`utils.ExistsFiles(path)`（定义在 `OsFilepathUtils.go`）
- 搜索结果缓存（`KeywordHistoryCache`）使用 epoch 机制：`cacheEpoch` 在每次 `installIndex` 时递增，缓存读写时校验 epoch，防止索引重建后返回过时结果（`search_executor.go:cachedResult`）
- 搜索入口：`UseApp().search.Page(param)`（导出）；内部 `pageAsync()` 三步：`loadIndex` → `tryCache`/`OnlyRepeat` → `doSearch`
- 文件操作通过 `s.notifyFileChanged(oldFile, updated, action)`（`searchService` 方法）统一更新索引 + SSE 通知

## 依赖注入模式

### 访问规则

| 场景 | 方式 | 示例 |
|------|------|------|
| **handler 层** | 通过 `UseApp()` 获取注入的依赖 | `app.search.FindById(id)` |
| **service 层内部** | 通过结构体字段访问 | `s.engine.FindById(id)`, `s.settings.Get()` |
| **包级辅助函数** | 通过 getter 获取（仅限必要） | `GetEngine().FindById(id)` |
| **禁止** | 直接引用全局单例 | ~~`service.SearchEngine.FindById()`~~ |

### 接口优先原则

- 新增依赖必须定义接口（`interfaces.go`）
- 构造函数接收接口而非具体类型
- 默认适配器提供开箱即用的实现（`DefaultSettings()`, `DefaultEventBus()`）

## CI（已废弃）

`gitlab-ci.yml` 引用的 `repo_workspace/` 路径与实际仓库结构不匹配，不可依赖。
