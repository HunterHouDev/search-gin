package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GetPlay 本地打开文件
func GetPlay(c *gin.Context) {
	id := c.Param("id")
	file := service.SearchApp.FindOne(id)
	if file.IsNull() {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	sanitizePath, err := utils.ValidatePath(file.Path, consts.GetOSSetting().Dirs)
	if err != nil {
		utils.InfoFormat("命令注入攻击尝试: %s, 错误: %v", file.Path, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("文件路径不在允许范围内"))
		return
	}

	utils.InfoFormat("GetPlay [%v]", sanitizePath)

	setting := consts.GetOSSetting()
	if setting.SystemPlayer == "ffplay" {
		go func() {
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
			if service.TempDir != "" {
				ffplayPath = filepath.Join(service.TempDir, "ffplay.exe")
			}

			params = append(params, sanitizePath)
			cmd := exec.Command(ffplayPath, params...)
			err := cmd.Start()
			if err != nil {
				utils.InfoFormat("播放失败: %v, 错误: %v", sanitizePath, err)
			}
		}()
	} else {
		utils.ExecCmdStart(sanitizePath)
	}

	res := utils.NewSuccessByMsg("播放成功")
	c.JSON(http.StatusOK, res)
}

// SetMovieType 设置类型
func SetMovieType(c *gin.Context) {
	id := c.Param("id")
	movieType := c.Param("movieType")
	file := service.SearchApp.FindOne(id)
	res := service.SearchApp.SetMovieType(file, movieType)
	c.JSON(http.StatusOK, res)
}

// GetInfo 获取Info信息
func GetInfo(c *gin.Context) {
	id := c.Param("id")

	// 远程转发
	if service.HandleRemoteByID(c, id, "info") {
		return
	}

	file := service.SearchApp.FindOne(id)
	c.JSON(http.StatusOK, file)
}

// PostRename 改名
func PostRename(c *gin.Context) {
	// 先读取 Body（仅读一次），用于转发和绑定
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("读取请求体失败"))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	currentFile := model.FileEdit{}
	err = c.ShouldBindJSON(&currentFile)
	if err != nil {
		utils.InfoNormal(err)
	}

	// 远程转发：恢复 Body 供 forwardRequest 读取
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if service.HandleRemoteByMovieEdit(c, currentFile, "rename") {
		return
	}

	utils.InfoFormat("PostRename :searchCnt[%v] \n\n", currentFile)
	res := service.SearchApp.Rename(currentFile)
	c.JSON(http.StatusOK, res)
}

func PostMove(c *gin.Context) {
	// 先读取 Body（仅读一次），用于转发和绑定
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.InfoNormal(err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("读取请求体失败"))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	currentFile := model.FileEdit{}
	err = c.ShouldBindJSON(&currentFile)
	if err != nil {
		utils.InfoNormal(err)
	}

	// 远程转发：恢复 Body 供 forwardRequest 读取
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if service.HandleRemoteByMovieEdit(c, currentFile, "move") {
		return
	}

	utils.InfoFormat("PostMove :[%v] \n\n", currentFile)
	res := service.SearchApp.Move(currentFile.Id, currentFile.Path, currentFile.Title)
	c.JSON(http.StatusOK, res)
}

// GetAddTag 添加标签
func GetAddTag(c *gin.Context) {
	idInt := c.Param("id")
	tag := c.Param("tag")

	// 远程转发
	if service.HandleRemoteByID(c, idInt, "addTag") {
		return
	}

	utils.InfoFormat("GetAddTag [%v] [%v]  \n", idInt, tag)
	res := service.SearchApp.AddTag(idInt, tag)
	c.JSON(http.StatusOK, res)
}

// GetClearTag 删除标签
func GetClearTag(c *gin.Context) {
	idInt := c.Param("id")
	tag := c.Param("tag")

	// 远程转发
	if service.HandleRemoteByID(c, idInt, "clearTag") {
		return
	}

	res := service.SearchApp.ClearTag(idInt, tag)
	c.JSON(http.StatusOK, res)
}

// GetDirInfo 文件夹信息 文件列表
func GetDirInfo(c *gin.Context) {
	// 使用读写锁保护并发访问
	consts.TempImageMutex.Lock()
	if len(consts.TempImage) > 1000 {
		consts.TempImage = make(map[string]model.FileItem)
	}
	id := c.Param("id")
	sort := c.Param("sort")
	file := service.SearchApp.FindOne(id)
	files := service.FileApp.Walk(file.DirPath, consts.Images, false)
	model.SortFileItems(files, "MTime", sort)
	for i := 0; i < len(files); i++ {
		consts.TempImage[files[i].Id] = files[i]
	}
	consts.TempImageMutex.Unlock()

	time.AfterFunc(30*time.Second, func() {
		consts.TempImageMutex.Lock()
		delete(consts.TempImage, id)
		consts.TempImageMutex.Unlock()
	})
	c.JSON(http.StatusOK, files)
}

// GetDelete 删除文件
func GetDelete(c *gin.Context) {
	id := c.Param("id")

	// 远程转发
	if service.HandleRemoteByID(c, id, "delete") {
		return
	}

	service.SearchApp.Delete(id)
	res := utils.NewSuccessByMsg("删除成功")
	c.JSON(http.StatusOK, res)
}

// GetRefreshIndex 刷新索引
func GetRefreshTargetIndex(c *gin.Context) {
	dir := c.Param("dir")
	baseDir, _ := url.QueryUnescape(dir)

	service.FileApp.ScanTarget(baseDir)
	res := utils.NewSuccessByMsg("扫描任务执行中")
	c.JSON(http.StatusOK, res)
}

func GetRefreshIndex(c *gin.Context) {
	cnt := service.FileApp.ScanAll()
	res := utils.NewSuccessByMsg("计划扫描：" + fmt.Sprint(cnt))
	c.JSON(http.StatusOK, res)
}

// GetTempImage 临时图片 特指浏览某个文件夹的所有图片
func GetTempImage(c *gin.Context) {
	id := c.Param("path")
	consts.TempImageMutex.RLock()
	file, exists := consts.TempImage[id]
	consts.TempImageMutex.RUnlock()

	if !exists || !utils.ExistsFiles(file.Path) {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}
	c.File(file.Path)
}

func GetFileByPathUseEncode(c *gin.Context) {
	escapeUrl := c.Param("path")
	decodedPath, err := url.QueryUnescape(escapeUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	// 验证路径是否在允许的目录内
	validatedPath, err := utils.ValidatePath(decodedPath, consts.GetOSSetting().Dirs)
	if err != nil {
		utils.InfoFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("访问被拒绝：路径不在允许范围内"))
		return
	}

	if utils.ExistsFiles(validatedPath) {
		c.File(validatedPath)
	} else {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
	}
}

func GetDeleteFileByPathUseEncode(c *gin.Context) {
	escapeUrl := c.Param("path")
	decodedPath, err := url.QueryUnescape(escapeUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("无效的文件路径"))
		return
	}

	// 验证路径是否在允许的目录内
	validatedPath, err := utils.ValidatePath(decodedPath, consts.GetOSSetting().Dirs)
	if err != nil {
		utils.InfoFormat("路径遍历攻击尝试: %s, 错误: %v", decodedPath, err)
		c.JSON(http.StatusForbidden, utils.NewFailByMsg("删除被拒绝：路径不在允许范围内"))
		return
	}

	if !utils.ExistsFiles(validatedPath) {
		c.JSON(http.StatusNotFound, utils.NewFailByMsg("文件不存在"))
		return
	}

	err = os.Remove(validatedPath)
	if err != nil {
		utils.InfoFormat("删除文件失败: %s, 错误: %v", validatedPath, err)
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("删除失败"))
		return
	}

	c.JSON(http.StatusOK, utils.NewSuccessByMsg("删除成功"))
}

// GetFile 获取文件流
func GetFile(c *gin.Context) {
	service.FileApp.GetFile(c)
}

// GetPng 获取Png流
func GetPng(c *gin.Context) {
	service.FileApp.GetPng(c)
}

// GetJpg 获取jpg流
func GetJpg(c *gin.Context) {
	service.FileApp.GetJpg(c)

}

// GetAuthorImage 获取脸谱的图片流
func GetAuthorImage(c *gin.Context) {
	path := c.Param("path")
	author := service.SearchEngine.FindAuthorByName(path)
	if author.IsNotEmpty() {
		for _, v := range author.Images {
			if utils.ExistsFiles(v) {
				c.File(v)
				return
			}
		}
	}
	c.Status(http.StatusNotFound)
}

func PostMerge(c *gin.Context) {
	searchParam := model.MergeParam{}
	err := c.Bind(&searchParam)
	if err != nil {
		utils.InfoFormat("PostMerge 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}
	utils.InfoFormat("PostMerge： [%v]", searchParam)

	var paths = []string{}
	var dir = ""
	for _, file := range searchParam.Files {
		curFile := service.SearchApp.FindOne(file)
		dir = curFile.DirPath
		paths = append(paths, curFile.Path)
	}

	if len(paths) == 0 {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("没有找到要合并的文件"))
		return
	}

	listPath := dir + "\\list.txt"
	file, err := os.Create(listPath)
	if err != nil {
		utils.InfoFormat("创建文件 list.txt 时出错: %v", err)
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("创建合并列表文件失败"))
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			utils.InfoFormat("关闭文件 list.txt 时出错: %v", closeErr)
		}
	}()

	for _, filePath := range paths {
		_, err := file.WriteString("file '" + filePath + "'\n")
		if err != nil {
			utils.InfoFormat("写入文件 list.txt 时出错: %v", err)
			c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("写入合并列表失败"))
			return
		}
	}

	if searchParam.Dest == "" {
		suffix := utils.GetSuffix(paths[0])
		searchParam.Dest = dir + fmt.Sprintf("\\%d.%s", time.Now().UnixMilli(), suffix)
	}

	task := model.NewMergeTask(paths, searchParam.Dest, listPath, searchParam.DeleteSource)
	task.SetStatus(model.StatusPending)
	consts.TransferTaskMutex.Lock()
	consts.TransferTask[task.CreateTime] = task
	consts.TransferTaskMutex.Unlock()
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("任务创建成功"))

}

func GetTransferToMp4(c *gin.Context) {
	id := c.Param("id")

	// 远程转发
	if service.HandleRemoteByID(c, id, "transferToMp4") {
		return
	}

	to := "mp4"
	xcode := c.Param("xcode")
	utils.InfoFormat("GetTransferToMp4 newFile [%v][%v] ", id, to)

	movieFile := service.SearchApp.FindOne(id)
	if !utils.ExistsFiles(movieFile.Path) {
		c.JSON(http.StatusOK, utils.NewFailByMsg("文件不存在"))
		return
	}
	from := utils.GetSuffix(movieFile.Path)
	if to == "" {
		to = "mp4"
	}

	exists := false
	consts.TransferTaskMutex.RLock()
	for _, taskModel := range consts.TransferTask {
		if taskModel.Path == movieFile.Path && taskModel.Status != "执行失败" {
			exists = true
			break
		}
	}
	consts.TransferTaskMutex.RUnlock()

	if exists {
		c.JSON(http.StatusOK, utils.NewFailByMsg("任务不可重复"))
		return
	} else {
		task := model.NewTask(movieFile.Path, movieFile.Name, from, to)
		task.SetStatus(model.StatusPending)
		if xcode != "" {
			task.VCode = xcode
		}
		consts.TransferTaskMutex.Lock()
		consts.TransferTask[task.CreateTime] = task
		consts.TransferTaskMutex.Unlock()
		c.JSON(http.StatusOK, utils.NewSuccessByMsg("任务创建成功"))
	}
}

func GetCutImage(c *gin.Context) {
	idInt := c.Param("id")
	typeImage := c.Param("typeImage")
	start := c.Param("start")

	// 远程转发
	if service.HandleRemoteByID(c, idInt, "cutImage") {
		return
	}

	movieFile := service.SearchApp.FindOne(idInt)
	if movieFile.IsNull() {
		r := utils.Fail()
		r.Message = "文件不存在"
		c.JSON(http.StatusOK, r)
		return
	}
	res := service.FileApp.CutImage(movieFile.Path, typeImage, start)
	c.JSON(http.StatusOK, res)
}

func GetCutMovie(c *gin.Context) {
	id := c.Param("id")

	// 远程转发
	if service.HandleRemoteByID(c, id, "cutMovie") {
		return
	}

	start := c.Param("start")
	end := c.Param("end")
	utils.InfoFormat("GetCutMovie [%v][%v][%v] ", id, start, end)

	movieFile := service.SearchApp.FindOne(id)
	if !utils.ExistsFiles(movieFile.Path) {
		c.JSON(http.StatusOK, utils.NewFailByMsg("文件不存在"))
		return
	}
	from := utils.GetSuffix(movieFile.Path)
	task := model.NewCutTask(movieFile.Path, movieFile.Name, start, end, from)
	task.SetStatus(model.StatusPending)
	consts.TransferTaskMutex.Lock()
	consts.TransferTask[task.CreateTime] = task
	consts.TransferTaskMutex.Unlock()
	c.JSON(http.StatusOK, utils.NewSuccessByMsg("任务创建成功"))

}

func GetTransferTask(c *gin.Context) {
	result := utils.NewSuccess()
	consts.TransferTaskMutex.RLock()
	tasks := make(map[time.Time]model.TransferTaskModel, len(consts.TransferTask))
	for k, v := range consts.TransferTask {
		tasks[k] = v
	}
	consts.TransferTaskMutex.RUnlock()
	result.Data = tasks
	c.JSON(http.StatusOK, result)
}

func GetDelTransferTask(c *gin.Context) {
	create := c.Param("create")
	consts.TransferTaskMutex.Lock()
	var ti time.Time
	var task model.TransferTaskModel
	for k, v := range consts.TransferTask {
		if v.Name == create {
			ti = k
			task = v
			break
		}
	}
	if task.Status == "执行中" {
		consts.TransferTaskMutex.Unlock()
		r := utils.Fail()
		r.Message = "执行中无法删除"
		c.JSON(http.StatusOK, r)
		return
	}
	delete(consts.TransferTask, ti)
	consts.TransferTaskMutex.Unlock()
	result := utils.NewSuccess()
	c.JSON(http.StatusOK, result)
}
