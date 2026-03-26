

# gosrc 文件管理系统

本地磁盘文件搜索与管理系统

## 项目简介

gosrc 是一款基于 Golang + Vue3 (Quasar) 开发的本地磁盘文件搜索与管理系统，支持文件检索、图片浏览、视频播放等多种功能。

## 技术架构

| 层级 | 技术栈 |
|------|--------|
| 后端 | Golang |
| Web 框架 | Gin |
| 前端框架 | Vue 3 + Quasar |
| 桌面应用 | Electron + Quasar |
| 视频处理 | FFmpeg |

## 功能特性

- **本地文件搜索**：快速检索本地磁盘中的文件
- **图片浏览**：支持图片文件的在线预览
- **视频播放**：集成视频播放功能，支持多种格式
- **文件管理**：对搜索结果进行管理和操作
- **系统设置**：灵活的系统配置选项

## 项目结构

```
gosrc/
├── buildQuasar.sh          # 构建脚本
├── electron_quasar/        # 前端项目 (Vue + Quasar + Electron)
│   ├── src-electron/       # Electron 源码
│   │   ├── electron-main.ts
│   │   └── electron-preload.ts
│   ├── src/                # Vue 源码
│   │   ├── pages/          # 页面组件
│   │   └── components/     # 公共组件
│   └── quasar.config.js    # Quasar 配置
└── qapp/                   # 打包后的 Web 应用
```

## 使用方式

### Web 系统部署

```bash
# 执行打包脚本
sh buildQuasar.sh 2

# 生成 qapp 文件夹（可移动）
# 点击 exe 启动 Web 服务
# 访问端口: http://localhost:10081
```

### 桌面应用部署

```bash
# 执行打包脚本
sh buildQuasar.sh 4

# 生成桌面应用包
# 目录: electron_quasar/dist/electron/Packaged/文件搜索系统-win32-x64
# 点击【文件搜索系统.exe】启动桌面软件
```

## 前端技术栈

- **Vue 3** - 渐进式前端框架
- **Quasar** - 基于 Vue 的 UI 框架
- **Electron** - 跨平台桌面应用框架
- **Axios** - HTTP 客户端
- **DPlayer** - 视频播放器组件

## 依赖要求

- Golang 1.18+
- Node.js 14+
- FFmpeg (用于视频处理)
- Quasar CLI

## 许可证

本项目仅供学习交流使用。