# search-gin 本地文件搜索与管理系统

基于 Golang + Vue 3 (Quasar) 的本地文件搜索、管理与媒体播放系统。通过 `//go:embed` 将前端构建产物嵌入 Go 二进制文件，单文件即可部署运行。

---

## 目录结构

```
search-gin/
├── main.go                  # Go 后端入口
├── assets_dev.go             # 开发环境资源加载（不嵌入）
├── assets_prod.go            # 生产环境 //go:embed 嵌入资源
├── go.mod / go.sum          # Go 依赖
├── setting.json              # 运行时配置文件
├── configs/                 # 配置文件目录
├── internal/                 # 后端核心代码
│   ├── handler/             # HTTP 请求处理器
│   ├── model/               # 数据模型
│   ├── repository/          # 数据仓库层
│   ├── router/              # 路由配置
│   └── service/             # 业务逻辑层
├── pkg/                     # 公共包
│   ├── consts/              # 常量定义
│   └── utils/              # 工具类（LRU缓存、日志、文件工具等）
├── middleware/               # Gin 中间件
├── frontend/                # Vue 3 + Quasar 前端源码
├── dist/                    # 前端构建产物（被 embed 嵌入）
├── scripts/                 # 构建/部署脚本
├── torrent_data/            # 磁力链下载数据目录
├── qapp/                    # 应用打包输出目录
├── ba_all_build.sh          # 全量构建脚本
├── bf_front_build.sh        # 前端构建脚本
├── bp_pc_build.sh          # PC Electron 打包脚本
├── ffmpeg.exe / ffplay.exe / ffprobe.exe  # 媒体处理工具
└── README.md
```

---

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Golang 1.25 + Gin 1.12 |
| 前端 | Vue 3 + Quasar 2.19 + Pinia 2.3 |
| 视频播放 | 原生 video 元素 + Web Audio API |
| 磁力链 | anacrolix/torrent |
| 媒体处理 | FFmpeg |
| 单文件部署 | `//go:embed dist`（build tag `prod` 控制） |
| 状态管理 | Pinia + pinia-plugin-persist |
| 桌面应用 | Electron（可选，依赖已安装） |

---

## 核心功能

### 文件搜索
- 本地文件全文索引，快速检索
- 支持按关键词、演员、标签、分类、文件路径等多维度筛选
- 分页、排序、过滤
- LRU 缓存搜索历史，提升重复搜索性能
- 重复文件检测

### 图片浏览（`PicturePage.vue`）
- 本地图片在线预览
- 缩略图网格展示
- 图片删除

### 视频播放（三个播放器组件）

| 播放器 | 场景 | 特点 |
|--------|------|------|
| `VideoPlayer.vue` | 主搜索页内嵌播放 | 支持画中画、剪辑参数面板、SearchPanel 搜索/图片侧边栏 |
| `VideoPlayerInPicture.vue` | 画中画小窗模式 | 悬浮播放、手势拖动、进度条、RAF 驱动进度更新 |
| `ImmersivePlayer.vue` | 沉浸式全屏播放 | 粒子背景特效、Web Audio 均衡器可视化、磁力链播放、本地图片 tab |

三个播放器共用 `frontend/src/components/utils/video.ts`（VideoClass 工具类，懒加载 video DOM）。

### 磁力链播放
- 解析磁力链（`POST /api/torrent/add`）
- 获取种子文件列表，选择指定文件播放
- 实时轮询下载进度（`GET /api/torrent/status/:infoHash`）
- 流式播放（`GET /api/torrent/stream/:infoHash?file=...`）
- 5 分钟轮询超时保护
- 自动清理完成的种子

### 视频剪辑
- 通过 FFmpeg 按时间范围剪切视频（`GET /api/cutMovie/:id/:start/:end`）
- 前端 `VideoCutParam.vue` 组件设置剪辑参数

### 文件管理
- 文件重命名、移动、删除
- 标签管理（添加/清除）
- 截图（`GET /api/cutImage/:id/:typeImage/:downFlag/:start`）
- 转换为 MP4（`GET /api/tranferToMp4/:id/:xcode`）
- 合并文件（`POST /api/mergeFiles`）
- 字幕合并（`GET /api/mergeSrt/:id`）

### 用户与系统
- 用户登录认证（`GET /api/login`）
- 用户管理（添加/删除/修改密码）
- 系统设置（运行时配置 `setting.json`）
- 心跳检测与文件变化扫描（`GET /api/heartBeat`）
- 内存日志（`GET /api/logMemery`，系统关闭或重启后消失）
- 远程关机（`GET /api/shutDown`）

---

## API 端点

### 文件操作

| 路径 | 方法 | 功能 |
|------|------|------|
| `/api/movieList` | POST | 电影文件搜索（分页） |
| `/api/actressList` | POST | 演员信息搜索 |
| `/api/file/:id` | GET | 获取文件信息 |
| `/api/file/rename` | GET | 重命名文件 |
| `/api/file/move` | POST | 移动文件 |
| `/api/delete/:id` | GET | 删除文件 |
| `/api/openFolder/:id` | GET | 打开文件所在文件夹 |

### 媒体

| 路径 | 方法 | 功能 |
|------|------|------|
| `/api/play/:id` | GET | 视频流播放 |
| `/api/png/:path` | GET | 获取图片 |
| `/api/jpg/:path` | GET | 获取 JPG 图片 |
| `/api/cutMovie/:id/:start/:end` | GET | 剪切视频 |
| `/api/cutImage/:id/:typeImage/:downFlag/:start` | GET | 截图 |
| `/api/tranferToMp4/:id/:xcode` | GET | 转换为 MP4 |

### 磁力链

| 路径 | 方法 | 功能 |
|------|------|------|
| `/api/torrent/add` | POST | 添加磁力链 |
| `/api/torrent/status/:infoHash` | GET | 查询下载状态 |
| `/api/torrent/stream/:infoHash` | GET | 流式播放（需 `?file=` 参数） |
| `/api/torrent/startDownload` | POST | 启动文件下载 |
| `/api/torrent/files/:infoHash` | GET | 获取种子文件列表 |
| `/api/torrent/:infoHash` | DELETE | 删除种子 |

### 系统

| 路径 | 方法 | 功能 |
|------|------|------|
| `/api/heartBeat` | GET | 心跳检测 / 触发文件扫描 |
| `/api/refreshIndex` | GET | 刷新文件索引 |
| `/api/logMemery` | GET | 内存日志（系统关闭或重启后消失） |
| `/api/setting` | GET/POST | 读取/更新系统设置 |
| `/api/login` | GET | 用户登录 |
| `/api/shutDown` | GET | 远程关机 |

---

## 构建与运行

### 开发环境

```bash
# 后端（configs/ 中的配置生效，不嵌入前端资源）
go run main.go

# 前端（另开终端）
cd frontend
yarn install
quasar dev
```

开发环境下前端通过 Vite dev server 运行，后端 API 通过代理转发，无需嵌入资源。

### 生产构建

```bash
# 全量构建（前端 + 后端 + embed）
bash ba_all_build.sh
# 输出：qapp/appQuaser.exe

# 仅构建前端（用于更新已部署环境的前端部分）
bash bf_front_build.sh

# PC Electron 打包
bash bp_pc_build.sh
```

`ba_all_build.sh` 流程：
1. `cd frontend && yarn build` — 构建前端，产物输出到 `frontend/dist/spa/`
2. 将前端产物复制到 `dist/`
3. `go build -o qapp/appQuaser.exe -ldflags "-H=windowsgui" -tags=prod` — 触发 `assets_prod.go` 中的 `//go:embed dist ffmpeg.exe ffplay.exe setting.json`

### 运行

```bash
# 直接运行（默认端口 10081，见 setting.json）
./qapp/appQuaser.exe

# 或指定配置文件
./qapp/appQuaser.exe -c configs/config.yaml
```

访问 `http://localhost:10081`

---

## `//go:embed` 实现方式

通过 build tag 区分开发/生产环境：

**`assets_dev.go`**（默认，无 build tag）：
```go
//go:build !prod

package main

func extractAll(tempDir string) error { return nil } // 开发环境不嵌入，直接读文件
```

**`assets_prod.go`**（需 `-tags=prod`）：
```go
//go:build prod

package main

//go:embed dist ffmpeg.exe ffplay.exe setting.json
var staticFiles embed.FS

func extractAll(tempDir string) error { /* 从嵌入 FS 解压到临时目录 */ }
```

---

## 主要依赖

### 后端（Go）

| 依赖 | 用途 |
|------|------|
| `github.com/gin-gonic/gin` | Web 框架 |
| `github.com/gin-contrib/cors` | CORS 中间件 |
| `github.com/anacrolix/torrent` | 磁力链/BT 下载 |
| `github.com/PuerkitoBio/goquery` | HTML 解析 |
| `github.com/sirupsen/logrus` | 结构化日志 |
| `github.com/toorop/gin-logrus` | Gin 日志中间件 |
| `golang.org/x/sync` | errgroup 等并发工具 |

### 前端（Node）

| 依赖 | 用途 |
|------|------|
| `vue@3` | 前端框架 |
| `quasar@2` | UI 组件库 |
| `pinia` | 状态管理 |
| `pinia-plugin-persist` | Pinia 持久化 |
| `@vueuse/core` | Vue 组合式工具集 |
| `axios` | HTTP 客户端 |
| `sortablejs` | 拖拽排序 |
| `electron` | 桌面应用（可选） |
