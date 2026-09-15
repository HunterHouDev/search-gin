package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"search-gin/internal/model"
	"search-gin/pkg/utils"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

type TorrentService struct {
	client   *torrent.Client
	torrents map[metainfo.Hash]*torrent.Torrent
	mu       sync.RWMutex
	dataDir  string
}

var TorrentApp *TorrentService

func NewTorrentService(dataDir string) error {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dataDir
	cfg.NoUpload = false
	cfg.Seed = true

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("创建 torrent 客户端失败: %v", err)
	}

	TorrentApp = &TorrentService{
		client:   client,
		torrents: make(map[metainfo.Hash]*torrent.Torrent),
		dataDir:  dataDir,
	}

	utils.InfoFormat("Torrent 服务已启动，数据目录: %s", dataDir)
	return nil
}

func (ts *TorrentService) Close() {
	if ts.client != nil {
		ts.client.Close()
	}
}

type AddMagnetResult struct {
	InfoHash string         `json:"infoHash"`
	Name     string         `json:"name"`
	Files    []*TorrentFile `json:"files"`
}

func (ts *TorrentService) AddMagnet(magnetURI string) (*AddMagnetResult, error) {
	t, err := ts.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("添加磁力链失败: %v", err)
	}

	select {
	case <-t.GotInfo():
	case <-time.After(60 * time.Second):
		t.Drop()
		return nil, fmt.Errorf("获取种子信息超时")
	}

	infoHash := t.InfoHash().HexString()

	ts.mu.Lock()
	ts.torrents[t.InfoHash()] = t
	ts.mu.Unlock()

	var files []*TorrentFile
	for _, f := range t.Files() {
		files = append(files, &TorrentFile{
			Name:   filepath.Base(f.Path()),
			Path:   f.Path(),
			Length: f.Length(),
		})
	}

	utils.InfoFormat("已添加磁力链: %s, InfoHash: %s, 文件数: %d", t.Name(), infoHash, len(files))
	return &AddMagnetResult{
		InfoHash: infoHash,
		Name:     t.Name(),
		Files:    files,
	}, nil
}

type StartDownloadResult struct {
	Skipped  bool   `json:"skipped"`
	FilePath string `json:"filePath"`
	FileSize int64  `json:"fileSize"`
	Message  string `json:"message"`
}

func (ts *TorrentService) StartDownload(infoHash, filePath string) (*StartDownloadResult, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return nil, err
	}

	if filePath != "" {
		for _, f := range t.Files() {
			if f.Path() == filePath {
				fullPath := filepath.Join(ts.dataDir, t.Name(), filePath)
				if info, err := os.Stat(fullPath); err == nil && !info.IsDir() && info.Size() > 0 {
					utils.InfoFormat("文件已存在，跳过下载: %s, 大小: %d bytes", fullPath, info.Size())
					return &StartDownloadResult{
						Skipped:  true,
						FilePath: fullPath,
						FileSize: info.Size(),
						Message:  "文件已存在",
					}, nil
				}
				f.Download()
				ts.createTorrentTask(t, infoHash, filePath, fullPath)
				utils.InfoFormat("开始下载文件: %s, InfoHash: %s", filePath, infoHash)
				return &StartDownloadResult{
					Skipped:  false,
					FilePath: fullPath,
					FileSize: 0,
					Message:  "开始下载",
				}, nil
			}
		}
		return nil, fmt.Errorf("未找到文件: %s", filePath)
	}

	t.DownloadAll()
	ts.createTorrentTask(t, infoHash, "", "")
	utils.InfoFormat("开始下载全部文件: %s", infoHash)
	return &StartDownloadResult{
		Skipped:  false,
		FilePath: "",
		FileSize: 0,
		Message:  "开始下载全部文件",
	}, nil
}

// createTorrentTask 为已开始下载的种子在统一任务列表中创建条目（TaskTypeTorrent）。
// 同一种子同一文件的执行中任务不重复创建；任务队列已满时仅记录日志，不阻断下载。
func (ts *TorrentService) createTorrentTask(t *torrent.Torrent, infoHash, torrentFile, dest string) {
	torrentName := t.Name()
	fileName := torrentName
	if torrentFile != "" {
		fileName = filepath.Base(torrentFile)
	}

	// 读锁查重：同一种子同一文件的执行中任务不重复创建
	TransferTaskMutex.RLock()
	for _, tk := range TransferTask {
		if tk.Type == model.TaskTypeTorrent &&
			strings.EqualFold(tk.InfoHash, infoHash) &&
			tk.TorrentFile == torrentFile &&
			(tk.Status == model.StatusPending || tk.Status == model.StatusExecuting) {
			TransferTaskMutex.RUnlock()
			return
		}
	}
	TransferTaskMutex.RUnlock()

	task := model.NewTorrentTask(infoHash, torrentName, fileName, torrentFile, dest)
	task.SetStatus(model.StatusExecuting)

	TransferTaskMutex.Lock()
	if len(TransferTask) >= MaxTransferTaskCount {
		TransferTaskMutex.Unlock()
		utils.InfoFormat("任务队列已满，磁力下载任务未入列表（下载继续）: %s/%s", torrentName, torrentFile)
		return
	}
	TransferTask[task.ID] = task
	TransferTaskMutex.Unlock()
	utils.InfoFormat("磁力下载任务已创建: %s/%s, InfoHash: %s", torrentName, torrentFile, infoHash)
}

// syncTorrentTasks 将种子实时进度同步到统一任务列表（TaskTypeTorrent，每 5 秒一次）。
// 注意：进度取自整个种子的下载统计，同一种子多个文件任务共享同一进度值。
// 完成时仅标记任务状态，不触发索引扫描（有意设计：磁力下载文件不自动入库）。
func (ts *TorrentService) syncTorrentTasks() {
	type taskSnap struct {
		key      string
		progress int
		size     int64
		complete bool
		missing  bool
	}
	var updates []taskSnap

	// 先在 torrent 侧收集状态快照，避免长时间持有 TransferTaskMutex
	TransferTaskMutex.RLock()
	for key, task := range TransferTask {
		if task.Type != model.TaskTypeTorrent || task.Status != model.StatusExecuting || task.InfoHash == "" {
			continue
		}
		var h metainfo.Hash
		if err := h.FromHexString(task.InfoHash); err != nil {
			continue
		}
		ts.mu.RLock()
		t, ok := ts.torrents[h]
		ts.mu.RUnlock()
		if !ok {
			// 种子已被移除/清理，任务标记取消
			updates = append(updates, taskSnap{key: key, missing: true})
			continue
		}
		stats := t.Stats()
		progress := 0
		if t.Length() > 0 {
			p := float64(stats.BytesReadUsefulIntendedData.Int64()) / float64(t.Length()) * 100
			if p > 100 {
				p = 100
			}
			progress = int(p)
		}
		updates = append(updates, taskSnap{
			key:      key,
			progress: progress,
			size:     stats.BytesReadUsefulIntendedData.Int64(),
			complete: t.Complete().Bool(),
		})
	}
	TransferTaskMutex.RUnlock()

	if len(updates) == 0 {
		return
	}

	TransferTaskMutex.Lock()
	for _, u := range updates {
		task, ok := TransferTask[u.key]
		if !ok || task.Status != model.StatusExecuting {
			continue
		}
		switch {
		case u.missing:
			task.SetStatus(model.StatusCancelled)
			task.FinishTime = time.Now()
		case u.complete:
			task.Progress = 100
			task.Size = u.size
			task.SetStatus(model.StatusCompleted)
			task.FinishTime = time.Now()
			utils.InfoFormat("磁力下载完成（不触发索引扫描）: %s", task.Name)
		default:
			task.Progress = u.progress
			task.Size = u.size
		}
		TransferTask[u.key] = task
	}
	TransferTaskMutex.Unlock()
}

// removeTorrentTasks 删除统一任务列表中指定种子的全部磁力下载任务条目
func removeTorrentTasks(infoHash string) {
	TransferTaskMutex.Lock()
	for key, task := range TransferTask {
		if task.Type == model.TaskTypeTorrent && strings.EqualFold(task.InfoHash, infoHash) {
			delete(TransferTask, key)
		}
	}
	TransferTaskMutex.Unlock()
}

// CancelTorrentTasks 取消指定种子的磁力下载并清理其全部任务条目。
// 供任务列表删除接口调用：删除执行中的磁力下载任务即取消下载。
func CancelTorrentTasks(infoHash string) {
	if infoHash == "" {
		return
	}
	if TorrentApp != nil {
		// 内部会连带清理任务条目；种子已不存在时忽略错误
		_ = TorrentApp.RemoveTorrent(infoHash)
	}
	removeTorrentTasks(infoHash)
}

func (ts *TorrentService) GetTorrent(infoHash string) (*torrent.Torrent, error) {
	var h metainfo.Hash
	if err := h.FromHexString(infoHash); err != nil {
		return nil, fmt.Errorf("无效的 infoHash: %v", err)
	}

	ts.mu.RLock()
	t, ok := ts.torrents[h]
	ts.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("未找到对应的种子")
	}
	return t, nil
}

func (ts *TorrentService) GetVideoFile(infoHash string) (*torrent.File, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return nil, err
	}

	videoExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".mov":  true,
		".webm": true,
		".ts":   true,
		".m4v":  true,
	}

	var candidates []*torrent.File
	for _, f := range t.Files() {
		ext := strings.ToLower(filepath.Ext(f.Path()))
		if videoExts[ext] {
			candidates = append(candidates, f)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("种子中没有找到视频文件")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Length() > candidates[j].Length()
	})

	return candidates[0], nil
}

type TorrentFile struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Length int64  `json:"length"`
}

func (ts *TorrentService) GetFiles(infoHash string) ([]*TorrentFile, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return nil, err
	}

	var files []*TorrentFile
	for _, f := range t.Files() {
		files = append(files, &TorrentFile{
			Name:   filepath.Base(f.Path()),
			Path:   f.Path(),
			Length: f.Length(),
		})
	}

	return files, nil
}

func (ts *TorrentService) GetFileByPath(infoHash, filePath string) (*torrent.File, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return nil, err
	}

	for _, f := range t.Files() {
		if f.Path() == filePath {
			return f, nil
		}
	}

	return nil, fmt.Errorf("未找到文件: %s", filePath)
}

// GetDownloadDir 返回磁力链文件在磁盘上的下载目录（默认打开整个种子目录，
// 传入 filePath 时打开该文件所在子目录）。路径经规范化并限制在 dataDir 之内，防止穿越。
func (ts *TorrentService) GetDownloadDir(infoHash, filePath string) (string, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return "", err
	}

	base := filepath.Join(ts.dataDir, t.Name())
	if filePath != "" {
		base = filepath.Join(base, filepath.Dir(filePath))
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absData, err := filepath.Abs(ts.dataDir)
	if err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(absData, absBase); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("非法的下载目录路径")
	}
	return absBase, nil
}

// videoMimeTypes 视频文件扩展名到 MIME 类型的映射，用于流式响应正确的 Content-Type
var videoMimeTypes = map[string]string{
	".mp4":  "video/mp4",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".mov":  "video/quicktime",
	".webm": "video/webm",
	".ts":   "video/mp2t",
	".m4v":  "video/x-m4v",
}

// contentTypeForFile 根据文件扩展名返回合适的视频 MIME 类型，未知扩展名回退为 video/mp4
func contentTypeForFile(path string) string {
	if ct, ok := videoMimeTypes[strings.ToLower(filepath.Ext(path))]; ok {
		return ct
	}
	return "video/mp4"
}

func (ts *TorrentService) StreamVideo(infoHash string, w http.ResponseWriter, r *http.Request) error {
	filePath := r.URL.Query().Get("file")
	var videoFile *torrent.File
	var err error

	if filePath != "" {
		videoFile, err = ts.GetFileByPath(infoHash, filePath)
		if err != nil {
			return err
		}
	} else {
		videoFile, err = ts.GetVideoFile(infoHash)
		if err != nil {
			return err
		}
	}

	fileSize := videoFile.Length()
	rangeHeader := r.Header.Get("Range")

	if rangeHeader == "" {
		w.Header().Set("Content-Type", contentTypeForFile(videoFile.Path()))
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)

		reader := videoFile.NewReader()
		defer reader.Close()
		buf := make([]byte, 256*1024)
		_, err = io.CopyBuffer(w, reader, buf)
		return err
	}

	var start, end int64
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
		start = 0
		end = fileSize - 1
	}
	if end == 0 || end >= fileSize {
		end = fileSize - 1
	}

	contentLength := end - start + 1

	w.Header().Set("Content-Type", contentTypeForFile(videoFile.Path()))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)

	reader := videoFile.NewReader()
	defer reader.Close()
	_, err = reader.Seek(start, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seek 失败: %v", err)
	}

	buf := make([]byte, 256*1024)
	_, err = io.CopyBuffer(w, io.LimitReader(reader, contentLength), buf)
	return err
}

type TorrentStatus struct {
	InfoHash   string  `json:"infoHash"`
	Name       string  `json:"name"`
	TotalSize  int64   `json:"totalSize"`
	Downloaded int64   `json:"downloaded"`
	Progress   float64 `json:"progress"`
	Speed      float64 `json:"speed"`
	Peers      int     `json:"peers"`
	State      string  `json:"state"`
	VideoFile  string  `json:"videoFile"`
	VideoSize  int64   `json:"videoSize"`
}

func (ts *TorrentService) GetStatus(infoHash string) (*TorrentStatus, error) {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return nil, err
	}

	stats := t.Stats()

	videoFile, _ := ts.GetVideoFile(infoHash)
	var videoName string
	var videoSize int64
	if videoFile != nil {
		videoName = videoFile.Path()
		videoSize = videoFile.Length()
	}

	state := "下载中"
	if t.Complete().Bool() {
		state = "已完成"
	} else if stats.ActivePeers == 0 {
		state = "等待连接"
	}

	progress := 0.0
	if t.Length() > 0 {
		progress = float64(stats.BytesReadUsefulIntendedData.Int64()) / float64(t.Length()) * 100
		if progress > 100 {
			progress = 100
		}
	}

	return &TorrentStatus{
		InfoHash:   infoHash,
		Name:       t.Name(),
		TotalSize:  t.Length(),
		Downloaded: stats.BytesReadUsefulIntendedData.Int64(),
		Progress:   progress,
		Speed:      float64(stats.BytesReadData.Int64()),
		Peers:      stats.ActivePeers,
		State:      state,
		VideoFile:  videoName,
		VideoSize:  videoSize,
	}, nil
}

func (ts *TorrentService) RemoveTorrent(infoHash string) error {
	t, err := ts.GetTorrent(infoHash)
	if err != nil {
		return err
	}

	ts.mu.Lock()
	delete(ts.torrents, t.InfoHash())
	ts.mu.Unlock()

	t.Drop()
	// 同步清理统一任务列表中的对应任务（用户主动移除种子 = 取消下载）
	removeTorrentTasks(infoHash)
	utils.InfoFormat("已移除种子: %s", infoHash)
	return nil
}

func (ts *TorrentService) StartCleanup(ctx context.Context) {
	syncTicker := time.NewTicker(5 * time.Second)
	cleanTicker := time.NewTicker(30 * time.Minute)
	defer syncTicker.Stop()
	defer cleanTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-syncTicker.C:
			ts.syncTorrentTasks()
		case <-cleanTicker.C:
			ts.mu.Lock()
			for hash, t := range ts.torrents {
				if t.Complete().Bool() {
					delete(ts.torrents, hash)
					t.Drop()
					utils.InfoFormat("自动清理已完成种子: %s", hash.HexString())
				}
			}
			ts.mu.Unlock()
		}
	}
}
