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

	paymentapi "github.com/bytedance-youthcamp/demo/api/payment"
	"github.com/bytedance-youthcamp/demo/internal/config"
	paymentService "github.com/bytedance-youthcamp/demo/internal/service/payment"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	viper.SetConfigName("payment")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/Users/Apple/Desktop/demo/configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var paymentConfig config.PaymentConfig
	if err := viper.Unmarshal(&paymentConfig); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	// 设置数据库连接
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		paymentConfig.Database.User,
		paymentConfig.Database.Password,
		paymentConfig.Database.Host,
		paymentConfig.Database.Port,
		paymentConfig.Database.Name,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// Configure connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(paymentConfig.Database.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(paymentConfig.Database.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(paymentConfig.Database.ConnectionMaxLifetime)

	// 创建服务实例
	service := paymentService.NewPaymentService(db)

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	paymentapi.RegisterPaymentServiceServer(grpcServer, service)

	// 处理优雅关闭
	go func() {
		log.Println("Payment Service started on :50054")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Use the context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use ctx for any cleanup operations that need a timeout
	_ = ctx

	log.Println("Shutting down Payment Service...")
	grpcServer.GracefulStop()
	log.Println("Payment Service stopped")
}
