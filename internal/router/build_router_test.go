package router

import (
	"os"
	"testing"

	"search-gin/internal/env"

	"github.com/stretchr/testify/assert"
)

func TestBuildCORSConfig_DevDefault(t *testing.T) {
	wasProd := env.IsProd
	env.IsProd = false
	defer func() { env.IsProd = wasProd }()

	config := buildCORSConfig()
	assert.Equal(t, []string{"*"}, config.AllowOrigins)
}

func TestBuildCORSConfig_ProdNoEnv(t *testing.T) {
	wasProd := env.IsProd
	env.IsProd = true
	defer func() { env.IsProd = wasProd }()

	os.Unsetenv("ALLOWED_ORIGINS")
	config := buildCORSConfig()
	assert.Equal(t, []string{"*"}, config.AllowOrigins)
}

func TestBuildCORSConfig_ProdWithEnv(t *testing.T) {
	wasProd := env.IsProd
	env.IsProd = true
	defer func() { env.IsProd = wasProd }()

	os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	config := buildCORSConfig()
	assert.Equal(t, []string{"https://example.com", "https://app.example.com"}, config.AllowOrigins)
}

func TestBuildCORSConfig_Headers(t *testing.T) {
	wasProd := env.IsProd
	env.IsProd = false
	defer func() { env.IsProd = wasProd }()

	config := buildCORSConfig()
	assert.Contains(t, config.AllowHeaders, "Origin")
	assert.Contains(t, config.AllowHeaders, "Content-Type")
	assert.Contains(t, config.ExposeHeaders, "Content-Length")
	assert.Contains(t, config.ExposeHeaders, "Content-Range")
}
