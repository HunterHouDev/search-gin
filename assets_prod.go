//go:build prod

package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed dist ffmpeg.exe ffplay.exe setting.json
var staticFiles embed.FS

func ExtractAll(dest string) error {
	if err := ExtractDist(dest); err != nil {
		return err
	}
	if err := ExtractFfmpeg(dest); err != nil {
		return err
	}
	if err := ExtractFfplay(dest); err != nil {
		return err
	}
	return ExtractSetting(dest)
}

func ExtractDist(dest string) error {
	return fs.WalkDir(staticFiles, "dist", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "dist" {
			return nil
		}
		relativePath := path[5:]
		if d.IsDir() {
			dirPath := filepath.Join(dest, "dist", relativePath)
			return os.MkdirAll(dirPath, 0755)
		}
		content, err := staticFiles.ReadFile(path)
		if err != nil {
			return err
		}
		filePath := filepath.Join(dest, "dist", relativePath)
		return os.WriteFile(filePath, content, 0644)
	})
}

func ExtractFfmpeg(dest string) error {
	content, err := staticFiles.ReadFile("ffmpeg.exe")
	if err != nil {
		return err
	}
	filePath := filepath.Join(dest, "ffmpeg.exe")
	return os.WriteFile(filePath, content, 0755)
}

func ExtractFfplay(dest string) error {
	content, err := staticFiles.ReadFile("ffplay.exe")
	if err != nil {
		return err
	}
	filePath := filepath.Join(dest, "ffplay.exe")
	return os.WriteFile(filePath, content, 0755)
}

func ExtractSetting(dest string) error {
	content, err := staticFiles.ReadFile("setting.json")
	if err != nil {
		return err
	}
	filePath := filepath.Join(dest, "setting.json")
	return os.WriteFile(filePath, content, 0644)
}

func ReadFile(path string) ([]byte, error) {
	return staticFiles.ReadFile(path)
}

func Open(path string) (fs.File, error) {
	return staticFiles.Open(path)
}
