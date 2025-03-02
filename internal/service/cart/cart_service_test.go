package cart

import (
	"context"
	"database/sql"
	"os"
	"testing"

	cartapi "github.com/bytedance-youthcamp/demo/api/cart"
	productapi "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockProductClient is a mock implementation of the ProductServiceClient
type MockProductClient struct {
	mock.Mock
}

func (m *MockProductClient) GetProduct(ctx context.Context, req *productapi.GetProductRequest, opts ...grpc.CallOption) (*productapi.GetProductResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*productapi.GetProductResponse), args.Error(1)
}

func (m *MockProductClient) CreateProduct(ctx context.Context, req *productapi.CreateProductRequest, opts ...grpc.CallOption) (*productapi.CreateProductResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*productapi.CreateProductResponse), args.Error(1)
}

func (m *MockProductClient) GetProducts(ctx context.Context, req *productapi.GetProductsRequest, opts ...grpc.CallOption) (*productapi.GetProductsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*productapi.GetProductsResponse), args.Error(1)
}

func (m *MockProductClient) UpdateProduct(ctx context.Context, req *productapi.UpdateProductRequest, opts ...grpc.CallOption) (*productapi.UpdateProductResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*productapi.UpdateProductResponse), args.Error(1)
}

func (m *MockProductClient) DeleteProduct(ctx context.Context, req *productapi.DeleteProductRequest, opts ...grpc.CallOption) (*productapi.DeleteProductResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*productapi.DeleteProductResponse), args.Error(1)
}

func setupTestDB(t *testing.T) *sql.DB {
	// Set test environment
	os.Setenv("GO_TEST_ENV", "true")

	// Open SQLite in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	// Create test tables
	err = createTestTables(db)
	assert.NoError(t, err)

	return db
}

func createTestCart(t *testing.T, service *CartService, userId int32) int32 {
	resp, err := service.CreateCart(context.Background(), &cartapi.CreateCartRequest{
		UserId: userId,
	})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	return resp.CartId
}

func TestCreateCart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, err := NewCartService(WithTestDatabase(db))
	assert.NoError(t, err)
	defer service.Close()

	tests := []struct {
		name     string
		userId   int32
		expected bool
	}{
		{"valid user", 1, true},
		{"invalid user", 0, false},
		{"duplicate cart", 1, true}, // Should return existing cart
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.CreateCart(context.Background(), &cartapi.CreateCartRequest{
				UserId: tt.userId,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Success)
		})
	}
}

func TestAddToCart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create mock product client
	mockProductClient := new(MockProductClient)
	service, err := NewCartService(WithTestDatabase(db))
	assert.NoError(t, err)
	defer service.Close()

	// Replace product client with mock
	service.productClient = mockProductClient

	// Create test cart
	cartId := createTestCart(t, service, 1)

	// Setup mock expectations
	mockProductClient.On("GetProduct", mock.Anything, &productapi.GetProductRequest{
		ProductId: 1,
	}).Return(&productapi.GetProductResponse{
		Success: true,
		Product: &productapi.Product{
			Id:    1,
			Name:  "Test Product",
			Price: 10.0,
			Stock: 100,
		},
	}, nil)

	tests := []struct {
		name        string
		cartId      int32
		productId   int32
		quantity    int32
		expected    bool
		errorMsg    string
	}{
		{"valid addition", cartId, 1, 5, true, ""},
		{"invalid cart", 0, 1, 5, false, "购物车ID无效"},
		{"invalid product", cartId, 0, 5, false, "商品ID无效"},
		{"invalid quantity", cartId, 1, 0, false, "商品数量必须大于0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.AddToCart(context.Background(), &cartapi.AddToCartRequest{
				CartId:    tt.cartId,
				ProductId: tt.productId,
				Quantity:  tt.quantity,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Success)
			if !tt.expected {
				assert.Equal(t, tt.errorMsg, resp.ErrorMessage)
			}
		})
	}
}

func TestGetCart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, err := NewCartService(WithTestDatabase(db))
	assert.NoError(t, err)
	defer service.Close()

	// Create test cart
	cartId := createTestCart(t, service, 1)

	tests := []struct {
		name     string
		cartId   int32
		expected bool
	}{
		{"valid cart", cartId, true},
		{"invalid cart", 0, false},
		{"non-existent cart", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetCart(context.Background(), &cartapi.GetCartRequest{
				CartId: tt.cartId,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Success)
		})
	}
}

func TestClearCart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, err := NewCartService(WithTestDatabase(db))
	assert.NoError(t, err)
	defer service.Close()

	// Create test cart
	cartId := createTestCart(t, service, 1)

	tests := []struct {
		name     string
		cartId   int32
		expected bool
	}{
		{"valid cart", cartId, true},
		{"invalid cart", 0, false},
		{"non-existent cart", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.ClearCart(context.Background(), &cartapi.ClearCartRequest{
				CartId: tt.cartId,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Success)
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create mock product client
	mockProductClient := new(MockProductClient)
	service, err := NewCartService(WithTestDatabase(db))
	assert.NoError(t, err)
	defer service.Close()

	// Replace product client with mock
	service.productClient = mockProductClient

	// Setup mock expectations
	mockProductClient.On("GetProduct", mock.Anything, &productapi.GetProductRequest{
		ProductId: 1,
	}).Return(&productapi.GetProductResponse{
		Success: true,
		Product: &productapi.Product{
			Id:    1,
			Name:  "Test Product",
			Price: 10.0,
			Stock: 100,
		},
	}, nil)

	// Create test cart and add item
	cartId := createTestCart(t, service, 1)
	addResp, err := service.AddToCart(context.Background(), &cartapi.AddToCartRequest{
		CartId:    cartId,
		ProductId: 1,
		Quantity:  1,
	})
	assert.NoError(t, err)
	assert.True(t, addResp.Success)

	// Get the cart item ID from the database
	var itemId int32
	err = db.QueryRow("SELECT id FROM cart_items WHERE cart_id = ? AND product_id = ?", cartId, 1).Scan(&itemId)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		cartId   int32
		itemId   int32
		expected bool
	}{
		{"valid removal", cartId, itemId, true},
		{"invalid cart", 0, itemId, false},
		{"invalid item", cartId, 0, false},
		{"non-existent item", cartId, 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.RemoveFromCart(context.Background(), &cartapi.RemoveFromCartRequest{
				CartId:     tt.cartId,
				CartItemId: tt.itemId,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Success)
		})
	}
}