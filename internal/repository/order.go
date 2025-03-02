package repository

import (
	"context"
	"time"
)

type OrderStatus int32

const (
	OrderStatusPending OrderStatus = iota
	OrderStatusPaid
	OrderStatusShipped
	OrderStatusDelivered
	OrderStatusCancelled
)

type Order struct {
	ID              string
	UserID          string
	Items           []*OrderItem
	TotalAmount     float64
	Status          OrderStatus
	ShippingAddress string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderItem struct {
	ProductID   string
	Quantity    int32
	Price       float64
	ProductName string
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	Get(ctx context.Context, orderID string) (*Order, error)
	Update(ctx context.Context, order *Order) error
	List(ctx context.Context, userID string, status *OrderStatus) ([]*Order, error)
	ListPendingOrdersOlderThan(ctx context.Context, duration time.Duration) ([]*Order, error)
	GetUserOrders(ctx context.Context, userID string, page, pageSize int, status OrderStatus) ([]*Order, int, error)
}
