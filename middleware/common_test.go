package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"search-gin/internal/service"

	"github.com/gin-gonic/gin"
)

// setupTestEngine 创建一个只挂 AuthMiddleware 的测试引擎，
// 所有路径通过一个 catch-all handler 返回 200
func setupTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	// catch-all handler 返回 200，用于验证中间件放行
	r.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// Test helper: 请求指定路径，检查被中间件拦截（401）
func assertBlockedByAuth(t *testing.T, r *gin.Engine, method, path string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("path %q should be blocked by auth, got %d", path, w.Code)
	}
}

// Test helper: 请求指定路径，检查不被中间件拦截（即中间件调用了 c.Next()）
func assertPassesAuth(t *testing.T, r *gin.Engine, method, path string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("path %q should be allowed without auth, got 401", path)
	}
}

// Test helper: 请求指定路径带 header，检查不被中间件拦截
func assertPassesAuthWithHeader(t *testing.T, r *gin.Engine, method, path, authHeader string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("path %q with auth header should be allowed, got 401", path)
	}
}

// Test helper: 请求指定路径带 header，检查被中间件拦截（401）
func assertBlockedByAuthWithHeader(t *testing.T, r *gin.Engine, method, path, authHeader string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("path %q with auth header should be blocked by auth, got %d", path, w.Code)
	}
}

func TestAuthMiddleware_SkipRootPath(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/")
}

func TestAuthMiddleware_SkipLogin(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "POST", "/api/login")
}

func TestAuthMiddleware_SkipWS(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/api/ws")
}

func TestAuthMiddleware_SkipEvents(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/api/events")
}

func TestAuthMiddleware_SkipHeartBeat(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/api/heartBeat")
}

func TestAuthMiddleware_SkipCSSPrefix(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/css/style.css")
}

func TestAuthMiddleware_SkipAssetsPrefix(t *testing.T) {
	assertPassesAuth(t, setupTestEngine(), "GET", "/assets/logo.png")
}

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	assertBlockedByAuth(t, setupTestEngine(), "GET", "/api/movieList")
}

func TestAuthMiddleware_ValidBearerToken_Returns200(t *testing.T) {
	service.SetToken("test-valid-token", time.Now().Add(1*time.Hour), "testuser", "admin", nil)
	assertPassesAuthWithHeader(t, setupTestEngine(), "GET", "/api/movieList", "Bearer test-valid-token")
}

func TestAuthMiddleware_QueryToken_Returns200(t *testing.T) {
	service.SetToken("test-query-token", time.Now().Add(1*time.Hour), "queryuser", "user", nil)
	assertPassesAuth(t, setupTestEngine(), "GET", "/api/movieList?token=test-query-token")
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	assertBlockedByAuthWithHeader(t, setupTestEngine(), "GET", "/api/movieList", "Bearer nonexistent-token")
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	service.SetToken("test-expired-token", time.Now().Add(-1*time.Hour), "expireduser", "user", nil)
	assertBlockedByAuthWithHeader(t, setupTestEngine(), "GET", "/api/movieList", "Bearer test-expired-token")
}
