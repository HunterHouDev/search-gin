package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"search-gin/internal/model"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	TaskCtx, TaskCancel = context.WithCancel(context.Background())
)

type fileService struct {
}

type stackItem struct {
	path       string
	queryChild bool
	visited    bool
}
var hwAccel = struct {
	h264  string
	h265  string
	mode  string
	det   bool
	mu    sync.Mutex
	dec   string
	force bool
}{}

var (
	noPic       []byte
	contentType = "image/png"
)

func init() {
	var buf bytes.Buffer
	if err := generatePlaceholderPNG(&buf); err != nil {
		panic("初始化默认图片失败: " + err.Error())
	}
	noPic = buf.Bytes()
}

func (fs *fileService) GetPng(c *gin.Context) {
	id := c.Param("path")
	file := SearchApp.FindOne(id)
	if !file.IsNull() {
		if utils.ExistsFiles(file.Png) {
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

func (fileService *fileService) HeartBeat() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-TaskCtx.Done():
			return
		case <-ticker.C:
			if !consts.GetOSSetting().EnableTimeScan || time.Since(consts.LastScanTime).Seconds() <= 180 {
				continue
			}
			for _, dir := range consts.GetOSSetting().Dirs {
				removeWalk(dir, true)
			}
		}
	}
}

// removeWalk 迭代方式删除空目录
func removeWalk(baseDir string, deep bool) {
	dirStack := []stackItem{{path: baseDir, queryChild: deep, visited: false}}

	for len(dirStack) > 0 {
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentDir := current.path
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentDir)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
				continue
			}

			if len(files) > 0 && current.queryChild {
				dirStack = append(dirStack, stackItem{path: currentDir, queryChild: current.queryChild, visited: true})

				for _, fi := range files {
					pathAbs := filepath.Join(currentDir, fi.Name())
					if fi.IsDir() {
						dirStack = append(dirStack, stackItem{path: pathAbs, queryChild: current.queryChild, visited: false})
					}
				}
			} else if len(files) == 0 {
				if err := os.Remove(currentDir); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				}
			}
		} else {
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
	c.Data(http.StatusOK, contentType, noPic)
}

// generatePlaceholderPNG 生成一个简单的占位PNG图片
func generatePlaceholderPNG(w io.Writer) error {
	width, height := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充灰色背景
	bgColor := color.RGBA{R: 204, G: 204, B: 204, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// 绘制简单的 "?" 图标（十字形）
	lineColor := color.RGBA{R: 153, G: 153, B: 153, A: 255}
	centerX, centerY := width/2, height/2
	thickness := 6
	size := 30

	// 水平线
	for x := centerX - size; x <= centerX+size; x++ {
		for dy := -thickness / 2; dy <= thickness/2; dy++ {
			img.Set(x, centerY+dy, lineColor)
		}
	}
	// 竖直线（下半部分，形成 ? 的竖）
	for y := centerY; y <= centerY+size; y++ {
		for dx := -thickness / 2; dx <= thickness/2; dx++ {
			img.Set(centerX+dx, y, lineColor)
		}
	}
	// 顶部弧线简化为小方块
	for x := centerX - size + 10; x < centerX+size-10; x++ {
		for y := centerY - size; y < centerY-size+thickness; y++ {
			img.Set(x, y, lineColor)
		}
	}
	// 左侧弧线
	for x := centerX - size; x < centerX-size+thickness; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}
	// 右侧弧线
	for x := centerX + size - thickness; x < centerX+size; x++ {
		for y := centerY - size; y < centerY; y++ {
			img.Set(x, y, lineColor)
		}
	}

	return png.Encode(w, img)
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
	postOrderStack := []stackItem{{path: dirname, visited: false}}

	for len(postOrderStack) > 0 {
		current := postOrderStack[len(postOrderStack)-1]
		postOrderStack = postOrderStack[:len(postOrderStack)-1]
		currentPath := current.path
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			postOrderStack = append(postOrderStack, stackItem{path: currentPath, visited: true})

			for i := len(files) - 1; i >= 0; i-- {
				ff := files[i]
				path := filepath.Join(currentPath, ff.Name())
				if ff.IsDir() {
					postOrderStack = append(postOrderStack, stackItem{path: path, visited: false})
				} else {
					if err := os.Remove(path); err != nil {
						utils.InfoFormat("删除文件失败: %s, 错误: %v", path, err)
					}
				}
			}
		} else {
			if err := os.Remove(currentPath); err != nil {
				utils.InfoFormat("删除目录失败: %s, 错误: %v", currentPath, err)
			}
		}
	}

	parentDir := filepath.Dir(dirname)
	fs.UpDirClear(parentDir)
}

// UpDirClear 迭代方式向上删除空文件夹
func (fs *fileService) UpDirClear(dirname string) {
	currentDir := dirname

	for {
		if filepath.Clean(currentDir) == "/" || filepath.Dir(currentDir) == currentDir {
			break
		}

		files, err := os.ReadDir(currentDir)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
			break
		}

		if len(files) == 0 {
			if err := os.Remove(currentDir); err != nil {
				utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				break
			}
			currentDir = filepath.Dir(currentDir)
		} else {
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
func (fs *fileService) ScanAll() int {
 setting := consts.GetOSSetting()
 dirCount := len(setting.Dirs)
 dirList := make([]string, dirCount)
 copy(dirList, setting.Dirs)
 AddLogMemory("Plan to ScanAll dirTotal: %d, dirList: %v", dirCount, dirList)
 if !atomic.CompareAndSwapInt32(&consts.IndexNumber, 0, int32(dirCount)) {
  AddLogMemory("索引构建任务正在执行中，剩余数量：%d", atomic.LoadInt32(&consts.IndexNumber))
  return dirCount
 }

 // 初始化扫描进度
 consts.SpMu.Lock()
 consts.Sp = consts.ScanProgress{
  Phase:            "scanning",
  TotalDirs:        dirCount,
  CompletedDirs:    0,
  CurrentDir:       "",
  ScannedFiles:     0,
  TotalBuckets:     dirCount,
  ProcessedBuckets: 0,
  CurrentPhase:     "正在扫描目录...",
 }
 consts.SpMu.Unlock()

 consts.TypeMenu.Clear()
 consts.SeriesCount.Clear()
 consts.TagMenu.Clear()
 consts.ClearSmallDir()

 queryTypes := make([]string, 0)
 queryTypes = utils.ExtendsItems(queryTypes, setting.VideoTypes)
 queryTypes = utils.ExtendsItems(queryTypes, setting.DocsTypes)
 queryTypes = utils.ExtendsItems(queryTypes, setting.ImageTypes)
 consts.InitFolderTime()
 fs.Walks(dirList, queryTypes)

 // 一致性检查：验证 bucket 数量和目录数量
 bucketCount := SearchEngin.BucketCount()
 indexNumber := atomic.LoadInt32(&consts.IndexNumber)
 AddLogMemory("ScanAll 一致性检查: BucketCount=%d, IndexNumber=%d, Expected=%d", bucketCount, indexNumber, dirCount)
 if bucketCount != int32(dirCount) {
  AddLogMemory("警告: BucketCount(%d) != Expected(%d)，可能存在并发问题", bucketCount, dirCount)
 }

 // 切换到索引构建阶段
 consts.SpMu.Lock()
 consts.Sp.Phase = "building"
 consts.Sp.CurrentPhase = "正在构建索引..."
 consts.SpMu.Unlock()

 consts.LastScanTime = time.Now()

 // 扫描完成
 consts.SpMu.Lock()
 consts.Sp.Phase = "done"
 consts.Sp.CompletedDirs = consts.Sp.TotalDirs
 consts.Sp.CurrentPhase = "扫描完成"
 consts.SpMu.Unlock()

 return dirCount
}

// ScanTarget 扫描指定文件夹
func (fs *fileService) ScanTarget(baseDir string) {
	scanQueue.AddTask(baseDir)
}

// Walks 并发扫描多文件夹并返回所有文件
func (fs *fileService) Walks(baseDir []string, types []string) []model.Movie {
	var wg sync.WaitGroup
	var result []model.Movie
	dirSize := len(baseDir)

	SearchEngin.Reset()

	resultChan := make(chan scanResult, dirSize)

	wg.Add(dirSize)
	for i := 0; i < dirSize; i++ {
		go func(dir string) {
			defer wg.Done()
			fs.goWalkWithResult(dir, types, resultChan)
		}(baseDir[i])
	}

	wg.Wait()
	close(resultChan)

	if dirSize == 0 {
		return result
	}

	// 收集所有扫描结果，批量重建索引
	buckets := make(map[string]*bucketFile, dirSize)
	for r := range resultChan {
		if r.bucket != nil && !r.bucket.isEmpty() {
			buckets[r.dir] = r.bucket
			for _, m := range r.bucket.FileLib {
				result = append(result, m)
			}
		}
	}

	AddLogMemory("Walks: 扫描完成, 共 %d 个目录, 准备重建索引", len(buckets))
	SearchEngin.rebuildWithBuckets(buckets)
	AddLogMemory("Walks: 索引重建完成")

	return result
}

// goWalkWithResult 协程方法扫描单个文件夹并返回结果
func (fs *fileService) goWalkWithResult(baseDir string, types []string, resultChan chan<- scanResult) {
	defer atomic.AddInt32(&consts.IndexNumber, -1)
	defer func() {
		consts.SpMu.Lock()
		consts.Sp.CompletedDirs++
		consts.SpMu.Unlock()
	}()

	// 更新当前正在扫描的目录
	consts.SpMu.Lock()
	consts.Sp.CurrentDir = baseDir
	consts.SpMu.Unlock()

	AddLogMemory("goWalkWithResult: 开始扫描目录 %s", baseDir)
	start := time.Now()
	files, size := fs.WalkInnter(baseDir, types, true, baseDir)

	AddLogMemory("goWalkWithResult: 扫描完成 %s, 发现 %d 个文件", baseDir, len(files))
	// 更新已扫描文件计数
	consts.SpMu.Lock()
	consts.Sp.ScannedFiles += int64(len(files))
	consts.SpMu.Unlock()

	bucket := newInstanceWithFiles(baseDir, files)

	ti := time.Since(start)
	thisTime := consts.MenuSize{
		Name:    baseDir,
		Cnt:     ti.Milliseconds(),
		Size:    int64(len(files)),
		SizeStr: utils.GetSizeStr(size),
	}
	AddLogMemory("扫描目录:[%s] 耗时:[%d] 大小:[%s],剩余目录数:%d", baseDir, ti.Milliseconds(), utils.GetSizeStr(size), atomic.LoadInt32(&consts.IndexNumber))
	consts.AddFolderTime(thisTime)

	select {
	case resultChan <- scanResult{dir: baseDir, bucket: bucket}:
	default:
	}
}

// scanResult 目录扫描结果
type scanResult struct {
	dir    string
	bucket *bucketFile
}

// Walk 遍历目录，获取指定类型文件列表（轻量版，不建索引）
func (fs *fileService) Walk(dirPath string, types []string, deep bool) []model.Movie {
	files, _ := fs.WalkInnter(dirPath, types, deep, dirPath)
	return files
}

func (fs *fileService) WalkInnter(currentDir string, types []string, queryChild bool, basePath string) ([]model.Movie, int64) {
	typeSet := utils.ToSet(types)

	dirStack := []stackItem{{path: currentDir, queryChild: queryChild, visited: false}}

	var allFiles []model.Movie
	sizeMap := make(map[string]int64)
	sizeMap[currentDir] = 0

	for len(dirStack) > 0 {
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentPath := current.path
		currentQueryChild := current.queryChild
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			dirStack = append(dirStack, stackItem{path: currentPath, queryChild: currentQueryChild, visited: true})

			for i := len(files) - 1; i >= 0; i-- {
				f := files[i]
				p := filepath.Join(currentPath, f.Name())

				if f.IsDir() && currentQueryChild {
					dirStack = append(dirStack, stackItem{path: p, queryChild: currentQueryChild, visited: false})
					sizeMap[p] = 0
				} else if !f.IsDir() {
					name := f.Name()
					suffix := utils.GetSuffix(name)

					if utils.HasItemSet(typeSet, suffix) {
						info, err := f.Info()
						if err != nil {
							utils.InfoFormat("获取文件信息失败: %s, 错误: %v", p, err)
							continue
						}
						movie := model.EasyFile(currentPath, p, name, suffix,
							info.Size(), info.ModTime(), basePath)
						allFiles = append(allFiles, movie)
						sizeMap[currentPath] += info.Size()
					} else {
						info, err := f.Info()
						if err != nil {
							utils.InfoFormat("获取文件信息失败: %s, 错误: %v", p, err)
							continue
						}
						sizeMap[currentPath] += info.Size()
					}
				}
			}

			if len(files) == 0 {
				if emptyFile, err := os.Stat(currentPath); err == nil {
					yesterday := time.Now().AddDate(0, 0, -1)
					if emptyFile.ModTime().Day() == yesterday.Day() &&
						emptyFile.ModTime().Month() == yesterday.Month() &&
						emptyFile.ModTime().Year() == yesterday.Year() {
						if utils.IndexOf(consts.GetOSSetting().Dirs, currentPath) < 0 {
							if err := os.RemoveAll(currentPath); err != nil {
								utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentPath, err)
							}
						}
					}
				}
			}
		} else {
			currentSize := sizeMap[currentPath]
			if currentSize <= 20000000 && utils.IndexOf(consts.GetOSSetting().Dirs, currentPath) < 0 {
				consts.AppendSmallDir(consts.NewMenuSizeFold(currentPath, currentSize, true))
			}

			if currentPath != currentDir {
				parentPath := filepath.Dir(currentPath)
				sizeMap[parentPath] += currentSize
			}
		}
	}

	return allFiles, sizeMap[currentDir]
}

// TaskExecuting 任务执行调度器
func (fs *fileService) TaskExecuting() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		fs.pollTasks()
		select {
		case <-TaskCtx.Done():
			utils.InfoFormat("任务调度器已停止")
			return
		case <-ticker.C:
		}
	}
}

// pollTasks 轮询并执行待处理任务
func (fs *fileService) pollTasks() {
	taskGroups := struct {
		todos           []model.TransferTaskModel
		todosCuts       []model.TransferTaskModel
		todosMerges     []model.TransferTaskModel
		executing       []model.TransferTaskModel
		executingCuts   []model.TransferTaskModel
		executingMerges []model.TransferTaskModel
	}{}

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
}

// TransferFormatter 视频转码格式化
func (fs *fileService) TransferFormatter(model model.TransferTaskModel) utils.Result {
	switch model.VCode {
	case "h264":
		return fs.TransferFormatter264(model)
	case "h265":
		return fs.TransferFormatter265(model)
	default:
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

	if suffix == model.To {
		if suffix == "mp4" {
			model.To = "mov"
		} else {
			model.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+model.To)
	thisNow := model.CreateTime

	args := []string{"-i", from, "-vcodec", "copy", dest}
	res := fs.ffmepgExec(args, thisNow)

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

	args := []string{"-f", "concat", "-safe", "0", "-i", model.ConcatFile, "-vcodec", "copy", model.Dest}
	res := fs.ffmepgExec(args, thisNow)

	if res.IsSuccess() && model.DeleteSource {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// CutFormatter 视频剪辑格式化
func (fs *fileService) CutFormatter(model model.TransferTaskModel) utils.Result {
	from := model.Path
	suffix := utils.GetSuffix(model.Path)

	toSuffix := "mkv"
	if suffix == "mkv" {
		toSuffix = "mp4"
	}

	dest := strings.ReplaceAll(model.Path, "."+suffix, "."+toSuffix)
	thisNow := model.CreateTime

	args := []string{"-i", from, "-ss", model.Start, "-t", model.End, "-c", "copy", dest}
	res := fs.ffmepgExec(args, thisNow)

	if res.IsSuccess() && consts.GetOSSetting().CutThenDelete {
		fs.cleanupSourceIfNeeded(model.Path)
	}

	return res
}

// CutImage 视频截图
func (fs *fileService) CutImage(path string, typeImage string, start string) utils.Result {
	res := utils.NewSuccess()

	snapshot := false
	if !strings.EqualFold(typeImage, "Png") && !strings.EqualFold(typeImage, "Jpg") {
		snapshot = true
		typeImage = "Jpg"
	}

	dest := strings.TrimSuffix(path, filepath.Ext(path))
	if snapshot {
		dest += time.Now().Format("-20060102150405")
	}
	dest += "." + strings.ToLower(typeImage)

	args := []string{"-y", "-ss", start}

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

	ffmpegPath := "ffmpeg.exe"
	if TempDir != "" {
		ffmpegPath = filepath.Join(TempDir, "ffmpeg.exe")
	}
	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		utils.InfoFormat("视频截图失败，输出: %v, 错误: %v", string(out), cmdErr)
		res = utils.NewFailByMsg("截图转换失败")

		if utils.ExistsFiles(dest) {
			res.Data = utils.ImageToString(dest)
		}
		return res
	}

	res.Data = utils.ImageToString(dest)
	return res
}

// ffmepgExec 执行ffmpeg命令
func (fs *fileService) ffmepgExec(args []string, thisNow time.Time) utils.Result {
	consts.TransferTaskMutex.Lock()
	task, exists := consts.TransferTask[thisNow]
	if !exists {
		consts.TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务不存在")
	}

	ffmpegPath := "ffmpeg.exe"
	if TempDir != "" {
		ffmpegPath = filepath.Join(TempDir, "ffmpeg.exe")
	}

	task.SetStatus("执行中")
	task.CreateTime = time.Now()
	task.Command = ffmpegPath + " " + strings.Join(args, "  ")
	consts.TransferTask[thisNow] = task
	consts.TransferTaskMutex.Unlock()

	utils.InfoFormat("执行命令: %v", task.Command)

	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()

	consts.TransferTaskMutex.Lock()
	task.SetLog(string(out))
	task.FinishTime = time.Now()

	if cmdErr != nil {
		task.SetStatus("执行失败")
		consts.TransferTask[thisNow] = task
		consts.TransferTaskMutex.Unlock()

		utils.InfoFormat("命令执行失败: %v, 错误: %v, 参数: %v", string(out), cmdErr, args)
		return utils.NewFailByMsg("转换失败")
	}

	task.SetStatus("成功")
	consts.TransferTask[thisNow] = task
	consts.TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("转换成功")
}

// detectHwAccel 检测平台上可用的最佳硬件编码器（惰性调用，首次转码时自动识别）
func (fs *fileService) detectHwAccel() {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()

	if hwAccel.det && !hwAccel.force {
		return
	}
	forceDetect := hwAccel.force
	hwAccel.force = false

	if forceDetect {
		hwAccel.h264 = ""
		hwAccel.h265 = ""
		hwAccel.mode = ""
		hwAccel.dec = ""
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
		return
	}

	output := string(out)

	type hwEncoder struct {
		h264   string
		h265   string
		name   string
		decode string
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
			hwAccel.h264 = e.h264
			hwAccel.h265 = e.h265
			hwAccel.mode = e.name
			hwAccel.dec = e.decode
			hwAccel.det = true
			utils.InfoFormat("硬件加速检测成功: %s (h264=%s, h265=%s) 解码参数=%s", e.name, e.h264, e.h265, e.decode)
			return
		}
	}

	for _, e := range encoders {
		if strings.Contains(output, e.h264) {
			hwAccel.h264 = e.h264
			hwAccel.mode = e.name
			hwAccel.dec = e.decode
			hwAccel.det = true
			utils.InfoFormat("硬件加速部分检测成功(仅H264): %s", e.name)
			return
		}
	}

	utils.InfoFormat("未检测到任何硬件加速编码器，将使用软件编码")
	hwAccel.det = true
}

// getH264Encoder 获取当前应使用的 H264 编码器
func (fs *fileService) getH264Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccel.h264 != "" {
			return hwAccel.h264
		}
	}
	return "libx264"
}

// getH265Encoder 获取当前应使用的 H265 编码器
func (fs *fileService) getH265Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccel.h265 != "" {
			return hwAccel.h265
		}
	}
	return "libx265"
}

// GetHwAccelModeName 暴露硬件加速模式名称给外部
func GetHwAccelModeName() string {
	return hwAccel.mode
}

// getHwDecodeParams 获取硬件解码参数（在 -i 之前插入）
func (fs *fileService) getHwDecodeParams() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccel.dec != "" {
			return hwAccel.dec
		}
	}
	return ""
}

// getHwQualityParam 获取硬件编码器的质量参数
func (fs *fileService) getHwQualityParam() string {
	if consts.GetOSSetting().HardwareAcceleration {
		fs.detectHwAccel()
		if hwAccel.h264 != "" || hwAccel.h265 != "" {
			return "-q"
		}
	}
	return "-crf"
}

// HwAccelSettingChanged 检查硬件加速设置是否发生变化（与上次保存时不同）
var lastHwAccelSetting bool

func HwAccelSettingChanged() bool {
	current := consts.GetOSSetting().HardwareAcceleration
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	if lastHwAccelSetting != current {
		lastHwAccelSetting = current
		return true
	}
	return false
}

// ForceHwAccelDetect 强制下次转码时重新检测硬件加速
func ForceHwAccelDetect() {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	hwAccel.force = true
	hwAccel.det = false
	utils.InfoFormat("硬件加速设置已更改，下次转码时将重新检测")
}
