package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"log"
)

// 从环境变量读取AES密钥，如果未设置则使用默认值（仅用于开发）
var key = getAESKey()

func getAESKey() []byte {
	envKey := os.Getenv("SEARCH_GIN_AES_KEY")
	if envKey == "" {
		// 开发环境默认值，生产环境必须设置环境变量
		log.Println("警告: 未设置SEARCH_GIN_AES_KEY环境变量，使用默认密钥（仅用于开发环境）")
		return []byte("1234567890123456")
	}
	
	// 验证密钥长度（AES-128: 16, AES-192: 24, AES-256: 32）
	keyLen := len(envKey)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		log.Fatalf("无效的AES密钥长度: %d，必须是16、24或32字节", keyLen)
	}
	
	return []byte(envKey)
}

func Encrypt(text string) (string, error) {
	plaintext := []byte(text)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext string) (string, error) {
	text, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(text) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := text[:nonceSize], text[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
