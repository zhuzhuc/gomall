package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	pb "github.com/bytedance-youthcamp/demo/api/payment"
)

type PaymentService struct {
	pb.UnimplementedPaymentServiceServer
	db *gorm.DB
}

type Payment struct {
	gorm.Model
	PaymentID     string             `gorm:"uniqueIndex:idx_payment_id,length:36"`
	OrderID       int32              `gorm:"not null"`
	Amount        float64            `gorm:"not null"`
	Status        pb.PaymentStatus   `gorm:"not null"`
	Method        pb.PaymentMethod   `gorm:"not null"`
	TransactionID string             `gorm:"default:null"`
}

func NewPaymentService(db *gorm.DB) *PaymentService {
	return &PaymentService{
		db: db,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	// 验证支付金额
	if req.Amount <= 0 {
		return nil, fmt.Errorf("invalid payment amount: must be greater than zero")
	}

	// 生成唯一的支付ID
	paymentID := uuid.New().String()

	// 创建支付记录
	payment := &Payment{
		PaymentID: paymentID,
		OrderID:   req.OrderId,
		Amount:    req.Amount,
		Status:    pb.PaymentStatus_PAYMENT_STATUS_PENDING,
		Method:    req.Method,
	}

	// 保存到数据库
	if err := s.db.Create(payment).Error; err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// 模拟生成支付URL（实际应该对接第三方支付）
	paymentURL := fmt.Sprintf("https://payment.example.com/pay?id=%s", paymentID)

	return &pb.CreatePaymentResponse{
		PaymentId:  paymentID,
		PaymentUrl: paymentURL,
	}, nil
}

func (s *PaymentService) QueryPayment(ctx context.Context, req *pb.QueryPaymentRequest) (*pb.QueryPaymentResponse, error) {
	var payment Payment
	if err := s.db.Where("payment_id = ?", req.PaymentId).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &pb.QueryPaymentResponse{
		PaymentId:     payment.PaymentID,
		OrderId:       payment.OrderID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		Method:        payment.Method,
		TransactionId: payment.TransactionID,
	}, nil
}

func (s *PaymentService) ProcessPaymentNotification(ctx context.Context, req *pb.PaymentNotificationRequest) (*pb.PaymentNotificationResponse, error) {
	// 查找支付记录
	var payment Payment
	if err := s.db.Where("payment_id = ?", req.PaymentId).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	// 验证状态转换规则
	switch payment.Status {
	case pb.PaymentStatus_PAYMENT_STATUS_PENDING:
		// 从 PENDING 只能转换到 SUCCESS 或 FAILED
		if req.Status != pb.PaymentStatus_PAYMENT_STATUS_SUCCESS &&
		   req.Status != pb.PaymentStatus_PAYMENT_STATUS_FAILED {
			return nil, fmt.Errorf("invalid status transition from PENDING")
		}
	case pb.PaymentStatus_PAYMENT_STATUS_SUCCESS:
		// 成功状态不能再转换
		return nil, fmt.Errorf("cannot change status from SUCCESS")
	case pb.PaymentStatus_PAYMENT_STATUS_FAILED:
		// 失败状态不能再转换
		return nil, fmt.Errorf("cannot change status from FAILED")
	default:
		// 未知状态
		return nil, fmt.Errorf("invalid current payment status")
	}

	// 更新支付状态
	payment.Status = req.Status
	payment.TransactionID = req.TransactionId

	if err := s.db.Save(&payment).Error; err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// 如果支付成功，记录日志
	if req.Status == pb.PaymentStatus_PAYMENT_STATUS_SUCCESS {
		fmt.Printf("Payment successful for order %d\n", req.OrderId)
	}

	return &pb.PaymentNotificationResponse{}, nil
}

// 模拟支付回调（实际应该由第三方支付系统调用）
func (s *PaymentService) SimulatePaymentCallback(ctx context.Context, req *pb.SimulatePaymentCallbackRequest) (*pb.SimulatePaymentCallbackResponse, error) {
	var payment Payment
	if err := s.db.Where("payment_id = ?", req.PaymentId).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	transactionID := fmt.Sprintf("simulated_callback_%s", uuid.New().String())

	_, err := s.ProcessPaymentNotification(ctx, &pb.PaymentNotificationRequest{
		PaymentId:     req.PaymentId,
		OrderId:       payment.OrderID,
		Status:        req.Status,
		TransactionId: transactionID,
	})
	if err != nil {
		return nil, err
	}

	return &pb.SimulatePaymentCallbackResponse{
		Success: true,
	}, nil
}
