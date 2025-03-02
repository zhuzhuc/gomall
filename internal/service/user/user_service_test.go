package user

import (
	"context"
	"database/sql"
	"testing"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	// userapi2 "github.com/bytedance-youthcamp/demo/api/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
)

func setupTestDatabase(t *testing.T) *sql.DB {
	// MySQL 连接配置
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/testdb?parseTime=true")
	require.NoError(t, err)

	// 删除现有表
	_, err = db.Exec(`DROP TABLE IF EXISTS two_factor_backup_codes`)
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE IF EXISTS user_backup_codes`)
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE IF EXISTS payments`)
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE IF EXISTS orders`)
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE IF EXISTS users`)
	require.NoError(t, err)

	// 创建测试表
	_, err = db.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			phone VARCHAR(20) UNIQUE NOT NULL,
			two_factor_enabled BOOLEAN DEFAULT FALSE,
			two_factor_token VARCHAR(255),
			two_factor_secret VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE orders (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			status VARCHAR(50) NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE payments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			order_id INT NOT NULL,
			status VARCHAR(50) NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE user_backup_codes (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			backup_code VARCHAR(255) UNIQUE NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE two_factor_backup_codes (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			backup_code VARCHAR(255) UNIQUE NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	return db
}

func registerUser(t *testing.T, userService *UserService, username, password, email, phone string) int32 {
	registerReq := &userapi.RegisterRequest{
		Username: username,
		Password: password,
		Email:    email,
		Phone:    phone,
	}
	resp, err := userService.Register(context.Background(), registerReq)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	return resp.UserId
}

func TestRegisterUser(t *testing.T) {
	testCases := []struct {
		name           string
		registerReq    *userapi.RegisterRequest
		expectedResult bool
	}{
		{
			name: "Valid Registration",
			registerReq: &userapi.RegisterRequest{
				Username: "testuser",
				Password: "StrongPass123!",
				Email:    "test@example.com",
				Phone:    "13800138000",
			},
			expectedResult: true,
		},
		{
			name: "Short Username",
			registerReq: &userapi.RegisterRequest{
				Username: "a",
				Password: "StrongPass123!",
				Email:    "short@example.com",
				Phone:    "13800138001",
			},
			expectedResult: false,
		},
		{
			name: "Weak Password",
			registerReq: &userapi.RegisterRequest{
				Username: "weakpassuser",
				Password: "weak",
				Email:    "weak@example.com",
				Phone:    "13800138002",
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userService, err := NewUserService()
			require.NoError(t, err)
			defer userService.Close()

			resp, err := userService.Register(context.Background(), tc.registerReq)

			switch tc.name {
			case "Valid Registration":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.Success)
				require.NotZero(t, resp.UserId)

			case "Short Username":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)

			case "Weak Password":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	// 设置测试数据库
	db := setupTestDatabase(t)
	defer db.Close()

	// 初始化 UserService
	userService, err := NewUserServiceWithDB(db)
	require.NoError(t, err)

	// 注册测试用户
	registerReq := &userapi.RegisterRequest{
		Username: "loginuser",
		Password: "StrongPass123!",
		Email:    "login@example.com",
		Phone:    "13800138000",
	}
	registerResp, err := userService.Register(context.Background(), registerReq)
	require.NoError(t, err)
	require.True(t, registerResp.Success)

	testCases := []struct {
		name           string
		loginReq       *userapi.LoginRequest
		expectedResult bool
		expectedError  string
	}{
		{
			name: "Valid Login",
			loginReq: &userapi.LoginRequest{
				Username: "loginuser",
				Password: "StrongPass123!",
			},
			expectedResult: true,
		},
		{
			name: "Invalid Password",
			loginReq: &userapi.LoginRequest{
				Username: "loginuser",
				Password: "WrongPassword123!",
			},
			expectedResult: false,
			expectedError:  "密码错误",
		},
		{
			name: "Non-existent User",
			loginReq: &userapi.LoginRequest{
				Username: "nonexistentuser",
				Password: "AnyPassword123!",
			},
			expectedResult: false,
			expectedError:  "用户不存在",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := userService.Login(context.Background(), tc.loginReq)

			switch tc.name {
			case "Valid Login":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.Success)
				require.NotEmpty(t, resp.Token)

			case "Invalid Password":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)
				require.Equal(t, tc.expectedError, resp.ErrorMessage)

			case "Non-existent User":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)
				require.Equal(t, tc.expectedError, resp.ErrorMessage)
			}
		})
	}
}
