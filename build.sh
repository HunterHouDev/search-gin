#!/bin/bash

# Search-Gin Project Build Script
# 使用方法: ./build.sh [选项]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 打印函数
print_header() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║           Search-Gin 统一构建系统                      ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}[步骤]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[成功]${NC} $1"
}

print_error() {
    echo -e "${RED}[错误]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[警告]${NC} $1"
}

print_info() {
    echo -e "${CYAN}[信息]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装. 请先安装 $1"
        exit 1
    fi
}

# 检查 Go 环境
check_go() {
    print_step "检查 Go 环境..."
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装. 请先安装 Go (https://golang.org/dl/)"
        exit 1
    fi
    go version
    print_success "Go 环境检查完成"
}

# 检查 Node 环境
check_node() {
    print_step "检查 Node.js 环境..."
    if ! command -v node &> /dev/null; then
        print_error "Node.js 未安装. 请先安装 Node.js (https://nodejs.org/)"
        exit 1
    fi
    node --version
    npm --version
    print_success "Node.js 环境检查完成"
}

# 初始化子模块
init_submodules() {
    print_step "初始化子模块..."
    if [ -d "gosrc" ]; then
        cd gosrc
        go mod download
        go mod tidy
        cd ..
        print_success "Go 依赖初始化完成"
    fi

    if [ -d "electron_quasar" ]; then
        cd electron_quasar
        if command -v yarn &> /dev/null; then
            yarn install
        else
            npm install
        fi
        cd ..
        print_success "前端依赖初始化完成"
    fi
}

# 代码检查
check_code() {
    print_step "代码质量检查..."

    if [ -d "gosrc" ]; then
        cd gosrc
        if command -v golangci-lint &> /dev/null; then
            golangci-lint run ./...
        else
            print_warning "golangci-lint 未安装, 跳过 Go 代码检查"
            print_info "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        fi
        go vet ./...
        cd ..
    fi

    if [ -d "electron_quasar" ]; then
        cd electron_quasar
        if [ -f "package.json" ]; then
            if command -v yarn &> /dev/null; then
                yarn lint || true
            else
                npm run lint || true
            fi
        fi
        cd ..
    fi

    print_success "代码质量检查完成"
}

# 运行测试
run_tests() {
    print_step "运行测试..."

    if [ -d "gosrc" ]; then
        cd gosrc
        print_info "运行 Go 测试..."
        go test -v ./... || true
        cd ..
    fi

    if [ -d "electron_quasar" ]; then
        cd electron_quasar
        if [ -f "package.json" ]; then
            print_info "运行前端测试..."
            if command -v yarn &> /dev/null; then
                yarn test || true
            else
                npm run test || true
            fi
        fi
        cd ..
    fi

    print_success "测试完成"
}

# 构建后端
build_backend() {
    print_step "构建后端..."

    cd gosrc

    OUTPUT_DIR="../dist"
    mkdir -p "$OUTPUT_DIR"

    case "$1" in
        web)
            print_info "构建 Web 版本..."
            go build -o "$OUTPUT_DIR/search-gin-web.exe" -ldflags "-H=windowsgui" .
            print_success "Web 版本构建完成: $OUTPUT_DIR/search-gin-web.exe"
            ;;
        console)
            print_info "构建控制台版本..."
            go build -o "$OUTPUT_DIR/search-gin-console.exe" .
            print_success "控制台版本构建完成: $OUTPUT_DIR/search-gin-console.exe"
            ;;
        linux)
            print_info "构建 Linux 版本..."
            GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/search-gin-linux" .
            print_success "Linux 版本构建完成: $OUTPUT_DIR/search-gin-linux"
            ;;
        all)
            print_info "构建所有平台..."
            mkdir -p "$OUTPUT_DIR"
            go build -o "$OUTPUT_DIR/search-gin-windows.exe" .
            GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/search-gin-linux" .
            print_success "所有平台版本构建完成"
            ;;
        *)
            go build -o "$OUTPUT_DIR/search-gin.exe" .
            print_success "后端构建完成: $OUTPUT_DIR/search-gin.exe"
            ;;
    esac

    cd ..
}

# 构建前端
build_frontend() {
    print_step "构建前端..."

    cd electron_quasar

    OUTPUT_DIR="../dist/front"

    case "$1" in
        web)
            print_info "构建 Web 版本..."
            if command -v yarn &> /dev/null; then
                yarn build
            else
                npm run build
            fi
            print_success "Web 版本构建完成"
            ;;
        electron)
            print_info "构建 Electron 版本..."
            if command -v yarn &> /dev/null; then
                yarn topc
            else
                npm run topc
            fi
            print_success "Electron 版本构建完成"
            ;;
        all)
            print_info "构建所有版本..."
            if command -v yarn &> /dev/null; then
                yarn build
                yarn topc
            else
                npm run build
                npm run topc
            fi
            print_success "所有前端版本构建完成"
            ;;
        *)
            if command -v yarn &> /dev/null; then
                yarn build
            else
                npm run build
            fi
            print_success "前端构建完成"
            ;;
    esac

    cd ..
}

# 打包完整应用
package_app() {
    print_step "打包完整应用..."

    BACKEND_DIR="dist"
    FRONTEND_DIR="electron_quasar/dist"
    OUTPUT_DIR="release"

    mkdir -p "$OUTPUT_DIR"

    print_info "准备打包文件..."

    if [ -d "$FRONTEND_DIR/electron/Packaged" ]; then
        print_info "复制 Electron 应用..."
        cp -r "$FRONTEND_DIR/electron/Packaged" "$OUTPUT_DIR/"
        print_success "Electron 应用已复制"
    fi

    if [ -d "$BACKEND_DIR" ]; then
        print_info "复制后端文件..."
        cp -r "$BACKEND_DIR" "$OUTPUT_DIR/"
        print_success "后端文件已复制"
    fi

    print_success "打包完成! 输出目录: $OUTPUT_DIR"
}

# 清理
clean_all() {
    print_step "清理构建文件..."

    rm -rf dist/
    rm -rf release/

    if [ -d "gosrc" ]; then
        cd gosrc
        rm -rf .testcoverage
        rm -f coverage.out coverage.html
        cd ..
    fi

    if [ -d "electron_quasar" ]; then
        cd electron_quasar
        rm -rf dist/
        rm -rf .quasar/
        rm -rf node_modules/.cache/
        cd ..
    fi

    print_success "清理完成"
}

# 显示帮助
show_help() {
    print_header
    echo "用法: $0 [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  init           初始化项目依赖"
    echo "  check          代码质量检查"
    echo "  test           运行测试"
    echo "  build          构建项目"
    echo "  build:backend  仅构建后端"
    echo "  build:frontend 仅构建前端"
    echo "  package        打包完整应用"
    echo "  clean          清理构建文件"
    echo "  help           显示帮助信息"
    echo ""
    echo "选项 (用于 build 命令):"
    echo "  web       Web 版本"
    echo "  console   控制台版本"
    echo "  electron  Electron 版本"
    echo "  linux     Linux 版本"
    echo "  all       所有平台版本"
    echo ""
    echo "示例:"
    echo "  $0 init                 # 初始化项目"
    echo "  $0 check                # 代码检查"
    echo "  $0 test                 # 运行测试"
    echo "  $0 build                # 构建项目"
    echo "  $0 build:backend web     # 构建后端 Web 版本"
    echo "  $0 build:frontend electron # 构建前端 Electron 版本"
    echo "  $0 package              # 打包应用"
    echo "  $0 clean                # 清理"
    echo ""
}

# 主函数
main() {
    print_header

    case "$1" in
        init)
            check_go
            check_node
            init_submodules
            ;;
        check)
            check_go
            check_node
            check_code
            ;;
        test)
            check_go
            check_node
            run_tests
            ;;
        build)
            check_go
            check_node
            build_backend "$2"
            build_frontend "$2"
            ;;
        build:backend)
            check_go
            build_backend "$2"
            ;;
        build:frontend)
            check_node
            build_frontend "$2"
            ;;
        package)
            package_app
            ;;
        clean)
            clean_all
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac

    echo ""
    print_success "所有任务完成!"
    echo ""
}

# 执行主函数
main "$@"
