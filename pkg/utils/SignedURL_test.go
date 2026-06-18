package utils

import (
	"net/url"
	"testing"
	"time"
)

func TestSignURL_VerifySuccess(t *testing.T) {
	InitSignSecret([]byte("test-secret-key-for-unit-test-32b"))
	defer func() { signSecret = make([]byte, 32) }()

	base := "http://192.168.1.100:10082"
	path := "/api/stream/png/abc123"
	signed := SignURL(base, path, 4*time.Hour)

	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("解析签名 URL 失败: %v", err)
	}

	if u.Path != path {
		t.Errorf("路径不匹配: got %s, want %s", u.Path, path)
	}

	if !VerifySignedRequest(u.Path, u.Query()) {
		t.Error("签名验证应该通过")
	}
}

func TestSignURL_EncodedPath(t *testing.T) {
	InitSignSecret([]byte("test-secret-key-for-unit-test-32b"))
	defer func() { signSecret = make([]byte, 32) }()

	base := "http://192.168.1.100:10082"
	path := "/api/stream/GetFileByPathUseEncode/C%3A%5Ctest%5Cfile.mp4"
	signed := SignURL(base, path, 4*time.Hour)

	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("解析签名 URL 失败: %v", err)
	}

	rawPath := u.RawPath
	if rawPath == "" {
		rawPath = u.Path
	}

	if !VerifySignedRequest(rawPath, u.Query()) {
		t.Errorf("编码路径签名验证失败: rawPath=%s", rawPath)
	}
}

func TestSignURL_Expired(t *testing.T) {
	InitSignSecret([]byte("test-secret-key-for-unit-test-32b"))
	defer func() { signSecret = make([]byte, 32) }()

	base := "http://192.168.1.100:10082"
	path := "/api/stream/png/abc123"
	signed := SignURL(base, path, -1*time.Second)

	u, _ := url.Parse(signed)

	if VerifySignedRequest(u.Path, u.Query()) {
		t.Error("过期签名应该验证失败")
	}
}

func TestSignURL_MissingParams(t *testing.T) {
	if VerifySignedRequest("/api/stream/png/abc123", url.Values{}) {
		t.Error("缺少签名参数应该验证失败")
	}
}

func TestSignURL_TamperedSign(t *testing.T) {
	InitSignSecret([]byte("test-secret-key-for-unit-test-32b"))
	defer func() { signSecret = make([]byte, 32) }()

	base := "http://192.168.1.100:10082"
	path := "/api/stream/png/abc123"
	signed := SignURL(base, path, 4*time.Hour)

	u, _ := url.Parse(signed)
	q := u.Query()
	q.Set("sign", "tamperedvalue")

	if VerifySignedRequest(u.Path, q) {
		t.Error("篡改签名应该验证失败")
	}
}
