package handler

import (
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// PingHost 通过系统 ping 检测目标主机是否在线
func PingHost(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("缺少 ip 参数"))
		return
	}

	if net.ParseIP(ip) == nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("ip 参数格式无效"))
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
	if !alive {
		outStr := strings.ToLower(string(output))
		if strings.Contains(outStr, "ttl=") || strings.Contains(outStr, "time=") || strings.Contains(outStr, "bytes from") || strings.Contains(outStr, "来自") {
			alive = true
		}
	}

	res := utils.NewSuccess()
	res.Data = gin.H{
		"alive": alive,
		"ip":    ip,
	}
	c.JSON(http.StatusOK, res)
}
