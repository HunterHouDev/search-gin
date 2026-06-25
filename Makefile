# ---------------------------------------------------------------------------
# search-gin Makefile
# Targets: dev | build | test | lint | clean | frontend-dev | help
# ---------------------------------------------------------------------------
.PHONY: dev build test lint clean frontend-dev help

APP_NAME   := search-gin
OUT_DIR    := qapp
OUT_BIN    := $(OUT_DIR)/appQuaser.exe
LDFLAGS    := -ldflags "-H=windowsgui -s -w"

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS=":.*## "; printf "\033[36mUsage:\033[0m\n  make \033[32m<target>\033[0m\n\n"; \
		{printf "  \033[32m%-20s\033[0m %s\n", $$1, $$2}}'

dev: ## 开发模式运行后端（热重载需 air）
	go run main.go

frontend-dev: ## 启动前端开发服务器（端口 9000，代理 API → :10081）
	cd frontend && quasar dev

build: ## 生产构建：前端 → Go embed → 单二进制
	@echo "==> Building frontend..."
	cd frontend && yarn build
	@echo "==> Copying frontend assets..."
	cp -r frontend/dist/spa/* dist/
	@echo "==> Building Go binary (prod)..."
	go build -tags=prod $(LDFLAGS) -o $(OUT_BIN) .
	@echo "==> Done: $(OUT_BIN)"

build-quick: ## 仅构建 Go（不重建前端）
	go build -tags=prod $(LDFLAGS) -o $(OUT_BIN) .

test: ## 运行全部测试
	go test ./... -v -count=1 -timeout=120s

test-short: ## 快速测试（跳过耗时用例）
	go test ./... -count=1 -timeout=30s -short

lint: ## 运行 lint（Go + 前端）
	golangci-lint run ./... --timeout=5m
	cd frontend && npx eslint --ext .js,.ts,.vue ./

vet: ## go vet 检查
	go vet ./...

frontend-lint: ## 前端 lint
	cd frontend && npx eslint --ext .js,.ts,.vue ./

frontend-tsc: ## TypeScript 类型检查
	cd frontend && npx vue-tsc --noEmit

clean: ## 清理构建产物
	rm -rf dist/*
	rm -f $(OUT_BIN)
	go clean -cache
	@echo "==> Cleaned"

tidy: ## 整理 Go 依赖
	go mod tidy
	go mod verify
