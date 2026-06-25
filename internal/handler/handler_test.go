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
	dirs []string
}

func (m *mockSettings) Get() model.Setting           { return model.Setting{Dirs: m.dirs} }
func (m *mockSettings) Set(s model.Setting)           {}
func (m *mockSettings) Flush(path string)             {}

// ============== Test Helpers ==============

func setupHandlerTest(t *testing.T, eng service.IndexEngine, fs service.FileService, s service.Settings) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	InitApp(eng, fs, s)
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
