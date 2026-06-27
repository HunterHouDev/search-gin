package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateStreamSecret_Returns64HexChars(t *testing.T) {
	secret := GenerateStreamSecret()
	assert.Len(t, secret, 64)
}

func TestSetStreamSecret_WrongLengthIgnored(t *testing.T) {
	SetStreamSecret("tooshort")
	assert.Nil(t, streamSecret)

	SetStreamSecret("")
	assert.Nil(t, streamSecret)
}

func TestSetStreamSecret_ValidKey(t *testing.T) {
	secret := GenerateStreamSecret()
	SetStreamSecret(secret)
	assert.NotNil(t, streamSecret)
	assert.Len(t, streamSecret, 32)
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	secret := GenerateStreamSecret()
	SetStreamSecret(secret)

	expire := time.Now().Add(1 * time.Hour).Unix()
	token, err := EncryptStreamToken(expire)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	decrypted, err := DecryptStreamToken(token)
	assert.NoError(t, err)
	assert.Equal(t, expire, decrypted)
}

func TestDecryptStreamToken_InvalidHex(t *testing.T) {
	_, err := DecryptStreamToken("not-hex!!")
	assert.Error(t, err)
}

func TestDecryptStreamToken_EmptyString(t *testing.T) {
	_, err := DecryptStreamToken("")
	assert.Error(t, err)
}

func TestDecryptStreamToken_WrongKey(t *testing.T) {
	secret := GenerateStreamSecret()
	SetStreamSecret(secret)

	expire := time.Now().Add(1 * time.Hour).Unix()
	token, err := EncryptStreamToken(expire)
	assert.NoError(t, err)

	// 换密钥后无法解密
	SetStreamSecret(GenerateStreamSecret())
	_, err = DecryptStreamToken(token)
	assert.Error(t, err)
}

func TestDecryptStreamToken_Expired(t *testing.T) {
	secret := GenerateStreamSecret()
	SetStreamSecret(secret)

	expire := time.Now().Add(-1 * time.Hour).Unix()
	token, err := EncryptStreamToken(expire)
	assert.NoError(t, err)

	decrypted, err := DecryptStreamToken(token)
	assert.NoError(t, err)
	assert.True(t, decrypted < time.Now().Unix(), "should be expired")
}

func TestStreamTokenTTL(t *testing.T) {
	assert.Equal(t, 4*time.Hour, StreamTokenTTL())
}
