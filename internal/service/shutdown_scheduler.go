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
		utils.InfoFormat("定时关机协程已启动")
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
					utils.InfoFormat("定时关机倒计时: %d -> %d", cur, next)
					// SSE 广播到所有前端
					sse.BroadcastEvent(model.SSEShutdownStatus, map[string]any{
						"remaining": next,
					})
					// 归零 → 执行系统关机
					if next <= 0 {
						utils.InfoFormat("定时关机倒计时结束，执行系统关机")
						// 先广播 remaining:0 确保前端显示归零，再执行关机
						// 因为关机命令会终止进程，之后的日志可能来不及写入
						ShutdownSystem()
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
	utils.InfoFormat("ScheduleShutdown 被调用: %d 秒", seconds)
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
	LogMem.Add("ShutdownSystem 被调用，准备执行系统关机")
	utils.InfoFormat("ShutdownSystem 被调用，准备执行系统关机")
	if runtime.GOOS != "windows" {
		LogMem.Add("非 Windows 系统，跳过 shutdown 命令")
		utils.InfoFormat("非 Windows 系统，跳过 shutdown 命令")
		return
	}
	// -f 强制关闭应用，避免弹窗阻止关机
	cmd := exec.Command("cmd", "/C", "shutdown -s -f -t 0")
	utils.FixOnWin(cmd)
	LogMem.Add("ShutdownSystem: 执行命令 %v", cmd.Args)
	utils.InfoFormat("ShutdownSystem: 执行命令 %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		LogMem.Add("执行系统关机失败: %v, 输出: %s", err, string(out))
		utils.ErrorFormat("执行系统关机失败: %v, 输出: %s", err, string(out))
	} else {
		LogMem.Add("关机命令已成功执行，输出: %s", string(out))
		utils.InfoFormat("ShutdownSystem: 关机命令已成功执行，输出: %s", string(out))
	}
}
