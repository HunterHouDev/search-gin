package handler

import (
	"bytes"
	"io"
	"net/http"

	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BindJSON 将请求体 JSON 绑定到泛型 T。
// 失败时自动写 400 响应（默认消息"参数绑定失败"，可选传入自定义消息），
// 调用方只需判断返回的 err，非 nil 时直接 return，无需重复编写响应代码。
// 出参：(T, error)
//
// 内部先调用 c.GetRawData() 缓存请求体，再执行 ShouldBindJSON。
// 这样后续 remote_operation.forwardRequest 通过 c.GetRawData() 仍能读取到完整 body，
// 确保远程节点转发不会因 body 已被消费而丢失数据。
func BindJSON[T any](c *gin.Context, msg ...string) (T, error) {
	// 先缓存 body，避免后续 forwardRequest 读取不到
	bodyBytes, _ := c.GetRawData()
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
