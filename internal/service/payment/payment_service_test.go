package payment_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	pb "github.com/bytedance-youthcamp/demo/api/payment"
	"github.com/bytedance-youthcamp/demo/internal/service/payment"
)

func setupTestPaymentService(t *testing.T) *payment.PaymentService {
	// 根据环境变量选择数据库主机
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	// 使用 MySQL 数据库进行测试
	dsn := fmt.Sprintf("root:root@tcp(%s:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", host)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	// 自动迁移数据库模型
	err = db.AutoMigrate(&payment.Payment{})
	assert.NoError(t, err)

	paymentService := payment.NewPaymentService(db)
	return paymentService
}

func TestCreatePayment(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name    string
		request *pb.CreatePaymentRequest
	}{
		{
			name: "创建支付订单",
			request: &pb.CreatePaymentRequest{
				OrderId: 1,
				Amount:  200.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := paymentService.CreatePayment(ctx, tc.request)
			assert.NoError(t, err)
			assert.NotEmpty(t, resp.PaymentId)
			assert.NotEmpty(t, resp.PaymentUrl)
		})
	}
}

func TestQueryPayment(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	// 先创建一个支付订单
	createResp, err := paymentService.CreatePayment(ctx, &pb.CreatePaymentRequest{
		OrderId: 1,
		Amount:  200.0,
		Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
	})
	assert.NoError(t, err)

	testCases := []struct {
		name          string
		paymentID     string
		expectedError bool
	}{
		{
			name:          "查询已存在的支付订单",
			paymentID:     createResp.PaymentId,
			expectedError: false,
		},
		{
			name:          "查询不存在的支付订单",
			paymentID:     "non_existent_payment",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := paymentService.QueryPayment(ctx, &pb.QueryPaymentRequest{
				PaymentId: tc.paymentID,
			})

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.paymentID, resp.PaymentId)
				assert.Equal(t, float64(200.0), resp.Amount)
				assert.Equal(t, pb.PaymentMethod_PAYMENT_METHOD_ALIPAY, resp.Method)
			}
		})
	}
}

func TestProcessPaymentNotification(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name           string
		initialStatus  pb.PaymentStatus
		notifyStatus   pb.PaymentStatus
		expectError    bool
		verifyCallback func(t *testing.T, resp *pb.PaymentNotificationResponse, err error)
	}{
		{
			name:           "支付成功通知",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_PENDING,
			notifyStatus:   pb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			expectError:    false,
			verifyCallback: func(t *testing.T, resp *pb.PaymentNotificationResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name:           "支付失败通知",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_PENDING,
			notifyStatus:   pb.PaymentStatus_PAYMENT_STATUS_FAILED,
			expectError:    false,
			verifyCallback: func(t *testing.T, resp *pb.PaymentNotificationResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建初始支付订单
			createResp, err := paymentService.CreatePayment(ctx, &pb.CreatePaymentRequest{
				OrderId: 1,
				Amount:  200.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
			})
			assert.NoError(t, err)

			// 处理状态通知
			resp, err := paymentService.ProcessPaymentNotification(ctx, &pb.PaymentNotificationRequest{
				PaymentId:     createResp.PaymentId,
				OrderId:       1,
				Status:        tc.notifyStatus,
				TransactionId: "trans_456",
			})

			// 验证回调
			tc.verifyCallback(t, resp, err)

			// 查询最终状态
			queryResp, err := paymentService.QueryPayment(ctx, &pb.QueryPaymentRequest{
				PaymentId: createResp.PaymentId,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.notifyStatus, queryResp.Status)
			assert.Equal(t, "trans_456", queryResp.TransactionId)
		})
	}
}

func TestPaymentStatusTransitions(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name           string
		initialStatus  pb.PaymentStatus
		transitionTo   pb.PaymentStatus
		shouldTransition bool
	}{
		{
			name:           "Success to Failed (Invalid)",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			transitionTo:   pb.PaymentStatus_PAYMENT_STATUS_FAILED,
			shouldTransition: false,
		},
		{
			name:           "Failed to Success (Invalid)",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_FAILED,
			transitionTo:   pb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			shouldTransition: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建初始支付订单
			createResp, err := paymentService.CreatePayment(ctx, &pb.CreatePaymentRequest{
				OrderId: 1,
				Amount:  200.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
			})
			assert.NoError(t, err)

			// 首先将支付状态转换到初始状态
			_, err = paymentService.ProcessPaymentNotification(ctx, &pb.PaymentNotificationRequest{
				PaymentId:     createResp.PaymentId,
				OrderId:       1,
				Status:        tc.initialStatus,
				TransactionId: "trans_initial",
			})
			assert.NoError(t, err)

			// 尝试状态转换
			_, err = paymentService.ProcessPaymentNotification(ctx, &pb.PaymentNotificationRequest{
				PaymentId:     createResp.PaymentId,
				OrderId:       1,
				Status:        tc.transitionTo,
				TransactionId: "trans_transition",
			})

			if tc.shouldTransition {
				assert.NoError(t, err)
				
				// 验证状态是否正确更新
				queryResp, err := paymentService.QueryPayment(ctx, &pb.QueryPaymentRequest{
					PaymentId: createResp.PaymentId,
				})
				assert.NoError(t, err)
				assert.Equal(t, tc.transitionTo, queryResp.Status)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestSimulatePaymentCallback(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name           string
		initialStatus  pb.PaymentStatus
		callbackStatus pb.PaymentStatus
		expectError    bool
	}{
		{
			name:           "模拟支付成功回调",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_PENDING,
			callbackStatus: pb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
			expectError:    false,
		},
		{
			name:           "模拟支付失败回调",
			initialStatus:  pb.PaymentStatus_PAYMENT_STATUS_PENDING,
			callbackStatus: pb.PaymentStatus_PAYMENT_STATUS_FAILED,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建初始支付订单
			createResp, err := paymentService.CreatePayment(ctx, &pb.CreatePaymentRequest{
				OrderId: 2,
				Amount:  200.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_WECHAT,
			})
			assert.NoError(t, err)

			// 模拟支付回调
			resp, err := paymentService.SimulatePaymentCallback(ctx, &pb.SimulatePaymentCallbackRequest{
				PaymentId: createResp.PaymentId,
				Status:    tc.callbackStatus,
			})

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				// 查询最终状态
				queryResp, err := paymentService.QueryPayment(ctx, &pb.QueryPaymentRequest{
					PaymentId: createResp.PaymentId,
				})
				assert.NoError(t, err)
				assert.Equal(t, tc.callbackStatus, queryResp.Status)
			}
		})
	}
}

func TestPaymentURLGeneration(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name    string
		request *pb.CreatePaymentRequest
	}{
		{
			name: "Alipay Payment",
			request: &pb.CreatePaymentRequest{
				OrderId: 1,
				Amount:  200.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
			},
		},
		{
			name: "WeChat Payment",
			request: &pb.CreatePaymentRequest{
				OrderId: 2,
				Amount:  150.0,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_WECHAT,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := paymentService.CreatePayment(ctx, tc.request)
			assert.NoError(t, err)

			// 验证 PaymentID 唯一性
			assert.NotEmpty(t, resp.PaymentId)
			assert.Len(t, resp.PaymentId, 36) // UUID 长度

			// 验证 PaymentURL 格式
			assert.NotEmpty(t, resp.PaymentUrl)
			assert.Contains(t, resp.PaymentUrl, "https://payment.example.com/pay")
			assert.Contains(t, resp.PaymentUrl, "id=")
		})
	}
}

func TestPaymentAmountValidation(t *testing.T) {
	paymentService := setupTestPaymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name        string
		amount      float64
		expectError bool
	}{
		{
			name:        "Valid Amount",
			amount:      100.0,
			expectError: false,
		},
		{
			name:        "Zero Amount",
			amount:      0.0,
			expectError: true,
		},
		{
			name:        "Negative Amount",
			amount:      -50.0,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := paymentService.CreatePayment(ctx, &pb.CreatePaymentRequest{
				OrderId: 1,
				Amount:  tc.amount,
				Method:  pb.PaymentMethod_PAYMENT_METHOD_ALIPAY,
			})

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
