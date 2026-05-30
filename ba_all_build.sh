#!/bin/bash

log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

clean_dist() {
    log "INFO" "清理 dist 目录..."
    rm -rf dist
    mkdir -p dist
}

build_frontend() {
    log "INFO" "开始打包前端文件..."
    cd frontend || { log "ERROR" "无法进入 frontend 目录"; exit 1; }
    if ! yarn build; then
        log "ERROR" "前端文件打包失败"
        exit 1
    fi
    log "INFO" "打包完成，开始移动到 dist 目录"
    cp -R dist/spa/* ../dist || { log "ERROR" "移动前端文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

build_go_app() {
    log "INFO" "开始打包 Go 应用..."
    if [ ! -d "dist" ]; then
        log "ERROR" "dist 目录不存在，请先执行前端打包"
        exit 1
    fi
    if ! go build -o qapp/appQuaser.exe -ldflags "-H=windowsgui -s -w" -tags=prod; then
        log "ERROR" "Go 应用打包失败"
        exit 1
    fi
    log "INFO" "Go 应用打包完成"
}

log "INFO" "开始打包流程..."

clean_dist

build_frontend

build_go_app

log "INFO" "打包流程完成！生成的可执行文件：qapp/appQuaser.exe"
