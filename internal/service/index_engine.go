package service

// 搜索引擎核心，分散在以下文件中：
// - snapshot_manager.go: searchEngineCore / searchSnapshot 类型定义、快照生命周期
// - index_builder.go: 批量/增量索引重建、聚合操作
// - search_executor.go: 搜索执行（分页搜索、演员搜索、查询）
// - index_engine_bucket.go: bucket 存储与搜索
// - index_engine_cache.go: 快照持久化
