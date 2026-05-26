package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
)

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
	if !utils.ExistsFiles(path) {
		_, err := os.Create(path)
		if err != nil {
			return
		}
	}
	outStream, openErr := os.OpenFile(path, os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if openErr != nil {
		fmt.Println("openErr", openErr)
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
		return
	}
	err = writer.Flush()
	if err != nil {
		return
	}
}

//func ReadDictionaryFromTxt(path string) model.Dictionary {
//	outStream, openErr := os.Open(path)
//	if openErr != nil {
//		fmt.Println("openErr", openErr)
//	}
//	defer func(outStream *os.File) {
//		err := outStream.Close()
//		if err != nil {
//
//		}
//	}(outStream)
//
//	reader := bufio.NewReader(outStream)
//	dict := model.NewDictionary()
//	for {
//		lineStr, err := reader.ReadString('\n')
//		if err != nil {
//			break
//		}
//		lineStr = strings.TrimSpace(lineStr)
//		if lineStr == "" {
//			continue
//		}
//		line := strings.Split(lineStr, "=")
//		dict.PutProperty(line[0], line[1])
//	}
//	return dict
//}
//func WriteDictionaryToText(path string, dict model.Dictionary) {
//	outStream, openErr := os.OpenFile(path, os.O_TRUNC|os.O_RDWR, os.ModePerm)
//	if openErr != nil {
//		fmt.Println("openErr", openErr)
//	}
//	defer func(outStream *os.File) {
//		err := outStream.Close()
//		if err != nil {
//
//		}
//	}(outStream)
//	writer := bufio.NewWriter(outStream)
//	for key, value := range dict.LibMap {
//		for _, v := range value {
//			_, err := writer.WriteString(key + "=" + v + "\n")
//			if err != nil {
//				return
//			}
//		}
//	}
//	err := writer.Flush()
//	if err != nil {
//		return
//	}
//}
