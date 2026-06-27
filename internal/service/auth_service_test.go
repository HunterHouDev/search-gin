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

func TestRequireAdmin(t *testing.T) {
	if !RequireAdmin(AdminRole) {
		t.Error("RequireAdmin should return true for super_admin")
	}
	if RequireAdmin("user") {
		t.Error("RequireAdmin should return false for user role")
	}
}

func TestRequireAdminWithName(t *testing.T) {
	if !RequireAdminWithName(AdminRole, AdminUsername) {
		t.Error("RequireAdminWithName should return true for admin/super_admin")
	}
	if RequireAdminWithName("user", "someone") {
		t.Error("RequireAdminWithName should return false for normal user")
	}
}
