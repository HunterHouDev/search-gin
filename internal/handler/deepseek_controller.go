package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"search-gin/internal/service"
	"search-gin/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ChatRequest 转发给 DeepSeek 的请求体（仅取需要的字段）
type ChatRequest struct {
	Messages []map[string]string `json:"messages"`
	Model    string              `json:"model"`
}

// PostChatDeepSeek 代理 DeepSeek Chat API，密钥仅存后端
func PostChatDeepSeek(c *gin.Context) {
	setting := service.GetOSSetting()
	apiKey := setting.DeepSeekApiKey
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("未配置 DeepSeek API Key"))
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.NewFailByMsg("参数绑定失败"))
		return
	}

	if req.Model == "" {
		req.Model = "deepseek-chat"
	}

	body := map[string]interface{}{
		"messages": req.Messages,
		"model":    req.Model,
	}
	bodyBytes, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions",
		strings.NewReader(string(bodyBytes)))
	if err != nil {
		utils.ErrorFormat("创建 DeepSeek 请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("请求失败"))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	deepSeekClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := deepSeekClient.Do(httpReq)
	if err != nil {
		utils.ErrorFormat("DeepSeek API 调用失败: %v", err)
		c.JSON(http.StatusBadGateway, utils.NewFailByMsg("调用 DeepSeek API 失败"))
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.ErrorFormat("读取 DeepSeek 响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("读取响应失败"))
		return
	}

	// 解析响应提取 content
	var deepSeekResp map[string]interface{}
	if err := json.Unmarshal(respBytes, &deepSeekResp); err != nil {
		c.JSON(http.StatusInternalServerError, utils.NewFailByMsg("解析响应失败"))
		return
	}

	// DeepSeek 返回格式同 OpenAI：choices[0].message.content
	choices, ok := deepSeekResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		c.Data(resp.StatusCode, "application/json; charset=utf-8", respBytes)
		return
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		c.Data(resp.StatusCode, "application/json; charset=utf-8", respBytes)
		return
	}
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		c.Data(resp.StatusCode, "application/json; charset=utf-8", respBytes)
		return
	}
	content, ok := message["content"].(string)
	if !ok {
		content = ""
	}

	result := utils.NewSuccess()
	result.Data = content
	c.JSON(http.StatusOK, result)
}
