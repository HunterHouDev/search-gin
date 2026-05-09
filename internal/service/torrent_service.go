package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
	cfg.NoUpload = true
	cfg.Seed = false

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

func (ts *TorrentService) AddMagnet(magnetURI string) (string, error) {
	t, err := ts.client.AddMagnet(magnetURI)
	if err != nil {
		return "", fmt.Errorf("添加磁力链失败: %v", err)
	}

	select {
	case <-t.GotInfo():
	case <-time.After(60 * time.Second):
		t.Drop()
		return "", fmt.Errorf("获取种子信息超时")
	}

	infoHash := t.InfoHash().HexString()

	ts.mu.Lock()
	ts.torrents[t.InfoHash()] = t
	ts.mu.Unlock()

	t.DownloadAll()

	utils.InfoFormat("已添加磁力链: %s, InfoHash: %s, 文件数: %d", t.Name(), infoHash, len(t.Files()))
	return infoHash, nil
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

func (ts *TorrentService) StreamVideo(infoHash string, w http.ResponseWriter, r *http.Request) error {
	videoFile, err := ts.GetVideoFile(infoHash)
	if err != nil {
		return err
	}

	fileSize := videoFile.Length()
	rangeHeader := r.Header.Get("Range")

	if rangeHeader == "" {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)

		reader := videoFile.NewReader()
		defer reader.Close()
		_, err = io.Copy(w, reader)
		return err
	}

	var start, end int64
	fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
	if end == 0 || end >= fileSize {
		end = fileSize - 1
	}

	contentLength := end - start + 1

	w.Header().Set("Content-Type", "video/mp4")
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

	_, err = io.CopyN(w, reader, contentLength)
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
		progress = float64(stats.BytesReadUseful) / float64(t.Length()) * 100
		if progress > 100 {
			progress = 100
		}
	}

	return &TorrentStatus{
		InfoHash:   infoHash,
		Name:       t.Name(),
		TotalSize:  t.Length(),
		Downloaded: stats.BytesReadUseful,
		Progress:   progress,
		Speed:      float64(stats.DownloadSpeed),
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
	utils.InfoFormat("已移除种子: %s", infoHash)
	return nil
}

func (ts *TorrentService) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
