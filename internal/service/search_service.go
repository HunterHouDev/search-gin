package service

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type searchService struct {
}

func (fs *searchService) SearchDataSource(searchParam model.SearchParam) utils.Page {
	result := utils.NewPage()
	searchResult := SearchEngin.PageAsync(searchParam)
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

func (fs *searchService) SetMovieType(movie model.Movie, movieType string) utils.Result {
	newMovieType := "{{" + movieType + "}}"

	if movie.MovieType != "" && movie.MovieType != "无" {
		originVideoType := utils.GetMovieType(movie.Path)
		if originVideoType == movieType {
			return utils.NewSuccessByMsg("执行成功")
		}

		originalPaths := []string{movie.Path, movie.Jpg, movie.Png, movie.Nfo, movie.Gif}
		newPaths := make([]string, len(originalPaths))
		for i, p := range originalPaths {
			if p != "" {
				newPaths[i] = strings.ReplaceAll(p, originVideoType, movieType)
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
		{movie.Nfo, newName + ".nfo"},
		{movie.Gif, newName + ".gif"},
	}

	for _, f := range files {
		if f.src != "" && utils.ExistsFiles(f.src) {
			if err := os.Rename(f.src, f.target); err != nil {
				utils.InfoFormat("rename failed: %v", err)
			}
		}
	}
	return utils.NewSuccessByMsg("执行成功")
}

func (fs *searchService) AddTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
	//video
	newTags := strings.Split(tag, ",")
	if len(movie.Tags) > 0 {
		originTagStr := utils.GetTagStr(movie.Path)
		if originTagStr == tag {
			return utils.NewSuccessByMsg("执行成功")
		}
		if strings.Contains(originTagStr, tag) {
			return utils.NewSuccessByMsg("已添加")
		}
		newTagStr := originTagStr
		for _, str := range newTags {
			if strings.Contains(originTagStr, str) {
				continue
			}
			newTagStr = newTagStr + "," + str
		}
		newTagStr = "《" + newTagStr + "》"
		originTagStr = "《" + utils.GetTagStr(movie.Path) + "》"
		path := strings.ReplaceAll(movie.Path, originTagStr, newTagStr)
		err := os.Rename(movie.Path, path)
		if err != nil {
			utils.InfoFormat("%v", err)
			return utils.NewFailByMsg(err.Error())
		}
		path = strings.ReplaceAll(movie.Jpg, originTagStr, newTagStr)
		err = os.Rename(movie.Jpg, path)
		if err != nil {
			utils.InfoFormat("%v", err)
		}
		path = strings.ReplaceAll(movie.Png, originTagStr, newTagStr)
		err = os.Rename(movie.Png, path)
		if err != nil {
			utils.InfoFormat("%v", err)
		}
		path = strings.ReplaceAll(movie.Nfo, originTagStr, newTagStr)
		err = os.Rename(movie.Nfo, path)
		if err != nil {
			utils.InfoFormat("%v", err)
		}
		path = strings.ReplaceAll(movie.Gif, originTagStr, newTagStr)
		err = os.Rename(movie.Gif, path)
		if err != nil {
			utils.InfoFormat("%v", err)
		}
		return utils.NewSuccessByMsg("执行成功")
	}

	newMovieType := "《" + tag + "》"
	utils.InfoFormat("%v", tag)
	suffix := "." + utils.GetSuffix(movie.Path)
	newSuffix := newMovieType + suffix
	newFilePath := strings.ReplaceAll(movie.Path, suffix, newSuffix)
	if strings.Contains(newFilePath, "《") && strings.Contains(newFilePath, "》") {
		newFilePath = strings.ReplaceAll(newFilePath, "《,》", "")
		newFilePath = strings.ReplaceAll(newFilePath, "《》", "")
	}
	err := os.Rename(movie.Path, newFilePath)
	newName := strings.TrimSuffix(newFilePath, suffix)
	if err != nil {
		utils.InfoFormat("%v", err)
		return utils.NewFailByMsg(err.Error())
	}
	//png
	if utils.ExistsFiles(movie.Png) {
		suffix = "." + utils.GetSuffix(movie.Png)
		os.Rename(movie.Png, newName+suffix)
	}

	//jpg
	if utils.ExistsFiles(movie.Jpg) {
		suffix = "." + utils.GetSuffix(movie.Jpg)
		os.Rename(movie.Jpg, newName+suffix)
	}

	//nfo
	if utils.ExistsFiles(movie.Nfo) {
		suffix = "." + utils.GetSuffix(movie.Nfo)
		os.Rename(movie.Nfo, newName+suffix)

	}
	//Gif
	if utils.ExistsFiles(movie.Gif) {
		suffix = "." + utils.GetSuffix(movie.Gif)
		os.Rename(movie.Gif, newName+suffix)

	}
	return utils.NewSuccessByMsg("执行成功")
}
func (fs *searchService) ClearTag(id string, tag string) utils.Result {
	movie := fs.FindOne(id)
	//video
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
	err := os.Rename(movie.Path, path)
	utils.InfoFormat("originPath [%s]", movie.Path)
	utils.InfoFormat("remove tag [%s]", tag)
	utils.InfoFormat("originTagStr [%s]", originTagStr)
	utils.InfoFormat("newTagStr [%s]", newTagStr)
	utils.InfoFormat("newPath [%s]", path)
	if err != nil {
		result := utils.Result{}
		utils.InfoFormat("重命名失败 [%s]", path)
		result.Message = "重命名失败" + path
		return result
	}
	newName := strings.TrimSuffix(path, "."+movie.FileType)
	os.Rename(movie.Jpg, newName+".jpg")
	os.Rename(movie.Png, newName+".png")
	os.Rename(movie.Nfo, newName+".nfo")
	os.Rename(movie.Gif, newName+".gif")
	return utils.NewSuccessByMsg("执行成功")
}

func (fs *searchService) MoveCut(srcFile model.Movie, toFile model.Movie) utils.Result {
	result := utils.Result{}
	root := srcFile.DirPath
	utils.InfoFormat("MoveCut： srcFile [%v] \n\n", srcFile)
	utils.InfoFormat("MoveCut： toFile [%v] \n\n", toFile)
	if toFile.Actress == "" && toFile.Code == "" {
		result.Message = "信息不全"
		return result
	}
	path := root + utils.PathSeparator + toFile.Actress
	if toFile.Studio != "" {
		path = path + utils.PathSeparator + toFile.Studio
	}
	title := toFile.Title
	title = strings.ReplaceAll(title, ":", "~")
	title = strings.ReplaceAll(title, ".", "~")
	title = strings.ReplaceAll(title, "!", "~")

	dirname := "[" + toFile.Actress + "] " + toFile.Code + " " + title
	dirpath := path + utils.PathSeparator + dirname
	os.MkdirAll(dirpath, os.ModePerm)
	filename := dirname + "." + utils.GetSuffix(srcFile.Path)
	finalPath := dirpath + utils.PathSeparator + filename
	if finalPath != srcFile.Path {
		os.Rename(srcFile.Path, finalPath)
	}
	jpgPath := utils.ConcatSuffix(finalPath, "jpg")
	pngPath := utils.ConcatSuffix(finalPath, "png")
	nfoPath := utils.ConcatSuffix(finalPath, "nfo")

	jpgOut, createErr := os.Create(jpgPath)
	if createErr != nil {
		//TODO 创建失败  标题 特殊字符处理 改为 演员+番号
		dirname = "[" + toFile.Actress + "]" + toFile.Code + ""
		dirpath = path + utils.PathSeparator + dirname
		os.MkdirAll(dirpath, os.ModePerm)
		filename = dirname + "." + utils.GetSuffix(srcFile.Path)
		finalPath = dirpath + utils.PathSeparator + filename
		jpgPath = utils.ConcatSuffix(finalPath, "jpg")
		jpgOut, createErr = os.Create(jpgPath)
		if createErr != nil {
			result.Fail()
			utils.InfoFormat("createErr: %v", createErr)
			os.Rename(finalPath, srcFile.Path)
			result.Message = "文件创建失败：" + jpgPath
			return result
		}
	}
	url := toFile.Jpg
	if !strings.Contains(url, consts.GetOSSetting().BaseUrl) {
		url = consts.GetOSSetting().BaseUrl + url
	}
	resp, downErr := httpGet(url)
	if downErr != nil {
		result.Fail()
		utils.InfoFormat("downErr: %v ", downErr)
		os.Rename(finalPath, srcFile.Path)
		result.Message = "文件下载失败：" + toFile.Jpg
		return result
	}
	body, readErr := io.ReadAll(bytes.NewReader(resp.Body()))
	if readErr != nil {
		result.Fail()
		utils.InfoFormat("readErr:%v", readErr)
		os.Rename(finalPath, srcFile.Path)
		result.Message = "请求读取response失败"
		return result
	}
	jpgOut.Write(body)
	jpgOut.Close()
	if toFile.Png == "" {
		pngErr := utils.ImageToPng(jpgPath)
		if pngErr != nil {
			result.Fail()
			utils.InfoFormat("pngErr:%v", pngErr)
			os.Rename(finalPath, srcFile.Path)
			result.Message = "png生成失败"
			// return result
		}
	} else {
		pngOut, createErr := os.Create(pngPath)
		if createErr != nil {
			result.Fail()
			utils.InfoFormat("downErr:%v", downErr)
			os.Rename(finalPath, srcFile.Path)
			result.Message = "png文件下载失败：" + toFile.Png
			return result
		}
		resp2, downErr := httpGet(url)
		if downErr != nil {
			result.Fail()
			utils.InfoFormat("downErr:%v", downErr)
			os.Rename(finalPath, srcFile.Path)
			result.Message = "文件下载失败：" + toFile.Jpg
			return result
		}
		body, readErr := io.ReadAll(bytes.NewReader(resp2.Body()))
		if readErr != nil {
			result.Fail()
			utils.InfoFormat("readErr:%v", readErr)
			os.Rename(finalPath, srcFile.Path)
			result.Message = "请求读取response失败"
			return result
		}
		pngOut.Write(body)
		pngOut.Close()
	}
	toFile.Jpg = jpgPath
	toFile.Nfo = nfoPath
	toFile.Png = pngPath
	result.Success()
	result.Message = "【" + dirname + "】" + result.Message
	return result

}

func (fs *searchService) DownJpgMakePng(finalPath string, url string, makePng bool) utils.Result {

	result := utils.Result{}
	jpgPath := utils.ConcatSuffix(finalPath, "jpg")
	jpgOut, createErr := os.Create(jpgPath)
	if createErr != nil {
		utils.InfoFormat("createErr:%v  \n\n\n", createErr)
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
	ti := time.Since(start)
	AddLogMemory("DownJpg  time:%d  %s %d", ti.Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		utils.InfoFormat("downErr:%v  \n\n", downErr)
		result.Message = "文件下载失败：" + url
		return result
	}
	//body, readErr := ioutil.ReadAll(bytes.NewReader(resp.Body()))
	//if readErr != nil {
	//	result.Fail()
	//	utils.InfoFormat("readErr:%v  \n\n", readErr)
	//	result.Message = "请求读取response失败"
	//	return result
	//}
	jpgOut.Write(resp.Body())
	if makePng {
		pngErr := utils.ImageToPng(jpgPath)
		if pngErr != nil {
			utils.InfoFormat("pngErr:%v  \n\n", pngErr)
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
	 utils.InfoFormat("createErr:%v  \n\n", createErr)
	 result.Fail()
	 return result
	}
	defer pngOut.Close()
	if !strings.Contains(url, "https") {
		url = consts.GetOSSetting().BaseUrl + url
	}
	start := time.Now()
	resp, downErr := httpGet(url)
	ti := time.Since(start)
	AddLogMemory("DownPng  time:%d  %s %d", ti.Milliseconds(), url, downErr)
	if downErr != nil {
		result.Fail()
		utils.InfoFormat("downErr:%v  \n\n", downErr)
		result.Message = "文件下载失败：" + url
		return result
	}
	//body, readErr := ioutil.ReadAll(resp.Body)
	//if readErr != nil {
	//	result.Fail()
	//	utils.InfoFormat("readErr:%v  \n\n", readErr)
	//	result.Message = "请求读取response失败"
	//	return result
	//}
	pngOut.Write(resp.Body())
	pngOut.Close()
	result.Success()
	return result
}

var httpClient = resty.New().
	SetTimeout(10 * time.Second).
	SetRetryCount(3).
	SetRetryWaitTime(1 * time.Second).
	SetRetryMaxWaitTime(5 * time.Second).
	SetHeaders(map[string]string{
		"Accept":           "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":  "zh-CN,zh;q=0.9,en;q=0.8",
		"Accept-Encoding":  "gzip, deflate, br",
		"Cache-Control":    "no-cache",
		"Pragma":           "no-cache",
		"sec-ch-ua":        `"Chromium";v="111", "Not_A Brand";v="8"`,
		"sec-ch-ua-mobile": "?0",
		"sec-ch-ua-platform": `"Windows"`,
		"Sec-Fetch-Dest":   "document",
		"Sec-Fetch-Mode":   "navigate",
		"Sec-Fetch-Site":   "none",
		"Sec-Fetch-User":   "?1",
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

func (fs *searchService) FindOne(Id string) model.Movie {
	return SearchEngin.FindById(Id)
}

func cleanPath(name string) string {
	newFilePath := strings.Trim(name, " ")
	newFilePath = strings.ReplaceAll(newFilePath, "《", "")
	newFilePath = strings.ReplaceAll(newFilePath, "》", "")
	newFilePath = strings.ReplaceAll(newFilePath, "{{", "")
	newFilePath = strings.ReplaceAll(newFilePath, "}}", "")
	return newFilePath
}

func (fs *searchService) Rename(movie model.MovieEdit) utils.Result {
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
	utils.InfoFormat("newTitle: %v\n\n", newDir)
	if movie.MoveOut {
		// os.MkdirAll(movie.Actress, os.ModePerm)
		if movie.Actress != "" {
			arr := strings.Split(newPath, utils.PathSeparator)
			if utils.HasItem(arr, movie.Actress) {
				arr2 := strings.Split(newPath, movie.Actress)
				newDir = arr2[0]
			}
			newDir += utils.PathSeparator + movie.Actress
		}
		utils.InfoFormat("newTitle: %v\n\n", newDir)
		if movie.Title != "" {
			newDir += utils.PathSeparator
			newCode := movie.Code
			if strings.HasPrefix(newCode, "-") {
				newCode = strings.Replace(newCode, "-", "", 1)
			}
			newDir += choose2To1(!strings.HasPrefix(movie.Title, movie.Actress), choose2To1(movie.Actress != "", movie.Actress, ""), "")
			newDir += choose2To1(!strings.Contains(movie.Title, newCode), choose2To1(newCode != "", " "+newCode, ""), "")
			newTitle := strings.Split(movie.Title, "{{")
			utils.InfoFormat("newDir: %v\n\n", newDir)
			// newTitle[0] 限制前10个字符
			newTitleStart := newTitle[0]
			if len(newTitleStart) > 10 {
				newTitleStart = newTitleStart[:10]
			}
			newDir += " " + cleanPath(newTitleStart)
		}
		utils.InfoFormat("newDir: %v\n\n", newDir)
		err := os.MkdirAll(newDir, os.ModePerm)
		if err != nil {
			utils.InfoFormat("err: %v\n\n", err)
			res.FailByMsg("执行失败")
			res.Data = err
			return res
		}
	}
	newPath = newDir + utils.PathSeparator + movie.Name
	err := os.Rename(oldPath, newPath)
	if err != nil {
		utils.InfoFormat("err: %v\n\n", err)
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}
	//png
	targetSuffix := ".png"
	suffix := "." + utils.GetSuffix(oldPath)
	oldPath = strings.ReplaceAll(oldPath, suffix, targetSuffix)
	newPath = strings.ReplaceAll(newPath, suffix, targetSuffix)
	if utils.ExistsFiles(oldPath) {
		err = os.Rename(oldPath, newPath)
		if err != nil {
			utils.InfoNormal(err)
		}
	}

	//gif
	targetSuffix = ".gif"
	suffix = "." + utils.GetSuffix(oldPath)
	oldPath = strings.ReplaceAll(oldPath, suffix, targetSuffix)
	newPath = strings.ReplaceAll(newPath, suffix, targetSuffix)
	if utils.ExistsFiles(oldPath) {
		err = os.Rename(oldPath, newPath)
		if err != nil {
			utils.InfoNormal(err)
		}
	}

	//jpg
	targetSuffix = ".jpg"
	suffix = "." + utils.GetSuffix(oldPath)
	oldPath = strings.ReplaceAll(oldPath, suffix, targetSuffix)
	newPath = strings.ReplaceAll(newPath, suffix, targetSuffix)
	if utils.ExistsFiles(oldPath) {
		err = os.Rename(oldPath, newPath)
		if err != nil {
			utils.InfoNormal(err)
		}
	}

	//nfo
	targetSuffix = ".nfo"
	suffix = "." + utils.GetSuffix(oldPath)
	oldPath = strings.ReplaceAll(oldPath, suffix, targetSuffix)
	newPath = strings.ReplaceAll(newPath, suffix, targetSuffix)
	if utils.ExistsFiles(oldPath) {
		err = os.Rename(oldPath, newPath)
		if err != nil {
			utils.InfoNormal(err)
		}
	}
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
	return res
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
		utils.InfoFormat("err: %v\n\n", err)
		res.FailByMsg("执行失败")
		res.Data = err
		return res
	}
	utils.InfoFormat("file move from [%s] to [%s]", oldPath, newPath+"."+movieLib.FileType)
	if utils.ExistsFiles(movieLib.Png) {
		os.Rename(movieLib.Png, newPath+".png")
		utils.InfoFormat("png move from [%s] to [%s]", movieLib.Png, newPath+".png")
	}
	if utils.ExistsFiles(movieLib.Jpg) {
		os.Rename(movieLib.Jpg, newPath+".jpg")
	}
	if utils.ExistsFiles(movieLib.Gif) {
		os.Rename(movieLib.Gif, newPath+".gif")
		utils.InfoFormat("gif move from [%s] to [%s]", movieLib.Gif, newPath+".gif")
	}
	return res
}

func choose2To1(tr bool, str1 string, str2 string) string {
	if tr {
		return str1
	} else {
		return str2
	}
}

func (fs *searchService) Delete(id string) {
	file := fs.FindOne(id)
	FileApp.DeleteOne(file.DirPath, file.Title)
}
