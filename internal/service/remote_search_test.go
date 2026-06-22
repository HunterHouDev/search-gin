package service

import (
	"net"
	"net/http/httptest"
	"search-gin/internal/model"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ── SetMovieNode 测试 ──

func TestSetMovieNode_AssignsHostAndName(t *testing.T) {
	LocalNodeHost = "mypc:10081"
	LocalNodeName = "书房电脑"

	m := &model.FileItem{Id: "abc123"}
	SetMovieNode(m)

	assert.Equal(t, "mypc:10081", m.NodeHost)
	assert.Equal(t, "书房电脑", m.NodeName)
	assert.Empty(t, m.StreamUrl, "SetMovieNode 不应修改 StreamUrl")
	assert.Empty(t, m.PngUrl, "SetMovieNode 不应修改 PngUrl")
	assert.Empty(t, m.JpgUrl, "SetMovieNode 不应修改 JpgUrl")
}

// ── dedupKey 测试 ──

func TestDedupKey_WithCode(t *testing.T) {
	m := model.FileItem{Code: "ABC-123", Size: 999999, Name: "test.mp4"}
	assert.Equal(t, "code:ABC-123:999999", dedupKey(m))
}

func TestDedupKey_WithoutCode_FallsBackToName(t *testing.T) {
	m := model.FileItem{Code: "", Name: "unique.mp4", Size: 500000}
	assert.Equal(t, "name:unique.mp4:500000", dedupKey(m))
}

func TestDedupKey_ZeroSize(t *testing.T) {
	m := model.FileItem{Code: "XYZ", Size: 0}
	assert.Equal(t, "code:XYZ:0", dedupKey(m))
}

// ── MergeResults 测试 ──
// MergeResults 不做去重，跨节点重复文件全部保留供用户知情后手动清理

func TestMergeResults_ConcatenatesLocalAndRemote(t *testing.T) {
	local := []model.FileItem{
		{Id: "1", Code: "ABC", Size: 100, Name: "local-a.mp4"},
	}
	remote := []model.FileItem{
		{Id: "2", Code: "ABC", Size: 100, Name: "remote-a.mp4"}, // Code+Size 相同但不同节点，保留
		{Id: "3", Code: "DEF", Size: 200, Name: "remote-b.mp4"},
	}
	merged := MergeResults(local, remote)
	assert.Len(t, merged, 3) // 不合并，全部保留
	assert.Equal(t, "1", merged[0].Id)
	assert.Equal(t, "2", merged[1].Id)
	assert.Equal(t, "3", merged[2].Id)
}

func TestMergeResults_KeepsNameSizeDuplicates(t *testing.T) {
	local := []model.FileItem{
		{Id: "1", Code: "", Name: "same.mp4", Size: 777},
	}
	remote := []model.FileItem{
		{Id: "2", Code: "", Name: "same.mp4", Size: 777}, // 不同节点同名同大小，保留
		{Id: "3", Code: "", Name: "same.mp4", Size: 888},
	}
	merged := MergeResults(local, remote)
	assert.Len(t, merged, 3) // 不合并，全部保留
	assert.Equal(t, "1", merged[0].Id)
	assert.Equal(t, "2", merged[1].Id)
	assert.Equal(t, "3", merged[2].Id)
}

func TestMergeResults_EdgeCases(t *testing.T) {
	assert.Len(t, MergeResults(nil, nil), 0)
	assert.Len(t, MergeResults([]model.FileItem{}, nil), 0)
	assert.Len(t, MergeResults(nil, []model.FileItem{}), 0)

	onlyLocal := []model.FileItem{{Id: "L", Code: "X", Size: 1}}
	assert.Len(t, MergeResults(onlyLocal, nil), 1)

	onlyRemote := []model.FileItem{{Id: "R", Code: "X", Size: 1}}
	assert.Len(t, MergeResults(nil, onlyRemote), 1)
}

// ── PaginateMovies 测试 ──

func TestPaginateMovies_AllCases(t *testing.T) {
	// 25 items, pageSize=10
	m25 := make([]model.FileItem, 25)
	for i := range m25 {
		m25[i].Id = string(rune('A' + i%26))
		m25[i].Size = int64(i)
	}

	// 第 1 页：10 条
	r, total := PaginateMovies(m25, 1, 10)
	assert.Len(t, r, 10)
	assert.Equal(t, 25, total)

	// 第 2 页：10 条
	r, total = PaginateMovies(m25, 2, 10)
	assert.Len(t, r, 10)
	assert.Equal(t, 25, total)

	// 第 3 页（末页）：5 条
	r, total = PaginateMovies(m25, 3, 10)
	assert.Len(t, r, 5)
	assert.Equal(t, 25, total)

	// 超出范围
	r, _ = PaginateMovies(m25, 10, 10)
	assert.Len(t, r, 0)

	// page=0 → 默认 page=1
	r, _ = PaginateMovies(m25, 0, 5)
	assert.Len(t, r, 5)

	// pageSize=0 → 默认 pageSize=20
	r, _ = PaginateMovies(m25, 1, 0)
	assert.Len(t, r, 20)
}

// ── FillURLs 测试 ──

func setupFillURLsTest() {
	gin.SetMode(gin.TestMode)
	LocalNodeHost = "mypc:10081"
	LocalNodeName = "测试机器"
	// 初始化 defaultManager（测试环境中需手动创建）
	if defaultManager == nil {
		defaultManager = &peerManager{
			peers: make(map[string]*Peer),
		}
	}
	defaultManager.mu.Lock()
	defaultManager.peers = make(map[string]*Peer)
	defaultManager.mu.Unlock()
}

func addFakePeer(id, ip string) {
	defaultManager.mu.Lock()
	defaultManager.peers[id] = &Peer{ID: id, IP: ip}
	defaultManager.mu.Unlock()
}

func newTestContext(remoteAddr string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/movieList", nil)
	c.Request.RemoteAddr = remoteAddr
	return c
}

func TestFillURLs_LocalFile_AssignsURLs(t *testing.T) {
	setupFillURLsTest()
	c := newTestContext("192.168.1.100:12345")

	movies := []model.FileItem{
		{Id: "file-a", NodeHost: ""},           // 空 → 视为本机
		{Id: "file-b", NodeHost: "mypc:10081"}, // 匹配本机
	}

	FillURLs(c, movies)

	for _, m := range movies {
		assert.Equal(t, "mypc:10081", m.NodeHost, "NodeHost 应被填充为本机")
		assert.Equal(t, "测试机器", m.NodeName, "NodeName 应被填充为本机")
		assert.Contains(t, m.StreamUrl, "/api/stream/GetFileByPathUseEncode/")
		assert.Contains(t, m.PngUrl, "/api/stream/png/"+m.Id)
		assert.Contains(t, m.JpgUrl, "/api/stream/jpg/"+m.Id)
	}
}

func TestFillURLs_RemoteFile_UsesPeerIP(t *testing.T) {
	setupFillURLsTest()
	addFakePeer("remote-pc:10081", "10.0.0.99")
	c := newTestContext("192.168.1.100:12345")

	movies := []model.FileItem{
		{Id: "r-file", NodeHost: "remote-pc:10081", NodeName: "远程机器"},
	}

	FillURLs(c, movies)

	m := movies[0]
	assert.Equal(t, "remote-pc:10081", m.NodeHost, "远程文件 NodeHost 不被覆盖")
	assert.Equal(t, "远程机器", m.NodeName, "远程文件 NodeName 不被覆盖")
	assert.Contains(t, m.StreamUrl, "10.0.0.99")
	assert.Contains(t, m.StreamUrl, "/api/stream/GetFileByPathUseEncode/")
	assert.Contains(t, m.PngUrl, "10.0.0.99")
	assert.Contains(t, m.JpgUrl, "10.0.0.99")
}

func TestFillURLs_RemoteFile_PeerOffline_NoURL(t *testing.T) {
	setupFillURLsTest()
	// 不注册 peer → ResolvePeerIP 返回 ""
	c := newTestContext("192.168.1.100:12345")

	movies := []model.FileItem{
		{Id: "off", NodeHost: "ghost-pc:10081", NodeName: "离线"},
	}

	FillURLs(c, movies)

	m := movies[0]
	assert.Equal(t, "ghost-pc:10081", m.NodeHost)
	assert.Empty(t, m.StreamUrl, "peer 离线时不应设置 URL")
	assert.Empty(t, m.PngUrl)
	assert.Empty(t, m.JpgUrl)
}

func TestFillURLs_EmptyList_NoPanic(t *testing.T) {
	setupFillURLsTest()
	c := newTestContext("192.168.1.100:12345")

	assert.NotPanics(t, func() { FillURLs(c, nil) })
	assert.NotPanics(t, func() { FillURLs(c, []model.FileItem{}) })
}

func TestFillURLs_UsesFilePort(t *testing.T) {
	setupFillURLsTest()
	c := newTestContext("192.168.1.100:12345")

	movies := []model.FileItem{{Id: "pt", NodeHost: "mypc:10081"}}
	FillURLs(c, movies)

	m := movies[0]
	filePort := strings.TrimPrefix(FilePortNo, ":")
	assert.Contains(t, m.StreamUrl, filePort, "StreamUrl 应使用文件流端口 10082")
}

// ── pickLocalIP / fallbackLocalIP 测试 ──

func TestPickLocalIP_Invalid(t *testing.T) {
	ip := pickLocalIP("not-an-ip")
	assert.NotEmpty(t, ip)
	assert.NotPanics(t, func() { pickLocalIP("") })
}

func TestFallbackLocalIP_ReturnsValidIP(t *testing.T) {
	ip := fallbackLocalIP()
	assert.NotEmpty(t, ip)
	parsed := net.ParseIP(ip)
	assert.NotNil(t, parsed, "fallbackLocalIP 应返回合法 IP: %s", ip)
}

func TestPickLocalIP_WithRealClientIP(t *testing.T) {
	fallback := fallbackLocalIP()
	if fallback == "127.0.0.1" {
		t.Skip("无外部网络接口")
	}
	ip := pickLocalIP(fallback)
	assert.NotEmpty(t, ip)
	assert.NotNil(t, net.ParseIP(ip))
}
