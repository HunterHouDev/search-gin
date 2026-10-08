package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsDataImage 校验内联 base64 图片的识别
func TestIsDataImage(t *testing.T) {
	assert.True(t, IsDataImage("data:image/jpeg;base64,/9j/4AAQSkZJRg=="))
	assert.True(t, IsDataImage("data:image/png;base64,iVBORw0KGgo="))
	// 非 base64 的 data URI（URL 编码的 svg 之类）不支持
	assert.False(t, IsDataImage("data:image/svg+xml,%3Csvg%3E"))
	assert.False(t, IsDataImage("http://example.com/a.jpg"))
	assert.False(t, IsDataImage("/relative/a.jpg"))
	assert.False(t, IsDataImage(""))
}

// TestDecodeDataImage 校验 base64 图片的解码与错误处理
func TestDecodeDataImage(t *testing.T) {
	want := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10}
	got, err := DecodeDataImage("data:image/jpeg;base64,/9j/4AAQ")
	assert.NoError(t, err)
	assert.Equal(t, want, got)

	// 非法 base64
	_, err = DecodeDataImage("data:image/jpeg;base64,!!!!")
	assert.Error(t, err)
	// 缺少分隔符
	_, err = DecodeDataImage("data:image/jpeg,abcdef")
	assert.Error(t, err)
	// 内容为空
	_, err = DecodeDataImage("data:image/jpeg;base64,")
	assert.Error(t, err)
}

// TestHasImageSource 校验封面字段是否可落盘（http 链接或内联 base64）
func TestHasImageSource(t *testing.T) {
	assert.True(t, hasImageSource("https://a.com/1.jpg"))
	assert.True(t, hasImageSource("http://a.com/1.jpg"))
	assert.True(t, hasImageSource("data:image/jpeg;base64,AAAA"))
	// 相对路径要交给 BaseUrl 拼接，不在这里判断
	assert.False(t, hasImageSource("/p/1.jpg"))
	assert.False(t, hasImageSource(""))
}
