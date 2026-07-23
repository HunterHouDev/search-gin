package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleRemote 判断 host 是否属远程节点，若是则转发请求
// host 为文件归属节点地址（host:port），空或本机地址（LocalNodeHost）表示本机文件。
// 返回 true 表示已转发并写了响应，调用方应 return；
// 返回 false 表示是本机文件，继续原逻辑。
// 转发时沿用原始请求方法（POST）与请求体，目标节点用 body 内的 Host 自行判定归属，
// 因此无需本机索引中存在该文件即可正确转发。
func HandleRemote(c *gin.Context, host string, action string) bool {
	if host == "" || host == LocalNodeHost {
		return false
	}

	peerIP := ResolvePeerIP(host)
	if peerIP == "" {
		c.JSON(http.StatusBadGateway, utils.NewFailByMsg("远程节点离线"))
		return true
	}

	apiPort := strings.TrimPrefix(PortNo, ":")
	targetURL := fmt.Sprintf("http://%s:%s%s", peerIP, apiPort, c.Request.URL.Path)

	resp, err := forwardRequest(targetURL, c)
	if err != nil {
		utils.ErrorFormat("远程操作失败 [%s]: %v", action, err)
		c.JSON(http.StatusBadGateway, utils.NewFailByMsg("远程操作失败"))
		return true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20)) // 100MB 上限
	if err != nil {
		utils.ErrorFormat("读取远程响应失败 [%s]: %v", action, err)
		c.JSON(http.StatusBadGateway, utils.NewFailByMsg("读取远程响应失败"))
		return true
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	return true
}

// forwardRequest 转发 HTTP 请求到目标节点
//
// 使用 c.GetRawData() 替代 io.ReadAll(c.Request.Body)，Gin 内部会缓存 body，
// 无论是否有其他中间件提前读取，都能获取完整 body 内容。
func forwardRequest(targetURL string, c *gin.Context) (*http.Response, error) {
	bodyBytes, err := c.GetRawData()
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	for k, v := range c.Request.Header {
		if k != "Host" && k != "Content-Length" && k != "Transfer-Encoding" && k != "Authorization" {
			req.Header[k] = v
		}
	}

	// 节点间认证使用 X-Search-Gin-Remote header，不转发用户的 Bearer token
	req.Header.Set("X-Search-Gin-Remote", "true")

	// 设置正确的 Content-Length
	req.ContentLength = int64(len(bodyBytes))

	req.URL.RawQuery = c.Request.URL.RawQuery

	return peerClient.Do(req)
}
