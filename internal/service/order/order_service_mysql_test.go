package order

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	orderapi "github.com/bytedance-youthcamp/demo/api/order"
	"github.com/bytedance-youthcamp/demo/internal/repository"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-sql-driver/mysql"
)

type OrderServiceMySQLTestSuite struct {
	suite.Suite
	db           *sql.DB
	orderService *orderService
	orderRepo    repository.OrderRepository
}

func (s *OrderServiceMySQLTestSuite) SetupSuite() {
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

func (s *OrderServiceMySQLTestSuite) TearDownSuite() {
	// Drop test tables
	if s.db != nil {
		s.dropTestTables()

		// Close database connection
		s.db.Close()
	}
}

func (s *OrderServiceMySQLTestSuite) setupTestDatabase() {
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

	// Insert test product
	_, err = s.db.Exec(`
		INSERT INTO products (id, name, description, price, stock, category, image_url)
		VALUES (1, 'Test Product', 'Test Description', 99.99, 10, 'Test Category', 'http://example.com/image.jpg')
	`)
	s.Require().NoError(err)

	// Create users table for testing
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

	// Insert test user
	_, err = s.db.Exec(`
		INSERT INTO users (id, username, email, phone)
		VALUES (1, 'testuser', 'test@example.com', '1234567890')
	`)
	s.Require().NoError(err)
}

func (s *OrderServiceMySQLTestSuite) dropTestTables() {
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

	// Drop tables that reference users first
	_, err = s.db.Exec("DROP TABLE IF EXISTS two_factor_backup_codes")
	s.Require().NoError(err)

	_, err = s.db.Exec("DROP TABLE IF EXISTS user_backup_codes")
	s.Require().NoError(err)

	// Now it's safe to drop the users table
	_, err = s.db.Exec("DROP TABLE IF EXISTS users")
	s.Require().NoError(err)
}

func (s *OrderServiceMySQLTestSuite) TestCreateOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// Create a test order
	req := &orderapi.CreateOrderRequest{
		UserId: 1,
		Items: []*orderapi.OrderItem{
			{
				ProductId:   1,
				ProductName: "Test Product",
				Quantity:    2,
				Price:       99.99,
			},
		},
		TotalPrice: 199.98,
	}

	// Create the order
	resp, err := s.orderService.CreateOrder(ctx, req)
	s.Require().NoError(err)
	s.True(resp.Success)
	s.NotZero(resp.OrderId)

	// Verify the order was created in the database
	order, err := s.orderRepo.Get(ctx, strconv.Itoa(int(resp.OrderId)))
	s.Require().NoError(err)
	s.Equal("1", order.UserID)
	s.Equal(199.98, order.TotalAmount)
	s.Equal(repository.OrderStatusPending, order.Status)
	s.Len(order.Items, 1)
	s.Equal("1", order.Items[0].ProductID)
	s.Equal("Test Product", order.Items[0].ProductName)
	s.Equal(int32(2), order.Items[0].Quantity)
	s.Equal(99.99, order.Items[0].Price)
}

func (s *OrderServiceMySQLTestSuite) TestUpdateOrder() {
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

	// Update the order
	updateReq := &orderapi.UpdateOrderRequest{
		OrderId: int32(s.parseOrderID(order.ID)),
		Status:  orderapi.OrderStatus_SHIPPING,
		Address: "New Shipping Address",
	}

	updateResp, err := s.orderService.UpdateOrder(ctx, updateReq)
	s.Require().NoError(err)
	s.True(updateResp.Success)

	// Verify the order was updated
	updatedOrder, err := s.orderRepo.Get(ctx, order.ID)
	s.Require().NoError(err)
	s.Equal(repository.OrderStatusShipped, updatedOrder.Status)
	s.Equal("New Shipping Address", updatedOrder.ShippingAddress)
}

func (s *OrderServiceMySQLTestSuite) TestCancelOrder() {
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

	// Cancel the order
	cancelReq := &orderapi.CancelOrderRequest{
		OrderId:      int32(s.parseOrderID(order.ID)),
		CancelReason: "Test cancellation",
	}

	cancelResp, err := s.orderService.CancelOrder(ctx, cancelReq)
	s.Require().NoError(err)
	s.True(cancelResp.Success)

	// Verify the order was cancelled
	cancelledOrder, err := s.orderRepo.Get(ctx, order.ID)
	s.Require().NoError(err)
	s.Equal(repository.OrderStatusCancelled, cancelledOrder.Status)
}

func (s *OrderServiceMySQLTestSuite) TestAutoCancelOrder() {
	if s.db == nil {
		s.T().Skip("Skipping test due to no database connection")
	}

	ctx := context.Background()

	// Create an order with a timestamp in the past
	pastTime := time.Now().Add(-35 * time.Minute) // 35 minutes ago
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
		CreatedAt: pastTime,
		UpdatedAt: pastTime,
	}

	err := s.orderRepo.Create(ctx, order)
	s.Require().NoError(err)

	// Get pending orders older than 30 minutes
	oldOrders, err := s.orderRepo.ListPendingOrdersOlderThan(ctx, 30*time.Minute)
	s.Require().NoError(err)
	s.NotEmpty(oldOrders)

	// Verify our order is in the list
	found := false
	for _, oldOrder := range oldOrders {
		if oldOrder.ID == order.ID {
			found = true
			break
		}
	}
	s.True(found)

	// Cancel the order
	for _, oldOrder := range oldOrders {
		oldOrder.Status = repository.OrderStatusCancelled
		oldOrder.UpdatedAt = time.Now()
		err = s.orderRepo.Update(ctx, oldOrder)
		s.Require().NoError(err)
	}

	// Verify the order was cancelled
	cancelledOrder, err := s.orderRepo.Get(ctx, order.ID)
	s.Require().NoError(err)
	s.Equal(repository.OrderStatusCancelled, cancelledOrder.Status)
}

// Helper function to parse order ID from string to int
func (s *OrderServiceMySQLTestSuite) parseOrderID(id string) int {
	var orderID int
	_, err := fmt.Sscanf(id, "%d", &orderID)
	s.Require().NoError(err)
	return orderID
}

func TestOrderServiceMySQL(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite.Run(t, new(OrderServiceMySQLTestSuite))
}
