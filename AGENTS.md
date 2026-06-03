# AGENTS.md — search-gin

## Build tag 系统（双重作用）

`prod` 构建标签同时控制两件事：

- **资源嵌入**：`assets_dev.go`（默认，不嵌入） vs `assets_prod.go`（`//go:embed dist ffmpeg.exe ffplay.exe setting.json`）
- **运行时模式**：`internal/env/config.go`（默认 `IsProd=false`） vs `prod_config.go`（`IsProd=true`），控制 Gin 运行模式、pprof（仅开发环境监听 `:6060`）、CORS 配置及日志级别。

默认 `go run main.go` = 开发模式。添加 `-tags=prod` 编译生产环境二进制。

## 多端口服务

应用启动时通过 `errgroup` 同时绑定三个端口：
- `:10081` — 主 API + 前端（ControllerHost）
- `:10082` — 图片服务（ImageHost）
- `:10083` — 视频流服务（StreamHost）

端口在 `pkg/consts/base_param.go:63-65` 硬编码。默认 `setting.json` 中的配置与此一致。修改端口时两个文件必须同步。

## 前端构建 / 嵌入流程

1. `cd frontend && yarn build` → 产物输出到 `frontend/dist/spa/`
2. 构建脚本将 `frontend/dist/spa/*` 复制到 `./dist/`
3. `go build -tags=prod` 嵌入 `./dist/` 并在启动时解压到当前工作目录

`go run main.go`（开发模式）不嵌入资源，直接从磁盘读取 `./dist/`。开发时若 `dist/` 未更新，需先重新构建前端。

## Go 模块与导入路径

模块名：`search-gin`。所有内部导入使用 `search-gin/internal/...` 和 `search-gin/pkg/...`。

注意：`pkg/` 会导入 `internal/`——这是本仓库的设计（例如 `pkg/consts/` 导入 `internal/model`，`pkg/utils/` 导入 `internal/env`）。

## 无数据库

`internal/repository/` 和 `configs/` 目录为空。所有数据存储在内存中（Go struct + `sync.Map`），通过文件系统扫描填充。`model/Movie.go` 中的 `xorm` 结构体标签是历史遗留，无实际作用。

## 认证

- 硬编码管理员账号：`admin` / `qwer`（`pkg/consts/setting_data.go:16-18`）
- Token 存储在内存中（`TokenStore` map），通过 `Authorization: Bearer <token>` 发送
- WebSocket 使用 `?token=` 查询参数传递（无法设置自定义 Header）
- 中间件跳过认证的路径：`/api/login`、`/`、`/index.html`、`/api/file/`、`/api/png/`、`/api/jpg/`、`/api/tempimage/`
- 前端 API 基础地址为 `http://localhost:10081`（`frontend/src/boot/axios.ts:18`）

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

# 运行唯一的 Go 测试套件
go test ./internal/model/

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

## Go 约定

- 2 空格缩进（`.editorconfig`），LF 换行
- 日志：使用 `utils.InfoFormat` / `utils.ErrorFormat`（封装 logrus；同时写入 stdout 和 `gin.log`）
- `main.go` 中启动的 goroutine 必须使用 `defer utils.RecoverPanic()`
- HTTP 错误响应使用 `utils.NewFailByMsg(msg)` 返回 JSON `{fail: true, msg: "..."}`，成功响应使用 `gin.H` 或 model 结构体
- `pkg/utils/Os.go` 导出 `PathSeparator`——请使用此常量而非直接使用 `os.PathSeparator`
- 文件存在判断：`utils.ExistsFiles(path)`（定义在 `OsFilepathUtils.go`）

## CI（已废弃）

`gitlab-ci.yml` 引用的 `repo_workspace/` 路径与实际仓库结构不匹配，不可依赖。
