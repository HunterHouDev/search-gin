package handler

import (
	"net/http"
	"os/exec"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetPlay(c *gin.Context) {
	id := c.Param("id")
	file := UseApp().search.FindById(id)
	if file.IsNull() {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	sanitizePath, ok := validatePathOrRespond(c, file.Path, "文件路径不在允许范围内")
	if !ok {
		return
	}

	utils.InfoFormat("GetPlay [%v]", sanitizePath)

	setting := UseApp().config.Get()
	if setting.SystemPlayer == "ffplay" {
		go func() {
			defer utils.RecoverPanic()
			params := []string{"-window_title", file.Title,
				"-alwaysontop",
				"-seek_interval", "30",
				"-stats",
			}
			if len(setting.SystemPlayerWidth) > 0 {
				arr := strings.Split(setting.SystemPlayerWidth, ",")
				params = append(params, "-x", arr[0])
				if len(arr) > 1 {
					params = append(params, "-y", arr[1])
				}
			}
			if len(setting.SystemPlayerVolumn) > 0 {
				params = append(params, "-volume", setting.SystemPlayerVolumn)
			}

			ffplayPath := "./ffplay.exe"
			if service.GetWorkDir() != "" {
				ffplayPath = filepath.Join(service.GetWorkDir(), "ffplay.exe")
			}

			params = append(params, sanitizePath)
			cmd := exec.Command(ffplayPath, params...)
			if err := cmd.Start(); err != nil {
				utils.InfoFormat("播放失败: %v, 错误: %v", sanitizePath, err)
				return
			}
			_ = cmd.Wait()
		}()
		c.JSON(http.StatusOK, utils.NewSuccessByMsg("播放成功"))
	} else if setting.SystemPlayer != "" {
		utils.ExecCmdStart(sanitizePath)
		c.JSON(http.StatusOK, utils.NewSuccessByMsg("播放成功"))
	} else {
		utils.InfoFormat("播放失败: 未配置播放器, 路径: %v", sanitizePath)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("请先在设置中配置播放器"))
	}
}

func GetInfo(c *gin.Context) {
	id := c.Param("id")
	if service.HandleRemoteByID(c, id, "info") {
		return
	}
	file := UseApp().search.FindById(id)
	if file.IsNull() {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}
	// 生成新的 streamToken 填充 URL，确保前端拿到的是有效 token
	items := []model.FileItem{file}
	service.FillURLs(c, items)
	c.JSON(http.StatusOK, items[0])
}

func GetAuthorImage(c *gin.Context) {
	path := c.Param("name")
	author := UseApp().search.FindAuthorByName(path)
	if author.IsNotEmpty() {
		for _, v := range author.Images {
			if v == "" || !utils.ExistsFiles(v) {
				continue
			}
			if validated, err := utils.ValidatePath(v, UseApp().config.Get().Dirs); err == nil {
				c.File(validated)
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, utils.NewFailByMsg("未找到作者图片"))
}
