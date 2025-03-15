#!/bin/bash

# 日志输出函数
log() {
    local level=$1
    local message=$2
    printf "[%s] %s\n" "$level" "$message"
}

# 验证输入参数是否为有效的数字
if ! [[ $1 =~ ^[0-4]$ ]]; then
    log "ERROR" "输入参数无效，请输入 0 到 4 之间的数字。"
    exit 1
fi

levelKey=${1}
log "INFO" "执行参数：levelKey[${levelKey}]"

# 根据 levelKey 设置 levelValue
get_level_value() {
    case $1 in
        0)
            echo "none"
            ;;
        1)
            echo "vue"
            ;;
        2)
            echo "vueGo"
            ;;
        *)
            echo ""
            ;;
    esac
}

levelValue=$(get_level_value "$levelKey")
log "INFO" "执行级别：levelKey[${levelKey}]--levelValue[${levelValue}]"

# 清理 qapp 目录
clean_qapp() {
    log "INFO" "清理 qapp 目录..."
    cd ./qapp || { log "ERROR" "无法进入 qapp 目录"; exit 1; }
    rm -rf dist
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

# 打包前端文件
build_frontend() {
    log "INFO" "开始打包前端文件..."
    cd electron_quasar || { log "ERROR" "无法进入 electron_quasar 目录"; exit 1; }
    if ! yarn build; then
        log "ERROR" "前端文件打包失败"
        exit 1
    fi
    log "INFO" "打包完成，开始移动到 app 目录"
    cp -R dist/spa ../qapp/dist || { log "ERROR" "移动前端文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
}

# 打包 APP
build_app() {
    log "INFO" "移动完成，打包 APP..."
    cd gosrc || { log "ERROR" "无法进入 gosrc 目录"; exit 1; }
    if ! go build -o ../qapp/appQuaser.exe -ldflags "-H=windowsgui" -tags=prod; then
        log "ERROR" "APP 打包失败"
        exit 1
    fi
    log "INFO" "移动配置文件 '*.*(1)'"
    cp setting.json '../qapp/setting.json(1)' || { log "ERROR" "移动 setting.json 文件失败"; exit 1; }
    cp ffmpeg.exe '../qapp/ffmpeg.exe' || { log "ERROR" "移动 ffmpeg.exe 文件失败"; exit 1; }
    cd .. || { log "ERROR" "无法返回上级目录"; exit 1; }
    log "INFO" "APP 打包完成！！！"
}

# 移动源到 Election 代码目录
move_to_electron() {
    log "INFO" "移动源到 Election 代码目录"
    cp -R qapp electron_quasar/src-electron/icons || { log "ERROR" "移动源文件到 Election 代码目录失败"; exit 1; }
    cd electron_quasar || { log "ERROR" "无法进入 electron_quasar 目录"; exit 1; }
    if ! yarn topc; then
        log "ERROR" "Electron 打包失败"
        exit 1
    fi
    log "INFO" "Electron Package OVER"
}

# 执行清理操作
clean_qapp

# 根据 levelKey 执行相应的操作
case $levelKey in
    1|2|3|4)
        build_frontend
        ;;
esac

case $levelKey in
    2|4)
        build_app
        ;;
esac

case $levelKey in
    3|4)
        move_to_electron
        ;;
esac

log "INFO" "SUCCESS,OVER !!!"
