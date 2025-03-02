package order

import (
	"context"
	"database/sql"
	"fmt"
	// "os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	// "gopkg.in/yaml.v2"

	orderapi "github.com/bytedance-youthcamp/demo/api/order"
	"github.com/bytedance-youthcamp/demo/internal/repository"
	_ "github.com/go-sql-driver/mysql"
)

type DatabaseConfig struct {
	MySQL struct {
		Host                 string `yaml:"host"`
		Port                 int    `yaml:"port"`
		Username             string `yaml:"username"`
		Password             string `yaml:"password"`
		Database             string `yaml:"database"`
		MaxOpenConnections   int    `yaml:"max_open_connections"`
		MaxIdleConnections   int    `yaml:"max_idle_connections"`
		ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime"`
	} `yaml:"mysql"`
}

type OrderServiceIntegrationTestSuite struct {
	suite.Suite
	db             *sql.DB
	orderService   *orderService
	orderRepo      repository.OrderRepository
	config         *DatabaseConfig
}

func (s *OrderServiceIntegrationTestSuite) loadConfig() {
	// Instead of loading from file, use hardcoded test values
	s.config = &DatabaseConfig{}
	s.config.MySQL.Host = "localhost"
	s.config.MySQL.Port = 3306
	s.config.MySQL.Username = "root"
	s.config.MySQL.Password = "root"
	s.config.MySQL.Database = "testdb"
	s.config.MySQL.MaxOpenConnections = 10
	s.config.MySQL.MaxIdleConnections = 5
	s.config.MySQL.ConnectionMaxLifetime = 60 * time.Second
}

func (s *OrderServiceIntegrationTestSuite) SetupSuite() {
	s.loadConfig()

	// 构建数据库连接字符串 - 直接使用localhost而不是配置文件中的地址
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		s.config.MySQL.Username,
		s.config.MySQL.Password,
		s.config.MySQL.Host,
		s.config.MySQL.Port,
		s.config.MySQL.Database,
	)

	// 连接 MySQL 数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		s.T().Fatalf("Failed to connect to database: %v", err)
	}
	db.SetMaxOpenConns(s.config.MySQL.MaxOpenConnections)
	db.SetMaxIdleConns(s.config.MySQL.MaxIdleConnections)
	db.SetConnMaxLifetime(s.config.MySQL.ConnectionMaxLifetime)

	// Test connection
	err = db.Ping()
	if err != nil {
		s.T().Skipf("Skipping test: could not connect to database: %v", err)
		return
	}

	s.db = db

	// 设置测试数据库
	s.setupTestDatabase()

	// 创建 OrderRepository
	s.orderRepo = repository.NewMySQLOrderRepository(s.db)

	// 创建 OrderService
	var serviceErr error
	s.orderService, serviceErr = NewOrderService(
		WithOrderRepository(s.orderRepo),
		WithTestDatabase(db),
	)
	if serviceErr != nil {
		s.T().Fatalf("Failed to create order service: %v", serviceErr)
	}
}

func (s *OrderServiceIntegrationTestSuite) TearDownSuite() {
	// 清理测试数据库
	if s.db != nil {
		s.dropTestTables()
		// 关闭数据库连接
		s.db.Close()
	}
}

func (s *OrderServiceIntegrationTestSuite) setupTestDatabase() {
	// 先删除已存在的表
	s.dropTestTables()

	// 创建 orders 表
	_, err := s.db.Exec(`
		CREATE TABLE orders (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL,
			total_amount DECIMAL(10, 2) NOT NULL,
			status INT NOT NULL,
			shipping_address TEXT,
			items JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	s.Require().NoError(err)

	// 创建 order_items 表
	_, err = s.db.Exec(`
		CREATE TABLE order_items (
			id INT AUTO_INCREMENT PRIMARY KEY,
			order_id INT NOT NULL,
			product_id VARCHAR(50) NOT NULL,
			product_name VARCHAR(255) NOT NULL,
			quantity INT NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
		)
	`)
	s.Require().NoError(err)

	// 创建测试产品数据
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL,
			stock INT NOT NULL DEFAULT 0,
			category VARCHAR(100),
			image_url VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	s.Require().NoError(err)

	// 插入测试产品
	_, err = s.db.Exec(`
		INSERT INTO products (id, name, description, price, stock, category, image_url)
		VALUES (1, 'Test Product', 'Test Description', 99.99, 10, 'Test Category', 'http://example.com/image.jpg')
	`)
	s.Require().NoError(err)

	// 创建测试用户数据
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(50) NOT NULL,
			email VARCHAR(100) NOT NULL,
			phone VARCHAR(20),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	s.Require().NoError(err)

	// 插入测试用户
	_, err = s.db.Exec(`
		INSERT INTO users (id, username, email, phone)
		VALUES (1, 'testuser', 'test@example.com', '1234567890')
	`)
	s.Require().NoError(err)
}

func (s *OrderServiceIntegrationTestSuite) dropTestTables() {
	// 按照外键约束的顺序删除表
	_, err := s.db.Exec("DROP TABLE IF EXISTS order_items")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS orders")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS products")
	s.Require().NoError(err)

	// Drop tables that reference users first
	_, err = s.db.Exec("DROP TABLE IF EXISTS two_factor_backup_codes")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS user_backup_codes")
	s.Require().NoError(err)

	// Now it's safe to drop the users table
	_, err = s.db.Exec("DROP TABLE IF EXISTS users")
	s.Require().NoError(err)
}

func (s *OrderServiceIntegrationTestSuite) TestCreateOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	req := &orderapi.CreateOrderRequest{
		UserId: 1, // 使用测试用户ID
		Items: []*orderapi.OrderItem{
			{
				ProductId:   1,
				ProductName: "测试商品",
				Quantity:    2,
				Price:       100.0,
			},
		},
		TotalPrice: 200.0,
	}

	resp, err := s.orderService.CreateOrder(ctx, req)
	s.NoError(err)
	s.True(resp.Success)
	s.NotZero(resp.OrderId)

	// 验证订单是否正确创建
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(resp.OrderId))
	s.NoError(err)
	s.Equal("1", order.UserID)
	s.Equal(200.0, order.TotalAmount)
	s.Equal(repository.OrderStatusPending, order.Status)
}

func (s *OrderServiceIntegrationTestSuite) TestSettleOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// 先创建一个订单
	createReq := &orderapi.CreateOrderRequest{
		UserId: 1,
		Items: []*orderapi.OrderItem{
			{
				ProductId:   1,
				ProductName: "测试商品",
				Quantity:    2,
				Price:       100.0,
			},
		},
		TotalPrice: 200.0,
	}

	createResp, err := s.orderService.CreateOrder(ctx, createReq)
	s.NoError(err)

	// 结算订单
	settleReq := &orderapi.SettleOrderRequest{
		OrderId:        createResp.OrderId,
		UserId:         1,
		PaymentMethod:  "credit_card",
	}

	settleResp, err := s.orderService.SettleOrder(ctx, settleReq)
	s.NoError(err)
	s.True(settleResp.Success)
	s.Equal(orderapi.OrderStatus_PAID, settleResp.Status)

	// 验证订单状态是否已更新
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(createResp.OrderId))
	s.NoError(err)
	s.Equal(repository.OrderStatusPaid, order.Status)
}

func (s *OrderServiceIntegrationTestSuite) TestCancelOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// 创建一个订单
	createReq := &orderapi.CreateOrderRequest{
		UserId: 1,
		Items: []*orderapi.OrderItem{
			{
				ProductId:   1,
				ProductName: "测试商品",
				Quantity:    2,
				Price:       100.0,
			},
		},
		TotalPrice: 200.0,
	}

	createResp, err := s.orderService.CreateOrder(ctx, createReq)
	s.NoError(err)

	// 取消订单
	cancelReq := &orderapi.CancelOrderRequest{
		OrderId:       createResp.OrderId,
		CancelReason: "测试取消",
	}

	cancelResp, err := s.orderService.CancelOrder(ctx, cancelReq)
	s.NoError(err)
	s.True(cancelResp.Success)

	// 验证订单状态是否已更新为取消
	order, err := s.orderRepo.Get(ctx, fmt.Sprint(createResp.OrderId))
	s.NoError(err)
	s.Equal(repository.OrderStatusCancelled, order.Status)
}

func TestOrderServiceIntegration(t *testing.T) {
	// 如果是短测试模式，跳过集成测试
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	suite.Run(t, new(OrderServiceIntegrationTestSuite))
}
