package order

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	orderapi "github.com/bytedance-youthcamp/demo/api/order"
	// productpb "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/bytedance-youthcamp/demo/internal/repository"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-sql-driver/mysql"
)

type OrderServiceSettleTestSuite struct {
	suite.Suite
	db           *sql.DB
	orderService *orderService
	orderRepo    repository.OrderRepository
}

func (s *OrderServiceSettleTestSuite) SetupSuite() {
	// Connect to MySQL
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/testdb?parseTime=true")
	s.Require().NoError(err)

	// Test connection
	err = db.Ping()
	if err != nil {
		s.T().Skipf("Skipping test: could not connect to database: %v", err)
		return
	}

	s.db = db

	// Create test tables
	s.setupTestDatabase()

	// Create repository
	s.orderRepo = repository.NewMySQLOrderRepository(db)

	// Create service
	s.orderService, err = NewOrderService(
		WithOrderRepository(s.orderRepo),
		WithTestDatabase(db),
	)
	s.Require().NoError(err)
}

func (s *OrderServiceSettleTestSuite) TearDownSuite() {
	// Drop test tables
	if s.db != nil {
		s.dropTestTables()

		// Close database connection
		s.db.Close()
	}
}

func (s *OrderServiceSettleTestSuite) setupTestDatabase() {
	// Drop tables if they exist
	s.dropTestTables()

	// Create orders table
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

	// Create order_items table
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

	// Create products table for testing
	_, err = s.db.Exec(`
		CREATE TABLE products (
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

	// Insert test product
	_, err = s.db.Exec(`
		INSERT INTO products (id, name, description, price, stock, category, image_url)
		VALUES (1, 'Test Product', 'Test Description', 99.99, 10, 'Test Category', 'http://example.com/image.jpg')
	`)
	s.Require().NoError(err)
}

func (s *OrderServiceSettleTestSuite) dropTestTables() {
	// Drop tables in reverse order to avoid foreign key constraints
	_, err := s.db.Exec("DROP TABLE IF EXISTS order_items")
	s.Require().NoError(err)

	// First check if payments table exists and drop it
	_, err = s.db.Exec("DROP TABLE IF EXISTS payments")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS orders")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS products")
	s.Require().NoError(err)
}

func (s *OrderServiceSettleTestSuite) TestSettleOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// First create an order
	order := &repository.Order{
		UserID:      "1",
		TotalAmount: 199.98,
		Status:      repository.OrderStatusPending,
		Items: []*repository.OrderItem{
			{
				ProductID:   "1",
				ProductName: "Test Product",
				Quantity:    2,
				Price:       99.99,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.orderRepo.Create(ctx, order)
	s.Require().NoError(err)
	s.NotEmpty(order.ID)

	// Update product stock in database to ensure there's enough for the test
	_, err = s.db.Exec("UPDATE products SET stock = 10 WHERE id = 1")
	s.Require().NoError(err)

	// Settle the order
	settleReq := &orderapi.SettleOrderRequest{
		OrderId:       int32(s.parseOrderID(order.ID)),
		UserId:        1,
		PaymentMethod: "credit_card",
	}

	settleResp, err := s.orderService.SettleOrder(ctx, settleReq)
	s.Require().NoError(err)
	s.True(settleResp.Success)
	s.Equal(orderapi.OrderStatus_PAID, settleResp.Status)

	// Verify the order was updated in the database
	updatedOrder, err := s.orderRepo.Get(ctx, order.ID)
	s.Require().NoError(err)
	s.Equal(repository.OrderStatusPaid, updatedOrder.Status)

	// Verify the product stock was reduced
	var newStock int
	err = s.db.QueryRow("SELECT stock FROM products WHERE id = 1").Scan(&newStock)
	s.Require().NoError(err)
	s.Equal(8, newStock) // 10 - 2 = 8
}

func (s *OrderServiceSettleTestSuite) TestSettleOrderInsufficientStock() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// First create an order
	order := &repository.Order{
		UserID:      "1",
		TotalAmount: 999.90,
		Status:      repository.OrderStatusPending,
		Items: []*repository.OrderItem{
			{
				ProductID:   "1",
				ProductName: "Test Product",
				Quantity:    10, // Order for 10 items
				Price:       99.99,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.orderRepo.Create(ctx, order)
	s.Require().NoError(err)
	s.NotEmpty(order.ID)

	// Set product stock to insufficient amount
	_, err = s.db.Exec("UPDATE products SET stock = 5 WHERE id = 1")
	s.Require().NoError(err)

	// Settle the order
	settleReq := &orderapi.SettleOrderRequest{
		OrderId:       int32(s.parseOrderID(order.ID)),
		UserId:        1,
		PaymentMethod: "credit_card",
	}

	settleResp, err := s.orderService.SettleOrder(ctx, settleReq)
	s.Require().NoError(err)
	s.False(settleResp.Success)
	s.Equal("Insufficient stock", settleResp.ErrorMessage)

	// Verify the order was not updated in the database
	updatedOrder, err := s.orderRepo.Get(ctx, order.ID)
	s.Require().NoError(err)
	s.Equal(repository.OrderStatusPending, updatedOrder.Status)

	// Verify the product stock was not changed
	var stock int
	err = s.db.QueryRow("SELECT stock FROM products WHERE id = 1").Scan(&stock)
	s.Require().NoError(err)
	s.Equal(5, stock)
}

// Helper function to parse order ID from string to int
func (s *OrderServiceSettleTestSuite) parseOrderID(id string) int {
	var orderID int
	_, err := fmt.Sscanf(id, "%d", &orderID)
	s.Require().NoError(err)
	return orderID
}

func TestOrderServiceSettle(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite.Run(t, new(OrderServiceSettleTestSuite))
}