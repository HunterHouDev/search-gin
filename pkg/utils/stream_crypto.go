package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

// streamSecret 由 SetStreamSecret 在启动时设置，二进制中不含密钥
var streamSecret []byte

// GenerateStreamSecret 生成 32 字节随机密钥并返回 hex 编码（64 字符）
func GenerateStreamSecret() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("生成 StreamSecret 失败: " + err.Error())
	}
	return hex.EncodeToString(key)
}

// SetStreamSecret 从外部设置 AES-256-GCM 密钥（hex 格式，64 十六进制字符）
// 应于启动初始化时调用。hexKey 为 64 字符 hex 串（对应 32 字节），长度不匹配时静默忽略。
func SetStreamSecret(hexKey string) {
	if len(hexKey) != 64 {
		return
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return
	}
	streamSecret = key
}

// EncryptStreamToken 将过期时间加密为 streamToken
// 格式：hex(nonce + ciphertext)，AES-256-GCM
func EncryptStreamToken(expireUnix int64) (string, error) {
	plaintext := strconv.FormatInt(expireUnix, 10)
	block, err := aes.NewCipher(streamSecret)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptStreamToken 解密 streamToken，返回过期时间戳
func DecryptStreamToken(token string) (int64, error) {
	ciphertext, err := hex.DecodeString(token)
	if err != nil {
		return 0, errors.New("streamToken 格式错误")
	}
	block, err := aes.NewCipher(streamSecret)
	if err != nil {
		return 0, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return 0, errors.New("streamToken 长度不足")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, errors.New("streamToken 解密失败")
	}
	return strconv.ParseInt(strings.TrimSpace(string(plaintext)), 10, 64)
}

// StreamTokenTTL 返回 streamToken 的最大有效期
func StreamTokenTTL() time.Duration {
	return 4 * time.Hour
}
