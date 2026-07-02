# 贡献指南

欢迎为 search-gin 贡献代码或反馈问题。

## 开发环境

| 工具 | 版本要求 |
|------|---------|
| Go | 1.25+ |
| Node.js | 20+ |
| Yarn | 1.22+ |
| FFmpeg | 可选（用于视频处理） |

### 快速开始

```bash
# 克隆仓库
git clone https://github.com/hunter/search-gin.git
cd search-gin

# 后端开发
go run main.go                # 启动 API 服务（默认 :10081）

# 前端开发（另开终端）
cd frontend
yarn install
quasar dev                    # 开发服务器（代理 /api → :10081）
```

## 分支策略

- `main` — 稳定分支，保持可发布状态
- `feat/*` — 功能分支，从 main 切出，合回 main
- `fix/*` — 修复分支
- `chore/*` — 工具/配置变更

## 提交信息规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <description>

[optional body]
```

| Type | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `refactor` | 重构 |
| `docs` | 文档 |
| `style` | 样式/格式 |
| `chore` | 工具/配置 |
| `test` | 测试 |
| `perf` | 性能优化 |
| `security` | 安全修复 |

示例：

```
feat(search): 添加模糊搜索支持
fix(auth): 修复 token 过期未正确跳转登录
docs(readme): 更新 API 文档链接
```

## 代码规范

详见 [AGENTS.md](AGENTS.md#代码风格)。

## PR 流程

1. 从 `main` 创建 feature/fix 分支
2. 提交代码，确保测试通过
3. 运行 `make lint`（Go + 前端）
4. 运行 `make test`
5. 创建 PR 到 `main`
6. 等待 CI 通过 + Code Review

## 测试

详见 [AGENTS.md](AGENTS.md#测试)。

## 架构笔记

关键架构文件：

- `internal/service/interfaces.go` — 核心接口定义
- `internal/service/index_engine_executor.go` — 无锁搜索引擎
- `internal/handler/handler.go` — 依赖注入入口

详细架构说明见 [AGENTS.md](AGENTS.md#后端架构)。
