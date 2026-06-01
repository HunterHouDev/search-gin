# 自动化日志

## 2026-06-01 - 索引构建并发问题修复

### 问题描述
最近优化代码导致构建索引失败，主要问题是并发竞态条件：
1. BucketCount 在 goroutine 中更新，但 buildIndexEngin 立即读取
2. IndexNumber 和 BucketCount 之间没有同步机制
3. 缺少详细的日志和健康检查接口

### 修复内容

#### 1. 添加 BucketCount 断言检查（index_engin.go）
- **位置**: `internal/service/index_engin.go:141-149`
- **修改**: 在 PageAsync 方法中添加零值检查
- **功能**: 防止 BucketCount 为 0 时创建无缓冲 channel 导致死锁
```go
if bucketCount <= 0 {
    AddLogMemory("警告: BucketCount=%d <= 0，可能存在竞态条件，跳过搜索", bucketCount)
    resultWrapper.FileList = []model.Movie{}
    return resultWrapper
}
```

#### 2. 添加 setBucket 日志（index_engin.go）
- **位置**: `internal/service/index_engin.go:252-256`
- **修改**: 添加详细的 setBucket 调用日志
```go
before := atomic.LoadInt32(&se.BucketCount)
se.SearchIndexMap.Store(baseDir, bucket)
after := atomic.AddInt32(&se.BucketCount, 1)
AddLogMemory("setBucket: %s, before=%d, after=%d", baseDir, before, after)
```

#### 3. 添加 buildIndexEngin 详细日志（index_engin.go）
- **位置**: `internal/service/index_engin.go:269-419`
- **修改**: 在索引构建的各个阶段添加日志记录
- **日志点**:
  - 开始构建索引
  - 开始遍历 SearchIndexMap
  - 处理每个 bucket 的详细信息
  - 遍历完成统计
  - 开始写入全局菜单数据
  - 构建完成统计

#### 4. 添加一致性检查（file_service.go）
- **位置**: `internal/service/file_service.go:375-381`
- **修改**: 在 ScanAll 方法中添加 IndexNumber 和 BucketCount 一致性检查
```go
bucketCount := atomic.LoadInt32(&SearchEngin.BucketCount)
indexNumber := atomic.LoadInt32(&consts.IndexNumber)
AddLogMemory("ScanAll 一致性检查: BucketCount=%d, IndexNumber=%d, Expected=%d", bucketCount, indexNumber, dirCount)
if bucketCount != int32(dirCount) {
    AddLogMemory("警告: BucketCount(%d) != Expected(%d)，可能存在并发问题", bucketCount, dirCount)
}
```

#### 5. 添加 goWalkWithResult 日志（file_service.go）
- **位置**: `internal/service/file_service.go:426-435`
- **修改**: 添加目录扫描开始和完成的日志
```go
AddLogMemory("goWalkWithResult: 开始扫描目录 %s", baseDir)
// ...
AddLogMemory("goWalkWithResult: 扫描完成 %s, 发现 %d 个文件，准备添加到索引", baseDir, len(files))
```

#### 6. 创建健康检查接口
- **文件**: `internal/handler/health_controller.go`
- **路由**: `GET /api/indexHealth`
- **功能**:
  - 返回 BucketCount、IndexNumber、ExpectedDirs
  - 返回 TotalCount、TotalSize
  - 返回状态: healthy/warning/error/empty
  - 提供问题建议
- **路由注册**: `internal/router/BuildRouter.go:119`

### 修改文件清单
- ✅ `internal/service/index_engin.go` - 添加断言、日志
- ✅ `internal/service/file_service.go` - 添加一致性检查、日志
- ✅ `internal/handler/health_controller.go` - 新建健康检查接口
- ✅ `internal/router/BuildRouter.go` - 注册健康检查路由

### 测试建议
1. 空目录扫描测试
2. 单目录扫描测试
3. 多目录并发扫描测试
4. 快速连续扫描测试
5. 调用 `/api/indexHealth` 接口验证状态

### 后续行动
- [ ] 完整测试覆盖
- [ ] 验证日志输出
- [ ] 部署到测试环境
- [ ] 监控健康检查接口

### 风险评估
- **风险等级**: 🟠 Medium
- **原因**: 主要增加了日志和检查，未改变核心逻辑
- **收益**: 大幅提升问题定位能力和系统可观测性

### 编译验证
- ✅ **编译状态**: 成功
- ✅ **可执行文件**: search-gin.exe 已生成
- ✅ **编译时间**: 2026-06-01 13:44:09
- ✅ **修复的问题**: 移除了 file_service.go 中未使用的 "log" 导入

### 前端修改

#### 1. 添加 IndexHealthQuery API 函数
- **文件**: `frontend/src/components/api/searchAPI.ts`
- **位置**: 第 101-105 行
- **功能**: 调用后端健康检查接口
```typescript
export const IndexHealthQuery = async () => {
  const res = await commonAxios().get('/api/indexHealth');
  return res && res.data;
};
```

#### 2. 更新 IndexButton 组件
- **文件**: `frontend/src/components/IndexButton.vue`
- **功能**:
  - 添加健康状态数据绑定
  - 动态颜色显示状态（绿色=健康，橙色=扫描中，红色=错误，灰色=默认）
  - 鼠标悬停显示详细信息（状态、文件数、建议）
  - 每次扫描完成后自动查询健康状态

- **新增数据字段**:
  - `healthStatus`: 健康状态 (healthy/warning/error/empty)
  - `totalCount`: 总文件数
  - `recommendations`: 建议列表
  - `bucketCount`: Bucket 数量
  - `expectedDirs`: 期望目录数

- **新增计算属性**:
  - `statusColor`: 根据状态返回 Quasar 颜色
  - `statusLabel`: 显示按钮文字

- **新增方法**:
  - `queryHealth()`: 查询并更新健康状态
