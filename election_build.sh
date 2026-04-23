#!/bin/bash

# 日志输出函数
log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

# 清理 qapp 目录
clean_qapp() {
    log "INFO" "清理 qapp 目录..."
    rm -rf qapp/dist
    mkdir -p qapp/dist
}

# 打包前端文件
build_frontend() {
    log "INFO" "开始打包前端文件..."
    cd electron_quasar || { log "ERROR" "无法进入 electron_quasar 目录"; exit 1; }
    if ! yarn build; then
        log "ERROR" "前端文件打包失败"
        exit 1
    fi
    log "INFO" "打包完成，开始移动到 gosrc 目录"
    # 复制 spa 目录下的所有文件到 gosrc/dist
    cp -R dist/spa/* ../gosrc/dist || { log "ERROR" "移动前端文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

# 打包 Go 应用
build_go_app() {
    log "INFO" "开始打包 Go 应用..."
    cd gosrc || { log "ERROR" "无法进入 gosrc 目录"; exit 1; }
    if ! go build -o ../qapp/appQuaser.exe -ldflags "-H=windowsgui" -tags=prod; then
        log "ERROR" "Go 应用打包失败"
        exit 1
    fi
    log "INFO" "Go 应用打包完成"
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

# 打包 Electron
build_electron() {
    log "INFO" "开始打包 Electron..."
    cp -R qapp electron_quasar/src-electron/icons || { log "ERROR" "移动源文件到 Electron 代码目录失败"; exit 1; }
    cd electron_quasar || { log "ERROR" "无法进入 electron_quasar 目录"; exit 1; }
    if ! yarn topc; then
        log "ERROR" "Electron 打包失败"
        exit 1
    fi
    log "INFO" "Electron 打包完成"
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

# 主执行流程
log "INFO" "开始 Electron 打包流程..."

# 1. 清理 qapp 目录
clean_qapp

# 2. 打包前端文件
build_frontend

# 3. 打包 Go 应用
build_go_app

# 4. 打包 Electron
build_electron

log "INFO" "Electron 打包完成！"