#!/usr/bin/env bash
# =============================================================================
# .husky/install.sh — 安装 Git pre-commit hooks
# 用于在项目根目录运行：bash .husky/install.sh
# =============================================================================
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_DIR="$PROJECT_DIR/.husky"

echo "🔧 安装 Git pre-commit hooks..."
git config --local core.hooksPath "$HOOKS_DIR"
echo "  ✅ hooksPath 已设置为: $HOOKS_DIR"

# 确保 hook 文件可执行
chmod +x "$HOOKS_DIR/pre-commit"
echo "  ✅ pre-commit hook 可执行权限已设置"

# 安装 lint-staged 依赖（如果尚未安装）
if [ ! -d "$PROJECT_DIR/node_modules/lint-staged" ]; then
  echo "  📦 安装 lint-staged 到根目录..."
  cd "$PROJECT_DIR/frontend"
  yarn install --frozen-lockfile 2>/dev/null || yarn install
fi

echo ""
echo "✅ pre-commit hooks 安装完成！"
echo "   下次 git commit 时自动运行："
echo "   • 前端: lint-staged (eslint + prettier)"
echo "   • Go: gofmt 格式检查 + go vet"
