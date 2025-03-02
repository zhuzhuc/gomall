package order

import (
	"context"

	orderapi "github.com/bytedance-youthcamp/demo/api/order"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *orderapi.CreateOrderRequest) (*orderapi.CreateOrderResponse, error)
	SettleOrder(ctx context.Context, req *orderapi.SettleOrderRequest) (*orderapi.SettleOrderResponse, error)
	GetOrderDetails(ctx context.Context, req *orderapi.GetOrderDetailsRequest) (*orderapi.GetOrderDetailsResponse, error)
	GetOrder(ctx context.Context, req *orderapi.GetOrderRequest) (*orderapi.GetOrderResponse, error)
	GetUserOrders(ctx context.Context, req *orderapi.GetUserOrdersRequest) (*orderapi.GetUserOrdersResponse, error)
	UpdateOrder(ctx context.Context, req *orderapi.UpdateOrderRequest) (*orderapi.UpdateOrderResponse, error)
	CancelOrder(ctx context.Context, req *orderapi.CancelOrderRequest) (*orderapi.CancelOrderResponse, error)
}
