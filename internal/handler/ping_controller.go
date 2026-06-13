package handler

import (
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// PingHost 通过系统 ping 检测目标主机是否在线
func PingHost(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"fail": true, "msg": "缺少 ip 参数"})
		return
	}

	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"ping", "-n", "1", "-w", "2000", ip}
	} else {
		args = []string{"ping", "-c", "1", "-W", "2", ip}
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	alive := err == nil
	// 某些语言环境下即使 ping 通也可能有非零退出码，再检查输出
	if !alive {
		outStr := strings.ToLower(string(output))
		if strings.Contains(outStr, "ttl=") || strings.Contains(outStr, "time=") || strings.Contains(outStr, "bytes from") || strings.Contains(outStr, "来自") {
			alive = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"alive": alive,
		"ip":    ip,
	})
}
