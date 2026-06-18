package service

import (
	"encoding/json"

	"os"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
)

// FlushDictionary 将当前设置持久化到配置文件
func FlushDictionary(path string) {
	WriteDictionaryToJson(path, consts.GetOSSetting())
}

func ReadDictionaryFromJson(path string) model.Setting {
	reader, err := os.ReadFile(path)
	if err != nil {
		utils.InfoFormat("读取配置文件失败: %v", err)
		return model.Setting{}
	}
	dict := model.Setting{}
	err = json.Unmarshal(reader, &dict)
	if err != nil {
		return model.Setting{}
	}
	return dict
}
func WriteDictionaryToJson(path string, dict model.Setting) {
	data, err := json.Marshal(dict)
	if err != nil {
		utils.InfoFormat("序列化配置文件失败: %v", err)
		return
	}
	err = os.WriteFile(path, data, os.ModePerm)
	if err != nil {
		utils.InfoFormat("写入配置文件失败: %v", err)
		return
	}
}

// newBool 返回指向给定 bool 值的指针
func newBool(v bool) *bool {
	return &v
}
