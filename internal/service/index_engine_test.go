package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ── 辅助函数 ──

func makeMovie(id, name, path, code, movieType, author string, size int64) model.FileItem {
	return model.FileItem{
		Id:        id,
		Name:      name,
		Path:      path,
		PathUpper: strings.ToUpper(path),
		Code:      code,
		MovieType: movieType,
		Author:    author,
		Size:      size,
		DirPath:   "/test",
		BaseDir:   "/test",
	}
}

func makeBucket(name string, movies ...model.FileItem) *bucketFile {
	b := newInstance(name)
	for _, m := range movies {
		b.put(m)
	}
	return b
}

// ── bucketFile 测试 ──

func TestBucketFile_NewInstance(t *testing.T) {
	b := newInstance("dir-a")
	assert.NotNil(t, b)
	assert.Equal(t, "dir-a", b.InstanceName)
	assert.Equal(t, int64(0), b.TotalSize)
	assert.Equal(t, 0, b.TotalCount)
	assert.Empty(t, b.FileLib)
	assert.Empty(t, b.TypeIndex)
}

func TestBucketFile_PutAndGet(t *testing.T) {
	b := newInstance("dir-a")
	m := makeMovie("1", "test.mp4", "/test/test.mp4", "ABC-123", "骑兵", "田中", 1024)

	b.put(m)

	assert.Equal(t, int64(1024), b.TotalSize)
	assert.Equal(t, 1, b.TotalCount)

	got := b.get("1")
	assert.Equal(t, "test.mp4", got.Name)
	assert.Equal(t, "ABC-123", got.Code)

	// 不存在的 id
	notFound := b.get("nonexist")
	assert.Nil(t, notFound)
}

func TestBucketFile_PutBatch(t *testing.T) {
	b := newInstance("dir-a")
	movies := []model.FileItem{
		makeMovie("1", "a.mp4", "/test/a.mp4", "AAA", "骑兵", "", 100),
		makeMovie("2", "b.mp4", "/test/b.mp4", "BBB", "步兵", "", 200),
		makeMovie("3", "c.mp4", "/test/c.mp4", "CCC", "骑兵", "", 300),
	}
	b.putBatch(movies)

	assert.Equal(t, 3, b.TotalCount)
	assert.Equal(t, int64(600), b.TotalSize)
	assert.Equal(t, 2, len(b.TypeIndex["骑兵"]))
	assert.Equal(t, 1, len(b.TypeIndex["步兵"]))
}

func TestBucketFile_IsEmpty(t *testing.T) {
	b1 := newInstance("empty")
	assert.True(t, b1.isEmpty())

	b2 := makeBucket("not-empty", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 10))
	assert.False(t, b2.isEmpty())
}

func TestBucketFile_TypeIndex(t *testing.T) {
	b := newInstance("dir")
	m1 := makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "", 100)
	m2 := makeMovie("2", "b.mp4", "/b.mp4", "", "步兵", "", 200)
	m3 := makeMovie("3", "c.mp4", "/c.mp4", "", "骑兵", "", 300)

	b.putBatch([]model.FileItem{m1, m2, m3})

	assert.Contains(t, b.TypeIndex["骑兵"], "1")
	assert.Contains(t, b.TypeIndex["骑兵"], "3")
	assert.Contains(t, b.TypeIndex["步兵"], "2")
	assert.NotContains(t, b.TypeIndex["骑兵"], "2")

	// 无类型的文件不加入索引
	noType := makeMovie("4", "d.txt", "/d.txt", "", "", "", 50)
	b.put(noType)
	_, exists := b.TypeIndex[""]
	assert.False(t, exists, "空类型不应加入 TypeIndex")
}

// ── buildIndexFromBuckets 测试 ──

func TestBuildIndexFromBuckets_AggregatesStats(t *testing.T) {
	b1 := makeBucket("dir-a",
		makeMovie("1", "a.mp4", "/a.mp4", "AAA", "骑兵", "田中", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "BBB", "骑兵", "佐藤", 200),
	)
	b2 := makeBucket("dir-b",
		makeMovie("3", "c.mp4", "/c.mp4", "CCC", "步兵", "田中", 300),
	)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir-a": b1, "dir-b": b2})

	assert.Equal(t, int64(600), index.totalSize)
	assert.Equal(t, 3, index.totalCount)
	assert.Equal(t, int32(2), index.bucketCount)
}

func TestBuildIndexFromBuckets_AuthorAggregation(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "田中", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "", "骑兵", "田中", 200),
		makeMovie("3", "c.mp4", "/c.mp4", "", "步兵", "佐藤", 150),
	)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})

	assert.Equal(t, 2, len(index.authorMap))
	assert.Equal(t, 2, index.authorMap["田中"].Cnt)
	assert.Equal(t, int64(300), index.authorMap["田中"].Size)
	assert.Equal(t, 1, index.authorMap["佐藤"].Cnt)
}

func TestBuildIndexFromBuckets_TypeMenu(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "", "步兵", "", 200),
		makeMovie("3", "c.mp4", "/c.mp4", "", "骑兵", "", 300),
	)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})

	assert.Equal(t, int64(600), index.typeMenu["全部"].Size)
	assert.Equal(t, int64(400), index.typeMenu["骑兵"].Size)
	assert.Equal(t, int64(200), index.typeMenu["步兵"].Size)
}

func TestBuildIndexFromBuckets_RepeatByCode(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "ABC-123", "", "", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "ABC-123", "", "", 100), // 同 Code+Size
	)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	// 两个文件 Size 相同且 Code 相同 → 标记为重复
	assert.GreaterOrEqual(t, len(index.repeatFiles), 2, "重复文件应被检测到")
}

func TestBuildIndexFromBuckets_EmptyBucket(t *testing.T) {
	b := newInstance("empty-dir")
	index := buildIndexFromBuckets(map[string]*bucketFile{"empty-dir": b})

	assert.Equal(t, int64(0), index.totalSize)
	assert.Equal(t, 0, index.totalCount)
	assert.Equal(t, int32(0), index.bucketCount)
}

func TestBuildIndexFromBuckets_NoTypeFallsback(t *testing.T) {
	m := makeMovie("1", "a.mp4", "/a.mp4", "", "", "", 100)
	m.MovieType = "" // 确保无类型
	b := makeBucket("dir", m)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})

	// 空类型应归为"无"
	assert.Contains(t, index.typeMenu, "无")
	assert.Equal(t, int64(100), index.typeMenu["无"].Size)
}

// ── searchEngineCore 测试 ──

func newTestEngine() searchEngineCore {
	return searchEngineCore{
		KeywordHistoryCache: utils.NewLRUCache(100),
		searchPool:          utils.NewGoroutinePool(4),
	}
}

func TestSearchEngineCore_IsEmpty(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	assert.True(t, core.IsEmpty())
}

func TestSearchEngineCore_InstallIndex(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	assert.False(t, core.IsEmpty())
	assert.Equal(t, 1, core.GetTotalCount())
	assert.Equal(t, int64(100), core.GetTotalSize())
	assert.Equal(t, int32(1), core.BucketCount())
}

func TestSearchEngineCore_Reset(t *testing.T) {
	core := newTestEngine()

	b := makeBucket("dir", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)
	core.installIndexSkipDisk(emptySearchIndex())

	assert.True(t, core.IsEmpty())
	assert.Equal(t, 0, core.GetTotalCount())
}

func TestSearchEngineCore_FindById(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir",
		makeMovie("id-a", "a.mp4", "/a.mp4", "", "", "", 100),
		makeMovie("id-b", "b.mp4", "/b.mp4", "", "", "", 200),
	)
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	found := core.FindById("id-a")
	assert.False(t, found.IsNull())
	assert.Equal(t, "a.mp4", found.Name)

	notFound := core.FindById("nonexist")
	assert.True(t, notFound.IsNull())
}

func TestSearchEngineCore_GetAuthorCount(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "田中", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "", "步兵", "佐藤", 200),
	)
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	assert.Equal(t, 2, core.GetAuthorCount())
}

func TestSearchEngineCore_FindAuthorByName(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "田中", 100),
	)
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	act := core.FindAuthorByName("田中")
	assert.True(t, act.IsNotEmpty())
	assert.Equal(t, "田中", act.Name)

	notFound := core.FindAuthorByName("不存在")
	assert.True(t, notFound.IsEmpty())
}

// ── bucketFile.searchBucket 测试 ──

func TestBucketFile_SearchBucket_NoKeyword(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/test/a.mp4", "", "骑兵", "", 100),
		makeMovie("2", "b.mp4", "/test/b.mp4", "", "步兵", "", 200),
	)

	param := model.SearchParam{Keyword: "", Page: 1, PageSize: 10}
	result := b.searchBucket(param)
	assert.Equal(t, 2, len(result.FileList))
}

func TestBucketFile_SearchBucket_KeywordMatch(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "alpha.mp4", "/test/alpha.mp4", "", "", "", 100),
		makeMovie("2", "beta.mp4", "/test/beta.mp4", "", "", "", 200),
		makeMovie("3", "gamma.mp4", "/test/gamma.mp4", "", "", "", 300),
	)

	param := model.SearchParam{Keyword: "alpha", Page: 1, PageSize: 10}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "alpha.mp4", result.FileList[0].Name)
}

func TestBucketFile_SearchBucket_MultiKeywords(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "abc-def.mp4", "/test/abc-def.mp4", "", "", "", 100),
		makeMovie("2", "abc-ghi.mp4", "/test/abc-ghi.mp4", "", "", "", 200),
		makeMovie("3", "xyz.mp4", "/test/xyz.mp4", "", "", "", 300),
	)

	// 空格分隔 = AND 匹配
	param := model.SearchParam{Keyword: "abc def", Page: 1, PageSize: 10}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "abc-def.mp4", result.FileList[0].Name)
}

func TestBucketFile_SearchKeyword_TypeFilter(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "", "步兵", "", 200),
		makeMovie("3", "c.mp4", "/c.mp4", "", "骑兵", "", 300),
	)

	param := model.SearchParam{Keyword: "", MovieType: "步兵", Page: 1, PageSize: 10}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "b.mp4", result.FileList[0].Name)
}

func TestBucketFile_SearchKeyword_NoMatch(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "cat.mp4", "/test/cat.mp4", "", "", "", 100),
	)

	param := model.SearchParam{Keyword: "nonexistent", Page: 1, PageSize: 10}
	result := b.searchBucket(param)
	assert.Equal(t, 0, len(result.FileList))
}

// ── rebuildWithBucket 测试 ──

func TestRebuildWithBucket_ReplacesExisting(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 设置配置目录使 rebuildWithBucket 不会跳过 bucket
	orig := GetOSSetting()
	SetOSSetting(model.Setting{
		Dirs: []string{"dir-a"},
	})
	defer SetOSSetting(orig)

	// 初始：dir-a 有文件
	b1 := makeBucket("dir-a", makeMovie("1", "old.mp4", "/old.mp4", "", "", "", 100))
	index1 := buildIndexFromBuckets(map[string]*bucketFile{"dir-a": b1})
	core.installIndex(index1)
	assert.Equal(t, 1, core.GetTotalCount())

	// 替换 dir-a 为新文件
	b2 := makeBucket("dir-a", makeMovie("2", "new.mp4", "/new.mp4", "", "", "", 200))
	core.rebuildWithBucketIncremental("dir-a", b2)

	assert.Equal(t, 1, core.GetTotalCount())
	assert.Equal(t, int64(200), core.GetTotalSize())

	found := core.FindById("1")
	assert.True(t, found.IsNull(), "旧文件 id=1 应已被替换")
	found = core.FindById("2")
	assert.False(t, found.IsNull(), "新文件 id=2 应可查")
}

func TestRebuildWithBucket_KeepsOtherBuckets(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 设置配置目录使 rebuildWithBucket 不会跳过这些 bucket
	orig := GetOSSetting()
	SetOSSetting(model.Setting{
		Dirs: []string{"dir-a", "dir-b"},
	})
	defer SetOSSetting(orig)

	bA := makeBucket("dir-a", makeMovie("1", "a.mp4", "/a.mp4", "", "", "", 100))
	bB := makeBucket("dir-b", makeMovie("2", "b.mp4", "/b.mp4", "", "", "", 200))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir-a": bA, "dir-b": bB})
	core.installIndex(index)

	// 只替换 dir-a 不影响 dir-b
	bA2 := makeBucket("dir-a", makeMovie("3", "a2.mp4", "/a2.mp4", "", "", "", 300))
	core.rebuildWithBucketIncremental("dir-a", bA2)

	assert.Equal(t, int32(2), core.BucketCount())
	assert.Equal(t, int64(500), core.GetTotalSize())
	_ = bB // dir-b 保持不变
}

// ── 多 bucket 并发搜索（pageAsync） ──

func Test_PageAsync_SearchAcrossAllBuckets(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	b1 := makeBucket("dir-a", makeMovie("1", "alpha.mp4", "/a/alpha.mp4", "", "", "", 100))
	b2 := makeBucket("dir-b", makeMovie("2", "beta.mp4", "/b/beta.mp4", "", "", "", 200))
	b3 := makeBucket("dir-c", makeMovie("3", "alpha-beta.mp4", "/c/alpha-beta.mp4", "", "", "", 300))

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir-a": b1, "dir-b": b2, "dir-c": b3})
	engine.installIndex(index)

	param := model.SearchParam{Keyword: "alpha", Page: 1, PageSize: 10, SortField: "Size", SortType: "desc"}
	result := engine.pageAsync(param)

	assert.Equal(t, 2, len(result.FileList), "应找到 2 个含 alpha 的文件")
	assert.Equal(t, "alpha-beta.mp4", result.FileList[0].Name, "应按 size desc 排序")
	assert.Equal(t, "alpha.mp4", result.FileList[1].Name)
}

func Test_PageAsync_NoMatchReturnsEmpty(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir", makeMovie("1", "cat.mp4", "/cat.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	engine.installIndex(index)

	param := model.SearchParam{Keyword: "dog", Page: 1, PageSize: 10}
	result := engine.pageAsync(param)
	assert.Equal(t, 0, len(result.FileList))
}

func Test_PageAsync_Pagination(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	movies := make([]model.FileItem, 25)
	for i := range movies {
		title := string(rune('A' + i))
		movies[i] = makeMovie(
			string(rune('a'+i)),
			title+".mp4",
			"/test/"+title+".mp4",
			"", "", "", int64(i+1),
		)
	}
	b := makeBucket("dir", movies...)
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	engine.installIndex(index)

	// 第1页 10条
	r1 := engine.pageAsync(model.SearchParam{Keyword: "", Page: 1, PageSize: 10})
	assert.Equal(t, 10, len(r1.FileList))

	// 第2页 10条
	r2 := engine.pageAsync(model.SearchParam{Keyword: "", Page: 2, PageSize: 10})
	assert.Equal(t, 10, len(r2.FileList))

	// 第3页 5条
	r3 := engine.pageAsync(model.SearchParam{Keyword: "", Page: 3, PageSize: 10})
	assert.Equal(t, 5, len(r3.FileList))
}

func Test_PageAsync_EmptyEngine(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	param := model.SearchParam{Keyword: "test", Page: 1, PageSize: 10}
	result := engine.pageAsync(param)
	assert.Equal(t, 0, len(result.FileList))
}

// ── repeat search ──

func Test_returnRepeatSearch(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 创建重复文件：同 Code + 同 Size
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "ABC", "", "", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "ABC", "", "", 100), // 重复
	)
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	engine.installIndex(index)

	param := model.SearchParam{OnlyRepeat: true}
	result := engine.pageAsync(param)
	assert.Greater(t, len(result.FileList), 0, "应检测到重复文件")
}

func emptySearchIndex() *searchIndex {
	return &searchIndex{
		buckets:   make(map[string]*bucketFile),
		authorMap: make(map[string]*model.Author),
	}
}

func TestGetTypeMenu_ReturnsAggregatedTypes(t *testing.T) {
	m1 := makeMovie("1", "movie1", "/d/f1.mp4", "AB-111", "mp4", "actor1", 100)
	m2 := makeMovie("2", "movie2", "/d/f2.mp4", "AB-222", "mp4", "actor2", 200)
	m3 := makeMovie("3", "movie3", "/d/f3.avi", "AB-333", "avi", "actor1", 300)

	bucket := makeBucket("d", m1, m2, m3)
	buckets := map[string]*bucketFile{"d": bucket}
	index := buildIndexFromBuckets(buckets)

	engine := newTestEngine()
	engine.installIndex(index)

	menu := engine.GetTypeMenu()
	if len(menu) < 2 {
		t.Fatalf("expected at least 2 types (mp4, avi), got %d", len(menu))
	}
	// mp4: 2 files = 300
	if v, ok := menu["mp4"]; !ok {
		t.Error("mp4 type not found in menu")
	} else if v.Size != 300 {
		t.Errorf("mp4 size = %d, want 300", v.Size)
	}
	// avi: 1 file = 300
	if v, ok := menu["avi"]; !ok {
		t.Error("avi type not found in menu")
	} else if v.Size != 300 {
		t.Errorf("avi size = %d, want 300", v.Size)
	}
	// 全部: 3 files = 600
	if v, ok := menu["全部"]; !ok {
		t.Error("'全部' type not found in menu")
	} else if v.Size != 600 {
		t.Errorf("'全部' size = %d, want 600", v.Size)
	}
}

func TestGetTagMenu_ReturnsAggregatedTags(t *testing.T) {
	m1 := makeMovie("1", "m1", "/d/f1.mp4", "AB-111", "mp4", "", 100)
	m1.Tags = []string{"tag1", "tag2"}
	m2 := makeMovie("2", "m2", "/d/f2.mp4", "AB-222", "mp4", "", 200)
	m2.Tags = []string{"tag1"}

	bucket := makeBucket("d", m1, m2)
	index := buildIndexFromBuckets(map[string]*bucketFile{"d": bucket})

	engine := newTestEngine()
	engine.installIndex(index)

	menu := engine.GetTagMenu()
	if len(menu) < 1 {
		t.Fatal("expected at least 1 tag")
	}
	if v, ok := menu["tag1"]; !ok {
		t.Error("tag1 not found")
	} else if v.Size != 300 { // 100 + 200
		t.Errorf("tag1 size = %d, want 300", v.Size)
	}
	if v, ok := menu["tag2"]; !ok {
		t.Error("tag2 not found")
	} else if v.Size != 100 {
		t.Errorf("tag2 size = %d, want 100", v.Size)
	}
}

func TestGetSeriesCount_ReturnsAggregatedSeries(t *testing.T) {
	m1 := makeMovie("1", "m1", "/d/f1.mp4", "AB-111", "mp4", "", 100)
	m1.Studio = "series1"
	m2 := makeMovie("2", "m2", "/d/f2.mp4", "AB-222", "mp4", "", 200)
	m2.Studio = "series1"

	bucket := makeBucket("d", m1, m2)
	index := buildIndexFromBuckets(map[string]*bucketFile{"d": bucket})

	engine := newTestEngine()
	engine.installIndex(index)

	sc := engine.GetSeriesCount()
	if v, ok := sc["series1"]; !ok {
		t.Error("series1 not found")
	} else if v.Size != 300 {
		t.Errorf("series1 size = %d, want 300", v.Size)
	}
}

func TestGetMenu_EmptyIndex(t *testing.T) {
	engine := newTestEngine()
	// 未安装索引时返回空 map
	menu := engine.GetTypeMenu()
	if menu == nil {
		t.Error("GetTypeMenu should return empty map, not nil")
	}
	if len(menu) != 0 {
		t.Errorf("expected empty menu, got %d items", len(menu))
	}
}

// ── matchAdvancedFiltersFast 测试（P3 修复验证） ──

func TestMatchAdvancedFiltersFast_DateRange(t *testing.T) {
	file := &model.FileItem{
		Size:  100,
		MTime: "2025-06-15 10:30:00",
		Name:  "test.mp4",
	}

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2025, 6, 30, 23, 59, 59, 999999999, time.Local)

	// 文件在范围内
	assert.True(t, matchAdvancedFiltersFast(file, 0, 0, &from, &to, nil))

	// 文件在范围前（2025-05-01 到 2025-05-31）
	beforeFrom := time.Date(2025, 5, 1, 0, 0, 0, 0, time.Local)
	beforeTo := time.Date(2025, 5, 31, 23, 59, 59, 999999999, time.Local)
	assert.False(t, matchAdvancedFiltersFast(file, 0, 0, &beforeFrom, &beforeTo, nil))

	// 文件在范围后（2025-07-01 到 2025-07-31）
	afterFrom := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
	afterTo := time.Date(2025, 7, 31, 23, 59, 59, 999999999, time.Local)
	assert.False(t, matchAdvancedFiltersFast(file, 0, 0, &afterFrom, &afterTo, nil))
}

func TestMatchAdvancedFiltersFast_ExtSet(t *testing.T) {
	file := &model.FileItem{
		Size: 100,
		Name: "test.mp4",
	}

	extSet := map[string]struct{}{
		"mp4": {},
		"avi": {},
	}

	// 匹配
	assert.True(t, matchAdvancedFiltersFast(file, 0, 0, nil, nil, extSet))

	// 不匹配
	file.Name = "test.mkv"
	assert.False(t, matchAdvancedFiltersFast(file, 0, 0, nil, nil, extSet))
}

func TestMatchAdvancedFiltersFast_SizeRange(t *testing.T) {
	file := &model.FileItem{
		Size: 500,
		Name: "test.mp4",
	}

	assert.True(t, matchAdvancedFiltersFast(file, 100, 1000, nil, nil, nil))
	assert.False(t, matchAdvancedFiltersFast(file, 600, 1000, nil, nil, nil))
	assert.False(t, matchAdvancedFiltersFast(file, 100, 400, nil, nil, nil))
}

// ── searchBucket 日期过滤测试（P3 修复验证） ──

func TestSearchBucket_DateFilter(t *testing.T) {
	b := makeBucket("dir",
		model.FileItem{Id: "1", Name: "old.mp4", Path: "/test/old.mp4", MTime: "2025-01-15 10:00:00", Size: 100},
		model.FileItem{Id: "2", Name: "new.mp4", Path: "/test/new.mp4", MTime: "2025-06-15 10:00:00", Size: 200},
		model.FileItem{Id: "3", Name: "mid.mp4", Path: "/test/mid.mp4", MTime: "2025-03-20 10:00:00", Size: 300},
	)

	// 只筛选 2025-06 之后的文件
	param := model.SearchParam{
		Keyword:  "",
		Page:     1,
		PageSize: 10,
		DateFrom: "2025-06-01",
	}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "new.mp4", result.FileList[0].Name)

	// 筛选 2025-01-01 到 2025-03-31
	param2 := model.SearchParam{
		Keyword:  "",
		Page:     1,
		PageSize: 10,
		DateFrom: "2025-01-01",
		DateTo:   "2025-03-31",
	}
	result2 := b.searchBucket(param2)
	assert.Equal(t, 2, len(result2.FileList))
}

// ── installIndexSkipDisk 缓存保持测试（P9 修复验证） ──

func TestInstallIndexSkipDisk_DoesNotClearCache(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 安装初始索引
	b := makeBucket("dir", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	// 向缓存写入一些数据
	core.KeywordHistoryCache.Set("test_key", "test_value")
	core.KeywordHistoryCache.Set("another_key", "another_value")

	// 用 installIndexSkipDisk 更新索引（模拟单文件操作）
	b2 := makeBucket("dir", makeMovie("2", "g.mp4", "/g.mp4", "", "", "", 200))
	index2 := buildIndexFromBuckets(map[string]*bucketFile{"dir": b2})
	core.installIndexSkipDisk(index2)

	// 验证缓存未被清空
	v1, ok1 := core.KeywordHistoryCache.Get("test_key")
	assert.True(t, ok1, "installIndexSkipDisk 不应清空 LRU 缓存")
	assert.Equal(t, "test_value", v1)

	v2, ok2 := core.KeywordHistoryCache.Get("another_key")
	assert.True(t, ok2)
	assert.Equal(t, "another_value", v2)
}

func TestInstallIndex_DoesClearCache(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 安装初始索引
	b := makeBucket("dir", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})
	core.installIndex(index)

	// 向缓存写入数据
	core.KeywordHistoryCache.Set("test_key", "test_value")

	// 用 installIndex 更新索引（模拟全量重建）
	b2 := makeBucket("dir", makeMovie("2", "g.mp4", "/g.mp4", "", "", "", 200))
	index2 := buildIndexFromBuckets(map[string]*bucketFile{"dir": b2})
	core.installIndex(index2)

	// 验证缓存已被清空
	_, ok := core.KeywordHistoryCache.Get("test_key")
	assert.False(t, ok, "installIndex 应清空 LRU 缓存")
}

// ── FindById O(1) 查找测试（P1 修复验证） ──

func TestFindById_UsesIdIndex(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 创建多个 bucket
	b1 := makeBucket("dir-a",
		makeMovie("id-1", "a.mp4", "/a.mp4", "", "", "", 100),
		makeMovie("id-2", "b.mp4", "/b.mp4", "", "", "", 200),
	)
	b2 := makeBucket("dir-b",
		makeMovie("id-3", "c.mp4", "/c.mp4", "", "", "", 300),
	)

	index := buildIndexFromBuckets(map[string]*bucketFile{"dir-a": b1, "dir-b": b2})
	core.installIndex(index)

	// 验证 idIndex 包含所有文件
	assert.Equal(t, 3, len(index.idIndex))

	// O(1) 查找
	f1 := core.FindById("id-1")
	assert.Equal(t, "a.mp4", f1.Name)

	f3 := core.FindById("id-3")
	assert.Equal(t, "c.mp4", f3.Name)

	// 不存在的 ID
	empty := core.FindById("nonexist")
	assert.True(t, empty.IsNull())
}

// ── BucketFile clone 测试 ──

func TestBucketFile_Clone_PreservesData(t *testing.T) {
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "ABC", "骑兵", "田中", 100),
		makeMovie("2", "b.mp4", "/b.mp4", "DEF", "步兵", "佐藤", 200),
	)

	cloned := b.clone()

	// 验证克隆保留所有数据
	assert.Equal(t, b.InstanceName, cloned.InstanceName)
	assert.Equal(t, b.TotalSize, cloned.TotalSize)
	assert.Equal(t, b.TotalCount, cloned.TotalCount)
	assert.Equal(t, len(b.FileLib), len(cloned.FileLib))
	assert.Equal(t, len(b.TypeIndex), len(cloned.TypeIndex))

	// 验证数据独立（修改克隆不影响原）
	clonedFile := cloned.get("1")
	assert.NotNil(t, clonedFile)
	clonedFile.Name = "modified.mp4"
	origFile := b.get("1")
	assert.Equal(t, "a.mp4", origFile.Name, "原 bucket 不应被修改")
}

func TestBucketFile_Clone_EmptyBucket(t *testing.T) {
	b := newInstance("empty")
	cloned := b.clone()

	assert.Equal(t, 0, cloned.TotalCount)
	assert.Equal(t, int64(0), cloned.TotalSize)
	assert.Empty(t, cloned.FileLib)
}

// ── searchBucket 扩展名过滤测试（P3 优化验证） ──

func TestSearchBucket_ExtFilter(t *testing.T) {
	b := makeBucket("dir",
		model.FileItem{Id: "1", Name: "video.mp4", Path: "/test/video.mp4", Size: 100, PathUpper: "/TEST/VIDEO.MP4"},
		model.FileItem{Id: "2", Name: "image.jpg", Path: "/test/image.jpg", Size: 50, PathUpper: "/TEST/IMAGE.JPG"},
		model.FileItem{Id: "3", Name: "movie.mkv", Path: "/test/movie.mkv", Size: 200, PathUpper: "/TEST/MOVIE.MKV"},
	)

	// 只筛选 mp4
	param := model.SearchParam{
		Keyword:  "",
		Page:     1,
		PageSize: 10,
		FileExts: []string{"mp4"},
	}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "video.mp4", result.FileList[0].Name)

	// 筛选 mp4 和 mkv
	param2 := model.SearchParam{
		Keyword:  "",
		Page:     1,
		PageSize: 10,
		FileExts: []string{"mp4", "mkv"},
	}
	result2 := b.searchBucket(param2)
	assert.Equal(t, 2, len(result2.FileList))
}

// ── author 缓存失效测试 ──

func TestAuthorCache_InvalidatedOnInstallIndex(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	// 安装初始索引
	b1 := makeBucket("dir", makeMovie("1", "a.mp4", "/a.mp4", "", "骑兵", "田中", 100))
	index1 := buildIndexFromBuckets(map[string]*bucketFile{"dir": b1})
	core.installIndex(index1)

	// 手动设置 author 缓存
	core.authorCacheMu.Lock()
	core.authorSizeCache = []model.Author{{Name: "old_author", Cnt: 1, Size: 100}}
	core.authorCountCache = []model.Author{{Name: "old_author", Cnt: 1, Size: 100}}
	core.authorCacheMu.Unlock()

	// 安装新索引
	b2 := makeBucket("dir", makeMovie("2", "b.mp4", "/b.mp4", "", "步兵", "佐藤", 200))
	index2 := buildIndexFromBuckets(map[string]*bucketFile{"dir": b2})
	core.installIndex(index2)

	// 验证缓存已清空
	core.authorCacheMu.RLock()
	assert.Nil(t, core.authorSizeCache, "installIndex 应清空 authorSizeCache")
	assert.Nil(t, core.authorCountCache, "installIndex 应清空 authorCountCache")
	core.authorCacheMu.RUnlock()
}

// ── hw_accel getter 函数测试（B3 修复验证） ──

func TestHwAccel_Getters_ReturnValidValues(t *testing.T) {
	// 保存原始设置
	orig := GetOSSetting()
	defer SetOSSetting(orig)

	// 禁用硬件加速时应返回软件编码器
	SetOSSetting(model.Setting{HardwareAcceleration: false})

	h264 := getH264Encoder()
	h265 := getH265Encoder()
	dec := getHwDecodeParams()
	qual := getHwQualityParam()
	mode := GetHwAccelModeName()

	assert.Equal(t, "libx264", h264, "禁用时应返回 libx264")
	assert.Equal(t, "libx265", h265, "禁用时应返回 libx265")
	assert.Equal(t, "", dec, "禁用时解码参数应为空")
	assert.Equal(t, "-crf", qual, "禁用时应返回 -crf")
	assert.Equal(t, "", mode, "禁用时模式应为空")
}

// ── searchBucket 组合过滤测试 ──

func TestSearchBucket_CombinedFilters(t *testing.T) {
	b := makeBucket("dir",
		model.FileItem{Id: "1", Name: "alpha.mp4", Path: "/test/alpha.mp4", Size: 100, MTime: "2025-06-15 10:00:00", PathUpper: "/TEST/ALPHA.MP4"},
		model.FileItem{Id: "2", Name: "beta.mkv", Path: "/test/beta.mkv", Size: 200, MTime: "2025-07-20 10:00:00", PathUpper: "/TEST/BETA.MKV"},
		model.FileItem{Id: "3", Name: "gamma.mp4", Path: "/test/gamma.mp4", Size: 50, MTime: "2025-03-10 10:00:00", PathUpper: "/TEST/GAMMA.MP4"},
	)

	// 关键词 + 扩展名 + 日期范围
	param := model.SearchParam{
		Keyword:  "alpha",
		Page:     1,
		PageSize: 10,
		FileExts: []string{"mp4"},
		DateFrom: "2025-01-01",
		DateTo:   "2025-12-31",
	}
	result := b.searchBucket(param)
	assert.Equal(t, 1, len(result.FileList))
	assert.Equal(t, "alpha.mp4", result.FileList[0].Name)

	// 无匹配（扩展名不匹配）
	param2 := model.SearchParam{
		Keyword:  "alpha",
		Page:     1,
		PageSize: 10,
		FileExts: []string{"mkv"},
	}
	result2 := b.searchBucket(param2)
	assert.Equal(t, 0, len(result2.FileList))
}

// ── cacheEpoch 递增测试 ──

func TestCacheEpoch_IncrementsOnInstall(t *testing.T) {
	core := newTestEngine()
	defer core.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir", makeMovie("1", "f.mp4", "/f.mp4", "", "", "", 100))
	index := buildIndexFromBuckets(map[string]*bucketFile{"dir": b})

	epoch1 := core.cacheEpoch.Load()
	core.installIndex(index)
	epoch2 := core.cacheEpoch.Load()

	assert.Greater(t, epoch2, epoch1, "installIndex 应递增 cacheEpoch")

	core.installIndexSkipDisk(emptySearchIndex())
	epoch3 := core.cacheEpoch.Load()

	assert.Greater(t, epoch3, epoch2, "installIndexSkipDisk 应递增 cacheEpoch")
}
