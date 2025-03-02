package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/bytedance-youthcamp/demo/internal/config"
	userService "github.com/bytedance-youthcamp/demo/internal/service/user"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Define the userServiceServer struct with explicit method implementations
type userServiceServer struct {
	userapi.UnimplementedUserServiceServer
	UserService *userService.UserService
}

// Explicitly implement Login to resolve ambiguity
func (s *userServiceServer) Login(ctx context.Context, req *userapi.LoginRequest) (*userapi.LoginResponse, error) {
	return s.UserService.Login(ctx, req)
}

// Explicitly implement DeleteUser to resolve ambiguity
func (s *userServiceServer) DeleteUser(ctx context.Context, req *userapi.DeleteUserRequest) (*userapi.DeleteUserResponse, error) {
	return s.UserService.DeleteUser(ctx, req)
}

// Explicitly implement GetUserInfo to resolve ambiguity
func (s *userServiceServer) GetUserInfo(ctx context.Context, req *userapi.GetUserInfoRequest) (*userapi.GetUserInfoResponse, error) {
	return s.UserService.GetUserInfo(ctx, req)
}

// Explicitly implement Logout to resolve ambiguity
func (s *userServiceServer) Logout(ctx context.Context, req *userapi.LogoutRequest) (*userapi.LogoutResponse, error) {
	return s.UserService.Logout(ctx, req)
}

// Explicitly implement Register to resolve ambiguity
func (s *userServiceServer) Register(ctx context.Context, req *userapi.RegisterRequest) (*userapi.RegisterResponse, error) {
	return s.UserService.Register(ctx, req)
}

// Implement the UpdateUserInfo method
func (s *userServiceServer) UpdateUserInfo(ctx context.Context, req *userapi.UpdateUserInfoRequest) (*userapi.UpdateUserInfoResponse, error) {
	return s.UserService.UpdateUserInfo(ctx, req)
}

func main() {
	// 加载配置
	configPath := "/Users/Apple/Desktop/demo/configs/user.yaml"

	// 打印配置文件的完整路径和内容
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v\nPath: %s", err, configPath)
	}
	log.Printf("Config File Contents:\n%s", string(configBytes))

	userConfig, err := config.LoadUserConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load user config: %v\n"+
			"Config Path: %s\n"+
			"Current Working Directory: %s",
			err, configPath, func() string {
				dir, _ := os.Getwd()
				return dir
			}())
	}

	// 打印详细的配置信息
	log.Printf("Loaded User Config:\n"+
		"  Service Name: %s\n"+
		"  Service Version: %s\n"+
		"  Database Host: %s\n"+
		"  Database Port: %d\n"+
		"  Database Name: %s\n"+
		"  Database User: %s\n"+
		"  Database Password: %s\n",
		userConfig.ServiceName,
		userConfig.ServiceVersion,
		userConfig.Database.Host,
		userConfig.Database.Port,
		userConfig.Database.Name,
		userConfig.Database.User,
		"****")

	// 如果数据库配置为空，尝试从环境变量获取
	if userConfig.Database.Host == "" {
		userConfig.Database.Host = os.Getenv("USER_DATABASE_HOST")
	}
	if userConfig.Database.Port == 0 {
		port, _ := strconv.Atoi(os.Getenv("USER_DATABASE_PORT"))
		userConfig.Database.Port = port
	}
	if userConfig.Database.Name == "" {
		userConfig.Database.Name = os.Getenv("USER_DATABASE_NAME")
	}
	if userConfig.Database.User == "" {
		userConfig.Database.User = os.Getenv("USER_DATABASE_USER")
	}
	if userConfig.Database.Password == "" {
		userConfig.Database.Password = os.Getenv("USER_DATABASE_PASSWORD")
	}

	// 建立数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		userConfig.Database.User,
		userConfig.Database.Password,
		userConfig.Database.Host,
		userConfig.Database.Port,
		userConfig.Database.Name)

	log.Printf("Database Connection Details:\n"+
		"  Host: %s\n"+
		"  Port: %d\n"+
		"  Database: %s\n"+
		"  User: %s\n"+
		"  DSN: %s",
		userConfig.Database.Host,
		userConfig.Database.Port,
		userConfig.Database.Name,
		userConfig.Database.User,
		strings.ReplaceAll(dsn, userConfig.Database.Password, "****"))

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 启用详细日志
	})
	if err != nil {
		log.Fatalf("Detailed database connection error: %v\n"+
			"Connection Details:\n"+
			"  Driver: %s\n"+
			"  Host: %s\n"+
			"  Port: %d\n"+
			"  Database: %s\n"+
			"  User: %s\n"+
			"  Password: %s\n",
			err,
			userConfig.Database.Driver,
			userConfig.Database.Host,
			userConfig.Database.Port,
			userConfig.Database.Name,
			userConfig.Database.User,
			"****")
	}

	// 配置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection pool: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(userConfig.Database.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(userConfig.Database.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(userConfig.Database.ConnectionMaxLifetime)

	// 创建 UserService
	userServiceInstance, err := userService.NewUserService(
		userService.WithGormDatabase(db),
	)
	if err != nil {
		log.Fatalf("Failed to create user service: %v", err)
	}

	// 创建 gRPC 服务器
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 注册 UserService
	userServiceImpl := &userServiceServer{
		UserService: userServiceInstance,
	}
	userapi.RegisterUserServiceServer(grpcServer, userServiceImpl)

	// 处理优雅关闭
	go func() {
		log.Printf("Starting gRPC server on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use ctx for any cleanup operations
	_ = ctx

	log.Println("Shutting down User Service...")
	grpcServer.GracefulStop()
	log.Println("User Service stopped")
}
