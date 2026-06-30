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
	m.SetNodeInfo(LocalNodeHost, LocalNodeName)

	assert.Equal(t, "mypc:10081", m.NodeHost)
	assert.Equal(t, "书房电脑", m.NodeName)
	assert.Empty(t, m.StreamUrl, "SetMovieNode 不应修改 StreamUrl")
	assert.Empty(t, m.PngUrl, "SetMovieNode 不应修改 PngUrl")
	assert.Empty(t, m.JpgUrl, "SetMovieNode 不应修改 JpgUrl")
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

func TestFillURLs_AlternatesPorts(t *testing.T) {
	setupFillURLsTest()
	c := newTestContext("192.168.1.100:12345")

	movies := []model.FileItem{
		{Id: "even", NodeHost: "mypc:10081"}, // i%2==0 → localBases[0] = filePort
		{Id: "odd", NodeHost: "mypc:10081"},  // i%2==1 → localBases[1] = apiPort
	}
	FillURLs(c, movies)

	apiPort := strings.TrimPrefix(PortNo, ":")
	filePort := strings.TrimPrefix(FilePortNo, ":")
	assert.Contains(t, movies[0].StreamUrl, ":"+filePort+"/", "偶数项应使用文件流端口 10082")
	assert.Contains(t, movies[1].StreamUrl, ":"+apiPort+"/", "奇数项应使用 API 端口 10081")
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

// ── ParseTotalSize 测试 ──

func TestParseTotalSize_Empty(t *testing.T) {
	assert.Equal(t, int64(0), ParseTotalSize(""))
	assert.Equal(t, int64(0), ParseTotalSize("  "))
}

func TestParseTotalSize_Bytes(t *testing.T) {
	assert.Equal(t, int64(100), ParseTotalSize("100 B"))
}

func TestParseTotalSize_KiloBytes(t *testing.T) {
	assert.Equal(t, int64(1*1024), ParseTotalSize("1 K"))
	assert.Equal(t, int64(500*1024), ParseTotalSize("500 K"))
}

func TestParseTotalSize_MegaBytes(t *testing.T) {
	assert.Equal(t, int64(100*1024*1024), ParseTotalSize("100 M"))
}

func TestParseTotalSize_GigaBytes(t *testing.T) {
	assert.Equal(t, int64(2*1024*1024*1024), ParseTotalSize("2 G"))
	// 23.53 * 1073741824 ≈ 25265145118.72 → float 精度 → 25265145118
	assert.Equal(t, int64(25265145118), ParseTotalSize("23.53 G"))
}

func TestParseTotalSize_TeraBytes(t *testing.T) {
	assert.Equal(t, int64(1*1024*1024*1024*1024), ParseTotalSize("1 T"))
}

func TestParseTotalSize_InvalidFormat(t *testing.T) {
	assert.Equal(t, int64(0), ParseTotalSize("abc"))
	assert.Equal(t, int64(0), ParseTotalSize("100"))
	assert.Equal(t, int64(0), ParseTotalSize("100 X")) // unknown unit
}

func TestParseTotalSize_TrimsSpaces(t *testing.T) {
	assert.Equal(t, int64(100), ParseTotalSize("  100 B  "))
}

func TestParseTotalSize_CaseInsensitiveUnit(t *testing.T) {
	assert.Equal(t, int64(1024), ParseTotalSize("1 k"))
	assert.Equal(t, int64(1024*1024), ParseTotalSize("1 m"))
	assert.Equal(t, int64(1024*1024*1024), ParseTotalSize("1 g"))
}
