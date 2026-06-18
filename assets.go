package main

import (
	"net/http"
	"os"
	"path/filepath"

	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// extractAssets 解压 setting.json（同步）及前端静态文件和二进制工具（异步）
func extractAssets(workDir string) chan struct{} {
	assetsExtracted := make(chan struct{})

	extractIfNotExist := func(path, label string, fn func(string) error) {
		if _, err := os.Stat(filepath.Join(workDir, path)); os.IsNotExist(err) {
			utils.InfoFormat("开始解压 %s...", label)
			if err := fn(workDir); err != nil {
				utils.InfoFormat("解压 %s 失败: %v", label, err)
				os.Exit(1)
			}
			utils.InfoFormat("%s 解压完成", label)
		} else {
			utils.InfoFormat("%s 已存在，跳过解压", label)
		}
	}

	extractIfNotExist("setting.json", "setting.json", ExtractSetting)

	go func() {
		defer utils.RecoverPanic()
		defer close(assetsExtracted)

		for _, a := range []struct {
			path  string
			label string
			fn    func(string) error
		}{
			{filepath.Join("dist", "index.html"), "前端静态文件", ExtractDist},
			{"ffmpeg.exe", "ffmpeg.exe", ExtractFfmpeg},
			{"ffplay.exe", "ffplay.exe", ExtractFfplay},
		} {
			extractIfNotExist(a.path, a.label, a.fn)
		}
	}()

	return assetsExtracted
}

// loadStaticFiles 加载前端静态文件（延迟执行，等待解压完成）
func loadStaticFiles(app *gin.Engine, workDir string, extracted <-chan struct{}) {
	<-extracted

	indexHtml := filepath.Join(workDir, "dist", "index.html")
	if !utils.ExistsFiles(indexHtml) {
		utils.InfoFormat("static not exists:%s", indexHtml)
		return
	}
	utils.InfoFormat("static exists:%s", indexHtml)
	app.LoadHTMLFiles(indexHtml)

	staticFs := map[string]string{
		"/css":    filepath.Join(workDir, "dist", "css"),
		"/js":     filepath.Join(workDir, "dist", "js"),
		"/assets": filepath.Join(workDir, "dist", "assets"),
	}
	for k, v := range staticFs {
		app.StaticFS(k, http.Dir(v))
		utils.InfoFormat("static exists:%s", k)
	}
}
