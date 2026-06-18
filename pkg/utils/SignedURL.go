package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

var signSecret []byte

func init() {
	signSecret = make([]byte, 32)
	if _, err := rand.Read(signSecret); err != nil {
		panic("生成签名密钥失败: " + err.Error())
	}
}

// InitSignSecret 使用外部密钥初始化（仅测试用）
func InitSignSecret(secret []byte) {
	signSecret = secret
}

// SignURL 为 URL 路径生成签名，附加 sign 和 expire 查询参数
// path: URL 路径（如 /api/stream/png/abc123）
// ttl: 签名有效期
// 返回带签名参数的完整 URL 字符串
func SignURL(baseURL, path string, ttl time.Duration) string {
	expire := time.Now().Add(ttl).Unix()
	signature := computeSignature(path, expire)
	params := url.Values{}
	params.Set("sign", signature)
	params.Set("expire", strconv.FormatInt(expire, 10))
	sep := "?"
	if containsQuery(baseURL+path) {
		sep = "&"
	}
	return baseURL + path + sep + params.Encode()
}

// VerifySignedRequest 验证请求的签名和过期时间
// path: 请求路径
// query: 请求查询参数
// 返回验证是否通过
func VerifySignedRequest(path string, query url.Values) bool {
	sign := query.Get("sign")
	expireStr := query.Get("expire")
	if sign == "" || expireStr == "" {
		return false
	}

	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return false
	}

	if time.Now().Unix() > expire {
		return false
	}

	expected := computeSignature(path, expire)
	return hmac.Equal([]byte(sign), []byte(expected))
}

// computeSignature 使用 HMAC-SHA256 对路径+过期时间计算签名
func computeSignature(path string, expire int64) string {
	message := path + "|" + strconv.FormatInt(expire, 10)
	mac := hmac.New(sha256.New, signSecret)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func containsQuery(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '?' {
			return true
		}
	}
	return false
}
