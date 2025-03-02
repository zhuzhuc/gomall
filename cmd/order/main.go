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

	orderapi "github.com/bytedance-youthcamp/demo/api/order"
	"github.com/bytedance-youthcamp/demo/internal/config"
	orderService "github.com/bytedance-youthcamp/demo/internal/service/order"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	viper.SetConfigName("order")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/Users/Apple/Desktop/demo/configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var orderConfig config.OrderConfig
	if err := viper.Unmarshal(&orderConfig); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	// 设置数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		orderConfig.Database.User,
		orderConfig.Database.Password,
		orderConfig.Database.Host,
		orderConfig.Database.Port,
		orderConfig.Database.Name,
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
	service, err := orderService.NewOrderService(
		orderService.WithTestDatabase(sqlDB),
	)
	if err != nil {
		log.Fatalf("Failed to create order service: %v", err)
	}

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	orderapi.RegisterOrderServiceServer(grpcServer, service)

	// 处理优雅关闭
	go func() {
		log.Println("Order Service started on :50053")
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

	log.Println("Shutting down Order Service...")
	grpcServer.GracefulStop()
	log.Println("Order Service stopped")
}
