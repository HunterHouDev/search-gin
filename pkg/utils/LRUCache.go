package utils

import (
	"container/list"
	"sync"
)

// LRUCache LRU缓存实现
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.RWMutex
}

// CacheItem 缓存项
type CacheItem struct {
	Key   string
	Value interface{}
}

// NewLRUCache 创建LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100
	}

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get 获取缓存值（读锁查找 + 写锁移动双检，降低锁竞争）
func (c *LRUCache) Get(key string) (interface{}, bool) {
 c.mu.RLock()
 element, ok := c.cache[key]
 if !ok {
  c.mu.RUnlock()
  return nil, false
 }
 c.mu.RUnlock()

 // 写锁下再次确认 + 移到链表头部
 c.mu.Lock()
 // double-check：在获取写锁后重新确认元素仍有效
 element, ok = c.cache[key]
 if !ok {
  c.mu.Unlock()
  return nil, false
 }
 c.list.MoveToFront(element)
 c.mu.Unlock()

 return element.Value.(*CacheItem).Value, true
}

// Set 设置缓存值
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已存在，更新值并移动到链表头部
	if element, ok := c.cache[key]; ok {
		element.Value.(*CacheItem).Value = value
		c.list.MoveToFront(element)
		return
	}

	// 如果缓存已满，删除最久未使用的项
	if c.list.Len() >= c.capacity {
		tail := c.list.Back()
		if tail != nil {
			delete(c.cache, tail.Value.(*CacheItem).Key)
			c.list.Remove(tail)
		}
	}

	// 添加新项到链表头部
	item := &CacheItem{Key: key, Value: value}
	element := c.list.PushFront(item)
	c.cache[key] = element
}

// Delete 删除缓存项
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.cache[key]; ok {
		delete(c.cache, key)
		c.list.Remove(element)
	}
}

// Clear 清空缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.list.Init()
}

// Len 获取缓存大小
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.list.Len()
}
