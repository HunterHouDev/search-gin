package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
	"slices"
	"strings"
)

// SetMovieType 设置电影类型
func (fo *searchService) SetMovieType(movie model.FileItem, movieType string) utils.Result {
	newMovieType := "{{" + movieType + "}}"

	if movie.MovieType != "" && movie.MovieType != "无" {
		originVideoType := utils.GetMovieType(movie.Path)
		if originVideoType == movieType {
			res := utils.NewSuccessByMsg("执行成功")
			res.Data = movie
			return res
		}

		newFilePath := filepath.Join(filepath.Dir(movie.Path), strings.Replace(filepath.Base(movie.Path), originVideoType, movieType, 1))
		newName := strings.TrimSuffix(newFilePath, "."+utils.GetSuffix(movie.Path))

		updated, err := movie.RenameAll(newFilePath, newName)
		if err != nil {
			return utils.NewFailByMsg("重命名失败: " + err.Error())
		}
		return notifyFileChanged(movie, updated, "type_change")
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newSuffix := newMovieType + suffix
	newFilePath := strings.Replace(movie.Path, suffix, newSuffix, 1)
	newName := strings.TrimSuffix(newFilePath, suffix)

	updated, err := movie.RenameAll(newFilePath, newName)
	if err != nil {
		return utils.NewFailByMsg("重命名视频失败: " + err.Error())
	}
	return notifyFileChanged(movie, updated, "type_change")
}

// AddTag 添加标签
func (fo *searchService) AddTag(id string, tag string) utils.Result {
	movie := SearchEngine.FindById(id)
	newTags := strings.Split(tag, ",")

	if len(movie.Tags) > 0 {
		originTagStr := utils.GetTagStr(movie.Path)
		for _, t := range movie.Tags {
			if t == tag {
				res := utils.NewSuccessByMsg("已添加")
				res.Data = movie
				return res
			}
		}

		newTagStr := originTagStr
		for _, str := range newTags {
			if !slices.Contains(movie.Tags, str) {
				newTagStr += "," + str
			}
		}
		newTagStr = "《" + newTagStr + "》"
		originTagStr = "《" + originTagStr + "》"

		newFilePath := strings.ReplaceAll(movie.Path, originTagStr, newTagStr)
		newName := strings.TrimSuffix(newFilePath, "."+utils.GetSuffix(movie.Path))
		updated, err := movie.RenameAll(newFilePath, newName)
		if err != nil {
			return utils.NewFailByMsg("重命名失败: " + err.Error())
		}
		return notifyFileChanged(movie, updated, "tag_change")
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newFilePath := strings.ReplaceAll(movie.Path, suffix, "《"+tag+"》"+suffix)
	newFilePath = strings.ReplaceAll(newFilePath, "《》", "")
	newName := strings.TrimSuffix(newFilePath, suffix)

	updated, err := movie.RenameAll(newFilePath, newName)
	if err != nil {
		return utils.NewFailByMsg("重命名失败: " + err.Error())
	}
	return notifyFileChanged(movie, updated, "tag_change")
}

// ClearTag 清除标签
func (fo *searchService) ClearTag(id string, tag string) utils.Result {
	movie := SearchEngine.FindById(id)
	if len(movie.Tags) == 0 {
		res := utils.NewSuccessByMsg("执行成功")
		res.Data = movie
		return res
	}

	originTagStr := utils.GetTagStr(movie.Path)
	newTagStr := strings.ReplaceAll(originTagStr, tag, "")
	if len(movie.Tags) == 1 {
		newTagStr = ""
	}
	newTagStr = strings.TrimSuffix(newTagStr, ",")
	newTagStr = strings.TrimPrefix(newTagStr, ",")
	var path string
	if newTagStr == "" {
		suffix := "." + utils.GetSuffix(movie.Path)
		path = strings.ReplaceAll(movie.Path, "《"+originTagStr+"》"+suffix, suffix)
	} else {
		path = strings.ReplaceAll(movie.Path, "《"+originTagStr+"》", "《"+newTagStr+"》")
	}

	newName := strings.TrimSuffix(path, "."+movie.FileType)
	updated, err := movie.RenameAll(path, newName)
	if err != nil {
		return utils.NewFailByMsg("重命名失败" + path)
	}
	return notifyFileChanged(movie, updated, "tag_change")
}

// Rename 重命名文件
func (fo *searchService) Rename(movie model.FileEdit) utils.Result {
	res := utils.NewSuccess()
	movieLib := SearchEngine.FindById(movie.Id)
	if movieLib.IsNull() {
		res.FailByMsg("数据不存在")
		return res
	}
	oldPath := movieLib.Path
	if !utils.ExistsFiles(oldPath) {
		res.FailByMsg("文件不存在")
		return res
	}

	newPath := cleanPath(movieLib.DirPath)
	newDir := newPath
	if movie.MoveOut {
		if movie.Author != "" {
			arr := strings.Split(newPath, utils.PathSeparator)
			if utils.HasItem(arr, movie.Author) {
				arr2 := strings.Split(newPath, movie.Author)
				newDir = arr2[0]
			}
			newDir += utils.PathSeparator + movie.Author
		}
		if movie.Title != "" {
			newDir += utils.PathSeparator
			newCode := movie.Code
			if strings.HasPrefix(newCode, "-") {
				newCode = strings.Replace(newCode, "-", "", 1)
			}
			newDir += choose2To1(!strings.HasPrefix(movie.Title, movie.Author),
				choose2To1(movie.Author != "", movie.Author, ""), "")
			newDir += choose2To1(!strings.Contains(movie.Title, newCode),
				choose2To1(newCode != "", " "+newCode, ""), "")
			newTitle := strings.Split(movie.Title, "{{")
			newTitleStart := newTitle[0]
			if len(newTitleStart) > 10 {
				newTitleStart = newTitleStart[:10]
			}
			newDir += " " + cleanPath(newTitleStart)
		}
		if err := os.MkdirAll(newDir, 0755); err != nil {
			res.FailByMsg("执行失败")
			res.Data = err
			return res
		}
	}
	newPath = newDir + utils.PathSeparator + movie.Name

	// 重命名主文件 + 附属文件
	newBaseName := strings.TrimSuffix(newPath, "."+utils.GetSuffix(newPath))
	updated, err := movieLib.RenameAll(newPath, newBaseName)
	if err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}

	// 下载 JPG/PNG
	if movie.Png != "" && strings.HasPrefix(movie.Png, "http") {
		if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
			res = Downloader.DownJpgMakePng(newPath, movie.Jpg, false)
			if !res.IsSuccess() {
				return res
			}
		}
		res = Downloader.DownJpgAsPng(newPath, movie.Png)
		if !res.IsSuccess() {
			return res
		}
	} else if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
		res = Downloader.DownJpgMakePng(newPath, movie.Jpg, true)
		if !res.IsSuccess() {
			return res
		}
	}
	return notifyFileChanged(movieLib, updated, "rename")
}

// Move 移动文件到新目录
func (fo *searchService) Move(id string, newDir string, title string) utils.Result {
	res := utils.NewSuccess()
	movieLib := SearchEngine.FindById(id)
	if movieLib.IsNull() {
		res.FailByMsg("数据不存在")
		return res
	}
	oldPath := movieLib.Path
	if !utils.ExistsFiles(oldPath) {
		res.FailByMsg("文件不存在")
		return res
	}
	if !utils.ExistsFiles(newDir) {
		os.MkdirAll(newDir, 0755)
	}
	newPath := newDir + utils.PathSeparator + title + "." + movieLib.FileType
	newBaseName := newDir + utils.PathSeparator + title
	updated, err := movieLib.RenameAll(newPath, newBaseName)
	if err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}
	return notifyFileChanged(movieLib, updated, "move")
}

// Delete 删除文件
func (fo *searchService) Delete(id string) {
	file := SearchEngine.FindById(id)
	if file.IsNull() {
		return
	}
	SearchApp.DeleteOne(file.DirPath, file.Title)
	sse.BroadcastEvent("file_changed", map[string]interface{}{
		"action": "delete",
		"id":     id,
		"path":   file.Path,
	})
}

// ── 私有辅助函数 ──────────────────────────────────────────────────

var (
	// 预编译的 replacer，避免热路径上多次分配
	pathReplacer = strings.NewReplacer("《", "", "》", "", "{{", "", "}}", "")
)

// cleanPath 清理文件名中的标记符号
func cleanPath(name string) string {
	return pathReplacer.Replace(strings.TrimSpace(name))
}

// choose2To1 三元选择
func choose2To1(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// notifyFileChanged 更新索引 + SSE 通知前端，返回 Result
func notifyFileChanged(oldFile, updated model.FileItem, action string) utils.Result {
	SearchEngine.ReplaceFile(oldFile, updated)
	res := utils.NewSuccessByMsg("执行成功")
	res.Data = updated
	sse.BroadcastEvent("file_changed", map[string]interface{}{
		"action": action,
		"id":     updated.Id,
		"old":    oldFile.Path,
		"new":    updated.Path,
	})
	return res
}
