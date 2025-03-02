package user

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	_ "github.com/mattn/go-sqlite3"

	// "github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

func setupSqliteTestDatabase(t *testing.T) *sql.DB {
	// 创建测试数据库连接
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to open test database")

	// 创建用户表
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			phone TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			two_factor_enabled INTEGER DEFAULT 0,
			two_factor_secret TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE user_backup_codes (
			user_id INTEGER,
			backup_code TEXT UNIQUE NOT NULL,
			used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);

		CREATE TABLE two_factor_backup_codes (
			user_id INTEGER,
			backup_code TEXT UNIQUE NOT NULL,
			used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);

		-- 创建角色表
		CREATE TABLE roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- 创建权限表
		CREATE TABLE permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- 创建角色-权限关联表
		CREATE TABLE role_permissions (
			role_id INTEGER,
			permission_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (permission_id) REFERENCES permissions(id)
		);

		-- 创建用户-角色关联表
		CREATE TABLE user_roles (
			user_id INTEGER,
			role_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
		);

		-- 创建权限缓存表
		CREATE TABLE permission_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			permissions TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- 添加默认角色
		INSERT INTO roles (name, description) VALUES
		('admin', '系统管理员'),
		('user', '普通用户'),
		('guest', '访客用户');

		-- 添加基础权限
		INSERT INTO permissions (name, description, resource, action) VALUES
		('user:read', '查看用户信息', 'user', 'read'),
		('user:write', '修改用户信息', 'user', 'write'),
		('user:delete', '删除用户', 'user', 'delete');

		-- 为管理员角色分配所有权限
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT 
			(SELECT id FROM roles WHERE name = 'admin'),
			id
		FROM permissions;

		-- 为普通用户分配基本权限
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT 
			(SELECT id FROM roles WHERE name = 'user'),
			id
		FROM permissions
		WHERE name IN ('user:read', 'user:write');
	`)
	require.NoError(t, err)

	return db
}

func setupUserService(t *testing.T) *UserService {
	testDB := setupSqliteTestDatabase(t)

	userService, err := NewUserService(
		WithTestDatabase(testDB),
	)
	require.NoError(t, err, "Failed to create UserService")

	t.Cleanup(func() {
		userService.Close()
	})
	return userService
}

func registerTestUser(t *testing.T, userService *UserService) (string, string, int32) {
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@example.com", username)
	phone := fmt.Sprintf("138%d", time.Now().UnixNano()%10000000000)

	// 注册用户
	registerResp, err := userService.Register(context.Background(), &userapi.RegisterRequest{
		Username: username,
		Password: "StrongPass123!",
		Email:    email,
		Phone:    phone,
	})
	require.NoError(t, err)
	assert.True(t, registerResp.Success)
	assert.NotZero(t, registerResp.UserId)

	// 为新注册的用户分配普通用户角色
	err = userService.rbacManager.AssignRole(context.Background(), registerResp.UserId, "user")
	require.NoError(t, err, "Failed to assign role to user")

	return username, "StrongPass123!", registerResp.UserId
}

func TestUserRegistrationAndLogin(t *testing.T) {
	userService := setupUserService(t)
	username, password, userId := registerTestUser(t, userService)

	// 登录
	loginResp, err := userService.Login(context.Background(), &userapi.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.Token)
	assert.Equal(t, userId, loginResp.UserId)

	// 获取用户信息
	userInfoResp, err := userService.GetUserInfo(context.Background(), &userapi.GetUserInfoRequest{
		UserId: userId,
	})
	require.NoError(t, err)
	assert.Equal(t, username, userInfoResp.Username)
}

func TestUserInfoUpdate(t *testing.T) {
	userService := setupUserService(t)
	_, _, userId := registerTestUser(t, userService)

	newEmail := fmt.Sprintf("new_email_%d@example.com", time.Now().UnixNano())
	newPhone := fmt.Sprintf("139%d", time.Now().UnixNano()%10000000000)

	// 更新用户信息
	updateResp, err := userService.UpdateUserInfo(context.Background(), &userapi.UpdateUserInfoRequest{
		UserId: userId,
		Email:  newEmail,
		Phone:  newPhone,
	})
	require.NoError(t, err)
	assert.True(t, updateResp.Success)

	// 获取更新后的用户信息
	userInfoResp, err := userService.GetUserInfo(context.Background(), &userapi.GetUserInfoRequest{
		UserId: userId,
	})
	require.NoError(t, err)
	assert.Equal(t, newEmail, userInfoResp.Email)
	assert.Equal(t, newPhone, userInfoResp.Phone)
}

func TestMain(m *testing.M) {
	// 设置测试环境变量
	os.Setenv("GO_TEST_ENV", "true")
	defer os.Unsetenv("GO_TEST_ENV")

	// 运行测试
	code := m.Run()

	os.Exit(code)
}
