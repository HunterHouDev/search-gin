#!/bin/bash

log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

clean_qapp_dist() {
    log "INFO" "清理 qapp/dist 目录..."
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
    log "INFO" "打包完成，开始移动到 qapp 目录"
    cp -R dist/spa ../qapp/dist || { log "ERROR" "移动前端文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
    log "INFO" "前端代码打包完成！"
}

log "INFO" "开始前端代码打包流程..."

clean_qapp_dist

build_frontend

log "INFO" "前端代码打包完成！"
