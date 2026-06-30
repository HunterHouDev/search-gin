//go:build !prod

// 开发模式：嵌入资源函数为空操作，前端从磁盘 ./dist/ 直接读取
package main

import (
	"io/fs"
)

// Dev mode: no-op，生产模式由 assets_prod.go 从 embed 读取
func ExtractAll(dest string) error {
	return nil
}

func ExtractDist(dest string) error {
	return nil
}

func ExtractFfmpeg(dest string) error {
	return nil
}

func ExtractFfplay(dest string) error {
	return nil
}

func ExtractSetting(dest string) error {
	return nil
}

func ReadFile(path string) ([]byte, error) {
	return nil, nil
}

func Open(path string) (fs.File, error) {
	return nil, nil
}
