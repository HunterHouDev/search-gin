# search-gin

基于 Golang + Vue 3 的本地文件搜索、管理与媒体播放系统。通过 `//go:embed` 将前端嵌入 Go 二进制，单文件即可部署。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/hunter/search-gin/actions/workflows/ci.yml/badge.svg)](https://github.com/hunter/search-gin/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-lightgrey)](README.md)

## 功能

- **文件搜索**：全文索引，支持按关键词、作者、标签、分类、路径等多维度筛选与分页
- **视频播放**：内嵌播放、画中画、沉浸式全屏三种模式
- **磁力链播放**：解析磁力链、边下边播、实时下载进度
- **视频剪辑**：通过 FFmpeg 按时间范围剪切、截图、转码
- **图片浏览**：缩略图网格、在线预览
- **文件管理**：重命名、移动、删除、标签管理
- **用户系统**：登录认证、多用户管理（管理员 + 普通用户）、运行时可配置、管理员密码支持 `setting.json` 覆盖
- **多节点集群**：HTTP 信令节点发现、跨节点搜索、跨节点文件操作（删除/重命名/转码等）、文件流直连 `:10082`
- **视频会议**：WebRTC 点对点视频通话
- **聊天系统**：WebSocket 实时聊天

## 技术栈

| 层级   | 技术                       |
| ------ | -------------------------- |
| 后端   | Go 1.25 + Gin 1.12        |
| 前端   | Vue 3 + Quasar 2 + Pinia  |
| 检索   | 内存全文索引 + LRU 缓存    |
| 磁力链 | anacrolix/torrent          |
| 媒体   | FFmpeg（ffmpeg.exe/ffplay） |
| 桌面   | Electron（可选）            |

## 快速开始

### 开发环境

```bash
# 后端（默认端口 :10081）
go run .

# 前端（另开终端，代理 /api → localhost:10081）
cd frontend && yarn install && quasar dev
```

### 生产构建

```bash
# 方式一：Makefile（推荐）
make build               # 全量构建 → qapp/appQuaser.exe

# 方式二：Shell 脚本
bash ball_build.sh       # 全量构建（前端 + Go embed）→ qapp/appQuaser.exe
bash bfront_build.sh     # 仅构建前端 → qapp/dist/
bash bpc_build.sh        # Electron 桌面打包
```

`make build` / `ball_build.sh` 流程：`yarn build` → 复制 `dist/spa/*` 到根 `dist/` → `go build -tags=prod -ldflags "-H=windowsgui -s -w"`

### Docker 部署

```bash
docker compose up -d
# 访问 http://localhost:10081
```

三段式构建：`node:20-alpine` 编译前端 → `golang:1.25-alpine` 编译 Go → `alpine:3.20` 运行时（安装 ffmpeg、ca-certificates）。详见 `Dockerfile` 和 `docker-compose.yml`。

### CI / 代码质量

项目使用 GitHub Actions CI（`.github/workflows/ci.yml`），自动运行：

- **golangci-lint**：`govet`、`staticcheck`、`gosimple`、`errcheck`、`ineffassign`、`unused`、`gosec`、`misspell`
- **前端 lint**：ESLint + Prettier
- **Pre-commit hooks**：Husky + lint-staged，提交前自动格式化和检查

其他 `make` 目标：

| 命令 | 说明 |
|------|------|
| `make dev` | 开发模式运行后端 |
| `make frontend-dev` | 启动前端开发服务器 |
| `make build-quick` | 仅构建 Go（不重建前端） |
| `make test` | 运行全部测试 |
| `make lint` | Go + 前端 lint |
| `make clean` | 清理构建产物 |

### 运行

```bash
./qapp/appQuaser.exe
# 访问 http://localhost:10081
# 登录：setting.json 中配置 users 数组（支持多用户 + 角色 + 权限）
# 或兼容旧版 adminPassword 字段（用户名为空或 admin）
```

### 首次初始化

系统启动时检查 `setting.json` 是否配置了密码。未配置时进入**未初始化状态**，所有 `/api/*` 请求返回 `412`。

**初始化流程：**

1. 启动后访问前端页面
2. 前端请求后端接口，收到 `412` 响应后自动跳转到初始化页面
3. 在初始化页面提交管理员密码
4. 密码保存到 `setting.json`，系统解除阻断，自动跳转到登录页

**判断逻辑：**
- 启动时 `InitInitializedFlag()` 检查 `AdminPassword` 是否为空
- 为空：`initialized = false`，`InitCheckMiddleware` 拦截所有 API（返回 412）
- 非空：`initialized = true`，系统正常运行
- `/api/init/setup` 未初始化时放行，已初始化后返回 403（防止重复初始化攻击）

## 文件流安全（:10082 端口）

从 `bb7a53a` 版本开始，文件流端口（10082）启用 **streamToken** 认证机制：

- `FillURLs()` 为每个文件生成 AES-256-GCM 加密的 streamToken（内含过期时间戳）
- 图片预览 token：**5 分钟**有效期
- 视频流 token：**4 小时**有效期
- `:10082` 侧 `StreamTokenAuth` 中间件解密验证 token，不依赖内存 session map
- AES-256-GCM 密钥通过 `setting.json` 的 `streamSecret` 字段配置（64 字符 hex）
  - 默认自动生成并持久化到 `setting.json`
  - 同一集群所有节点应使用相同密钥以实现跨节点流媒体互通
- 旧版 HMAC 签名（`SignAuthMiddleware`）保留但不注册

## 用户认证与权限

支持多用户、角色和权限管理。所有配置通过 `setting.json` 进行，无编译回退。

### 管理员密码（兼容模式）

```json
{
  "adminPassword": "your-password"
}
```

- 未配置时系统进入**未初始化状态**，所有 API 返回 412，需通过 `/api/init/setup` 完成首次配置
- 登录时支持用户名留空（仅凭密码匹配管理员）

### 多用户 + 角色权限（推荐）

```json
{
  "users": [
    {
      "username": "admin",
      "password": "strong-password",
      "role": "super_admin",
      "expireDate": "",
      "permissions": []
    }
  ],
  "roles": [
    {
      "name": "custom_role",
      "label": "自定义角色",
      "permissions": ["menu:home", "op:edit", "op:scan"]
    }
  ],
  "defaultPermissions": ["menu:home", "menu:search"]
}
```

| 字段 | 说明 |
|------|------|
| `users` | 用户数组，每项含用户名、密码、角色、过期时间、个人权限 |
| `roles` | 自定义角色定义（名称、标签、权限列表） |
| `defaultPermissions` | 新用户默认权限列表（叠加在角色之上） |

支持的角色：`super_admin` / `admin` / `user` + 自定义角色。`requireAdmin()` 兼容旧 token（role 为空时自动放行）。

- `GetSettingInfo` API 不返回密码字段
- Token 存内存 map（`sync.RWMutex` 保护），`time.AfterFunc` 到期自动删除，无定时轮询

## 项目结构

```
search-gin/
├── main.go              # 入口，依赖组装、信号处理、优雅关闭
├── assets.go            # 资源解压、静态文件加载
├── assets_dev.go        # 开发环境（不嵌入资源）
├── assets_prod.go       # 生产环境 //go:embed
├── internal/
│   ├── handler/         # HTTP 处理器（依赖注入：IndexEngine + FileService + Settings）
│   ├── model/           # 数据模型（FileItem、FileInfo、搜索参数、任务模型）
│   ├── router/          # 路由注册（API 路由 + 文件流路由）
│   ├── server/          # HTTP 服务创建、端口解析、优雅关闭
│   ├── service/         # 业务逻辑（显式依赖注入模式）
│   │   ├── service.go               # 构造函数（NewSearchEngine/NewSearchService）+ 默认适配器
│   │   ├── interfaces.go            # 接口定义（IndexEngine/FileService/Settings/EventBus）
│   │   ├── index_engine_manager.go  # searchEngineCore + atomic.Value 索引指针
│   │   ├── index_engine_builder.go  # 索引构建（全量/增量/替换/删除）
│   │   ├── index_engine_executor.go # Page() 搜索入口 / pageAsync() / tryCache()
│   │   ├── index_engine_bucket.go   # bucketFile 文件桶 + searchBucket()
│   │   ├── index_engine_cache.go    # 快照磁盘缓存（gob 序列化）
│   │   ├── index_stats.go          # 扫描计时、内存日志、小文件目录
│   │   ├── index_param.go          # 端口常量、IndexNumber、最后扫描时间
│   │   ├── file_operations.go      # SetMovieType / AddTag / Rename / Move / Delete
│   │   ├── file_scanner.go         # ScanAll / Walk / WalkInner
│   │   ├── file_video_processor.go # TransferFormatter / CutImage / MergeFiles
│   │   ├── file_downloader.go      # DownJpgMakePng / DownJpgAsPng
│   │   ├── file_directory_cleaner.go # DeleteOne / DownDeleteDir / removeWalk
│   │   ├── hw_accel.go             # 硬件加速检测 / 编码器选择
│   │   ├── task_scheduler.go       # TaskExecuting / HeartBeat + 扫描任务队列
│   │   ├── task_service.go         # 传输任务管理（TransferTask / ClearCompletedTasks）
│   │   ├── scan_progress.go         # 扫描进度追踪（ScanProgress struct + 读锁安全拷贝）
│   │   ├── auth_service.go         # 认证（Token 签发/校验、time.AfterFunc 自动过期）
│   │   ├── authz_service.go        # 权限管理（PermissionDef / AllPermissions / RBAC）
│   │   ├── node_discovery.go       # 集群节点管理（HTTP 信令 + 反向心跳）
│   │   ├── remote_search.go        # 跨节点搜索 + 合并去重 + streamToken URL 生成
│   │   ├── remote_operation.go     # 跨节点文件操作转发（c.GetRawData() 读取 body）
│   │   └── torrent_service.go      # 磁力链/BT 下载管理
│   ├── sse/             # Server-Sent Events 广播（atomic.Bool 防递归启动）
│   ├── ws/              # WebSocket Hub（聊天/视频会议信令，atomic.Bool 防递归启动）
│   └── env/             # 环境配置（prod/dev build tag）
├── pkg/
│   ├── consts/          # 基础常量（端口等，逐步迁移至 internal）
│   ├── types/           # 类型定义（Setting, User, TransferTaskModel, Role）
│   └── utils/           # 日志、LRU 缓存、FNV 哈希、文件工具、协程池、stream_crypto
├── middleware/           # Gin 中间件
│   ├── common.go        # AuthMiddleware / InitCheckMiddleware / SlowRequestLogger
│   ├── stream_auth.go   # StreamTokenAuth（AES-256-GCM 文件流 token 校验）
│   └── recovery.go      # CustomRecovery（panic 恢复 → 500 JSON）
├── frontend/            # Vue 3 + Quasar 前端源码
├── dist/                # 前端构建产物（被 embed 嵌入）
├── setting.json         # 运行时配置（Dirs / adminPassword / users / roles / 多节点等）
├── Makefile             # 构建辅助（make dev / make build / make test / make lint）
├── Dockerfile           # Docker 三段式构建（node → golang → alpine）
├── docker-compose.yml   # Docker 编排（暴露 :10081/:10082，挂载 /media 和 data 卷）
└── ffmpeg.exe ffplay.exe  # 媒体处理工具
```

## 依赖注入架构

2024 年重构采用**显式依赖注入**模式，消除全局单例依赖。

### 核心依赖图

```
main.go
  ├─ NewSearchEngine()         → *searchEngineCore
  ├─ NewScanQueue(engine)      → *taskQueue
  ├─ NewSearchService(engine, settings, events, scanQueue) → *searchService
  ├─ InitService(engine, search)   → 注册全局 getter（内部使用）
  └─ handler.InitApp(engine, search, settings) → handler 层 DI
```

### 服务层结构

| 结构体 | 字段 | 职责 |
|--------|------|------|
| `searchService` | `engine`, `settings`, `events`, `scanQueue` | 文件操作 / 扫描 / 流媒体 / 目录清理 |
| `searchEngineCore` | `index`, `KeywordHistoryCache`, `searchPool` | 搜索引擎：索引加载、分页搜索、缓存 |
| `taskQueue` | `tasks`, `engine`, `settings`, `walkInner` | 扫描任务队列（容量 100 channel） |

### 接口定义（`interfaces.go`）

| 接口 | 方法 | 说明 |
|------|------|------|
| `IndexEngine` | `Page`, `FindById`, `ReplaceFileOnIndex`, `DeleteOnIndex`, `GetTypeMenu` 等 | 搜索引擎抽象 |
| `FileService` | `SetMovieType`, `AddTag`, `Rename`, `Move`, `Delete` 等 | 文件操作抽象 |
| `Settings` | `Get`, `Set`, `Flush` | 配置读写抽象，替代全局 `GetOSSetting()` |
| `EventBus` | `Broadcast` | 事件广播抽象，替代直接调用 `sse.BroadcastEvent()` |

### Handler 层

```go
type AppHandle struct {
    search  service.IndexEngine
    files   service.FileService
    config  service.Settings
}

func InitApp(search, files, config)  // main.go 调用
func UseApp() *AppHandle             // 获取全局 handler
```

### 访问规则

| 场景 | 方式 | 示例 |
|------|------|------|
| handler 层 | 通过 `UseApp()` 获取注入的依赖 | `app.search.FindById(id)` |
| service 层内部 | 通过结构体字段访问 | `s.engine.FindById(id)` |
| 包级辅助函数 | 通过 getter 获取（仅限必要） | `GetEngine().FindById(id)` |
| 禁止 | 直接引用全局单例 | ~`service.SearchEngine.FindById()`~ |

## ID 生成

文件 ID 由 `pkg/utils/OsFilepathUtils.go` 中的 `DirpathForId` 函数生成，基于 **FNV-1a** 哈希算法。

### 算法

```go
// pkg/utils/OsFilepathUtils.go
func DirpathForId(path string) string {
    h := fnv.New64a()
    h.Write([]byte(path))
    id := fmt.Sprintf("%x", h.Sum64())
    return id
}
```

### 特性

- **确定性**：相同路径始终生成相同 ID
- **零分配**：无内存分配，单次调用 ~10ns
- **非加密**：FNV-1a 是快速散列，不适合安全场景

### 碰撞概率与实际容量

哈希冲突遵循[生日悖论](https://en.wikipedia.org/wiki/Birthday_problem)——约 `sqrt(πN/2)` 个条目后预期出现首次碰撞（N = 2ⁿ）。

| 位数 | 首次碰撞约在 | 对媒体库的结论 |
|------|-------------|---------------|
| 32-bit | ~7.7 万 | 大型媒体库有风险 |
| **64-bit（当前）** | **~50 亿** | 远超任何实际场景 |
| 128-bit | ~2×10¹⁹ | 宇宙级冗余 |

实际媒体库文件数通常在 1 万 ~ 50 万之间，64-bit FNV-1a 碰撞概率远低于硬件误码率，无需担心。

### 修改指南

如需调整 ID 生成方式，修改 `pkg/utils/OsFilepathUtils.go` 中的 `DirpathForId` 函数即可。注意：

- **修改哈希算法会导致旧缓存 `search_cache.gob` 失效**，首次启动会全量重建索引
- 确保新算法是**确定性的**（同输入同输出）
- 64-bit 已满足绝大多数场景，无需升级到 128-bit

## 部署说明

- **Windows 平台**：主要目标平台，使用 `ffmpeg.exe`、`-H=windowsgui` 等 Windows 特性
- **embed 机制**：`-tags=prod` 将 `dist/`、`ffmpeg.exe`、`ffplay.exe`、`setting.json` 嵌入二进制，启动时自动解压到工作目录
- **端口分配**：
  - `:10081` — API + 前端（Token 认证）
  - `:10082` — 文件/图片/视频流（streamToken 认证：AES-256-GCM）
- **认证**：支持多用户 + 角色权限（`setting.json` 的 `users` 数组配置），也兼容旧版 `adminPassword` 字段
  - 启动时如未配置密码，系统进入**未初始化状态**（`InitCheckMiddleware` 拦截所有 API 返回 412）
  - 通过 `/api/init/setup` 完成首次配置后自动解除阻断
  - Token 存储在内存中（`sync.RWMutex` 保护），每个 token 到期自动删除（`time.AfterFunc`），无定时轮询
- **无数据库**：所有数据为内存存储，通过文件系统扫描填充。索引快照自动持久化到 `search_cache.gob`（gob 序列化），重启后优先加载缓存，用户无空白等待期；后台继续扫描以同步最新文件变更
- **多节点**：`setting.json` 中配置 `enableLanDiscovery: true` 并在 `discoveryPeers` 中添加对端地址，节点间通过 HTTP 信令 + 反向心跳自动发现，文件流通过 `:10082` 直连传输（streamToken 认证）
- **集群安全认证**：跨节点 API 请求携带 `X-Search-Gin-Remote: true` header 绕过 Token 认证，但来源 IP 必须为集群内已知 peer。首次遇到未知 IP 时自动反向心跳验证（GET 该 IP 的 `/api/heartBeat`），通过后自动加入集群并持久化到 `setting.json`，后续请求直达免验证

## 设计决策

本项目是 **LAN 应用**，安全标准与公有云不同。以下均为有意设计：

| 决策 | 原因 |
|------|------|
| 反向心跳自动发现 | 未知 IP 自动验证并加入集群，与 Redis Gossip 协议同理 |
| `/api/lanPeers` 无认证 | 节点发现需要无认证访问，扫描方此时不知道目标节点 |
| `/api/heartBeat` 无认证 | LAN 扫描探测存活用，返回的只是文件数量 |
| WebSocket `CheckOrigin` 返回 true | 局域网场景，安全靠 token 不靠 origin |
| SSE `/api/events` 无认证 | SSE 是只读推送，安全靠前端 token 校验 |
| LRU Cache Get 不移到头部 | 读并发性能优于标准 LRU 实现 |
| 无数据库 | 所有数据存内存，索引快照用 gob 持久化，简化设计 |
| 每节点独立 AES 密钥 | 默认自动生成并持久化到 `setting.json` 的 `streamSecret`，集群节点应配置相同密钥以互通 |

## 主要依赖

| 依赖                         | 用途            |
| ---------------------------- | --------------- |
| github.com/gin-gonic/gin     | Web 框架        |
| github.com/gin-contrib/cors  | CORS 中间件     |
| github.com/anacrolix/torrent | 磁力链/BT 下载   |
| github.com/gorilla/websocket | WebSocket       |
| github.com/sirupsen/logrus   | 结构化日志       |
| github.com/shirou/gopsutil/v3| 系统监控         |
| github.com/stretchr/testify  | 测试断言        |
| github.com/go-resty/resty/v2 | HTTP 客户端     |
| golang.org/x/sync            | errgroup 并发管理 |

---

## 架构解析：无锁内存搜索引擎

### 设计目标

- **读不阻塞写**：索引构建/更新期间，搜索请求不受影响
- **高并发读**：不加锁、不等待，多 goroutine 可同时搜索
- **增量更新**：单目录重扫不重建全局索引

### 三层架构

```
┌──────────────────────────────────────────────────────────┐
│                     searchEngineCore                     │
│  ┌────────────────────────────────────────────────────┐  │
│  │  index (atomic.Value) → *searchIndex             │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐              │  │
│  │  │bucket   │ │bucket   │ │bucket   │  ...          │  │
│  │  │ dir A   │ │ dir B   │ │ dir C   │              │  │
│  │  │FileLib  │ │FileLib  │ │FileLib  │              │  │
│  │  │TypeIdx  │ │TypeIdx  │ │TypeIdx  │              │  │
│  │  └─────────┘ └─────────┘ └─────────┘              │  │
│  ├────────────────────────────────────────────────────┤  │
│  │  KeywordHistoryCache  (LRU, 500 条)                │  │
│  │  searchPool           (goroutine 池, 20 并发)       │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

#### 1. bucketFile — 数据分片

每个扫描目录对应一个 `bucketFile`，内部包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| `FileLib` | `map[string]Movie` | 主存储，文件 ID → 文件对象 O(1) 查找 |
| `TypeIndex` | `map[string]map[string]struct{}` | 倒排索引，文件类型 → 文件 ID 集合 |
| `mu` | `sync.RWMutex` | 每 bucket 独立读写锁 |

每个 bucket 有自有的 `RWMutex`，写入时只锁单个 bucket，不影响其他 bucket 并发读。

#### 2. searchIndex — 只读快照

`searchIndex` 是一个不可变结构体，通过 `atomic.Value` 原子替换，包含：

- **buckets**：所有 bucket 的引用（pointer 共享）
- **预聚合数据**：`authorMap`、`typeMenu`、`tagMenu`、`seriesCount`、`repeatFiles`
- **统计**：`totalSize`、`totalCount`、`bucketCount`

预聚合数据在索引构建时一次性算好，搜索时零计算开销。
Handler 层通过 `GetTypeMenu()` / `GetTagMenu()` / `GetSeriesCount()` 方法从快照直接读取，消除 `sync.Map` 全局变量。

#### 3. searchEngineCore — 引擎门面

核心使用 `atomic.Value` 存储当前索引指针：

```go
type searchEngineCore struct {
    index               atomic.Value    // *searchIndex
    KeywordHistoryCache *utils.LRUCache // 搜索结果 LRU 缓存
    searchPool          *utils.GoroutinePool
    rebuildMu           sync.Mutex      // 防止并发重建
    cacheEpoch          atomic.Int64    // 缓存失效纪元
}
```

### 搜索流程

```
UseApp().search.Page(param)            ← handler 调用的 API 入口
  └─ pageAsync(param)                  ← 内部引擎方法
       │
       ├─ loadIndex()                  ← atomic.Value.Load（无锁）
       ├─ OnlyRepeat？ → returnRepeatSearch(index)
       ├─ tryCache(param)              ← LRU + epoch 校验
       │     └─ 命中 & epoch 匹配 → 直接分页返回
       │
       └─ doSearch(index, param)
             ├─ 遍历 buckets → 提交 goroutine 池
             │     ├─ searchBucket():
             │     │   ├─ 空关键词 + 类型筛选 → TypeIndex 倒排
             │     │   └─ 有关键词 → strings.Contains (AND 匹配)
             │     └─ 结果 → resultChan
             ├─ collectResults() ← channel 合并 + 超时处理
             ├─ 排序 → 写入 LRU 缓存 → 分页返回
             └─ GetPageOfFiles()
```

### 索引构建流程（影子索引）

```
ScanAll()
  │
  ├─ 并发 WalkInner() 扫描所有配置目录
  │   每个目录产出一个 bucketFile
  │
  ├─ buildSnapshotFromBuckets()
  │   ├─ 复制 bucket 指针
  │   ├─ 遍历所有文件 → 聚合作者/类型/标签/系列/重复检测
  │   └─ 返回完整 searchIndex
  │
  └─ installIndex()
      ├─ index.Store(newSnap)       ← 原子切换
      ├─ 清空 LRU 缓存
      ├─ cacheEpoch.Add(1)          ← 递增纪元，旧缓存自动失效
      ├─ 清空作者缓存
      └─ 设置最后扫描时间
```

**增量扫描**（单目录）走 `rebuildWithBucketIncremental()`：
1. 加载当前快照
2. 复制除目标目录外的所有 bucket 引用
3. 放入新 bucket
4. 在新集合上重新聚合作者/类型/标签/系列/重复检测
5. `syncIndex` 原子替换

### 快照磁盘缓存（填补启动空窗期）

`installIndex` 每次执行时异步将当前索引序列化（`encoding/gob`）写入工作目录的 `search_cache.gob`：

```
installIndex(newSnap)
  ├─ index.Store(newSnap)
  ├─ 清空 LRU 缓存 / 递增 epoch
  └─ saveIndexToCache(newSnap)         ← 异步 goroutine
       ├─ 遍历 buckets，持有 RLock 复制数据
       ├─ gob.Encode → .tmp 文件
       └─ os.Rename → search_cache.gob   ← 原子替换，防碎裂
```

**启动时**（`main.go`）在 HTTP 服务启动前加载缓存：

```
main()
  ├─ NewSearchEngine() / NewSearchService()
  ├─ engine.LoadCachedIndex()              ← 加载磁盘缓存
  │     └─ syncIndex(loaded)              ← 用户立刻可搜
  ├─ InitSetting()
  ├─ StartScanQueue() / ScanAll()         ← 后台扫描, 完成后原子替换
  └─ server.ListenAndServe()
```

**设计要点：**

| 决策 | 理由 |
|------|------|
| 每次 `installIndex` 都保存 | 保证缓存与内存状态一致，无不一致窗口 |
| 空快照跳过（`len(buckets)==0`） | 防止 `Reset()` 清空磁盘缓存 |
| 异步写入 goroutine | 不阻塞搜索/扫描路径 |
| `encoding/gob` 而非 JSON | 二进制紧凑、支持 Go 原生类型、无需 tag |
| `.tmp` + `Rename` 原子写入 | 防止写入中断导致文件损坏 |
| 启动静默降级 | 缓存不存在/损坏/版本不匹配时打日志后继续正常扫描 |

### 并发安全设计

| 场景 | 机制 | 级别 |
|------|------|------|
| 搜索读 | `atomic.Value.Load` (`loadIndex()`) | **完全无锁** |
| 索引重建写 | `rebuildMu` + 影子索引 | 写时排他，读不受影响 |
| Bucket 内部写入 | `bucketFile.mu` (RWMutex) | 细粒度，不影响其他 bucket |
| 菜单读取 | 从索引快照 `map` 直接读取（getter 方法） | 快照不可变，无锁 |
| LRU 缓存 | `sync.RWMutex` + epoch 校验 | 高并发读优化，索引更新后自动失效 |
| 快照磁盘缓存 | 异步 goroutine + 原子 rename | 不阻塞任何路径，写中断不损坏文件 |

### 性能特征

| 操作 | 复杂度 | 说明 |
|------|--------|------|
| 文件查找 (by ID) | O(1) | map 直接寻址 |
| 关键词搜索 | O(n) | 遍历所有文件，`strings.Contains` 匹配 |
| 类型筛选 | O(匹配数) | 走 TypeIndex 倒排索引 |
| 索引构建 | O(文件总数) | 全量扫描，单次构建 |
| 增量更新 | O(目标目录文件数) | 只重建一个 bucket |
| 缓存加载 | O(文件总数) | 启动时 gob 反序列化，代替扫描 |
| 缓存保存 | O(文件总数) | scan 完成后异步 gob 序列化 + 原子写盘 |

> 关键词搜索 O(n) 是当前瓶颈。文件数 10 万级时搜索延时仍在可接受范围（毫秒级），百万级需引入倒排索引。

### 测试覆盖

引擎测试位于 `internal/service/index_engine_test.go`，覆盖：

| 测试类别 | 用例数 | 覆盖范围 |
|----------|--------|----------|
| bucketFile | 5 | 创建/写入/读取/批量/空判断/索引 |
| buildSnapshot | 5 | 聚合统计/作者/菜单/重复/空 |
| searchEngineCore | 6 | 生命周期/查找/重置 |
| searchBucket | 5 | 关键词/多词/类型过滤/无匹配 |
| rebuildWithBucket | 2 | 替换/保留其他 |
| pageAsync | 4 | 跨 bucket/分页/空引擎/重复搜索 |

```bash
go test ./internal/service/ -run "Test" -v
```

## License

[MIT](LICENSE)
