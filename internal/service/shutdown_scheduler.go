package service

import (
	"os/exec"
	"runtime"
	"search-gin/internal/model"
	"search-gin/internal/sse"
	"search-gin/pkg/utils"
	"sync/atomic"
	"time"
)

// shutdownScheduler 后端定时关机管理器
// 所有前端通过 SSE shutdown_status 事件实时同步剩余时间
var shutdownScheduler = struct {
	remaining atomic.Int64 // 剩余秒数，≤0 表示无定时任务
}{}

// InitShutdownScheduler 启动定时关机协程（每秒递减 + SSE 广播）
func InitShutdownScheduler() {
	go func() {
		defer utils.RecoverPanic()
		for {
			time.Sleep(1 * time.Second)

			// CAS 安全递减（防止与 ScheduleShutdown/CancelShutdown 并发写冲突）
			for {
				cur := shutdownScheduler.remaining.Load()
				if cur <= 0 {
					break
				}
				next := cur - 1
				if shutdownScheduler.remaining.CompareAndSwap(cur, next) {
					// SSE 广播到所有前端
					sse.BroadcastEvent(model.SSEShutdownStatus, map[string]any{
						"remaining": next,
					})
					// 归零 → 执行系统关机
					if next <= 0 {
						utils.InfoFormat("定时关机倒计时结束，执行系统关机")
						go ShutdownSystem()
					}
					break
				}
				// CAS 失败（被其他 goroutine 修改），重试
			}
		}
	}()
}

// ScheduleShutdown 设置定时关机（秒数）
func ScheduleShutdown(seconds int) {
	shutdownScheduler.remaining.Store(int64(seconds))
	utils.InfoFormat("定时关机已设置: %d 秒后执行", seconds)

	// 立即广播初始值
	sse.BroadcastEvent(model.SSEShutdownStatus, map[string]any{
		"remaining": seconds,
	})
}

// CancelShutdown 取消定时关机
func CancelShutdown() {
	shutdownScheduler.remaining.Store(0)
	utils.InfoFormat("定时关机已取消")

	sse.BroadcastEvent(model.SSEShutdownStatus, map[string]any{
		"remaining": 0,
	})
}

// GetShutdownRemaining 获取当前剩余秒数
func GetShutdownRemaining() int64 {
	return shutdownScheduler.remaining.Load()
}

// ShutdownSystem 执行操作系统关机（Windows shutdown -s -t 0）
func ShutdownSystem() {
	if runtime.GOOS != "windows" {
		utils.InfoFormat("非 Windows 系统，跳过 shutdown 命令")
		return
	}
	cmd := exec.Command("cmd", "/C", "shutdown -s -t 0")
	utils.FixOnWin(cmd)
	if err := cmd.Run(); err != nil {
		utils.ErrorFormat("执行系统关机失败: %v", err)
	}
}
