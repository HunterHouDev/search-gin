package handler

import (
	"net/http"

	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BindJSON 将请求体 JSON 绑定到泛型 T。
// 失败时自动写 400 响应（默认消息"参数绑定失败"，可选传入自定义消息），
// 调用方只需判断返回的 err，非 nil 时直接 return，无需重复编写响应代码。
// 出参：(T, error)
func BindJSON[T any](c *gin.Context, msg ...string) (T, error) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		m := "参数绑定失败"
		if len(msg) > 0 && msg[0] != "" {
			m = msg[0]
		}
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg(m))
		return req, err
	}
	return req, nil
}
