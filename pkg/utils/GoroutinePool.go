package utils

import (
	"sync"
	"sync/atomic"
)

// GoroutinePool goroutine池

type GoroutinePool struct {
	capacity int
	jobs     chan func()
	wg       sync.WaitGroup
	closed   atomic.Bool
}

// NewGoroutinePool 创建goroutine池
func NewGoroutinePool(capacity int) *GoroutinePool {
	if capacity <= 0 {
		capacity = 10
	}

	pool := &GoroutinePool{
		capacity: capacity,
		jobs:     make(chan func(), 1000),
	}

	// 启动goroutine
	for i := 0; i < capacity; i++ {
		go func() {
			for job := range pool.jobs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							InfoFormat("GoroutinePool: 任务 panic 已恢复: %v", r)
						}
					}()
					job()
				}()
			}
		}()
	}

	return pool
}

// Submit 提交任务，非阻塞；若池满则在当前 goroutine 直接执行
// 若池已关闭则直接在当前 goroutine 执行，避免向已关闭 channel 发送 panic
func (p *GoroutinePool) Submit(job func()) {
	p.wg.Add(1)
	wrapped := func() {
		defer p.wg.Done()
		job()
	}
	if p.closed.Load() {
		wrapped()
		return
	}
	select {
	case p.jobs <- wrapped:
	default:
		wrapped()
	}
}

// Wait 等待所有已提交的任务完成
func (p *GoroutinePool) Wait() {
	p.wg.Wait()
}

// Cap 返回池容量
func (p *GoroutinePool) Cap() int {
	return p.capacity
}

// Close 关闭goroutine池，停止接收新任务
func (p *GoroutinePool) Close() {
	if p.closed.CompareAndSwap(false, true) {
		close(p.jobs)
	}
}
