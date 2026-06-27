package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleRemote 判断 movie 是否属远程节点，若是则转发请求
// 返回 true 表示已转发并写了响应，调用方应 return
// 返回 false 表示是本机文件，继续原逻辑
func HandleRemote(c *gin.Context, movie model.FileItem, action string) bool {
	if movie.NodeHost == "" || movie.NodeHost == LocalNodeHost {
		return false
	}

	peerIP := ResolvePeerIP(movie.NodeHost)
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

// HandleRemoteByID 根据 id 查找 Movie，若远程则转发
func HandleRemoteByID(c *gin.Context, id string, action string) bool {
	movie := GetEngine().FindById(id)
	return HandleRemote(c, movie, action)
}

// HandleRemoteByMovieEdit 从 MovieEdit 提取 id 查找 Movie，若远程则转发
func HandleRemoteByMovieEdit(c *gin.Context, edit model.FileEdit, action string) bool {
	movie := GetEngine().FindById(edit.Id)
	return HandleRemote(c, movie, action)
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
		if k != "Host" && k != "Content-Length" && k != "Transfer-Encoding" {
			req.Header[k] = v
		}
	}

	// 转发认证 token
	if token := c.GetHeader("Authorization"); token != "" {
		req.Header.Set("Authorization", token)
	}

	// 设置正确的 Content-Length
	req.ContentLength = int64(len(bodyBytes))

	req.URL.RawQuery = c.Request.URL.RawQuery

	return peerClient.Do(req)
}
