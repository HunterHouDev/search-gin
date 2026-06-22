package service

import (
	"sync/atomic"
	"time"
)

var lastScanTime atomic.Int64

// SetLastScanTime 设置最近扫描时间（并发安全）
func SetLastScanTime(t time.Time) {
	lastScanTime.Store(t.UnixNano())
}

// GetLastScanTime 获取最近扫描时间（并发安全）
func GetLastScanTime() time.Time {
	return time.Unix(0, lastScanTime.Load())
}

// IndexNumber 索引构建中的目录数量（并发安全）
var IndexNumber atomic.Int32

var PortNo = ":10081"
var FilePortNo = ":10082"
