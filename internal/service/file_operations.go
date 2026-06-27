package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"slices"
	"strings"
)

// SetMovieType 设置电影类型
func (s *searchService) SetMovieType(movie model.FileItem, movieType string) utils.Result {
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
		return s.notifyFileChanged(movie, updated, "type_change")
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newSuffix := newMovieType + suffix
	extIdx := strings.LastIndex(movie.Path, suffix)
	if extIdx < 0 {
		extIdx = len(movie.Path)
	}
	newFilePath := movie.Path[:extIdx] + newSuffix
	newName := strings.TrimSuffix(newFilePath, suffix)

	updated, err := movie.RenameAll(newFilePath, newName)
	if err != nil {
		return utils.NewFailByMsg("重命名视频失败: " + err.Error())
	}
	return s.notifyFileChanged(movie, updated, "type_change")
}

// AddTag 添加标签
func (s *searchService) AddTag(id string, tag string) utils.Result {
	movie := s.engine.FindById(id)
	newTags := strings.Split(tag, ",")

	if len(movie.Tags) > 0 {
		originTagStr := utils.GetTagStr(movie.Path)
		// 检查每个新标签是否已存在，全部已存在则直接返回
		allExist := true
		for _, nt := range newTags {
			if !slices.Contains(movie.Tags, nt) {
				allExist = false
				break
			}
		}
		if allExist {
			res := utils.NewSuccessByMsg("已添加")
			res.Data = movie
			return res
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
		return s.notifyFileChanged(movie, updated, "tag_change")
	}

	tag = strings.TrimSpace(tag)
	if tag == "" {
		res := utils.NewSuccessByMsg("标签为空，跳过")
		res.Data = movie
		return res
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	extIdx := strings.LastIndex(movie.Path, suffix)
	if extIdx < 0 {
		extIdx = len(movie.Path)
	}
	newFilePath := movie.Path[:extIdx] + "《" + tag + "》" + suffix
	newFilePath = strings.Replace(newFilePath, "《》", "", 1)
	newName := strings.TrimSuffix(newFilePath, suffix)

	updated, err := movie.RenameAll(newFilePath, newName)
	if err != nil {
		return utils.NewFailByMsg("重命名失败: " + err.Error())
	}
	return s.notifyFileChanged(movie, updated, "tag_change")
}

// ClearTag 清除标签
func (s *searchService) ClearTag(id string, tag string) utils.Result {
	movie := s.engine.FindById(id)
	if len(movie.Tags) == 0 {
		res := utils.NewSuccessByMsg("执行成功")
		res.Data = movie
		return res
	}

	// 按逗号分割后精确匹配，避免子串误删（如清除 "comedy" 时误伤 "comedy-drama"）
	originTagStr := utils.GetTagStr(movie.Path)
	originalTags := strings.Split(originTagStr, ",")
	var remaining []string
	for _, t := range originalTags {
		if strings.TrimSpace(t) != strings.TrimSpace(tag) {
			remaining = append(remaining, t)
		}
	}

	var newTagStr string
	if len(remaining) > 0 {
		newTagStr = strings.Join(remaining, ",")
	}

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
	return s.notifyFileChanged(movie, updated, "tag_change")
}

// Rename 重命名文件
func (s *searchService) Rename(movie model.FileEdit) utils.Result {
	res := utils.NewSuccess()
	movieLib := s.engine.FindById(movie.Id)
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
			res = DownJpgMakePng(newPath, movie.Jpg, false)
			if !res.IsSuccess() {
				return res
			}
		}
		res = DownJpgAsPng(newPath, movie.Png)
		if !res.IsSuccess() {
			return res
		}
	} else if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
		res = DownJpgMakePng(newPath, movie.Jpg, true)
		if !res.IsSuccess() {
			return res
		}
	}
	return s.notifyFileChanged(movieLib, updated, "rename")
}

// Move 移动文件到新目录
func (s *searchService) Move(id string, newDir string, title string) utils.Result {
	res := utils.NewSuccess()
	movieLib := s.engine.FindById(id)
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
		if err := os.MkdirAll(newDir, 0755); err != nil {
			utils.ErrorFormat("创建目录失败 %s: %v", newDir, err)
		}
	}
	newPath := newDir + utils.PathSeparator + title + "." + movieLib.FileType
	newBaseName := newDir + utils.PathSeparator + title
	updated, err := movieLib.RenameAll(newPath, newBaseName)
	if err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}
	return s.notifyFileChanged(movieLib, updated, "move")
}

// Delete 删除文件（物理删除 + 索引移除 + SSE 通知）
// 先删磁盘再删索引，避免磁盘删除失败后索引已丢失导致状态不一致
func (s *searchService) Delete(id string) utils.Result {
	file := s.engine.FindById(id)
	if file.IsNull() {
		return utils.NewFailByMsg("文件不存在")
	}
	s.DeleteFilesOnDisk(file.DirPath, file.Title)
	s.engine.DeleteOnIndex(file) // 入队索引删除，由 flushLoop 批量应用
	s.events.Broadcast("file_changed", map[string]interface{}{
		"action": "delete",
		"id":     id,
		"path":   file.Path,
	})
	return utils.NewSuccessByMsg("删除成功")
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
func (s *searchService) notifyFileChanged(oldFile, updated model.FileItem, action string) utils.Result {
	s.engine.ReplaceFileOnIndex(oldFile, updated)
	res := utils.NewSuccessByMsg("执行成功")
	res.Data = updated
	s.events.Broadcast("file_changed", map[string]interface{}{
		"action": action,
		"id":     updated.Id,
		"old":    oldFile.Path,
		"new":    updated.Path,
	})
	return res
}
