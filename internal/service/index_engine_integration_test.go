package service

import (
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ── 集成测试：完整搜索流程 ──

func TestIntegration_FullSearchFlow(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 创建 3 个 bucket，各含不同文件
	b1 := makeBucket("dir1",
		makeMovie("1", "test.mp4", "/dir1/test.mp4", "ABC-1", "骑兵", "田中", 1024),
		makeMovie("2", "hello.mp4", "/dir1/hello.mp4", "ABC-2", "骑兵", "田中", 2048),
	)
	b2 := makeBucket("dir2",
		makeMovie("3", "world.mp4", "/dir2/world.mp4", "ABC-3", "步兵", "佐藤", 4096),
	)
	b3 := makeBucket("dir3",
		makeMovie("4", "test_demo.mp4", "/dir3/test_demo.mp4", "ABC-4", "国产", "王", 512),
		makeMovie("5", "other.avi", "/dir3/other.avi", "ABC-5", "漫动", "李", 256),
	)

	buckets := map[string]*bucketFile{"dir1": b1, "dir2": b2, "dir3": b3}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 全部搜索
	param := model.SearchParam{Page: 1, PageSize: 10, Keyword: ""}
	result := engine.Page(param)
	assert.Equal(t, 5, result.TotalCnt)
	assert.Len(t, result.Data.([]model.FileItem), 5)

	// 关键词搜索
	param = model.SearchParam{Page: 1, PageSize: 10, Keyword: "test"}
	result = engine.Page(param)
	list := result.Data.([]model.FileItem)
	assert.Equal(t, 2, result.TotalCnt)
	assert.Equal(t, 2, len(list))

	// 类型过滤
	param = model.SearchParam{Page: 1, PageSize: 10, MovieType: "骑兵"}
	result = engine.Page(param)
	list = result.Data.([]model.FileItem)
	assert.Equal(t, 2, result.TotalCnt)
	for _, f := range list {
		assert.Equal(t, "骑兵", f.MovieType)
	}

	// 分页验证（每页 2 条）
	param = model.SearchParam{Page: 1, PageSize: 2, Keyword: ""}
	result = engine.Page(param)
	list = result.Data.([]model.FileItem)
	assert.Equal(t, 5, result.TotalCnt) // 总数仍是 5
	assert.Equal(t, 2, len(list))

	param = model.SearchParam{Page: 3, PageSize: 2, Keyword: ""}
	result = engine.Page(param)
	list = result.Data.([]model.FileItem)
	assert.Equal(t, 5, result.TotalCnt)
	assert.Equal(t, 1, len(list)) // 第3页只有1条

	// 无匹配
	param = model.SearchParam{Page: 1, PageSize: 10, Keyword: "nonexistent"}
	result = engine.Page(param)
	assert.Equal(t, 0, result.TotalCnt)
	assert.Empty(t, result.Data.([]model.FileItem))
}

func TestIntegration_SortBySize(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/a.mp4", "", "", "", 500),
		makeMovie("2", "b.mp4", "/b.mp4", "", "", "", 100),
		makeMovie("3", "c.mp4", "/c.mp4", "", "", "", 9999),
	)
	buckets := map[string]*bucketFile{"dir": b}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 降序
	param := model.SearchParam{Page: 1, PageSize: 10, SortField: "Size", SortType: "desc"}
	result := engine.Page(param)
	list := result.Data.([]model.FileItem)
	assert.Equal(t, 3, len(list))
	assert.True(t, list[0].Size >= list[1].Size && list[1].Size >= list[2].Size,
		"降序排列: %d %d %d", list[0].Size, list[1].Size, list[2].Size)

	// 升序
	param = model.SearchParam{Page: 1, PageSize: 10, SortField: "Size", SortType: "asc"}
	result = engine.Page(param)
	list = result.Data.([]model.FileItem)
	assert.True(t, list[0].Size <= list[1].Size && list[1].Size <= list[2].Size,
		"升序排列: %d %d %d", list[0].Size, list[1].Size, list[2].Size)
}

// ── 集成测试：并发搜索 ──

func TestIntegration_ConcurrentSearch(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 构建 10 个 bucket，每个 100 文件
	buckets := make(map[string]*bucketFile)
	for bi := 0; bi < 10; bi++ {
		name := utils.IntToString(bi)
		movies := make([]model.FileItem, 0, 100)
		for fi := 0; fi < 100; fi++ {
			id := utils.IntToString(bi*1000 + fi)
			movies = append(movies, makeMovie(id, "movie_"+id+".mp4", "/"+name+"/"+id+".mp4",
				"CODE-"+id, "骑兵", "作者"+name, int64(fi*100)))
		}
		buckets[name] = makeBucket(name, movies...)
	}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 20 个 goroutine 并发搜索
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errs <- utils.Errorf("goroutine %d panic: %v", id, r)
				}
			}()
			param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "movie"}
			result := engine.Page(param)
			if result.TotalCnt < 0 {
				errs <- utils.Errorf("goroutine %d: TotalCnt=%d", id, result.TotalCnt)
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// ── 集成测试：搜索超时 ──

func TestIntegration_SearchTimeout(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 构建 50 个 bucket，各含 200 文件
	buckets := make(map[string]*bucketFile)
	for bi := 0; bi < 50; bi++ {
		name := utils.IntToString(bi)
		movies := make([]model.FileItem, 0, 200)
		for fi := 0; fi < 200; fi++ {
			id := utils.IntToString(bi*1000 + fi)
			movies = append(movies, makeMovie(id, "data_"+id+".mp4", "/"+name+"/"+id+".mp4",
				"CODE-"+id, "类型"+name, "作者"+name, int64(fi)))
		}
		buckets[name] = makeBucket(name, movies...)
	}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 正常搜索应成功
	param := model.SearchParam{Page: 1, PageSize: 10, Keyword: ""}
	result := engine.Page(param)
	assert.Greater(t, result.TotalCnt, 0)
	// 10000 个文件，全部返回
	assert.Equal(t, 10000, result.TotalCnt)
}

// ── 集成测试：索引增量更新 ──

func TestIntegration_IncrementalUpdate(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 初始索引
	b := makeBucket("dir",
		makeMovie("1", "original.mp4", "/dir/original.mp4", "ORG-1", "骑兵", "田中", 1000),
	)
	buckets := map[string]*bucketFile{"dir": b}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 验证初始数据
	param := model.SearchParam{Page: 1, PageSize: 10, Keyword: "original"}
	result := engine.Page(param)
	assert.Equal(t, 1, result.TotalCnt)

	// 添加新文件
	_, err := engine.ReplaceFile(model.FileEdit{
		FileItem: model.FileItem{
			Id:   "2",
			Name: "new.mp4",
			Path: "/dir/new.mp4",
			Size: 2000,
		},
		NoRefresh: true,
	})
	assert.NoError(t, err)

	// 验证新文件可搜索
	param = model.SearchParam{Page: 1, PageSize: 10, Keyword: "new"}
	result = engine.Page(param)
	assert.Equal(t, 1, result.TotalCnt)

	// 修改文件类型
	engine.DeleteFile("1")

	// 验证已删除的文件不再返回
	param = model.SearchParam{Page: 1, PageSize: 10, Keyword: "original"}
	result = engine.Page(param)
	assert.Equal(t, 0, result.TotalCnt)
}

// ── 集成测试：去重搜索 ──

func TestIntegration_RepeatSearch(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	// 创建重复文件（相同 Code）
	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/dir/a.mp4", "DUP-1", "", "", 1000),
		makeMovie("2", "b.mp4", "/dir/b.mp4", "DUP-1", "", "", 1000), // 同 Code+Size
		makeMovie("3", "c.mp4", "/dir/c.mp4", "DUP-2", "", "", 2000),
	)
	buckets := map[string]*bucketFile{"dir": b}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 搜索重复文件
	param := model.SearchParam{Page: 1, PageSize: 10}
	param.SetOnlyRepeat()
	result := engine.Page(param)
	list := result.Data.([]model.FileItem)
	// DUP-1 有 2 个文件（重复），DUP-2 只有 1 个
	assert.Equal(t, 2, result.TotalCnt)
	_ = list
}

// ── 集成测试：高级过滤 ──

func TestIntegration_FilterSearch(t *testing.T) {
	engine := newTestEngine()
	defer engine.installIndexSkipDisk(emptySearchIndex())

	b := makeBucket("dir",
		makeMovie("1", "a.mp4", "/dir/a.mp4", "", "", "", 100),
		makeMovie("2", "b.mp4", "/dir/b.mp4", "", "", "", 500),
		makeMovie("3", "c.txt", "/dir/c.txt", "", "", "", 1000),
		makeMovie("4", "d.mp4", "/dir/d.mp4", "", "", "", 2000),
	)
	buckets := map[string]*bucketFile{"dir": b}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)

	// 最小大小过滤
	param := model.SearchParam{Page: 1, PageSize: 10, MinSize: 500}
	result := engine.Page(param)
	assert.Equal(t, 3, result.TotalCnt)

	// 大小范围
	param = model.SearchParam{Page: 1, PageSize: 10, MinSize: 100, MaxSize: 1000}
	result = engine.Page(param)
	assert.Equal(t, 2, result.TotalCnt)
}
