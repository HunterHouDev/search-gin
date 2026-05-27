package service

import (
	"context"
	"search-gin/pkg/consts"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// TaskCtx 任务调度上下文，用于优雅关闭
var (
	TaskCtx, TaskCancel = context.WithCancel(context.Background())
)

type fileService struct {
}

// 硬件加速相关（惰性检测，首次转码时自动识别）
 var (
	hwAccelH264Encoder string
	hwAccelH265Encoder string
	hwAccelModeName    string
	hwAccelDetected    bool
	hwAccelMutex       sync.Mutex

	// 硬件解码参数（与编码器匹配，如 "-hwaccel cuda -hwaccel_output_format cuda"）
	hwAccelDecodeParams string

	// 强制重新检测标志（用户切换硬件加速设置时重置）
	hwAccelForceDetect bool
)

var (
	noPic       []byte
	contentType string
	noPicMutex  sync.RWMutex // 保护noPic和contentType的并发访问
)

// GetPng 获取PNG图片
func (fs *fileService) GetPng(c *gin.Context) {
	id := c.Param("path")
	file := SearchApp.FindOne(id)
	if !file.IsNull() {
		// 按优先级检查图片文件
		if utils.ExistsFiles(file.Png) {
			data, err := utils.CompressPngIfNeed(file.Png)
			if err == nil {
				c.Data(http.StatusOK, "image/png", data)
				return
			}
			c.File(file.Png)
			return
		} else if utils.ExistsFiles(file.Jpg) {
			c.File(file.Jpg)
			return
		} else if utils.ExistsFiles(file.Gif) {
			c.File(file.Gif)
			return
		}
	}
	fs.writeNoPic(c)
}

// GetJpg 获取JPG图片
func (fs *fileService) GetJpg(c *gin.Context) {
	id := c.Param("path")
	file := SearchApp.FindOne(id)
	if !file.IsNull() {
		// 按优先级检查图片文件
		jpeg := utils.ConcatSuffix(file.Path, "jpeg")
		if utils.ExistsFiles(file.Jpg) {
			c.File(file.Jpg)
			return
		} else if utils.ExistsFiles(jpeg) {
			c.File(jpeg)
			return
		} else if utils.ExistsFiles(file.Png) {
			c.File(file.Png)
			return
		} else if utils.ExistsFiles(file.Gif) {
			c.File(file.Gif)
			return
		}
	}
	fs.writeNoPic(c)
}

// GetFile 获取文件
func (fs *fileService) GetFile(c *gin.Context) {
	id := c.Param("id")
	file := SearchApp.FindOne(id)
	if utils.ExistsFiles(file.Path) {
		c.File(file.Path)
	} else {
		c.Status(http.StatusNotFound)
	}
}

// HeartBeat 心跳与定时
func (fileService *fileService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		if consts.OSSetting.EnableTimeScan {
			now := time.Now()
			diff := now.Sub(consts.LastScanTime)
			if diff.Seconds() > 180 {
				for _, v := range consts.OSSetting.Dirs {
					removeWalk(v, true)
				}
			}
		}
		<-ticker.C
	}
}

// removeWalk 迭代方式删除空目录
func removeWalk(baseDir string, deep bool) {
	// 使用栈来模拟递归过程
	dirStack := []struct {
		path    string
		depth   bool
		visited bool // 标记是否已访问过该目录（第二次检查是否为空）
	}{
		{path: baseDir, depth: deep, visited: false},
	}

	for len(dirStack) > 0 {
		// 弹出栈顶目录
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentDir := current.path
		currentDeep := current.depth
		visited := current.visited

		// 第一次访问：处理子目录
		if !visited {
			files, err := os.ReadDir(currentDir)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
				continue
			}

			if len(files) > 0 && currentDeep {
				// 先重新压入当前目录（标记为已访问），这样处理完子目录后会再次检查
				dirStack = append(dirStack, struct {
					path    string
					depth   bool
					visited bool
				}{path: currentDir, depth: currentDeep, visited: true})

				// 压入所有子目录（未访问）
				for _, path := range files {
					pathAbs := filepath.Join(currentDir, path.Name())
					if path.IsDir() {
						dirStack = append(dirStack, struct {
							path    string
							depth   bool
							visited bool
						}{path: pathAbs, depth: currentDeep, visited: false})
					}
				}
			} else if len(files) == 0 {
				// 空目录直接删除
				if err := os.Remove(currentDir); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				}
			}
		} else {
			// 第二次访问：检查并删除空目录
			if files, err := os.ReadDir(currentDir); err == nil && len(files) == 0 {
				if err := os.Remove(currentDir); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				}
			}
		}
	}
}

// writeNoPic 无图时返回默认图片
func (fs *fileService) writeNoPic(c *gin.Context) {
	// 先尝试读取已缓存的图片
	noPicMutex.RLock()
	if noPic != nil && contentType != "" {
		defer noPicMutex.RUnlock()
		c.Data(http.StatusOK, contentType, noPic)
		return
	}
	noPicMutex.RUnlock()

	// 缓存未命中，获取默认图片
	noPicMutex.Lock()
	defer noPicMutex.Unlock()

	// 双重检查锁定模式
	if noPic != nil && contentType != "" {
		c.Data(http.StatusOK, contentType, noPic)
		return
	}

	// 从网络获取默认图片
	imgURL := "https://gimg2.baidu.com/image_search/src=http%3A%2F%2Fwww.bianminchewu.com%2Fimgs%2F18%2F0804%2F1533370482927057.png&refer=http%3A%2F%2Fwww.bianminchewu.com&app=2002&size=f9999,10000&q=a80&n=0&g=0n&fmt=auto?sec=1666008344&t=9da005a04a6c6209595f46dd05477c0f"
	response, err := httpClient.Get(imgURL)
	if err != nil || response.StatusCode != http.StatusOK {
		utils.InfoFormat("获取默认图片失败: %v", err)
		c.Status(http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			utils.InfoFormat("关闭响应体失败: %v", closeErr)
		}
	}()

	noPic, err = io.ReadAll(response.Body)
	if err != nil {
		utils.InfoFormat("读取默认图片失败: %v", err)
		c.Status(http.StatusServiceUnavailable)
		return
	}

	contentType = response.Header.Get("Content-Type")
	c.Data(http.StatusOK, contentType, noPic)
}

// DeleteOne 删除指定文件夹下的指定文件名的文件
func (fs *fileService) DeleteOne(dirName string, fileName string) {
	if len(fileName) == 0 {
		return
	}

	files, err := os.ReadDir(dirName)
	if err != nil {
		utils.InfoFormat("读取目录失败: %s, 错误: %v", dirName, err)
		return
	}

	deleted := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), fileName) {
			path := filepath.Join(dirName, f.Name())
			if err := os.Remove(path); err != nil {
				utils.InfoFormat("删除文件失败: %s, 错误: %v", path, err)
			} else {
				deleted = true
			}
		}
	}

	// 如果有删除操作，检查目录是否为空
	if deleted {
		filesThen, err := os.ReadDir(dirName)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", dirName, err)
			return
		}
		if len(filesThen) == 0 {
			fs.UpDirClear(dirName)
		}
	}
}

// DownDeleteDir 迭代方式删除文件夹及其内容
func (fs *fileService) DownDeleteDir(dirname string) {
	// 后序遍历的处理栈
	type stackItem struct {
		path    string
		visited bool
	}
	postOrderStack := []stackItem{{path: dirname, visited: false}}

	for len(postOrderStack) > 0 {
		current := postOrderStack[len(postOrderStack)-1]
		postOrderStack = postOrderStack[:len(postOrderStack)-1]
		currentPath := current.path
		visited := current.visited

		if !visited {
			// 第一次访问，先处理子项
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			// 重新压入当前目录，标记为已访问
			postOrderStack = append(postOrderStack, stackItem{path: currentPath, visited: true})

			// 压入所有子项（先文件后目录，以确保目录在最后处理）
			// 注意：我们需要逆序压入目录，以确保处理顺序与原递归一致
			for i := len(files) - 1; i >= 0; i-- {
				ff := files[i]
				path := filepath.Join(currentPath, ff.Name())
				if ff.IsDir() {
					postOrderStack = append(postOrderStack, stackItem{path: path, visited: false})
				} else {
					// 直接删除文件
					if err := os.Remove(path); err != nil {
						utils.InfoFormat("删除文件失败: %s, 错误: %v", path, err)
					}
				}
			}
		} else {
			// 第二次访问，删除空目录
			if err := os.Remove(currentPath); err != nil {
				utils.InfoFormat("删除目录失败: %s, 错误: %v", currentPath, err)
			}
		}
	}

	// 删除完所有内容后，处理父目录
	parentDir := filepath.Dir(dirname)
	fs.UpDirClear(parentDir)
}

// UpDirClear 迭代方式向上删除空文件夹
func (fs *fileService) UpDirClear(dirname string) {
	currentDir := dirname

	// 迭代处理父目录链
	for {
		// 避免删除根目录或系统目录
		if filepath.Clean(currentDir) == "/" || filepath.Dir(currentDir) == currentDir {
			break
		}

		files, err := os.ReadDir(currentDir)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
			break // 读取失败则停止处理
		}

		if len(files) == 0 {
			if err := os.Remove(currentDir); err != nil {
				utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				break // 删除失败则停止处理
			}
			// 继续处理父目录
			currentDir = filepath.Dir(currentDir)
		} else {
			// 目录不为空，停止处理
			break
		}
	}
}

func GetIpAddr() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		utils.InfoFormat("GetIpAddrError:%v \n\n", err)
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(localAddr.String(), ":")[0]
	return ip
}

// ScanAll 全局扫描
func (fs *fileService) ScanAll() {
	// 检查是否已经有索引构建任务在执行
	if atomic.LoadInt32(&consts.IndexDone) > 0 {
		AddLogMemory("索引构建任务正在执行中，跳过本次扫描")
		return
	}

	// 统计初始化
	consts.TypeMenu.Clear()
	consts.SeriesCount.Clear()
	consts.TagMenu.Clear()
	consts.SmallDir = []consts.MenuSize{}

	// 初始化查询条件
	setting := consts.OSSetting
	dirList := make([]string, len(setting.Dirs))
	copy(dirList, setting.Dirs)

	queryTypes := make([]string, 0)
	queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
	queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)

	consts.InitFolderTime()
	// 设置索引构建状态
	atomic.StoreInt32(&consts.IndexDone, 1)
	defer atomic.StoreInt32(&consts.IndexDone, 0)

	fs.Walks(dirList, queryTypes)
	SearchEngin.buildIndexEngin()
	consts.LastScanTime = time.Now()

	// 清空切片
	clear(queryTypes)
	clear(dirList)
}

// ScanTarget 扫描指定文件夹
func (fs *fileService) ScanTarget(baseDir string) {
	// 添加任务到队列
	scanQueue.AddTask(baseDir)
}

// Walks 并发扫描多文件夹并返回所有文件
func (fs *fileService) Walks(baseDir []string, types []string) []model.Movie {
	var wg sync.WaitGroup
	var result []model.Movie
	dirSize := len(baseDir)

	// 检查是否已经有索引构建任务在执行
	if atomic.LoadInt32(&consts.IndexDone) == 0 {
		// 只有当没有索引构建任务时，才设置IndexDone
		atomic.StoreInt32(&consts.IndexDone, int32(dirSize))
		defer atomic.StoreInt32(&consts.IndexDone, 0)
	}

	// 重置搜索引擎
	SearchEngin.Reset()

	// 创建一个通道来收集扫描结果
	resultChan := make(chan []model.Movie, dirSize)
	defer close(resultChan)

	wg.Add(dirSize)
	for i := 0; i < dirSize; i++ {
		go func(dir string) {
			defer wg.Done()
			fs.goWalkWithResult(dir, types, resultChan)
		}(baseDir[i])
	}

	// 等待所有扫描完成
	wg.Wait()

	// 收集结果
	for i := 0; i < dirSize; i++ {
		if files := <-resultChan; len(files) > 0 {
			result = append(result, files...)
		}
	}

	return result
}

// goWalkWithResult 协程方法扫描单个文件夹并返回结果
func (fs *fileService) goWalkWithResult(baseDir string, types []string, resultChan chan<- []model.Movie) {
	defer atomic.AddInt32(&consts.IndexDone, -1)

	start := time.Now()
	files, size := fs.WalkInnter(baseDir, types, 0, true, baseDir)

	// 存储扫描结果
	SearchEngin.setBucket(baseDir, newInstanceWithFiles(baseDir, files))

	// 记录扫描统计信息
	ti := time.Since(start)
	thisTime := consts.MenuSize{
		Name:    baseDir,
		Cnt:     ti.Milliseconds(),
		Size:    int64(len(files)),
		SizeStr: utils.GetSizeStr(size),
	}

	AddLogMemory("扫描目录:[%s] 耗时:[%d] 大小:[%s]", baseDir, ti.Milliseconds(), utils.GetSizeStr(size))
	consts.AddFolderTime(thisTime)

	// 发送结果到通道
	resultChan <- files
}

// Walk 迭代方式遍历目录获取文件库
func (fs *fileService) Walk(baseDir string, types []string, deep bool) []model.Movie {
	var result []model.Movie
	typeSet := utils.ToSet(types)

	// 使用栈来模拟递归过程
	dirStack := []string{baseDir}

	for len(dirStack) > 0 {
		// 弹出当前目录
		currentDir := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]

		files, err := os.ReadDir(currentDir)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
			continue
		}

		if len(files) > 0 {
			// 逆序压入子目录，确保处理顺序与原递归一致
			for i := len(files) - 1; i >= 0; i-- {
				path := files[i]
				pathAbs := filepath.Join(currentDir, path.Name())

				if path.IsDir() && deep {
					// 目录压入栈中，稍后处理
					dirStack = append(dirStack, pathAbs)
				} else {
					// 直接处理文件
					info, err := path.Info()
					if err != nil {
						utils.InfoFormat("获取文件信息失败: %s, 错误: %v", pathAbs, err)
						continue
					}

					name := path.Name()
					suffix := utils.GetSuffix(name)
					if utils.HasItemSet(typeSet, suffix) {
						file := model.EasyFile(currentDir, pathAbs, name, suffix, info.Size(), info.ModTime(), "")
						result = append(result, file)
					}
				}
			}
		} else {
			// 尝试删除空目录
			if err := os.Remove(currentDir); err != nil {
				utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
			}
		}
	}

	return result
}

// WalkInnter 内部文件夹搜索方法（迭代实现）
/**
currentDir 文件夹路径
types 扫描类型
totalSize 总数
queryChild 是否递归
basePath 基础路径
*/
func (fs *fileService) WalkInnter(currentDir string, types []string, totalSize int64, queryChild bool, basePath string) ([]model.Movie, int64) {
	var result []model.Movie
	typeSet := utils.ToSet(types)
	// 使用栈来模拟递归过程
	type stackItem struct {
		path       string
		queryChild bool
		visited    bool
	}
	dirStack := []stackItem{{path: currentDir, queryChild: queryChild, visited: false}}

	// 用于跟踪每个目录的大小
	sizeMap := make(map[string]int64)
	// 用于存储每个目录找到的文件
	fileMap := make(map[string][]model.Movie)

	// 先将根目录的初始大小设为0
	sizeMap[currentDir] = 0
	fileMap[currentDir] = []model.Movie{}

	for len(dirStack) > 0 {
		// 弹出当前目录
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentPath := current.path
		currentQueryChild := current.queryChild
		visited := current.visited

		if !visited {
			// 第一次访问，处理子项
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			// 重新压入当前目录，标记为已访问
			dirStack = append(dirStack, stackItem{path: currentPath, queryChild: currentQueryChild, visited: true})

			if len(files) > 0 {
				// 逆序压入子目录，确保处理顺序与原递归一致
				for i := len(files) - 1; i >= 0; i-- {
					path := files[i]
					pathAbs := filepath.Join(currentPath, path.Name())

					if path.IsDir() && currentQueryChild {
						// 压入子目录，并初始化其大小和文件列表
						dirStack = append(dirStack, stackItem{path: pathAbs, queryChild: currentQueryChild, visited: false})
						sizeMap[pathAbs] = 0
						fileMap[pathAbs] = []model.Movie{}
					} else {
						// 直接处理文件
						info, err := path.Info()
						if err != nil {
							utils.InfoFormat("获取文件信息失败: %s, 错误: %v", pathAbs, err)
							continue
						}

						// 更新当前目录的大小
						sizeMap[currentPath] += info.Size()

						name := path.Name()
						suffix := utils.GetSuffix(name)

						if utils.HasItemSet(typeSet, suffix) {
							file := model.EasyFile(currentPath, pathAbs, name, suffix, info.Size(), info.ModTime(), basePath)
							fileMap[currentPath] = append(fileMap[currentPath], file)
						}
					}
				}
			} else {
				// 尝试删除昨天创建的空目录
				if emptyFile, err := os.Stat(currentPath); err == nil {
					yesterday := time.Now().AddDate(0, 0, -1)
					if emptyFile.ModTime().Day() == yesterday.Day() &&
						emptyFile.ModTime().Month() == yesterday.Month() &&
						emptyFile.ModTime().Year() == yesterday.Year() {
						if err := os.Remove(currentPath); err != nil {
							utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentPath, err)
						}
					}
				}
			}
		} else {
			// 第二次访问，处理目录级别的操作
			// 记录小目录信息
			currentSize := sizeMap[currentPath]
			if currentSize <= 20000000 && utils.IndexOf(consts.OSSetting.Dirs, currentPath) < 0 {
				consts.SmallDir = append(consts.SmallDir, consts.NewMenuSizeFold(currentPath, currentSize, true))
			}

			// 如果当前不是根目录，将当前目录的大小和文件合并到父目录
			if currentPath != currentDir {
				parentPath := filepath.Dir(currentPath)
				// 更新父目录的大小
				sizeMap[parentPath] += currentSize
				// 合并文件列表
				fileMap[parentPath] = append(fileMap[parentPath], fileMap[currentPath]...)
			} else {
				// 根目录的文件直接添加到结果中
				result = append(result, fileMap[currentPath]...)
			}
		}
	}

	totalSize += sizeMap[currentDir]
	return result, sizeMap[currentDir]
}

// TaskExecuting 任务执行调度器
func (fs *fileService) TaskExecuting() {
	// 任务分类
	taskGroups := struct {
		todos         []model.TransferTaskModel
		todosCuts     []model.TransferTaskModel
		todosMerges   []model.TransferTaskModel
		executing     []model.TransferTaskModel
		executingCuts []model.TransferTaskModel
		executingMerges []model.TransferTaskModel
	}{}

	// 遍历并分类任务
	consts.TransferTaskMutex.RLock()
	for _, t := range consts.TransferTask {
		switch {
		case strings.EqualFold(t.Status, model.StatusPending):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				taskGroups.todosCuts = append(taskGroups.todosCuts, t)
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				taskGroups.todosMerges = append(taskGroups.todosMerges, t)
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				taskGroups.todos = append(taskGroups.todos, t)
			}
		case strings.EqualFold(t.Status, model.StatusExecuting):
			switch {
			case strings.EqualFold(t.Type, model.TaskTypeCut):
				taskGroups.executingCuts = append(taskGroups.executingCuts, t)
			case strings.EqualFold(t.Type, model.TaskTypeMerge):
				taskGroups.executingMerges = append(taskGroups.executingMerges, t)
			case strings.EqualFold(t.Type, model.TaskTypeTrans):
				taskGroups.executing = append(taskGroups.executing, t)
			}
		}
	}
	consts.TransferTaskMutex.RUnlock()

	if len(taskGroups.executing) == 0 && len(taskGroups.todos) > 0 {
		go fs.TransferFormatter(taskGroups.todos[0])
	}
	if len(taskGroups.executingCuts) == 0 && len(taskGroups.todosCuts) > 0 {
		go fs.CutFormatter(taskGroups.todosCuts[0])
	}
	if len(taskGroups.executingMerges) == 0 && len(taskGroups.todosMerges) > 0 {
		go fs.MergeFiles(taskGroups.todosMerges[0])
	}

	// 继续调度（支持优雅退出）
	if TaskCtx.Err() == nil {
		time.AfterFunc(2*time.Second, func() {
			select {
			case <-TaskCtx.Done():
				utils.InfoFormat("任务调度器已停止")
				return
			default:
				fs.TaskExecuting()
			}
		})
	}
}


// TransferFormatter 视频转码格式化
func (fs *fileService) TransferFormatter(model model.TransferTaskModel) utils.Result {
	// 根据视频编码选择对应的转码方法
	switch model.VCode {
	case "h264":
		return fs.TransferFormatter264(model)
	case "h265":
		return fs.TransferFormatter265(model)
	default:
		// 默认使用直接复制编码
		return fs.transferFormatWithCopy(model)
	}
}

// cleanupSourceIfNeeded 如果配置了转码后删除源文件，则执行删除
func (fs *fileService) cleanupSourceIfNeeded(path string) {
	if consts.GetOSSetting().CutThenDelete {
		if err := os.Remove(path); err != nil {
			utils.InfoFormat("删除源文件失败: %s, 错误: %v", path, err)
		}
	}
}
func (fs *fileService) transferFormatWithCopy(model model.TransferTaskModel) utils.Result {
	from := model.Path
	suffix := utils.GetSuffix(model.Path)

	// 确保输出格式与输入不同
	if suffix == model.To {
		if suffix == "mp4" {
			model.To = "mov"
		} else {
			model.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+model.To)
	thisNow := model.CreateTime

	// 直接复制编码
	args := []string{"-i", from, "-vcodec", "copy", dest}
	res := fs.ffmepgExec(args, thisNow)

	// 成功后删除源文件（如果配置了）
	if res.IsSuccess() {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// TransferFormatter264 H264编码转码
func (fs *fileService) TransferFormatter264(model model.TransferTaskModel) utils.Result {
	from := model.Path
	suffix := utils.GetSuffix(model.Path)

	if suffix == model.To {
		if suffix == "mp4" {
			model.To = "mov"
		} else {
			model.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+model.To)
	thisNow := model.CreateTime

	encoder := fs.getH264Encoder()
	decodeParams := fs.getHwDecodeParams()
	qualityParam := fs.getHwQualityParam()
	args := []string{}
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}
	args = append(args, "-i", from, "-c:v", encoder, qualityParam, "23", dest)
	res := fs.ffmepgExec(args, thisNow)

	if res.IsSuccess() {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// TransferFormatter265 H265编码转码
func (fs *fileService) TransferFormatter265(model model.TransferTaskModel) utils.Result {
	from := model.Path
	suffix := utils.GetSuffix(model.Path)

	if suffix == model.To {
		if suffix == "mp4" {
			model.To = "mov"
		} else {
			model.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+model.To)
	thisNow := model.CreateTime

	encoder := fs.getH265Encoder()
	decodeParams := fs.getHwDecodeParams()
	qualityParam := fs.getHwQualityParam()
	args := []string{}
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}
	args = append(args, "-i", from, "-c:v", encoder, qualityParam, "28", dest)
	res := fs.ffmepgExec(args, thisNow)

	if res.IsSuccess() {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// MergeFiles 合并文件
func (fs *fileService) MergeFiles(model model.TransferTaskModel) utils.Result {
	thisNow := model.CreateTime

	// 构建ffmpeg参数
	args := []string{"-f", "concat", "-safe", "0", "-i", model.ConcatFile, "-vcodec", "copy", model.Dest}
	res := fs.ffmepgExec(args, thisNow)

	// 成功后删除源文件（如果配置了）
	if res.IsSuccess() && model.DeleteSource {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// CutFormatter 视频剪辑格式化
func (fs *fileService) CutFormatter(model model.TransferTaskModel) utils.Result {
	from := model.Path
	suffix := utils.GetSuffix(model.Path)

	// 选择目标格式（与源格式不同）
	toSuffix := "mkv"
	if suffix == "mkv" {
		toSuffix = "mp4"
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+toSuffix)
	thisNow := model.CreateTime

	// 构建ffmpeg参数
	args := []string{"-i", from, "-ss", model.Start, "-t", model.End, "-c", "copy", dest}
	res := fs.ffmepgExec(args, thisNow)

	// 成功后删除源文件（如果配置了）
	if res.IsSuccess() && consts.GetOSSetting().CutThenDelete {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// CutImage 视频截图
func (fs *fileService) CutImage(path string, typeImage string, start string) utils.Result {
	// 初始化响应结果
	res := utils.NewSuccess()

	// 确定图片类型
	snapshot := false
	if !strings.EqualFold(typeImage, "Png") && !strings.EqualFold(typeImage, "Jpg") {
		snapshot = true
		typeImage = "Jpg" // 默认使用JPG格式
	}

	// 构建目标文件路径
	dest := strings.TrimSuffix(path, filepath.Ext(path))
	if snapshot {
		dest += time.Now().Format("-20060102150405")
	}
	dest += "." + strings.ToLower(typeImage)

	// 构建ffmpeg参数
	args := []string{"-y", "-ss", start}

	// 如果启用硬件加速，插入硬件解码参数加快截图速度
	decodeParams := fs.getHwDecodeParams()
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}

	args = append(args, "-i", path,
		"-f", "image2",
		"-vframes", "1",
		"-an",
		"-vcodec", "mjpeg",
		dest,
	)

	// 构建ffmpeg路径
	ffmpegPath := "ffmpeg.exe"
	if TempDir != "" {
		ffmpegPath = filepath.Join(TempDir, "ffmpeg.exe")
	}
	// 执行ffmpeg命令
	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		utils.InfoFormat("视频截图失败，输出: %v, 错误: %v", string(out), cmdErr)
		res = utils.NewFailByMsg("截图转换失败")

		// 检查是否成功生成了文件
		if utils.ExistsFiles(dest) {
			res.Data = utils.ImageToString(dest)
		}
		return res
	}

	// 成功时返回图片数据
	res.Data = utils.ImageToString(dest)
	return res
}

// ffmepgExec 执行ffmpeg命令
func (fs *fileService) ffmepgExec(args []string, thisNow time.Time) utils.Result {
	// 检查任务是否存在
	consts.TransferTaskMutex.Lock()
	task, exists := consts.TransferTask[thisNow]
	if !exists {
		consts.TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务不存在")
	}

	// 构建ffmpeg命令路径
	ffmpegPath := "ffmpeg.exe"
	if TempDir != "" {
		ffmpegPath = filepath.Join(TempDir, "ffmpeg.exe")
	}

	// 更新任务状态为执行中
	task.SetStatus("执行中")
	task.CreateTime = time.Now()
	task.Command = ffmpegPath + " " + strings.Join(args, "  ")
	consts.TransferTask[thisNow] = task
	consts.TransferTaskMutex.Unlock()

	utils.InfoFormat("执行命令: %v", task.Command)

	// 创建并执行命令
	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	// 获取命令输出
	out, cmdErr := cmd.CombinedOutput()

	consts.TransferTaskMutex.Lock()
	// 更新任务日志
	task.SetLog(string(out))
	task.FinishTime = time.Now()

	// 处理执行结果
	if cmdErr != nil {
		task.SetStatus("执行失败")
		consts.TransferTask[thisNow] = task
		consts.TransferTaskMutex.Unlock()

		utils.InfoFormat("命令执行失败: %v, 错误: %v, 参数: %v", string(out), cmdErr, args)
		return utils.NewFailByMsg("转换失败")
	}

	// 执行成功
	task.SetStatus("成功")
	consts.TransferTask[thisNow] = task
	consts.TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("转换成功")
}

// detectHwAccel 检测平台上可用的最佳硬件编码器（惰性调用，首次转码时自动识别）
// 如果 hwAccelForceDetect 为 true，则强制重新检测并重置标志
func (fs *fileService) detectHwAccel() {
	hwAccelMutex.Lock()
	defer hwAccelMutex.Unlock()

	// 如果未检测到过，或用户要求强制重新检测
	if hwAccelDetected && !hwAccelForceDetect {
		return
	}
	forceDetect := hwAccelForceDetect
	hwAccelForceDetect = false // 重置标志

	// 清除之前的检测结果（强制重新检测时）
	if forceDetect {
		hwAccelH264Encoder = ""
		hwAccelH265Encoder = ""
		hwAccelModeName = ""
		hwAccelDecodeParams = ""
	}

	ffmpegPath := "ffmpeg.exe"
	if TempDir != "" {
		ffmpegPath = filepath.Join(TempDir, "ffmpeg.exe")
	}

	cmd := exec.Command(ffmpegPath, "-encoders")
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		utils.InfoFormat("硬件加速检测失败(ffmpeg -encoders): %v", err)
		// 失败时不标记 hwAccelDetected，允许后续重新检测
		return
	}

	output := string(out)

	// 按优先级检测可用编码器：NVENC > AMF > QSV > VAAPI > VideoToolbox
	type hwEncoder struct {
		h264   string
		h265   string
		name   string
		decode string // 对应的硬件解码参数
	}
	encoders := []hwEncoder{
		{"h264_nvenc", "hevc_nvenc", "NVIDIA NVENC", "-hwaccel cuda -hwaccel_output_format cuda"},
		{"h264_amf", "hevc_amf", "AMD AMF", "-hwaccel dxva2"},
		{"h264_qsv", "hevc_qsv", "Intel QSV", "-hwaccel qsv -hwaccel_output_format qsv"},
		{"h264_vaapi", "hevc_vaapi", "VAAPI", "-hwaccel vaapi -hwaccel_output_format vaapi"},
		{"h264_videotoolbox", "hevc_videotoolbox", "VideoToolbox", "-hwaccel videotoolbox"},
	}

	for _, e := range encoders {
		h264Ok := strings.Contains(output, e.h264)
		h265Ok := strings.Contains(output, e.h265)
		if h264Ok && h265Ok {
			hwAccelH264Encoder = e.h264
			hwAccelH265Encoder = e.h265
			hwAccelModeName = e.name
			hwAccelDecodeParams = e.decode
			hwAccelDetected = true // 成功检测后才标记
			utils.InfoFormat("硬件加速检测成功: %s (h264=%s, h265=%s) 解码参数=%s", e.name, e.h264, e.h265, e.decode)
			return
		}
	}

	// 回退：只检测到 H264 编码器也凑合能用
	for _, e := range encoders {
		if strings.Contains(output, e.h264) {
			hwAccelH264Encoder = e.h264
			hwAccelModeName = e.name
			hwAccelDecodeParams = e.decode
			hwAccelDetected = true // 成功检测后才标记
			utils.InfoFormat("硬件加速部分检测成功(仅H264): %s", e.name)
			return
		}
	}

	utils.InfoFormat("未检测到任何硬件加速编码器，将使用软件编码")
	hwAccelDetected = true // 检测完毕但未找到，标记为已检测过（避免每次转码都跑 ffmpeg）
}

// getH264Encoder 获取当前应使用的 H264 编码器
func (fs *fileService) getH264Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccelH264Encoder != "" {
			return hwAccelH264Encoder
		}
	}
	return "libx264"
}

// getH265Encoder 获取当前应使用的 H265 编码器
func (fs *fileService) getH265Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccelH265Encoder != "" {
			return hwAccelH265Encoder
		}
	}
	return "libx265"
}

// GetHwAccelModeName 暴露硬件加速模式名称给外部
func GetHwAccelModeName() string {
	return hwAccelModeName
}

// getHwDecodeParams 获取硬件解码参数（在 -i 之前插入）
func (fs *fileService) getHwDecodeParams() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccelDecodeParams != "" {
			return hwAccelDecodeParams
		}
	}
	return ""
}

// getHwQualityParam 获取硬件编码器的质量参数
// 软件编码器(libx264/libx265)用 -crf，硬件编码器用 -q
func (fs *fileService) getHwQualityParam() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccelH264Encoder != "" || hwAccelH265Encoder != "" {
			return "-q"
		}
	}
	return "-crf"
}

// HwAccelSettingChanged 检查硬件加速设置是否发生变化（与上次保存时不同）
var lastHwAccelSetting bool
func HwAccelSettingChanged() bool {
	current := consts.GetOSSetting().HardwareAcceleration
	hwAccelMutex.Lock()
	defer hwAccelMutex.Unlock()
	if lastHwAccelSetting != current {
		lastHwAccelSetting = current
		return true
	}
	return false
}

// ForceHwAccelDetect 强制下次转码时重新检测硬件加速
func ForceHwAccelDetect() {
	hwAccelMutex.Lock()
	defer hwAccelMutex.Unlock()
	hwAccelForceDetect = true
	hwAccelDetected = false
	utils.InfoFormat("硬件加速设置已更改，下次转码时将重新检测")
}
