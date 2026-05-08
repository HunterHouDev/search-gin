package consts

import (
	"search-gin/internal/model"
	"sync"
)

var (
	OSSetting    = model.Setting{}
	settingMutex sync.RWMutex
)

func init() {
	OSSetting = model.Setting{
		IsDb:               true,
		IsJavBus:           false,
		EnableTimeScan:     true,
		SystemPlayerVolumn: "30",
		SystemPlayerWidth:  "1280",
		SelfPath:           "setting.json",
		ControllerHost:     "127.0.0.1:17001",
		ImageHost:          "127.0.0.1:17002",
		StreamHost:         "127.0.0.1:17003",
		BaseUrl:            "https://www.busjav.blog/",
		ImageUrl:           "",
		OMUrl:              "https://www.busjav.blog/",
		Remark:             "",
		SystemHtml:         "",
		Dirs: []string{
			"e://emby",
			"e://code",
		},
		Tags: []string{
			"東京熱",
		},
		ImageTypes: []string{GIF, PNG, JPG},
		DocsTypes:  []string{TXT, XLSX},
		VideoTypes: []string{AVI, MKV, WMV, MP4},
		Types:      []string{AVI, MKV, WMV, MP4, GIF, PNG, JPG, TXT, XLSX},
		Buttons:    []string{"刮图", "删除", "移动"},
		MovieTypes: []string{"骑兵", "步兵", "国产", "漫动"},
		Pages:      []string{"10", "12", "15", "27", "50", "100"},
	}
}

// GetOSSetting 获取系统配置（线程安全）
func GetOSSetting() model.Setting {
	settingMutex.RLock()
	defer settingMutex.RUnlock()
	return OSSetting
}

// SetOSSetting 设置系统配置（线程安全）
func SetOSSetting(setting model.Setting) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	OSSetting = setting
}
