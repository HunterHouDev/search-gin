#!/bin/bash

log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

clean_qapp() {
    log "INFO" "清理 qapp 目录..."
    rm -rf qapp/dist
    mkdir -p qapp/dist
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
    if ! go build -o qapp/appQuaser.exe -ldflags "-H=windowsgui" -tags=prod; then
        log "ERROR" "Go 应用打包失败"
        exit 1
    fi
    log "INFO" "Go 应用打包完成"
}

build_electron() {
    log "INFO" "开始打包 Electron..."
    cp -R qapp frontend/src-electron/icons || { log "ERROR" "移动源文件到 Electron 代码目录失败"; exit 1; }
    cd frontend || { log "ERROR" "无法进入 frontend 目录"; exit 1; }
    if ! yarn topc; then
        log "ERROR" "Electron 打包失败"
        exit 1
    fi
    log "INFO" "Electron 打包完成"
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

log "INFO" "开始 Electron 打包流程..."

clean_qapp

build_frontend

build_go_app

build_electron

log "INFO" "Electron 打包完成！"
