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

// 固定密钥，所有节点共用，streamToken 可跨节点解密
var streamSecret = []byte{
	0x2b, 0x7e, 0x15, 0x16, 0x28, 0xae, 0xd2, 0xa6,
	0xab, 0xf7, 0x15, 0x88, 0x09, 0xcf, 0x4f, 0x3c,
	0x1a, 0xb3, 0x7c, 0x8d, 0x2e, 0xf4, 0x61, 0x99,
	0xdd, 0x22, 0xaa, 0x3b, 0x5f, 0x10, 0xcd, 0x77,
}

// SetStreamSecret 从外部设置 AES-256-GCM 密钥（hex 格式，64 十六进制字符）
// 未调用时使用代码内建固定密钥。应于启动初始化时调用。
// hexKey 为 64 字符 hex 串（对应 32 字节），长度不匹配时静默忽略。
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
