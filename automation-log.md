# 自动化操作日志

## 2026-05-25 高危Bug修复 + 性能瓶颈修复

### Bug修复

#### Bug1: GetPlay空值判空
- **文件**: `internal/handler/file_controller.go`
- **修复**: FindOne返回空Movie后先判空，避免对空路径调ValidatePath

#### Bug4: GetActressImage无响应
- **文件**: `internal/handler/file_controller.go`
- **修复**: actress为空或图片不存在时返回404

#### Bug5: GetCutImage IsNull判断缺return
- **文件**: `internal/handler/file_controller.go`
- **修复**: IsNull判断后添加return，避免对空文件继续操作

#### Bug9: LogMemory并发不安全
- **文件**: `pkg/consts/analysis_data.go`
- **修复**: 添加logMemoryMutex保护append和截断操作，截断时重新分配底层数组避免内存泄漏

#### Bug10: SmallDir并发不安全
- **文件**: `pkg/consts/analysis_data.go`
- **修复**: 添加smallDirMutex和AppendSmallDir/GetSmallDir/ClearSmallDir方法

#### Bug25: PostMovies/PostActress Bind失败无响应
- **文件**: `internal/handler/search_controller.go`
- **修复**: Bind失败时返回400错误响应，而非直接return挂起连接

#### Bug27+13: PostSetting绑定失败+OSSetting未加锁
- **文件**: `internal/handler/setting_controller.go`
- **修复**: 绑定失败返回400；使用GetOSSetting()/SetOSSetting()代替直接访问

#### Bug29: WriteDictionaryToJson OpenFile失败后继续执行
- **文件**: `internal/service/config_service.go`
- **修复**: OpenFile失败时添加return，避免对nil指针操作panic

#### Bug33: GetPageOfFiles/GetActressPageOfFiles越界panic
- **文件**: `internal/model/Movie.go`, `internal/model/Actress.go`
- **修复**: 添加start>=length越界检查

#### Bug35: GetLast运算优先级错误
- **文件**: `internal/model/TransferTask.go`
- **修复**: `(FinishTime.Unix() - CreateTime.Unix()) / 1000`，加括号修正运算顺序

#### Bug42: ValidatePath HasPrefix路径遍历绕过
- **文件**: `pkg/utils/OsFilepathUtils.go`
- **修复**: 改为`absPath == absAllowed || HasPrefix(absPath, absAllowed+Separator)`

#### Bug54: SetResultCnt除零panic
- **文件**: `pkg/utils/Page.go`
- **修复**: PageSize为0时默认设为10

#### Bug17: GetIpAddr连接未关闭
- **文件**: `internal/service/file_service.go`
- **修复**: 添加`defer conn.Close()`

### 性能修复

#### 性能21: buildIndexEngin goroutine无等待
- **文件**: `internal/service/index_engin.go`
- **修复**: 添加sync.WaitGroup等待3个goroutine完成，消除数据竞争和数据不一致

---

## 2026-05-25 第二轮高危修复

### Bug修复

#### Bug7: ImageToPng os.Open错误被忽略
- **文件**: `pkg/utils/ImageUtils.go`
- **修复**: 检查os.Open错误，失败时return

#### Bug22+23: PostOpenFolderByPath/PostDeleteFolerByPath路径遍历
- **文件**: `internal/handler/dir_controller.go`
- **修复**: 添加ValidatePath校验 + 绑定失败返回400 + 删除返回"删除成功"

#### Bug48: GetSettingInfo暴露密码
- **文件**: `internal/handler/setting_controller.go`
- **修复**: 返回前清空Users字段脱敏

#### Bug50: GetPlay goroutine内用gin.Context
- **文件**: `internal/handler/file_controller.go`
- **修复**: 移除goroutine内的c.JSON调用，错误只记日志

### 性能修复

#### 性能1: 搜索路径每次重复ToUpper
- **文件**: `internal/model/Movie.go`, `internal/service/index_engin_bucket.go`
- **修复**: Movie添加PathUpper字段，put时预计算，searchBucket优先使用

#### 性能7: LRU Cache Get用写锁
- **文件**: `pkg/utils/LRUCache.go`
- **修复**: Get改为先RLock读值再Lock移动，减少锁竞争

#### 性能2: HasItem线性查找改map
- **文件**: `pkg/utils/CollectionsUtils.go`, `internal/service/file_service.go`
- **修复**: 添加HasItemSet/ToSet函数，Walk/WalkInnter入口转map，查找O(1)

### 修改文件汇总
- internal/handler/file_controller.go
- internal/handler/search_controller.go
- internal/handler/setting_controller.go
- internal/handler/dir_controller.go
- internal/service/config_service.go
- internal/service/file_service.go
- internal/service/index_engin.go
- internal/service/index_engin_bucket.go
- internal/model/Movie.go
- internal/model/Actress.go
- internal/model/TransferTask.go
- pkg/consts/analysis_data.go
- pkg/utils/OsFilepathUtils.go
- pkg/utils/Page.go
- pkg/utils/ImageUtils.go
- pkg/utils/LRUCache.go
- pkg/utils/CollectionsUtils.go
