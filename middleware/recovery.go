package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"search-gin/pkg/utils"
)

func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				utils.ErrorNormal("请求处理发生异常:", err)
				utils.ErrorNormal("请求路径:", c.Request.URL.Path)
				utils.ErrorNormal("请求方法:", c.Request.Method)
				utils.ErrorNormal("堆栈信息:", string(debug.Stack()))
				c.JSON(500, gin.H{
					"error": "服务器内部错误",
					"msg":   "系统发生异常，请查看日志获取详细信息",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
