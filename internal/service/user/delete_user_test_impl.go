package user

import (
	"context"
	"database/sql"
	"testing"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/stretchr/testify/require"
)

func SetupTestDatabase(t *testing.T) *sql.DB {
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

func TestDeleteUser(t *testing.T) {
	// 设置测试数据库
	db := SetupTestDatabase(t)
	defer db.Close()

	// 初始化 UserService
	userService, err := NewUserServiceWithDB(db)
	require.NoError(t, err)

	// 注册测试用户
	registerReq := &userapi.RegisterRequest{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
		Phone:    "1234567890",
	}
	registerResp, err := userService.Register(context.Background(), registerReq)
	require.NoError(t, err)
	require.True(t, registerResp.Success)
	userId := registerResp.UserId

	testCases := []struct {
		name           string
		deleteReq      *userapi.DeleteUserRequest
		expectedResult bool
		expectedError  string
	}{
		{
			name: "Valid Deletion",
			deleteReq: &userapi.DeleteUserRequest{
				UserId:   userId,
				Password: "password123",
			},
			expectedResult: true,
		},
		{
			name: "Wrong Password",
			deleteReq: &userapi.DeleteUserRequest{
				UserId:   userId,
				Password: "wrongpassword",
			},
			expectedResult: false,
			expectedError:  "密码错误",
		},
		{
			name: "User Not Found",
			deleteReq: &userapi.DeleteUserRequest{
				UserId:   9999, // 不存在的用户ID
				Password: "password123",
			},
			expectedResult: false,
			expectedError:  "用户不存在",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := userService.DeleteUser(context.Background(), tc.deleteReq)

			switch tc.name {
			case "Valid Deletion":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.Success)

				// 验证用户是否已删除
				query := "SELECT COUNT(*) FROM users WHERE id = ?"
				var count int
				err := db.QueryRow(query, userId).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 0, count)

			case "Wrong Password":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)
				require.Equal(t, tc.expectedError, resp.ErrorMessage)

			case "User Not Found":
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.False(t, resp.Success)
				require.Equal(t, tc.expectedError, resp.ErrorMessage)
			}
		})
	}
}

func TestDeleteUser_Success(t *testing.T) {
	// 设置测试数据库
	db := SetupTestDatabase(t)
	defer db.Close()

	// 初始化 UserService
	userService, err := NewUserServiceWithDB(db)
	require.NoError(t, err)

	// 注册测试用户
	registerReq := &userapi.RegisterRequest{
		Username: "deleteuser",
		Password: "password123",
		Email:    "delete@example.com",
		Phone:    "9876543210",
	}
	registerResp, err := userService.Register(context.Background(), registerReq)
	require.NoError(t, err)
	require.True(t, registerResp.Success)
	userId := registerResp.UserId

	// 创建关联的订单
	_, err = db.Exec("INSERT INTO orders (user_id, status) VALUES (?, ?)", userId, "pending")
	require.NoError(t, err)

	// 执行删除用户操作
	deleteReq := &userapi.DeleteUserRequest{
		UserId:   userId,
		Password: "password123",
	}
	resp, err := userService.DeleteUser(context.Background(), deleteReq)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)

	// 验证用户是否已删除
	query := "SELECT COUNT(*) FROM users WHERE id = ?"
	var count int
	err = db.QueryRow(query, userId).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// 验证关联的订单是否已删除
	query = "SELECT COUNT(*) FROM orders WHERE user_id = ?"
	err = db.QueryRow(query, userId).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestDeleteUser_UserNotFound(t *testing.T) {
	// 设置测试数据库
	db := SetupTestDatabase(t)
	defer db.Close()

	// 初始化 UserService
	userService, err := NewUserServiceWithDB(db)
	require.NoError(t, err)

	// 尝试删除不存在的用户
	deleteReq := &userapi.DeleteUserRequest{
		UserId:   9999, // 不存在的用户ID
		Password: "anypassword",
	}
	resp, err := userService.DeleteUser(context.Background(), deleteReq)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.Success)
	require.Equal(t, "用户不存在", resp.ErrorMessage)
}
