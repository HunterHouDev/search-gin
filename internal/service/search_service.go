package service

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type searchService struct{}

func (fs *searchService) SearchDataSource(searchParam model.SearchParam) utils.Page {
	result := utils.NewPage()
	searchResult := SearchEngine.PageAsync(searchParam)
	result.TotalCnt = searchResult.SearchCount
	result.TotalSize = utils.GetSizeStr(searchResult.SearchSize)
	result.PageSize = searchParam.PageSize
	result.ResultSize = utils.GetSizeStr(searchResult.SearchSize)
	result.SetResultCnt(searchResult.SearchCount, searchParam.Page)
	result.CurSize = utils.GetSizeStr(searchResult.ResultSize)
	result.CurCnt = searchResult.ResultCount
	for i := range searchResult.FileList {
		searchResult.FileList[i].PageNo = searchParam.Page
	}
	result.Data = searchResult.FileList
	return result
}

func (fs *searchService) SetMovieType(movie model.FileItem, movieType string) utils.Result {
	newMovieType := "{{" + movieType + "}}"

	if movie.MovieType != "" && movie.MovieType != "无" {
		originVideoType := utils.GetMovieType(movie.Path)
		if originVideoType == movieType {
			return utils.NewSuccessByMsg("执行成功")
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
		return utils.NewSuccessByMsg("执行成功")
	}

	suffix := "." + utils.GetSuffix(movie.Path)
	newSuffix := newMovieType + suffix
	newFilePath := strings.ReplaceAll(movie.Path, suffix, newSuffix)

	if err := os.Rename(movie.Path, newFilePath); err != nil {
		return utils.NewFailByMsg("重命名视频失败: " + err.Error())
	}

	newName := strings.TrimSuffix(newFilePath, suffix)

	files := []struct{ src, target string }{
		{movie.Png, newName + ".png"},
		{movie.Jpg, newName + ".jpg"},
		{movie.Gif, newName + ".gif"},
	}

	for _, f := range files {
		if f.src != "" && utils.ExistsFiles(f.src) {
			if err := os.Rename(f.src, f.target); err != nil {
				utils.InfoFormat("rename failed: %v", err)
			}
		}
	}
	replaceIndexAfterRename(movie.Path, newFilePath, movie.BaseDir)
	return utils.NewSuccessByMsg("执行成功")
}

func (fs *searchService) AddTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
	newTags := strings.Split(tag, ",")

	if len(movie.Tags) > 0 {
		originTagStr := utils.GetTagStr(movie.Path)
		if originTagStr == tag || strings.Contains(originTagStr, tag) {
			return utils.NewSuccessByMsg("已添加")
		}

		newTagStr := originTagStr
		for _, str := range newTags {
			if !strings.Contains(originTagStr, str) {
				newTagStr += "," + str
			}
		}
		newTagStr = "《" + newTagStr + "》"
		originTagStr = "《" + originTagStr + "》"

		files := []string{movie.Path, movie.Jpg, movie.Png, movie.Gif}
		for _, file := range files {
			newPath := strings.ReplaceAll(file, originTagStr, newTagStr)
			if err := os.Rename(file, newPath); err != nil {
				utils.InfoFormat("rename %s failed: %v", file, err)
			}
		}
		replaceIndexAfterRename(movie.Path, strings.ReplaceAll(movie.Path, "《"+utils.GetTagStr(movie.Path)+"》", "《"+newTagStr+"》"), movie.BaseDir)
		return utils.NewSuccessByMsg("执行成功")
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
	replaceIndexAfterRename(movie.Path, newFilePath, movie.BaseDir)
	return utils.NewSuccessByMsg("执行成功")
}
func (fs *searchService) ClearTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
	if len(movie.Tags) == 0 {
		return utils.NewSuccessByMsg("执行成功")
	}

	originTagStr := utils.GetTagStr(movie.Path)
	newTagStr := strings.ReplaceAll(originTagStr, tag, "")
	if len(movie.Tags) == 1 {
		newTagStr = ""
	}
	newTagStr = strings.TrimSuffix(newTagStr, ",")
	path := strings.ReplaceAll(movie.Path, "《"+originTagStr+"》", "《"+newTagStr+"》")

	if err := os.Rename(movie.Path, path); err != nil {
		result := utils.Result{}
		result.Message = "重命名失败" + path
		return result
	}

	newName := strings.TrimSuffix(path, "."+movie.FileType)
	files := []string{movie.Jpg, movie.Png, movie.Gif}
	for _, f := range files {
		os.Rename(f, newName+"."+utils.GetSuffix(f))
	}
	replaceIndexAfterRename(movie.Path, path, movie.BaseDir)
	return utils.NewSuccessByMsg("执行成功")
}

func (fs *searchService) MoveCut(srcFile model.FileItem, toFile model.FileItem) utils.Result {
	result := utils.Result{}

	if toFile.Author == "" && toFile.Code == "" {
		result.Message = "信息不全"
		return result
	}

	// 构建目标路径
	title := toFile.Title
	title = strings.ReplaceAll(title, ":", "~")
	title = strings.ReplaceAll(title, ".", "~")
	title = strings.ReplaceAll(title, "!", "~")

	dirname := "[" + toFile.Author + "] " + toFile.Code + " " + title
	path := srcFile.DirPath + utils.PathSeparator + toFile.Author
	if toFile.Studio != "" {
		path += utils.PathSeparator + toFile.Studio
	}
	dirpath := path + utils.PathSeparator + dirname
	os.MkdirAll(dirpath, os.ModePerm)

	filename := dirname + "." + utils.GetSuffix(srcFile.Path)
	finalPath := dirpath + utils.PathSeparator + filename
	jpgPath := utils.ConcatSuffix(finalPath, "jpg")
	pngPath := utils.ConcatSuffix(finalPath, "png")

	// 创建 JPG 文件
	var jpgOut *os.File
	var createErr error
	if jpgOut, createErr = os.Create(jpgPath); createErr != nil {
		// 创建失败时，简化目录名重试
		dirname = "[" + toFile.Author + "]" + toFile.Code
		dirpath = path + utils.PathSeparator + dirname
		os.MkdirAll(dirpath, os.ModePerm)
		filename = dirname + "." + utils.GetSuffix(srcFile.Path)
		finalPath = dirpath + utils.PathSeparator + filename
		jpgPath = utils.ConcatSuffix(finalPath, "jpg")
		if jpgOut, createErr = os.Create(jpgPath); createErr != nil {
			result.Fail()
			result.Message = "文件创建失败：" + jpgPath
			os.Rename(finalPath, srcFile.Path)
			return result
		}
	}

	// 下载 JPG
	url := toFile.Jpg
	if !strings.Contains(url, consts.GetOSSetting().BaseUrl) {
		url = consts.GetOSSetting().BaseUrl + url
	}
	if err := downloadFile(jpgOut, url); err != nil {
		result.Fail()
		result.Message = "文件下载失败：" + toFile.Jpg
		os.Rename(finalPath, srcFile.Path)
		return result
	}
	jpgOut.Close()

	// 生成或下载 PNG
	if toFile.Png == "" {
		if err := utils.ImageToPng(jpgPath); err != nil {
			result.Fail()
			result.Message = "png生成失败"
			os.Rename(finalPath, srcFile.Path)
			return result
		}
	} else {
		pngOut, err := os.Create(pngPath)
		if err != nil {
			result.Fail()
			result.Message = "png文件下载失败：" + toFile.Png
			os.Rename(finalPath, srcFile.Path)
			return result
		}
		if err := downloadFile(pngOut, toFile.Png); err != nil {
			result.Fail()
			result.Message = "png下载失败"
			os.Rename(finalPath, srcFile.Path)
			return result
		}
		pngOut.Close()
	}

	// 更新文件路径
	toFile.Jpg = jpgPath
	toFile.Png = pngPath

	result.Success()
	result.Message = "【" + dirname + "】" + result.Message
	return result
}

// downloadFile 将指定 URL 的内容下载到文件
func downloadFile(f *os.File, url string) error {
	resp, err := httpGet(url)
	if err != nil {
		return err
	}
	if _, err = f.Write(resp.Body()); err != nil {
		return fmt.Errorf("读取 response 失败: %w", err)
	}
	return nil
}
func (fs *searchService) DownJpgMakePng(finalPath string, url string, makePng bool) utils.Result {
	result := utils.Result{}
	jpgPath := utils.ConcatSuffix(finalPath, "jpg")
	jpgOut, createErr := os.Create(jpgPath)
	if createErr != nil {
		result.Fail()
		result.Message = "文件创建失败：" + jpgPath
		return result
	}
	defer jpgOut.Close()

	if !strings.Contains(url, "https") {
		url = consts.GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	AddLogMemory("DownJpg  time:%d  %s %d", time.Since(start).Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		result.Message = "文件下载失败：" + url
		return result
	}
	jpgOut.Write(resp.Body())
	if makePng {
		pngErr := utils.ImageToPng(jpgPath)
		if pngErr != nil {
			utils.InfoFormat("pngErr:%v", pngErr)
		}
	}
	result.Success()
	return result
}

func (fs *searchService) DownJpgAsPng(finalPath string, url string) utils.Result {
	result := utils.Result{}
	pngPath := utils.ConcatSuffix(finalPath, "png")
	pngOut, createErr := os.Create(pngPath)
	if createErr != nil {
		result.Fail()
		return result
	}
	defer pngOut.Close()

	if !strings.Contains(url, "https") {
		url = consts.GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	AddLogMemory("DownPng  time:%d  %s %d", time.Since(start).Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		result.Message = "文件下载失败：" + url
		return result
	}
	pngOut.Write(resp.Body())
	result.Success()
	return result
}

var httpClient = resty.New().
	SetTimeout(10 * time.Second).
	SetRetryCount(3).
	SetRetryWaitTime(1 * time.Second).
	SetRetryMaxWaitTime(5 * time.Second).
	SetHeaders(map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Accept-Encoding":           "gzip, deflate, br",
		"Cache-Control":             "no-cache",
		"Pragma":                    "no-cache",
		"sec-ch-ua":                 `"Chromium";v="111", "Not_A Brand";v="8"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        `"Windows"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}).
	// 每次请求随机UA，防止被统一特征拦截
	OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		ua := browsers[rand.Intn(len(browsers))]
		r.SetHeader("User-Agent", ua)
		r.SetHeader("Cookie", "random="+strconv.Itoa(rand.Intn(999999)))
		return nil
	}).
	OnError(func(req *resty.Request, err error) {
		utils.InfoNormal("http请求失败:", err)
	})

// 常见浏览器UA列表，随机切换
var browsers = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/111.0",
}

func httpGet(url string) (*resty.Response, error) {
	return httpClient.R().EnableTrace().Get(url)
}

func (fs *searchService) FindOne(Id string) model.FileItem {
	return SearchEngine.FindById(Id)
}

func cleanPath(name string) string {
	newFilePath := strings.Trim(name, " ")
	newFilePath = strings.ReplaceAll(newFilePath, "《", "")
	newFilePath = strings.ReplaceAll(newFilePath, "》", "")
	newFilePath = strings.ReplaceAll(newFilePath, "{{", "")
	newFilePath = strings.ReplaceAll(newFilePath, "}}", "")
	return newFilePath
}

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
			newDir += choose2To1(!strings.HasPrefix(movie.Title, movie.Author), choose2To1(movie.Author != "", movie.Author, ""), "")
			newDir += choose2To1(!strings.Contains(movie.Title, newCode), choose2To1(newCode != "", " "+newCode, ""), "")
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
		renamed := renameFile(suffix, ext, newPath, movieLib)
		_ = renamed
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
	} else {
		if movie.Jpg != "" && strings.HasPrefix(movie.Jpg, "http") {
			res = fs.DownJpgMakePng(newPath, movie.Jpg, true)
			if !res.IsSuccess() {
				return res
			}
		}
	}
	replaceIndexAfterRename(oldPath, newPath, movieLib.BaseDir)
	return res
}

// renameFile 重命名附属文件（如 jpg/png/gif
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
	newPath := newDir + utils.PathSeparator + title
	err := os.Rename(oldPath, newPath+"."+movieLib.FileType)
	if err != nil {
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}

	if utils.ExistsFiles(movieLib.Png) {
		os.Rename(movieLib.Png, newPath+".png")
	}
	if utils.ExistsFiles(movieLib.Jpg) {
		os.Rename(movieLib.Jpg, newPath+".jpg")
	}
	if utils.ExistsFiles(movieLib.Gif) {
		os.Rename(movieLib.Gif, newPath+".gif")
	}
	return res
}

func (fs *searchService) Delete(id string) {
	file := fs.FindOne(id)
	FileApp.DeleteOne(file.DirPath, file.Title)
}

func choose2To1(tr bool, str1 string, str2 string) string {
	if tr {
		return str1
	}
	return str2
}

// replaceIndexAfterRename 重命名文件后更新搜索引擎索引（无需全量扫描）
// oldPath: 原完整路径；newPath: 新完整路径；baseDir: 所属扫描根目录
func replaceIndexAfterRename(oldPath, newPath, baseDir string) {
	info, err := os.Stat(newPath)
	if err != nil {
		return
	}
	suffix := utils.GetSuffix(newPath)
	name := filepath.Base(newPath)
	newFile := model.EasyFile(filepath.Dir(newPath), newPath, name, suffix,
		info.Size(), info.ModTime(), baseDir)

	// 用旧路径构造旧文件的 Id
	oldId, _ := utils.DirpathForId(oldPath)
	oldFile := model.FileItem{Id: oldId, BaseDir: baseDir}

	SearchEngine.ReplaceFile(oldFile, newFile)
}
