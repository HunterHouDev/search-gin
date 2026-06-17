package service

import (
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
)

// SetMovieType 设置电影类型
func (fs *searchService) SetMovieType(movie model.FileItem, movieType string) utils.Result {
	newMovieType := "{{" + movieType + "}}"

	if movie.MovieType != "" && movie.MovieType != "无" {
		originVideoType := utils.GetMovieType(movie.Path)
		if originVideoType == movieType {
			res := utils.NewSuccessByMsg("执行成功")
			res.Data = movie
			return res
		}

		originalPaths := []string{movie.Path, movie.Jpg, movie.Png, movie.Gif}
		newPaths := make([]string, len(originalPaths))
		for i, p := range originalPaths {
			if p != "" {
				newPaths[i] = filepath.Join(filepath.Dir(p), strings.ReplaceAll(filepath.Base(p), originVideoType, movieType))
			}
		}

		successCount := 0
		for i := range originalPaths {
			if originalPaths[i] == "" || !utils.ExistsFiles(originalPaths[i]) {
				continue
			}
			if err := os.Rename(originalPaths[i], newPaths[i]); err != nil {
				utils.InfoFormat("rename failed: %v", err)
				// 回滚已成功的操作
				for j := 0; j < successCount; j++ {
					if originalPaths[j] != "" && utils.ExistsFiles(newPaths[j]) {
						os.Rename(newPaths[j], originalPaths[j])
					}
				}
				return utils.NewFailByMsg("重命名失败: " + err.Error())
			}
			successCount++
		}
		updated := replaceIndexAfterRename(movie.Id, newPaths[0], movie.BaseDir)
		res := utils.NewSuccessByMsg("执行成功")
		res.Data = updated
		return res
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newSuffix := newMovieType + suffix
	newFilePath := strings.ReplaceAll(movie.Path, suffix, newSuffix)

	if err := os.Rename(movie.Path, newFilePath); err != nil {
		return utils.NewFailByMsg("重命名视频失败: " + err.Error())
	}

	newName := strings.TrimSuffix(newFilePath, suffix)
	for _, f := range []struct{ src, target string }{
		{movie.Png, newName + ".png"},
		{movie.Jpg, newName + ".jpg"},
		{movie.Gif, newName + ".gif"},
	} {
		if f.src != "" && utils.ExistsFiles(f.src) {
			if err := os.Rename(f.src, f.target); err != nil {
				utils.InfoFormat("rename failed: %v", err)
			}
		}
	}
	updated := replaceIndexAfterRename(movie.Id, newFilePath, movie.BaseDir)
	res := utils.NewSuccessByMsg("执行成功")
	res.Data = updated
	return res
}

// AddTag 添加标签
func (fs *searchService) AddTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
	newTags := strings.Split(tag, ",")

	if len(movie.Tags) > 0 {
		originTagStr := utils.GetTagStr(movie.Path)
		if originTagStr == tag || strings.Contains(originTagStr, tag) {
			res := utils.NewSuccessByMsg("已添加")
			res.Data = movie
			return res
		}

		newTagStr := originTagStr
		for _, str := range newTags {
			if !strings.Contains(originTagStr, str) {
				newTagStr += "," + str
			}
		}
		newTagStr = "《" + newTagStr + "》"
		originTagStr = "《" + originTagStr + "》"

		for _, file := range []string{movie.Path, movie.Jpg, movie.Png, movie.Gif} {
			newPath := strings.ReplaceAll(file, originTagStr, newTagStr)
			if err := os.Rename(file, newPath); err != nil {
				utils.InfoFormat("rename %s failed: %v", file, err)
			}
		}
		newTagPath := strings.ReplaceAll(movie.Path, "《"+utils.GetTagStr(movie.Path)+"》", "《"+newTagStr+"》")
		updated := replaceIndexAfterRename(movie.Id, newTagPath, movie.BaseDir)
		res := utils.NewSuccessByMsg("执行成功")
		res.Data = updated
		return res
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newFilePath := strings.ReplaceAll(movie.Path, suffix, "《"+tag+"》"+suffix)
	newFilePath = strings.ReplaceAll(newFilePath, "《》", "")
	newName := strings.TrimSuffix(newFilePath, suffix)

	if err := os.Rename(movie.Path, newFilePath); err != nil {
		return utils.NewFailByMsg("重命名失败: " + err.Error())
	}

	for _, file := range []string{movie.Png, movie.Jpg, movie.Gif} {
		if file != "" && utils.ExistsFiles(file) {
			ext := "." + utils.GetSuffix(file)
			os.Rename(file, newName+ext)
		}
	}
	updated := replaceIndexAfterRename(movie.Id, newFilePath, movie.BaseDir)
	res := utils.NewSuccessByMsg("执行成功")
	res.Data = updated
	return res
}

// ClearTag 清除标签
func (fs *searchService) ClearTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
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
	path := strings.ReplaceAll(movie.Path, "《"+originTagStr+"》", "《"+newTagStr+"》")

	if err := os.Rename(movie.Path, path); err != nil {
		return utils.NewFailByMsg("重命名失败" + path)
	}

	newName := strings.TrimSuffix(path, "."+movie.FileType)
	for _, f := range []string{movie.Jpg, movie.Png, movie.Gif} {
		os.Rename(f, newName+"."+utils.GetSuffix(f))
	}
	updated := replaceIndexAfterRename(movie.Id, path, movie.BaseDir)
	res := utils.NewSuccessByMsg("执行成功")
	res.Data = updated
	return res
}

// Rename 重命名文件
func (fs *searchService) Rename(movie model.FileEdit) utils.Result {
	res := utils.NewSuccess()
	movieLib := fs.FindOne(movie.Id)
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
		if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
			res.FailByMsg("执行失败")
			res.Data = err
			return res
		}
	}
	newPath = newDir + utils.PathSeparator + movie.Name
	if err := os.Rename(oldPath, newPath); err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}

	// 重命名附属文件
	suffix := "." + utils.GetSuffix(movieLib.Path)
	for _, ext := range []string{".png", ".gif", ".jpg"} {
		renameFile(suffix, ext, newPath, movieLib)
	}

	// 下载 JPG/PNG
	if movie.Png != "" && strings.HasPrefix(movie.Png, "http") {
		if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
			res = fs.DownJpgMakePng(newPath, movie.Jpg, false)
			if !res.IsSuccess() {
				return res
			}
		}
		res = fs.DownJpgAsPng(newPath, movie.Png)
		if !res.IsSuccess() {
			return res
		}
	} else if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
		res = fs.DownJpgMakePng(newPath, movie.Jpg, true)
		if !res.IsSuccess() {
			return res
		}
	}
	updated := replaceIndexAfterRename(movieLib.Id, newPath, movieLib.BaseDir)
	res.Data = updated
	return res
}

// Move 移动文件到新目录
func (fs *searchService) Move(id string, newDir string, title string) utils.Result {
	res := utils.NewSuccess()
	movieLib := fs.FindOne(id)
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
		os.MkdirAll(newDir, os.ModePerm)
	}
	newPath := newDir + utils.PathSeparator + title + "." + movieLib.FileType
	if err := os.Rename(oldPath, newPath); err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}

	renameCompanionFiles(movieLib, newDir+utils.PathSeparator+title)
	updated := replaceIndexAfterRename(id, newPath, movieLib.BaseDir)
	res.Data = updated
	return res
}

// Delete 删除文件
func (fs *searchService) Delete(id string) {
	file := fs.FindOne(id)
	FileApp.DeleteOne(file.DirPath, file.Title)
}

// ── 私有辅助函数 ──────────────────────────────────────────────────

// renameCompanionFiles 将视频的附属图片文件（jpg/png/gif）一起重命名
// newBaseName: 不含后缀的新基本名（如 "/path/to/newfile"）
func renameCompanionFiles(movie model.FileItem, newBaseName string) {
	for _, file := range []string{movie.Jpg, movie.Png, movie.Gif} {
		if file == "" || !utils.ExistsFiles(file) {
			continue
		}
		suffix := "." + utils.GetSuffix(file)
		if err := os.Rename(file, newBaseName+suffix); err != nil {
			utils.InfoFormat("rename companion file failed: %v -> %s%s", err, newBaseName, suffix)
		}
	}
}

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

// renameFile 重命名附属文件
func renameFile(oldSuffix, newSuffix, newPath string, movieLib model.FileItem) bool {
	oldPath := strings.ReplaceAll(movieLib.Path, oldSuffix, newSuffix)
	if !utils.ExistsFiles(oldPath) {
		return false
	}
	if err := os.Rename(oldPath, strings.ReplaceAll(newPath, oldSuffix, newSuffix)); err != nil {
		utils.InfoNormal(err)
		return false
	}
	return true
}

// replaceIndexAfterRename 重命名文件后更新搜索引擎索引（无需全量扫描），返回更新后的 FileItem
// oldId: 旧文件 ID（由调用方从旧路径计算），同时赋给新文件
func replaceIndexAfterRename(oldId, newPath, baseDir string) model.FileItem {
	info, err := os.Stat(newPath)
	if err != nil {
		return model.FileItem{}
	}
	suffix := utils.GetSuffix(newPath)
	name := filepath.Base(newPath)
	newFile := model.EasyFile(filepath.Dir(newPath), newPath, name, suffix,
		info.Size(), info.ModTime(), baseDir)
	newFile.Id = oldId

	oldFile := model.FileItem{Id: oldId, BaseDir: baseDir}

	SearchEngine.ReplaceFile(oldFile, newFile)
	return newFile
}
