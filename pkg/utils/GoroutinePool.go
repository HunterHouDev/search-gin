package utils

import (
	"sync"
)

// GoroutinePool goroutine池

type GoroutinePool struct {
	capacity int
	jobs     chan func()
	wg       sync.WaitGroup
	closed   bool
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
		pool.wg.Add(1)
		go func() {
			defer pool.wg.Done()
			for job := range pool.jobs {
				job()
			}
		}()
	}

	return pool
}

// Submit 提交任务
func (p *GoroutinePool) Submit(job func()) {
	if p.closed {
		return
	}
	p.jobs <- job
}

// Wait 等待所有任务完成并关闭池
func (p *GoroutinePool) Wait() {
	p.closed = true
	close(p.jobs)
	p.wg.Wait()
}
