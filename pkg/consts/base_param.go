package consts

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"time"
)

var LastScanTime time.Time

var QueryTypes []string

var MovieFields = utils.InterfaceFields(model.Movie{})

var Types = []string{PNG, JPG, GIF, XLSX, TXT, MP4, WMV, MKV, AVI, JAVA, XML}
var Images = []string{PNG, JPG, GIF}

var IndexHtml = "./dist/index.html"
var StaticFs = map[string]string{
	"/css":    "./dist/css",
	"/js":     "./dist/js",
	"/assets": "./dist/assets",
}

// IndexDone 索引构建中标记
var IndexDone = int32(0)

var TempImage = make(map[string]model.Movie)
var TempImageMutex sync.RWMutex // 保护TempImage的并发访问

var TransferTask = map[time.Time]model.TransferTaskModel{}
var TransferTaskMutex sync.RWMutex // 保护TransferTask的并发访问

// PNG Base Dictory
const PNG = "png"
const JPG = "jpg"
const GIF = "gif"
const XLSX = "xlsx"
const TXT = "txt"
const MP4 = "mp4"
const WMV = "wmv"
const MKV = "mkv"
const AVI = "avi"
const JAVA = "java"
const XML = "xml"

var PortNo = ":10081"
var PortNo2 = ":10082"
var PortNo3 = ":10083"
