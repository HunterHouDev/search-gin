# search-gin

基于 Golang + Vue 3 的本地文件搜索、管理与媒体播放系统。通过 `//go:embed` 将前端嵌入 Go 二进制，单文件即可部署。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 功能

- **文件搜索**：全文索引，支持按关键词、演员、标签、分类、路径等多维度筛选与分页
- **视频播放**：内嵌播放、画中画、沉浸式全屏三种模式，Web Audio 均衡器可视化
- **磁力链播放**：解析磁力链、边下边播、实时下载进度
- **视频剪辑**：通过 FFmpeg 按时间范围剪切、截图、转码
- **图片浏览**：缩略图网格、在线预览
- **文件管理**：重命名、移动、删除、标签管理
- **用户系统**：登录认证、多用户管理（管理员 + 普通用户）、运行时可配置
- **多节点集群**：局域网内 UDP 组播自动发现、跨节点搜索、跨节点文件操作（删除/重命名/转码等）、文件流直连 `:10082`

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
go run main.go

# 前端（另开终端，代理 /api → localhost:10081）
cd frontend && yarn install && quasar dev
```

### 生产构建

```bash
# 全量构建（前端 + Go embed）→ qapp/appQuaser.exe
bash ball_build.sh

# 仅构建前端 → qapp/dist/
bash bfront_build.sh

# Electron 桌面打包
bash bpc_build.sh
```

`ball_build.sh` 流程：`yarn build` → 复制 `dist/spa/*` 到根 `dist/` → `go build -tags=prod -ldflags "-H=windowsgui -s -w"`

### 运行

```bash
./qapp/appQuaser.exe
# 访问 http://localhost:10081
```

## 项目结构

```
search-gin/
├── main.go              # 入口，HTTP 服务
├── assets_dev.go        # 开发环境（不嵌入资源）
├── assets_prod.go       # 生产环境 //go:embed
├── internal/
│   ├── handler/         # HTTP 处理器
│   ├── model/           # 数据模型
│   ├── router/          # 路由注册
│   ├── service/         # 业务逻辑（索引、搜索、文件扫描、多节点集群）
│   │   ├── lan_discovery.go    # UDP 组播节点发现
│   │   ├── remote_search.go    # 跨节点搜索 + 合并去重
│   │   └── remote_operation.go # 跨节点文件操作转发
│   └── env/             # 环境配置（prod/dev build tag）
├── pkg/
│   ├── consts/          # 常量、配置、Token 管理
│   └── utils/           # 日志、LRU 缓存、文件工具、协程池
├── middleware/           # Gin 中间件（认证、recovery）
├── frontend/            # Vue 3 + Quasar 前端源码
├── dist/                # 前端构建产物（被 embed 嵌入）
├── setting.json         # 运行时配置（扫描目录、文件类型、多节点配置等）
└── ffmpeg.exe ffplay.exe  # 媒体处理工具
```

## 部署说明

- **Windows 平台**：主要目标平台，使用 `ffmpeg.exe`、`-H=windowsgui` 等 Windows 特性
- **embed 机制**：`-tags=prod` 将 `dist/`、`ffmpeg.exe`、`ffplay.exe`、`setting.json` 嵌入二进制，启动时自动解压到工作目录
- **端口分配**：
  - `:10081` — API + 前端（需认证）
  - `:10082` — 文件/图片/视频流（无需认证，跨节点直连使用）
  - `:10083` — UDP 组播（多节点自动发现）
- **认证**：默认管理员 `admin` / `qwer`，Token 存储在内存中
- **无数据库**：所有数据为内存存储，通过文件系统扫描填充，重启后需重新索引
- **多节点**：`setting.json` 中配置 `enableLanDiscovery: true` 开启 UDP 组播发现，节点间通过 `:10082` 直连传输文件流

## 主要依赖

| 依赖                         | 用途            |
| ---------------------------- | --------------- |
| github.com/gin-gonic/gin     | Web 框架        |
| github.com/anacrolix/torrent | 磁力链/BT 下载   |
| github.com/sirupsen/logrus   | 结构化日志       |
| github.com/stretchr/testify  | 测试断言        |
| github.com/go-resty/resty/v2 | HTTP 客户端     |

## License

[MIT](LICENSE)
