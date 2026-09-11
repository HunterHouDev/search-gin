package service

import (
	"testing"
	"time"

	"search-gin/internal/model"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	hash := HashPassword("test123")
	if hash == "" {
		t.Fatal("HashPassword returned empty")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("test123")); err != nil {
		t.Error("Hash/verify mismatch")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash := HashPassword("mypass")
	if !VerifyPassword("mypass", hash) {
		t.Error("VerifyPassword should return true for correct password")
	}
	if VerifyPassword("wrong", hash) {
		t.Error("VerifyPassword should return false for wrong password")
	}
}

func TestSetAndValidateToken(t *testing.T) {
	now := time.Now()
	expire := now.Add(1 * time.Hour)

	SetToken("tok_abc", expire, "testuser", "admin", nil)
	info, ok := ValidateTokenWithInfo("tok_abc")
	if !ok {
		t.Fatal("ValidateTokenWithInfo should return true for valid token")
	}
	if info.Username != "testuser" {
		t.Errorf("Username = %q, want %q", info.Username, "testuser")
	}
	if info.Role != "admin" {
		t.Errorf("Role = %q, want %q", info.Role, "admin")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	expired := time.Now().Add(-1 * time.Hour)
	SetToken("tok_expired", expired, "user", "role", nil)

	_, ok := ValidateTokenWithInfo("tok_expired")
	if ok {
		t.Error("ValidateTokenWithInfo should return false for expired token")
	}
}

func TestValidateToken_NotFound(t *testing.T) {
	_, ok := ValidateTokenWithInfo("tok_nonexist")
	if ok {
		t.Error("ValidateTokenWithInfo should return false for non-existent token")
	}
}

func TestCacheAdminPasswordHash(t *testing.T) {
	// 保存旧设置，测试后恢复
	old := GetOSSetting()
	defer SetOSSetting(old)

	SetOSSetting(model.Setting{AdminPassword: "testpass"})
	CacheAdminPasswordHash()
	// 验证缓存生效：登录应成功
	result := LoginUser("admin", "testpass")
	if !result.Success {
		t.Error("Login should succeed with correct password after CacheAdminPasswordHash")
	}
}

func TestRevokeToken(t *testing.T) {
	SetToken("tok_revoke", time.Now().Add(1*time.Hour), "testuser", "admin", nil)
	if _, ok := ValidateTokenWithInfo("tok_revoke"); !ok {
		t.Fatal("吊销前 token 应有效")
	}

	RevokeToken("tok_revoke")
	if _, ok := ValidateTokenWithInfo("tok_revoke"); ok {
		t.Error("吊销后 token 应立即失效")
	}
}

func TestRevokeToken_Idempotent(t *testing.T) {
	// 重复吊销与空 token 都不应 panic
	RevokeToken("tok_not_exist")
	RevokeToken("")
	RevokeToken("tok_not_exist")
}

func TestRequireAdminWithName(t *testing.T) {
	if !RequireAdminWithName(AdminRole, AdminUsername) {
		t.Error("RequireAdminWithName should return true for admin/super_admin")
	}
	if RequireAdminWithName("user", "someone") {
		t.Error("RequireAdminWithName should return false for normal user")
	}
}
