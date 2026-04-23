#!/bin/bash

# 日志输出函数
log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

# 清理 qapp/dist 目录
clean_qapp_dist() {
    log "INFO" "清理 qapp/dist 目录..."
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
    log "INFO" "打包完成，开始移动到 qapp 目录"
    cp -R dist/spa ../qapp/dist || { log "ERROR" "移动前端文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
    log "INFO" "前端文件打包完成！"
}

# 主执行流程
log "INFO" "开始前端代码打包流程..."

# 1. 清理 qapp/dist 目录
clean_qapp_dist

# 2. 打包前端文件
build_frontend

log "INFO" "前端代码打包完成！"