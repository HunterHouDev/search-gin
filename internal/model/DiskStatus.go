package model

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

type DiskStatus struct {
	Path    string  `json:"Path"`
	All     int64   `json:"All"`
	Used    int64   `json:"Used"`
	Free    int64   `json:"Free"`
	Percent float64 `json:"Percent"`
}

func GetDiskUsage(path string) (*DiskStatus, error) {
	rootPath := getRootPath(path)
	usage, err := disk.Usage(rootPath)
	if err != nil {
		return nil, err
	}
	return &DiskStatus{
		Path:    path,
		All:     int64(usage.Total),
		Used:    int64(usage.Used),
		Free:    int64(usage.Free),
		Percent: usage.UsedPercent,
	}, nil
}

func getRootPath(path string) string {
	if runtime.GOOS == "windows" {
		path = filepath.ToSlash(path)
		parts := strings.Split(path, "/")
		if len(parts) > 0 && len(parts[0]) == 2 && parts[0][1] == ':' {
			return strings.ToUpper(parts[0]) + "/"
		}
	}
	return filepath.VolumeName(path)
}
