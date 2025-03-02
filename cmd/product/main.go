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

	productapi "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/bytedance-youthcamp/demo/internal/config"
	product "github.com/bytedance-youthcamp/demo/internal/service/product"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Define wrapper methods to resolve ambiguity
type productServiceServer struct {
	*product.ProductService
	productapi.UnimplementedProductServiceServer
}

// Implement the necessary methods to satisfy the ProductServiceServer interface
func (s *productServiceServer) CreateProduct(ctx context.Context, req *productapi.CreateProductRequest) (*productapi.CreateProductResponse, error) {
	return s.ProductService.CreateProduct(ctx, req)
}

func (s *productServiceServer) GetProduct(ctx context.Context, req *productapi.GetProductRequest) (*productapi.GetProductResponse, error) {
	return s.ProductService.GetProduct(ctx, req)
}

func (s *productServiceServer) GetProducts(ctx context.Context, req *productapi.GetProductsRequest) (*productapi.GetProductsResponse, error) {
	return s.ProductService.GetProducts(ctx, req)
}

func (s *productServiceServer) UpdateProduct(ctx context.Context, req *productapi.UpdateProductRequest) (*productapi.UpdateProductResponse, error) {
	return s.ProductService.UpdateProduct(ctx, req)
}

func (s *productServiceServer) DeleteProduct(ctx context.Context, req *productapi.DeleteProductRequest) (*productapi.DeleteProductResponse, error) {
	return s.ProductService.DeleteProduct(ctx, req)
}

// The UnimplementedProductServiceServer is embedded in the struct, so we don't need this method

func main() {
	// 加载配置
	viper.SetConfigName("product")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/Users/Apple/Desktop/demo/configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var productConfig config.ProductConfig
	if err := viper.Unmarshal(&productConfig); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	// 设置数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		productConfig.Database.User,
		productConfig.Database.Password,
		productConfig.Database.Host,
		productConfig.Database.Port,
		productConfig.Database.Name,
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
	service, err := product.NewProductService(
		product.WithTestDatabase(sqlDB),
	)
	if err != nil {
		log.Fatalf("Failed to create product service: %v", err)
	}

	// 创建包装后的服务
	server := &productServiceServer{ProductService: service}

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	productapi.RegisterProductServiceServer(grpcServer, server)

	// 处理优雅关闭
	go func() {
		log.Println("Product Service started on :50052")
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

	log.Println("Shutting down Product Service...")
	grpcServer.GracefulStop()
	log.Println("Product Service stopped")
}
