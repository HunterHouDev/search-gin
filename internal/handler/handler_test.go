package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"search-gin/internal/model"
	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============== Mock 实现 ==============

type mockIndexEngine struct {
	empty      bool
	totalCount int
	totalSize  int64
	bucketCnt  int32
	authCnt    int
	typeMenu   map[string]model.FileInfo
	tagMenu    map[string]model.FileInfo
	seriesCnt  map[string]model.FileInfo
}

func (m *mockIndexEngine) Page(param model.SearchParam) utils.Page {
	return utils.Page{Data: []model.FileItem{}}
}
func (m *mockIndexEngine) PageAuthor(param model.SearchParam) model.PageAuthorResultWrapper {
	return model.PageAuthorResultWrapper{}
}
func (m *mockIndexEngine) FindById(id string) model.FileItem                         { return model.FileItem{} }
func (m *mockIndexEngine) FindAuthorByName(name string) model.Author                  { return model.Author{} }
func (m *mockIndexEngine) GetAuthorCount() int                                        { return m.authCnt }
func (m *mockIndexEngine) IsEmpty() bool                                              { return m.empty }
func (m *mockIndexEngine) GetTotalCount() int                                         { return m.totalCount }
func (m *mockIndexEngine) GetTotalSize() int64                                        { return m.totalSize }
func (m *mockIndexEngine) BucketCount() int32                                         { return m.bucketCnt }
func (m *mockIndexEngine) DeleteOnIndex(file model.FileItem)                          {}
func (m *mockIndexEngine) ReplaceFileOnIndex(oldFile, newFile model.FileItem)         {}
func (m *mockIndexEngine) GetTypeMenu() map[string]model.FileInfo                     { return m.typeMenu }
func (m *mockIndexEngine) GetTagMenu() map[string]model.FileInfo                      { return m.tagMenu }
func (m *mockIndexEngine) GetSeriesCount() map[string]model.FileInfo                  { return m.seriesCnt }

type mockFileService struct{}

func (m *mockFileService) SetMovieType(movie model.FileItem, movieType string) utils.Result { return utils.NewSuccess() }
func (m *mockFileService) AddTag(id string, tag string) utils.Result                         { return utils.NewSuccess() }
func (m *mockFileService) ClearTag(id string, tag string) utils.Result                       { return utils.NewSuccess() }
func (m *mockFileService) Rename(movie model.FileEdit) utils.Result                          { return utils.NewSuccess() }
func (m *mockFileService) Move(id string, newDir string, title string) utils.Result          { return utils.NewSuccess() }
func (m *mockFileService) Delete(id string) utils.Result                                     { return utils.NewSuccess() }
func (m *mockFileService) ScanAll() int                                                      { return 0 }
func (m *mockFileService) ScanTarget(baseDir string)                                         {}
func (m *mockFileService) Walk(dir string, types []string, withSub bool) []model.FileItem    { return nil }
func (m *mockFileService) DeleteFilesOnDisk(dirName string, fileName string)                 {}
func (m *mockFileService) DownDeleteDir(dirname string)                                      {}

type mockSettings struct {
	dirs           []string
	controllerHost string
}

func (m *mockSettings) Get() model.Setting {
	return model.Setting{Dirs: m.dirs, ControllerHost: m.controllerHost}
}
func (m *mockSettings) Set(s model.Setting)    {}
func (m *mockSettings) Flush(path string) error { return nil }

// ============== Test Helpers ==============

func setupHandlerTest(t *testing.T, eng service.IndexEngine, fs service.FileService, s service.Settings) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	InitApp(eng, fs, s)
}

func setTestAuth(c *gin.Context) {
	c.Set("role", "super_admin")
	c.Set("username", "admin")
	c.Set("permissions", service.AllPermissionKeys())
}

func performGet(t *testing.T, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handler(c)
	return w
}

func performPost(t *testing.T, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return w
}

func performGetWithAuth(t *testing.T, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	setTestAuth(c)
	handler(c)
	return w
}

func performPostWithAuth(t *testing.T, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	setTestAuth(c)
	handler(c)
	return w
}

// ============== Health Controller Tests ==============

func TestGetIndexHealthCheck_HappyPath(t *testing.T) {
	eng := &mockIndexEngine{
		totalCount: 1000,
		totalSize:  1024 * 1024 * 1024, // 1GB
		bucketCnt:  3,
		authCnt:    50,
		typeMenu:   map[string]model.FileInfo{"全部": {}, "动画": {}, "电影": {}},
		seriesCnt:  map[string]model.FileInfo{"系列A": {Name: "系列A"}},
		empty:      false,
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: []string{"D:/media", "E:/media", "F:/media"}})

	// 重置全局状态
	service.IndexNumber.Store(1)
	service.Sp.SetPhase("done", "扫描完成")
	service.Sp.Init(3)
	service.Sp.Complete()

	w := performGet(t, GetIndexHealthCheck)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp IndexHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int32(3), resp.BucketCount)
	assert.Equal(t, 1000, resp.TotalCount)
	assert.Equal(t, 3, resp.ExpectedDirs)
	assert.Equal(t, 50, resp.AuthorCount)
	assert.Equal(t, "healthy", resp.Status)
}

func TestGetIndexHealthCheck_EmptyIndex(t *testing.T) {
	eng := &mockIndexEngine{
		totalCount: 0,
		bucketCnt:  0,
		typeMenu:   map[string]model.FileInfo{"全部": {}},
		empty:      true,
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: nil})

	service.IndexNumber.Store(0)
	service.Sp.SetPhase("done", "扫描完成")

	w := performGet(t, GetIndexHealthCheck)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp IndexHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "empty", resp.Status)
}

func TestGetIndexHealthCheck_Scanning(t *testing.T) {
	eng := &mockIndexEngine{
		totalCount: 500,
		bucketCnt:  1,
		typeMenu:   map[string]model.FileInfo{"全部": {}},
		empty:      false,
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: []string{"D:/media", "E:/media"}})

	service.IndexNumber.Store(1)
	service.FullScanInProgress.Store(true)
	service.Sp.Init(2)

	w := performGet(t, GetIndexHealthCheck)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp IndexHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "scanning", resp.Status)
	assert.True(t, len(resp.Recommendations) > 0)
}

func TestGetIndexHealthCheck_AllTypes(t *testing.T) {
	tests := []struct {
		name         string
		bucketCnt    int32
		indexNum     int32
		fullScan     bool
		spPhase      string
		expectedSt   string
		expectRec    bool
	}{
		{"scanning_building", 0, 1, false, "building", "building", true},
		{"bucket_mismatch_warning", 1, 1, false, "done", "warning", true},
		{"bucket_mismatch_error", 1, 0, false, "done", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := &mockIndexEngine{
				totalCount: 100,
				bucketCnt:  tt.bucketCnt,
				typeMenu:   map[string]model.FileInfo{"全部": {}},
				empty:      false,
			}
			setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: []string{"D:/media", "E:/media"}})

			service.IndexNumber.Store(tt.indexNum)
			service.FullScanInProgress.Store(tt.fullScan)
			service.Sp.SetPhase(tt.spPhase, "test")

			w := performGet(t, GetIndexHealthCheck)

			var resp IndexHealth
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, tt.expectedSt, resp.Status)
		})
	}
}

// ============== Auth Controller Tests ==============

func TestLogin_Success(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.SetOSSetting(service.GetOSSetting())
	old := service.GetOSSetting()
	old.AdminPassword = "qwer"
	service.SetOSSetting(old)
	service.CacheAdminPasswordHash()

	w := performPost(t, Login, `{"username":"admin","password":"qwer"}`)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Result
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.IsSuccess())
}

func TestLogin_WrongPassword(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, Login, `{"username":"admin","password":"wrong"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp utils.Result
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.IsSuccess())
}

func TestLogin_EmptyBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, Login, `{}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_InvalidJSON(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, Login, `{invalid`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============== Search Controller Tests ==============

func TestPostMovies_EmptySearch(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{empty: false}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostMovies, `{"page":1,"pageSize":14}`)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostMovies_InvalidBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{empty: false}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostMovies, `{invalid json`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============== Home Controller Tests ==============

func TestGetHeartBeat(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.IndexNumber.Store(3)

	w := performGet(t, GetHeartBeat)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "3")
}

func TestGetLogMemory_ReturnsJSON(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.LogMem.Add("test log entry")

	w := performGet(t, GetLogMemory)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test log entry")
}

func TestGetTypeSize_EmptyIndex(t *testing.T) {
	eng := &mockIndexEngine{
		typeMenu: map[string]model.FileInfo{},
		empty:    true,
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: nil})

	service.SmallDir = nil

	w := performGet(t, GetTypeSize)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTypeSize_WithTypes(t *testing.T) {
	eng := &mockIndexEngine{
		typeMenu: map[string]model.FileInfo{
			"电影": {Name: "电影", Size: 100},
			"动画": {Name: "动画", Size: 200},
		},
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{dirs: nil})

	w := performGet(t, GetTypeSize)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []model.FileInfo
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, len(resp) >= 2)
}

func TestGetTagSize(t *testing.T) {
	eng := &mockIndexEngine{
		tagMenu: map[string]model.FileInfo{
			"action": {Name: "action", Size: 50},
		},
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetTagSize)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []model.FileInfo
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "action", resp[0].Name)
}

func TestGetSeriesSize(t *testing.T) {
	eng := &mockIndexEngine{
		seriesCnt: map[string]model.FileInfo{
			"系列A": {Name: "系列A", Cnt: 5},
		},
	}
	setupHandlerTest(t, eng, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetSeriesSize)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetScanTime_Empty(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetScanTime)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", w.Body.String())
}

func TestGetDiskUsage_EmptyDirs(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{dirs: nil})

	w := performGet(t, GetDiskUsage)
	assert.Equal(t, http.StatusOK, w.Code)
	// nil dirs → range over nil → nil slice → JSON "null"
	assert.Equal(t, "null", w.Body.String())
}

// ============== System Controller Tests ==============

func TestGetSettingInfo_RedactsSensitive(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{dirs: []string{"D:/media"}})

	old := service.GetOSSetting()
	service.SetOSSetting(model.Setting{
		Dirs:          []string{"D:/media"},
		AdminPassword: "secret",
		Users:         []model.User{{Username: "admin"}},
	})
	defer service.SetOSSetting(old)

	w := performGet(t, GetSettingInfo)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Empty(t, resp["AdminPassword"])
	assert.Nil(t, resp["Users"])
	assert.NotEmpty(t, resp["Dirs"])
}

func TestGetServerPort_Default(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetServerPort)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["Data"].(map[string]any)
	assert.Equal(t, ":10081", data["runningPort"])
}

func TestGetServerPort_ConfiguredDifferent(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{controllerHost: "0.0.0.0:20081"})

	w := performGet(t, GetServerPort)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["Data"].(map[string]any)
	assert.Equal(t, ":20081", data["configuredPort"])
	assert.Equal(t, true, data["changed"])
}

// ============== Lan Controller Tests ==============

func TestGetLanPeers_Empty(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.LocalNodeHost = "test-pc:10081"
	service.LocalNodeName = "测试机"
	t.Cleanup(func() {
		service.LocalNodeHost = ""
		service.LocalNodeName = ""
	})

	w := performGet(t, GetLanPeers)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["Data"].(map[string]any)
	assert.Equal(t, "test-pc:10081", data["localNodeHost"])
	assert.Equal(t, "测试机", data["localNodeName"])
	// defaultManager 为 nil 时 peers 为 null，unmarshal 后为 nil
}

func TestGetPeerStats_MissingNode(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/peerStats?node=", nil)
	GetPeerStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetLanPeersWithStats_Empty(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetLanPeersWithStats)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddLanPeer_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, AddLanPeer, `{"addr":"10.0.0.1:10081"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRemoveLanPeer_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, RemoveLanPeer, `{"id":"10.0.0.1:10081"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTogglePeer_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, TogglePeer, `{"id":"10.0.0.1:10081","disabled":true}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCleanLanPeers_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, CleanLanPeers)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDiscoverLanPeers_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, DiscoverLanPeers, `{"subnet":"192.168.1"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============== splitLines 测试 ==============

func TestSplitLines_Empty(t *testing.T) {
	assert.Empty(t, splitLines(""))
}

func TestSplitLines_SingleLine(t *testing.T) {
	assert.Equal(t, []string{"hello"}, splitLines("hello"))
}

func TestSplitLines_MultipleLines(t *testing.T) {
	result := splitLines("line1\nline2\nline3")
	assert.Equal(t, []string{"line1", "line2", "line3"}, result)
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	result := splitLines("a\nb\n")
	assert.Equal(t, []string{"a", "b"}, result)
}

// ============== Ping Controller Tests ==============

func TestPingHost_MissingIP(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	PingHost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "缺少 ip")
}

func TestPingHost_InvalidIP(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/ping?ip=not-an-ip", nil)
	PingHost(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "格式无效")
}

// ============== Torrent Controller Tests ==============

func TestPostAddMagnet_NoBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPostWithAuth(t, PostAddMagnet, `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostAddMagnet_TorrentAppNil(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPostWithAuth(t, PostAddMagnet, `{"magnetURI":"magnet:?xt=urn:btih:xxx"}`)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "未启动")
}

func TestPostStartDownload_NoBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPostWithAuth(t, PostStartDownload, `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostStartDownload_TorrentAppNil(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPostWithAuth(t, PostStartDownload, `{"infoHash":"xxx"}`)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetTorrentStream_MissingHash(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	GetTorrentStream(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "缺少 infoHash")
}

func TestGetTorrentStream_TorrentAppNil(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "infoHash", Value: "xxx"}}
	GetTorrentStream(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "未启动")
}

func TestGetTorrentStatus_MissingHash(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	GetTorrentStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTorrentFiles_MissingHash(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	GetTorrentFiles(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteTorrent_MissingHash(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	setTestAuth(c)
	DeleteTorrent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteTorrent_TorrentAppNil(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	c.Params = []gin.Param{{Key: "infoHash", Value: "xxx"}}
	setTestAuth(c)
	DeleteTorrent(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ============== User Controller Tests ==============

func TestGetUsers_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetUsers)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAddUser_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, AddUser, `{"username":"newuser","password":"pass"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteUser_NoAdminReturns403(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, DeleteUser, `{"username":"someuser"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAddUser_EmptyUsername(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.SetOSSetting(model.Setting{})
	service.CacheAdminPasswordHash()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"username":"","password":"pass"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("role", "super_admin")
	AddUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddUser_EmptyPassword(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	service.SetOSSetting(model.Setting{})
	service.CacheAdminPasswordHash()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"username":"newuser","password":""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("role", "super_admin")
	AddUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddUser_DuplicateAdminUsername(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})
	old := service.GetOSSetting()
	service.SetOSSetting(model.Setting{})
	defer service.SetOSSetting(old)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"username":"admin","password":"pass"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("role", "super_admin")
	AddUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "已存在")
}

// ============== Dir Controller Tests ==============

func TestGetOpenFolder_NotFound(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	GetOpenFolder(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "文件不存在")
}

func TestPostOpenFolderByPath_InvalidBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostOpenFolderByPath, `{invalid`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostOpenFolderByPath_EmptyBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostOpenFolderByPath, `{}`)
	// empty dirpath with no Dirs → ValidatePath rejects
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostDeleteFolderByPath_InvalidBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostDeleteFolderByPath, `{invalid`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============== File Stream Controller Tests ==============

func TestGetFileByPathUseEncode_InvalidPath(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "path", Value: "%zz"}}
	GetFileByPathUseEncode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "无效的文件路径")
}

func TestGetDeleteFileByPathUseEncode_NoAdmin(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	c.Params = []gin.Param{{Key: "path", Value: "test"}}
	GetDeleteFileByPathUseEncode(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetRefreshTargetIndex_ForbiddenPath(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{dirs: []string{"D:/media"}})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "dir", Value: "C%3A%5Cwindows"}}
	GetRefreshTargetIndex(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetRefreshIndex_ReturnsOK(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{dirs: []string{"D:/media", "E:/media"}})

	w := performGetWithAuth(t, GetRefreshIndex)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "2")
}

// ============== File Play Controller Tests ==============

func TestGetAuthorImage_NotFound(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "path", Value: "nonexistent"}}
	GetAuthorImage(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============== File Task Controller Tests ==============

func TestGetDelTransferTask_InvalidTime(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "create", Value: "not-a-time"}}
	GetDelTransferTask(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// key 改为 string ID，不存在的直接返回"任务不存在"
	assert.Contains(t, w.Body.String(), "任务不存在")
}

func TestGetDelTransferTask_NotFound(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = []gin.Param{{Key: "create", Value: "2024-01-01T00:00:00Z"}}
	GetDelTransferTask(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "任务不存在")
}

func TestPostMerge_InvalidBody(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPostWithAuth(t, PostMerge, `{invalid`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostClearCompletedTasks_ReturnsOK(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostClearCompletedTasks, ``)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostClearFailedTasks_ReturnsOK(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostClearFailedTasks, ``)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostClearAllTasks_ReturnsOK(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performPost(t, PostClearAllTasks, ``)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTransferTask_ReturnsOK(t *testing.T) {
	setupHandlerTest(t, &mockIndexEngine{}, &mockFileService{}, &mockSettings{})

	w := performGet(t, GetTransferTask)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "tasks")
}


