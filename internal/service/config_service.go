package service

import (
	"bufio"
	"encoding/json"

	"os"
	"path/filepath"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

// InitSetting 读取配置文件并初始化全局设置
func InitSetting() {
	curDir, _ := filepath.Abs(".")
	osSetting := consts.OSSetting
	settingPath := curDir + utils.PathSeparator + consts.OSSetting.SelfPath
	dict := ReadDictionaryFromJson(settingPath)
	dict.SelfPath = osSetting.SelfPath
	// ip := GetIpAddr()
	dict.ControllerHost =   consts.PortNo
	dict.ImageHost =  consts.PortNo2
	dict.StreamHost =  consts.PortNo3

	// 如果启用硬件加速，主动检测并同步模式名称
	if dict.HardwareAcceleration {
		FileApp.detectHwAccel()
		dict.HardwareAccelMode = GetHwAccelModeName()
	}

	consts.OSSetting = dict
}

// FlushDictionary 将当前设置持久化到配置文件
func FlushDictionary(path string) {
	WriteDictionaryToJson(path, consts.OSSetting)
}

func ReadDictionaryFromJson(path string) model.Setting {
	reader, _ := os.ReadFile(path)
	dict := model.Setting{}
	err := json.Unmarshal(reader, &dict)
	if err != nil {
		return model.Setting{}
	}
	return dict
}
func WriteDictionaryToJson(path string, dict model.Setting) {
	data, _ := json.Marshal(dict)
	outStream, openErr := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if openErr != nil {
		utils.InfoFormat("openErr: %v", openErr)
		return
	}
	defer func(outStream *os.File) {
		err := outStream.Close()
		if err != nil {

		}
	}(outStream)
	writer := bufio.NewWriter(outStream)
	_, err := writer.Write(data)
	if err != nil {
		utils.InfoFormat("写入配置文件失败: %v", err)
		return
	}
	err = writer.Flush()
	if err != nil {
		utils.InfoFormat("刷新配置文件缓冲失败: %v", err)
		return
	}
}

