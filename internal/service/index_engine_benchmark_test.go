package service

import (
	"fmt"
	"search-gin/internal/model"
	"testing"
)

// ── 辅助：构建指定数量的测试数据 ──

func buildBenchData(b *testing.B, bucketCount, filesPerBucket int) *searchEngineCore {
	engine := newTestEngine()

	buckets := make(map[string]*bucketFile)
	for bi := 0; bi < bucketCount; bi++ {
		name := fmt.Sprint(bi)
		movies := make([]model.FileItem, 0, filesPerBucket)
		for fi := 0; fi < filesPerBucket; fi++ {
			code := fmt.Sprint(fi)
			if fi%2 == 0 {
				movies = append(movies, makeMovie(
					fmt.Sprint(bi*1000000+fi),
					"test_"+code+".mp4",
					"/"+name+"/test_"+code+".mp4",
					"CODE-"+code,
					[]string{"骑兵", "步兵", "国产", "漫动"}[fi%4],
					"作者"+fmt.Sprint(fi%100),
					int64(fi*1000),
				))
			} else {
				movies = append(movies, makeMovie(
					fmt.Sprint(bi*1000000+fi),
					"data_"+code+".mp4",
					"/"+name+"/data_"+code+".mp4",
					"OTHER-"+code,
					[]string{"骑兵", "步兵", "国产", "漫动"}[fi%4],
					"作者"+fmt.Sprint(fi%100),
					int64(fi*1000),
				))
			}
		}
		buckets[name] = makeBucket(name, movies...)
	}
	index := buildIndexFromBuckets(buckets)
	engine.installIndex(index)
	return &engine
}

// BenchmarkPage_1k 1K 文件 + 1 个 bucket 搜索
func BenchmarkPage_1k(b *testing.B) {
	engine := buildBenchData(b, 1, 1000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "test"}
	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}

// BenchmarkPage_10k 10K 文件 + 5 个 bucket 搜索
func BenchmarkPage_10k(b *testing.B) {
	engine := buildBenchData(b, 5, 2000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "test"}
	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}

// BenchmarkPage_50k 50K 文件 + 10 个 bucket 搜索
func BenchmarkPage_50k(b *testing.B) {
	engine := buildBenchData(b, 10, 5000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "test"}
	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}

// BenchmarkPage_Parallel 并发搜索（50K 文件）
func BenchmarkPage_Parallel(b *testing.B) {
	engine := buildBenchData(b, 10, 5000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "test"}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine.Page(param)
		}
	})
}

// BenchmarkPage_NoKeyword 无关键词搜索（返回所有，50K 文件）
func BenchmarkPage_NoKeyword(b *testing.B) {
	engine := buildBenchData(b, 10, 5000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: ""}
	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}

// BenchmarkPage_TypeFilter 类型过滤搜索（50K 文件）
func BenchmarkPage_TypeFilter(b *testing.B) {
	engine := buildBenchData(b, 10, 5000)
	b.ResetTimer()

	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "", MovieType: "骑兵"}
	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}

// BenchmarkPage_CacheHit 缓存命中（搜索相同关键词 2 次）
func BenchmarkPage_CacheHit(b *testing.B) {
	engine := buildBenchData(b, 10, 5000)

	// 首次搜索填充缓存
	param := model.SearchParam{Page: 1, PageSize: 20, Keyword: "test"}
	engine.Page(param)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.Page(param)
	}
}
