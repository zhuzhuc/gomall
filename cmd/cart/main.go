package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	cartapi "github.com/bytedance-youthcamp/demo/api/cart"
	"github.com/bytedance-youthcamp/demo/internal/config"
	cart "github.com/bytedance-youthcamp/demo/internal/service/cart"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Define wrapper methods to resolve ambiguity
type cartServiceServer struct {
	*cart.CartService
}

// Implement the necessary methods to satisfy the CartServiceServer interface
func (s *cartServiceServer) CreateCart(ctx context.Context, req *cartapi.CreateCartRequest) (*cartapi.CreateCartResponse, error) {
	return s.CartService.CreateCart(ctx, req)
}

func (s *cartServiceServer) ClearCart(ctx context.Context, req *cartapi.ClearCartRequest) (*cartapi.ClearCartResponse, error) {
	return s.CartService.ClearCart(ctx, req)
}

func (s *cartServiceServer) GetCart(ctx context.Context, req *cartapi.GetCartRequest) (*cartapi.GetCartResponse, error) {
	return s.CartService.GetCart(ctx, req)
}

func (s *cartServiceServer) AddToCart(ctx context.Context, req *cartapi.AddToCartRequest) (*cartapi.AddToCartResponse, error) {
	return s.CartService.AddToCart(ctx, req)
}

func (s *cartServiceServer) RemoveFromCart(ctx context.Context, req *cartapi.RemoveFromCartRequest) (*cartapi.RemoveFromCartResponse, error) {
	return s.CartService.RemoveFromCart(ctx, req)
}

func (s *cartServiceServer) UpdateCartItem(ctx context.Context, req *cartapi.UpdateCartItemRequest) (*cartapi.UpdateCartItemResponse, error) {
	return s.CartService.UpdateCartItem(ctx, req)
}

// Must embed the unimplemented server
func (s *cartServiceServer) mustEmbedUnimplementedCartServiceServer() {}

func main() {
	// 加载配置
	viper.SetConfigName("cart")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/Users/Apple/Desktop/demo/configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var cartConfig config.CartConfig
	if err := viper.Unmarshal(&cartConfig); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	// 设置数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cartConfig.Database.User,
		cartConfig.Database.Password,
		cartConfig.Database.Host,
		cartConfig.Database.Port,
		cartConfig.Database.Name,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 获取底层的 *sql.DB 对象
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}

	// 创建服务实例
	service, err := cart.NewCartService(
		cart.WithTestDatabase(sqlDB),
	)
	if err != nil {
		log.Fatalf("Failed to create cart service: %v", err)
	}

	// 创建包装后的服务
	server := &cartServiceServer{CartService: service}

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":50055")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	cartapi.RegisterCartServiceServer(grpcServer, server)

	// 处理优雅关闭
	go func() {
		log.Println("Cart Service started on :50055")
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

	log.Println("Shutting down Cart Service...")
	grpcServer.GracefulStop()
	log.Println("Cart Service stopped")
}
