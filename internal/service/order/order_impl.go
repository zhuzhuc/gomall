package order

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	cartpb "github.com/bytedance-youthcamp/demo/api/cart"
	orderapi "github.com/bytedance-youthcamp/demo/api/order"
	productpb "github.com/bytedance-youthcamp/demo/api/product"
	userpb "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/bytedance-youthcamp/demo/internal/repository"
	// 添加日志导入
)

type orderService struct {
	orderapi.UnimplementedOrderServiceServer
	productClient productpb.ProductServiceClient
	cartClient    cartpb.CartServiceClient
	userClient    userpb.UserServiceClient
	orderRepo     repository.OrderRepository
	db            *sql.DB
}

func NewOrderService(opts ...Option) (*orderService, error) {
	service := &orderService{}

	for _, opt := range opts {
		opt(service)
	}

	// 启动自动取消订单的定时任务
	// 每5分钟检查一次，取消超过30分钟未支付的订单
	service.StartOrderCancellationTask(5*time.Minute, 30*time.Minute)

	return service, nil
}

type Option func(*orderService)

func WithProductClient(client productpb.ProductServiceClient) Option {
	return func(s *orderService) {
		s.productClient = client
	}
}

func WithCartClient(client cartpb.CartServiceClient) Option {
	return func(s *orderService) {
		s.cartClient = client
	}
}

func WithUserClient(client userpb.UserServiceClient) Option {
	return func(s *orderService) {
		s.userClient = client
	}
}

func WithOrderRepository(repo repository.OrderRepository) Option {
	return func(s *orderService) {
		s.orderRepo = repo
	}
}

func WithTestDatabase(db *sql.DB) Option {
	return func(s *orderService) {
		s.db = db
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req *orderapi.CreateOrderRequest) (*orderapi.CreateOrderResponse, error) {
	// 验证用户信息，在测试环境中允许跳过
	if s.userClient != nil {
		_, err := s.userClient.GetUserInfo(ctx, &userpb.GetUserInfoRequest{UserId: req.UserId})
		if err != nil {
			return &orderapi.CreateOrderResponse{
				Success:       false,
				ErrorMessage: "Invalid user",
			}, nil
		}
	}

	// 准备订单项目
	orderItems := make([]*repository.OrderItem, len(req.Items))
	for i, item := range req.Items {
		orderItems[i] = &repository.OrderItem{
			ProductID:   strconv.Itoa(int(item.ProductId)),
			ProductName: item.ProductName,
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		}
	}

	// 创建订单
	order := &repository.Order{
		UserID:      strconv.Itoa(int(req.UserId)),
		Status:      repository.OrderStatusPending,
		Items:       orderItems,
		TotalAmount: req.TotalPrice,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存订单
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return &orderapi.CreateOrderResponse{
			Success:       false,
			ErrorMessage: "Failed to create order",
		}, nil
	}

	// 解析订单ID
	orderID, _ := strconv.Atoi(order.ID)

	return &orderapi.CreateOrderResponse{
		Success:  true,
		OrderId: int32(orderID),
	}, nil
}

func (s *orderService) SettleOrder(ctx context.Context, req *orderapi.SettleOrderRequest) (*orderapi.SettleOrderResponse, error) {
	// 获取订单
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(req.OrderId))
	if err != nil {
		return &orderapi.SettleOrderResponse{
			Success:       false,
			ErrorMessage: "Order not found",
		}, nil
	}

	// 检查订单状态
	if order.Status != repository.OrderStatusPending {
		return &orderapi.SettleOrderResponse{
			Success:       false,
			ErrorMessage: "Order cannot be settled",
		}, nil
	}

	// 在测试环境中，我们可能没有产品服务客户端，但有直接的数据库连接
	if s.productClient != nil {
		// 使用产品服务客户端减少库存
		for _, item := range order.Items {
			productID, _ := strconv.Atoi(item.ProductID)
			
			// 先获取产品信息
			getResp, err := s.productClient.GetProduct(ctx, &productpb.GetProductRequest{
				ProductId: int32(productID),
			})
			if err != nil || !getResp.Success {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Failed to get product information",
				}, nil
			}
			
			// 计算新的库存
			newStock := getResp.Product.Stock - int32(item.Quantity)
			if newStock < 0 {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Insufficient stock",
				}, nil
			}
			
			// 更新产品库存
			_, err = s.productClient.UpdateProduct(ctx, &productpb.UpdateProductRequest{
				ProductId: int32(productID),
				Name:      getResp.Product.Name,
				Stock:     newStock,
				Price:     getResp.Product.Price,
				Category:  getResp.Product.Category,
				ImageUrl:  getResp.Product.ImageUrl,
			})
			if err != nil {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Failed to reduce stock",
				}, nil
			}
		}
	} else if s.db != nil {
		// 在测试环境中直接使用数据库连接更新库存
		for _, item := range order.Items {
			productID, _ := strconv.Atoi(item.ProductID)
			
			// 先获取当前库存
			var currentStock int
			err := s.db.QueryRow("SELECT stock FROM products WHERE id = ?", productID).Scan(&currentStock)
			if err != nil {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Failed to get product information",
				}, nil
			}
			
			// 计算新的库存
			newStock := currentStock - int(item.Quantity)
			if newStock < 0 {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Insufficient stock",
				}, nil
			}
			
			// 更新产品库存
			_, err = s.db.Exec("UPDATE products SET stock = ? WHERE id = ?", newStock, productID)
			if err != nil {
				return &orderapi.SettleOrderResponse{
					Success:      false,
					ErrorMessage: "Failed to reduce stock",
				}, nil
			}
		}
	}

	// 更新订单状态
	order.Status = repository.OrderStatusPaid
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return &orderapi.SettleOrderResponse{
			Success:      false,
			ErrorMessage: "Failed to update order status",
		}, nil
	}

	return &orderapi.SettleOrderResponse{
		Success:  true,
		OrderId: req.OrderId,
		Status:  orderapi.OrderStatus_PAID,
	}, nil
}

func (s *orderService) GetOrderDetails(ctx context.Context, req *orderapi.GetOrderDetailsRequest) (*orderapi.GetOrderDetailsResponse, error) {
	// 获取订单
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(req.OrderId))
	if err != nil {
		return &orderapi.GetOrderDetailsResponse{
			Success:      false,
			ErrorMessage: "Order not found",
		}, nil
	}

	// 检查用户权限
	if order.UserID != strconv.Itoa(int(req.UserId)) {
		return &orderapi.GetOrderDetailsResponse{
			Success:      false,
			ErrorMessage: "Unauthorized access",
		}, nil
	}

	// 转换订单项目
	pbItems := make([]*orderapi.OrderItem, len(order.Items))
	for i, item := range order.Items {
		productID, _ := strconv.Atoi(item.ProductID)
		pbItems[i] = &orderapi.OrderItem{
			ProductId:   int32(productID),
			ProductName: item.ProductName,
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		}
	}

	// 转换订单ID
	orderID, _ := strconv.Atoi(order.ID)
	userID, _ := strconv.Atoi(order.UserID)

	return &orderapi.GetOrderDetailsResponse{
		Success: true,
		Order: &orderapi.Order{
			Id:          int32(orderID),
			UserId:      int32(userID),
			Status:      orderapi.OrderStatus(order.Status),
			TotalAmount: order.TotalAmount,
			Items:       pbItems,
			CreatedAt:   order.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   order.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *orderService) GetOrder(ctx context.Context, req *orderapi.GetOrderRequest) (*orderapi.GetOrderResponse, error) {
	// 获取订单
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(req.OrderId))
	if err != nil {
		return &orderapi.GetOrderResponse{
			Success:      false,
			ErrorMessage: "Order not found",
		}, nil
	}

	// 转换订单项目
	pbItems := make([]*orderapi.OrderItem, len(order.Items))
	for i, item := range order.Items {
		productID, _ := strconv.Atoi(item.ProductID)
		pbItems[i] = &orderapi.OrderItem{
			ProductId:   int32(productID),
			ProductName: item.ProductName,
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		}
	}

	// 转换订单ID
	orderID, _ := strconv.Atoi(order.ID)
	userID, _ := strconv.Atoi(order.UserID)

	return &orderapi.GetOrderResponse{
		Order: &orderapi.Order{
			Id:          int32(orderID),
			UserId:      int32(userID),
			Status:      orderapi.OrderStatus(order.Status),
			TotalAmount: order.TotalAmount,
			Items:       pbItems,
			CreatedAt:   order.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   order.UpdatedAt.Format(time.RFC3339),
		},
		Success: true,
	}, nil
}

func (s *orderService) GetUserOrders(ctx context.Context, req *orderapi.GetUserOrdersRequest) (*orderapi.GetUserOrdersResponse, error) {
	// 转换状态
	var repoStatus repository.OrderStatus
	switch req.Status {
	case orderapi.OrderStatus_PENDING:
		repoStatus = repository.OrderStatusPending
	case orderapi.OrderStatus_PAID:
		repoStatus = repository.OrderStatusPaid
	case orderapi.OrderStatus_SHIPPING:
		repoStatus = repository.OrderStatusShipped
	case orderapi.OrderStatus_COMPLETED:
		repoStatus = repository.OrderStatusDelivered
	case orderapi.OrderStatus_CANCELLED:
		repoStatus = repository.OrderStatusCancelled
	default:
		repoStatus = repository.OrderStatusPending
	}

	// 获取用户订单
	orders, total, err := s.orderRepo.GetUserOrders(ctx, strconv.Itoa(int(req.UserId)), int(req.Page), int(req.PageSize), repoStatus)
	if err != nil {
		return &orderapi.GetUserOrdersResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 转换订单
	pbOrders := make([]*orderapi.Order, len(orders))
	for i, order := range orders {
		pbItems := make([]*orderapi.OrderItem, len(order.Items))
		for j, item := range order.Items {
			productID, _ := strconv.Atoi(item.ProductID)
			pbItems[j] = &orderapi.OrderItem{
				ProductId:   int32(productID),
				ProductName: item.ProductName,
				Quantity:    item.Quantity,
				Price:       item.Price,
			}
		}

		orderID, _ := strconv.Atoi(order.ID)
		userID, _ := strconv.Atoi(order.UserID)

		pbOrders[i] = &orderapi.Order{
			Id:          int32(orderID),
			UserId:      int32(userID),
			Items:       pbItems,
			TotalAmount: order.TotalAmount,
			Status:      orderapi.OrderStatus(order.Status),
			Address:     order.ShippingAddress,
			CreatedAt:   order.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   order.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &orderapi.GetUserOrdersResponse{
		Orders:  pbOrders,
		Total:   int32(total),
		Success: true,
	}, nil
}

func (s *orderService) UpdateOrder(ctx context.Context, req *orderapi.UpdateOrderRequest) (*orderapi.UpdateOrderResponse, error) {
	// 获取订单
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(req.OrderId))
	if err != nil {
		return &orderapi.UpdateOrderResponse{
			Success:      false,
			ErrorMessage: "Order not found",
		}, nil
	}

	// 更新订单信息
	order.ShippingAddress = req.Address
	order.UpdatedAt = time.Now()

	// 转换状态
	switch req.Status {
	case orderapi.OrderStatus_PENDING:
		order.Status = repository.OrderStatusPending
	case orderapi.OrderStatus_PAID:
		order.Status = repository.OrderStatusPaid
	case orderapi.OrderStatus_SHIPPING:
		order.Status = repository.OrderStatusShipped
	case orderapi.OrderStatus_COMPLETED:
		order.Status = repository.OrderStatusDelivered
	case orderapi.OrderStatus_CANCELLED:
		order.Status = repository.OrderStatusCancelled
	}

	// 保存更新
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return &orderapi.UpdateOrderResponse{
			Success:      false,
			ErrorMessage: "Failed to update order",
		}, nil
	}

	return &orderapi.UpdateOrderResponse{
		Success: true,
	}, nil
}

func (s *orderService) CancelOrder(ctx context.Context, req *orderapi.CancelOrderRequest) (*orderapi.CancelOrderResponse, error) {
	// 获取订单
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(req.OrderId))
	if err != nil {
		return &orderapi.CancelOrderResponse{
			Success:      false,
			ErrorMessage: "Order not found",
		}, nil
	}

	// 检查订单状态是否允许取消
	if order.Status != repository.OrderStatusPending && order.Status != repository.OrderStatusPaid {
		return &orderapi.CancelOrderResponse{
			Success:      false,
			ErrorMessage: "Order cannot be cancelled",
		}, nil
	}

	// 更新订单状态
	order.Status = repository.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	// 保存更新
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return &orderapi.CancelOrderResponse{
			Success:      false,
			ErrorMessage: "Failed to cancel order",
		}, nil
	}

	return &orderapi.CancelOrderResponse{
		Success: true,
	}, nil
}

func (s *orderService) scheduleOrderCancellation(orderID string, minutes int) {
	time.Sleep(time.Duration(minutes) * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	order, err := s.orderRepo.Get(ctx, orderID)
	if err != nil {
		log.Printf("Error retrieving order %s: %v", orderID, err)
		return
	}

	if order.Status == repository.OrderStatusPending {
		order.Status = repository.OrderStatusCancelled
		order.UpdatedAt = time.Now()

		if err := s.orderRepo.Update(ctx, order); err != nil {
			log.Printf("Error cancelling order %s: %v", orderID, err)
		}
	}
}

func (s *orderService) StartOrderCancellationTask(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			ctx := context.Background()
			orders, err := s.orderRepo.ListPendingOrdersOlderThan(ctx, timeout)
			if err != nil {
				log.Printf("Error fetching pending orders: %v", err)
				continue
			}

			for _, order := range orders {
				order.Status = repository.OrderStatusCancelled
				order.UpdatedAt = time.Now()

				if err := s.orderRepo.Update(ctx, order); err != nil {
					log.Printf("Error cancelling order %s: %v", order.ID, err)
				} else {
					log.Printf("Auto-cancelled order %s due to timeout", order.ID)
				}
			}
		}
	}()
}
