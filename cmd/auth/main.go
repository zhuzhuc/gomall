package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	authapi "github.com/bytedance-youthcamp/demo/auth"
	"github.com/bytedance-youthcamp/demo/internal/config"
	auth "github.com/bytedance-youthcamp/demo/internal/service/auth"

	"google.golang.org/grpc"
)

// Define the authServiceServer struct with proper method implementation
type authServiceServer struct {
	*auth.AuthService
	authapi.UnimplementedAuthServiceServer
}

// Implement the DeliverTokenByRPC method as a method of authServiceServer
func (s *authServiceServer) DeliverTokenByRPC(ctx context.Context, req *authapi.DeliverTokenReq) (*authapi.DeliveryResp, error) {
	// Convert API type to internal type
	internalReq := &auth.DeliverTokenReq{
		UserId: req.GetUserId(),
	}
	
	// Call the internal service method
	internalResp, err := s.AuthService.DeliverTokenByRPC(context.Background(), internalReq)
	if err != nil {
		return nil, err
	}
	
	// Convert internal response to API response
	return &authapi.DeliveryResp{
		Token: internalResp.Token,
	}, nil
}

// Implement the VerifyTokenByRPC method as a method of authServiceServer
func (s *authServiceServer) VerifyTokenByRPC(ctx context.Context, req *authapi.VerifyTokenReq) (*authapi.VerifyResp, error) {
	// Convert API type to internal type
	internalReq := &auth.VerifyTokenReq{
		Token: req.GetToken(),
	}
	
	// Call the internal service method
	internalResp, err := s.AuthService.VerifyTokenByRPC(context.Background(), internalReq)
	if err != nil {
		return nil, err
	}
	
	// Convert internal response to API response
	return &authapi.VerifyResp{
		Res: internalResp.Res,
	}, nil
}

// Implement the RenewTokenByRPC method as a method of authServiceServer
func (s *authServiceServer) RenewTokenByRPC(ctx context.Context, req *authapi.RenewTokenReq) (*authapi.RenewTokenResp, error) {
	// Convert API type to internal type
	internalReq := &auth.RenewTokenReq{
		Token: req.GetToken(),
	}
	
	// Call the internal service method
	internalResp, err := s.AuthService.RenewTokenByRPC(context.Background(), internalReq)
	if err != nil {
		return nil, err
	}
	
	// Convert internal response to API response
	return &authapi.RenewTokenResp{
		NewToken: internalResp.NewToken,
	}, nil
}

func main() {
	// 加载配置
	configPath := "/Users/Apple/Desktop/demo/configs/auth.yaml"
	_, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建 gRPC 服务器
	lis, err := net.Listen("tcp", ":50050")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 初始化服务
	service, err := auth.NewAuthService()
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	// 创建包装后的服务
	server := &authServiceServer{AuthService: service}

	// 注册服务
	authapi.RegisterAuthServiceServer(grpcServer, server)

	// 处理优雅关闭
	go func() {
		log.Printf("Starting gRPC server on :50050")
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

	log.Println("Shutting down Auth Service...")
	grpcServer.GracefulStop()
	log.Println("Auth Service stopped")
}
