package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"gorm.io/gorm"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/bytedance-youthcamp/demo/internal/config"
	"github.com/bytedance-youthcamp/demo/internal/service/auth"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type UserServiceOption func(*UserService) error

func WithTestDatabase(db *sql.DB) UserServiceOption {
	return func(us *UserService) error {
		us.db = db
		return nil
	}
}

func WithGormDatabase(db *gorm.DB) UserServiceOption {
	return func(us *UserService) error {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}
		us.db = sqlDB
		return nil
	}
}

func NewUserService(opts ...UserServiceOption) (*UserService, error) {
	// 加载用户配置
	configPath := "/Users/Apple/Desktop/demo/configs/user.yaml"
	userConfig, err := config.LoadUserConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %v", err)
	}

	// 创建 UserService 实例
	service := &UserService{
		config:      userConfig,
		authService: nil, // 需要初始化 authService
	}

	// 处理数据库连接选项
	if len(opts) > 0 {
		for _, opt := range opts {
			if err := opt(service); err != nil {
				return nil, fmt.Errorf("failed to apply service option: %w", err)
			}
		}
	}

	// 如果没有通过选项设置数据库，使用默认连接
	if service.db == nil {
		// 打开数据库连接
		dbURL := "postgres://localhost:5432/demo?sslmode=disable"
		driver := "postgres"

		// 如果是测试环境，使用SQLite
		if os.Getenv("GO_TEST_ENV") == "true" {
			dbURL = ":memory:"
			driver = "sqlite3"
		} else if os.Getenv("TEST_DB_URL") != "" {
			dbURL = os.Getenv("TEST_DB_URL")
		}

		db, err := sql.Open(driver, dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %v", err)
		}
		service.db = db
	}

	// 如果是测试环境，创建表
	if os.Getenv("GO_TEST_ENV") == "true" {
		err = createTestTables(service.db)
		if err != nil {
			service.db.Close()
			return nil, fmt.Errorf("failed to create test tables: %v", err)
		}
	}

	// 初始化 AuthService
	authService, err := auth.NewAuthService()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth service: %w", err)
	}
	service.authService = authService

	// 初始化 RBACManager
	service.rbacManager = NewRBACManager(service.db)

	return service, nil
}

func NewUserServiceWithDB(db *sql.DB) (*UserService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	// 加载用户配置
	configPath := "/Users/Apple/Desktop/demo/configs/user.yaml"
	userConfig, err := config.LoadUserConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %v", err)
	}

	// 初始化 AuthService
	authService, err := auth.NewAuthService()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth service: %w", err)
	}

	userService := &UserService{
		db:          db,
		config:      userConfig,
		authService: authService,
	}

	return userService, nil
}

func createTestTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			phone TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			two_factor_enabled INTEGER DEFAULT 0,
			two_factor_secret TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_backup_codes (
			user_id INTEGER,
			backup_code TEXT UNIQUE NOT NULL,
			used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS two_factor_backup_codes (
			user_id INTEGER,
			backup_code TEXT UNIQUE NOT NULL,
			used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
	`)
	return err
}

type UserService struct {
	db          *sql.DB
	config      *config.UserConfig
	authService *auth.AuthService
	rbacManager *RBACManager
}

func (s *UserService) Close() {
	if s.db != nil {
		// 检查是否为测试环境
		if os.Getenv("GO_TEST_ENV") != "true" {
			if err := s.db.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}
	}
}

func (s *UserService) Register(ctx context.Context, req *userapi.RegisterRequest) (*userapi.RegisterResponse, error) {
	// 验证用户名和密码
	if err := s.validateRegistration(req); err != nil {
		return &userapi.RegisterResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 对密码进行哈希处理
	hashedPassword := s.hashPassword(req.Password)

	// 插入用户到数据库
	var userID int32
	query := `
		INSERT INTO users (username, password_hash, email, phone, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		req.Username,
		hashedPassword,
		req.Email,
		req.Phone,
		time.Now(),
	)
	if err != nil {
		// 处理可能的唯一性约束冲突
		if err.Error() == "Error 1062 (23000): Duplicate entry" {
			return &userapi.RegisterResponse{
				Success:      false,
				ErrorMessage: "用户名或邮箱已存在",
			}, nil
		}
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	// 获取最后插入的 ID
	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	userID = int32(lastID)

	return &userapi.RegisterResponse{
		UserId:  userID,
		Success: true,
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *userapi.LoginRequest) (*userapi.LoginResponse, error) {
	// 查询用户
	query := `
		SELECT id, password_hash, two_factor_enabled, COALESCE(two_factor_secret, '') 
		FROM users 
		WHERE username = ?
	`

	var (
		userID           int32
		passwordHash     string
		twoFactorEnabled bool
		twoFactorSecret  string
	)

	err := s.db.QueryRowContext(ctx, query, req.Username).Scan(
		&userID,
		&passwordHash,
		&twoFactorEnabled,
		&twoFactorSecret,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &userapi.LoginResponse{
				Success:      false,
				ErrorMessage: "用户不存在",
			}, nil
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return &userapi.LoginResponse{
			Success:      false,
			ErrorMessage: "密码错误",
		}, nil
	}

	// 检查两因素认证
	if twoFactorEnabled {
		// 如果启用了两因素认证，但未提供验证码
		if req.TwoFactorCode == "" {
			return &userapi.LoginResponse{
				Success:      false,
				ErrorMessage: "需要两因素认证码",
			}, nil
		}

		// 验证两因素认证码
		valid, err := s.validateTwoFactorCode(ctx, userID, req.TwoFactorCode)
		if err != nil {
			return nil, fmt.Errorf("两因素认证验证失败: %w", err)
		}
		if !valid {
			return &userapi.LoginResponse{
				Success:      false,
				ErrorMessage: "两因素认证码无效",
			}, nil
		}
	}

	// 生成认证令牌
	tokenResp, err := s.authService.DeliverTokenByRPC(ctx, &auth.DeliverTokenReq{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &userapi.LoginResponse{
		UserId:  userID,
		Token:   tokenResp.Token,
		Success: true,
	}, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, req *userapi.GetUserInfoRequest) (*userapi.GetUserInfoResponse, error) {
	query := `
		SELECT username, email, phone 
		FROM users 
		WHERE id = ?
	`

	var (
		username string
		email    string
		phone    string
	)

	err := s.db.QueryRowContext(ctx, query, req.UserId).Scan(
		&username,
		&email,
		&phone,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &userapi.GetUserInfoResponse{
		Username: username,
		Email:    email,
		Phone:    phone,
	}, nil
}

func (s *UserService) UpdateUserInfo(ctx context.Context, req *userapi.UpdateUserInfoRequest) (*userapi.UpdateUserInfoResponse, error) {
	// 检查用户是否有修改用户信息的权限
	hasPermission, err := s.rbacManager.HasPermission(ctx, req.UserId, "user", "write")
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return &userapi.UpdateUserInfoResponse{
			Success:      false,
			ErrorMessage: "权限不足：用户没有修改权限",
		}, nil
	}

	query := `
		UPDATE users 
		SET email = ?, phone = ? 
		WHERE id = ?
	`
	_, err = s.db.ExecContext(ctx, query, req.Email, req.Phone, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to update user info: %w", err)
	}

	return &userapi.UpdateUserInfoResponse{
		Success: true,
	}, nil
}

func (s *UserService) validateRegistration(req *userapi.RegisterRequest) error {
	// 验证用户名
	if len(req.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}

	// 验证密码
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// 验证密码复杂度
	if s.config.Security.PasswordComplexityEnabled {
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(req.Password)
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(req.Password)
		hasDigit := regexp.MustCompile(`\d`).MatchString(req.Password)
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(req.Password)

		if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
			return fmt.Errorf("password must contain at least one lowercase letter, one uppercase letter, one digit, and one special character")
		}
	}

	// 验证邮箱
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func (s *UserService) hashPassword(password string) string {
	// 使用 bcrypt 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// 处理哈希错误，通常在生产环境中会记录日志
		return ""
	}
	return string(hashedPassword)
}

func (s *UserService) Logout(ctx context.Context, req *userapi.LogoutRequest) (*userapi.LogoutResponse, error) {
	if req.Token == "" {
		return &userapi.LogoutResponse{
			Success:      false,
			ErrorMessage: "token is required",
		}, nil
	}

	// 验证令牌
	verifyResp, err := s.authService.VerifyTokenByRPC(ctx, &auth.VerifyTokenReq{Token: req.Token})
	if err != nil || !verifyResp.Res {
		return &userapi.LogoutResponse{
			Success:      false,
			ErrorMessage: "invalid token",
		}, nil
	}

	// 将令牌加入黑名单
	s.authService.BlacklistToken(req.Token)

	return &userapi.LogoutResponse{
		Success: true,
	}, nil
}
