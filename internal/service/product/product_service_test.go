package product

import (
	"context"
	"database/sql"
	"testing"

	productapi "github.com/bytedance-youthcamp/demo/api/product"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

func setupTestDatabase(t *testing.T) *sql.DB {
	// 创建测试数据库连接
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to open test database")

	// 创建产品表
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			price REAL NOT NULL,
			stock INTEGER NOT NULL,
			category TEXT,
			image_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err, "Failed to create tables")

	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	})

	return db
}

func setupProductService(t *testing.T) *ProductService {
	testDB := setupTestDatabase(t)

	productService, err := NewProductService(
		WithTestDatabase(testDB),
	)
	require.NoError(t, err, "Failed to create ProductService")

	t.Cleanup(func() {
		productService.Close()
	})
	return productService
}

func createTestProduct(t *testing.T, productService *ProductService) int32 {
	name := "Test Product"
	description := "This is a test product"
	price := 19.99
	stock := int32(100)
	category := "Test Category"
	imageURL := "https://example.com/image.jpg"

	// 创建产品
	createResp, err := productService.CreateProduct(context.Background(), &productapi.CreateProductRequest{
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		Category:    category,
		ImageUrl:    imageURL,
	})
	require.NoError(t, err)
	assert.True(t, createResp.Success)
	assert.NotZero(t, createResp.ProductId)

	return createResp.ProductId
}

func TestCreateProduct(t *testing.T) {
	productService := setupProductService(t)

	testCases := []struct {
		name           string
		createReq      *productapi.CreateProductRequest
		expectedResult bool
	}{
		{
			name: "Valid Product",
			createReq: &productapi.CreateProductRequest{
				Name:        "Test Product",
				Description: "This is a test product",
				Price:       19.99,
				Stock:       100,
				Category:    "Test Category",
				ImageUrl:    "https://example.com/image.jpg",
			},
			expectedResult: true,
		},
		{
			name: "Empty Name",
			createReq: &productapi.CreateProductRequest{
				Name:        "",
				Description: "Product with empty name",
				Price:       9.99,
				Stock:       50,
				Category:    "Test Category",
				ImageUrl:    "https://example.com/image2.jpg",
			},
			expectedResult: false,
		},
		{
			name: "Negative Price",
			createReq: &productapi.CreateProductRequest{
				Name:        "Negative Price Product",
				Description: "Product with negative price",
				Price:       -9.99,
				Stock:       50,
				Category:    "Test Category",
				ImageUrl:    "https://example.com/image3.jpg",
			},
			expectedResult: false,
		},
		{
			name: "Negative Stock",
			createReq: &productapi.CreateProductRequest{
				Name:        "Negative Stock Product",
				Description: "Product with negative stock",
				Price:       9.99,
				Stock:       -50,
				Category:    "Test Category",
				ImageUrl:    "https://example.com/image4.jpg",
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := productService.CreateProduct(context.Background(), tc.createReq)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, resp.Success)
		})
	}
}

func TestGetProduct(t *testing.T) {
	productService := setupProductService(t)

	// 创建一个测试产品
	productId := createTestProduct(t, productService)

	// 测试获取存在的产品
	getResp, err := productService.GetProduct(context.Background(), &productapi.GetProductRequest{
		ProductId: productId,
	})
	require.NoError(t, err)
	assert.True(t, getResp.Success)
	assert.NotNil(t, getResp.Product)
	assert.Equal(t, productId, getResp.Product.Id)
	assert.Equal(t, "Test Product", getResp.Product.Name)
	assert.Equal(t, "This is a test product", getResp.Product.Description)
	assert.Equal(t, 19.99, getResp.Product.Price)
	assert.Equal(t, int32(100), getResp.Product.Stock)
	assert.Equal(t, "Test Category", getResp.Product.Category)
	assert.Equal(t, "https://example.com/image.jpg", getResp.Product.ImageUrl)

	// 测试获取不存在的产品
	nonExistentResp, err := productService.GetProduct(context.Background(), &productapi.GetProductRequest{
		ProductId: 9999,
	})
	require.NoError(t, err)
	assert.False(t, nonExistentResp.Success)
	assert.Equal(t, "商品不存在", nonExistentResp.ErrorMessage)
}

func TestGetProducts(t *testing.T) {
	productService := setupProductService(t)

	// 创建多个测试产品
	productIds := make([]int32, 0, 5)
	for i := 0; i < 5; i++ {
		productIds = append(productIds, createTestProduct(t, productService))
	}

	// 测试按ID列表查询
	getByIdsResp, err := productService.GetProducts(context.Background(), &productapi.GetProductsRequest{
		ProductIds: productIds,
		Page:       1,
		PageSize:   10,
	})
	require.NoError(t, err)
	assert.True(t, getByIdsResp.Success)
	assert.Equal(t, int32(5), getByIdsResp.Total)
	assert.Len(t, getByIdsResp.Products, 5)

	// 测试按分类查询
	getByCategoryResp, err := productService.GetProducts(context.Background(), &productapi.GetProductsRequest{
		Category: "Test Category",
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.True(t, getByCategoryResp.Success)
	assert.Equal(t, int32(5), getByCategoryResp.Total)
	assert.Len(t, getByCategoryResp.Products, 5)

	// 测试分页
	getPagedResp, err := productService.GetProducts(context.Background(), &productapi.GetProductsRequest{
		Category: "Test Category",
		Page:     1,
		PageSize: 2,
	})
	require.NoError(t, err)
	assert.True(t, getPagedResp.Success)
	assert.Equal(t, int32(5), getPagedResp.Total)
	assert.Len(t, getPagedResp.Products, 2)
}

func TestUpdateProduct(t *testing.T) {
	productService := setupProductService(t)

	// 创建一个测试产品
	productId := createTestProduct(t, productService)

	// 更新产品信息
	updateResp, err := productService.UpdateProduct(context.Background(), &productapi.UpdateProductRequest{
		ProductId:   productId,
		Name:        "Updated Product",
		Description: "This is an updated product",
		Price:       29.99,
		Stock:       200,
		Category:    "Updated Category",
		ImageUrl:    "https://example.com/updated-image.jpg",
	})
	require.NoError(t, err)
	assert.True(t, updateResp.Success)

	// 验证更新后的产品信息
	getResp, err := productService.GetProduct(context.Background(), &productapi.GetProductRequest{
		ProductId: productId,
	})
	require.NoError(t, err)
	assert.True(t, getResp.Success)
	assert.NotNil(t, getResp.Product)
	assert.Equal(t, productId, getResp.Product.Id)
	assert.Equal(t, "Updated Product", getResp.Product.Name)
	assert.Equal(t, "This is an updated product", getResp.Product.Description)
	assert.Equal(t, 29.99, getResp.Product.Price)
	assert.Equal(t, int32(200), getResp.Product.Stock)
	assert.Equal(t, "Updated Category", getResp.Product.Category)
	assert.Equal(t, "https://example.com/updated-image.jpg", getResp.Product.ImageUrl)

	// 测试更新不存在的产品
	nonExistentResp, err := productService.UpdateProduct(context.Background(), &productapi.UpdateProductRequest{
		ProductId:   9999,
		Name:        "Non-existent Product",
		Description: "This product doesn't exist",
		Price:       9.99,
		Stock:       50,
		Category:    "Test Category",
		ImageUrl:    "https://example.com/image.jpg",
	})
	require.NoError(t, err)
	assert.False(t, nonExistentResp.Success)
	assert.Equal(t, "商品不存在", nonExistentResp.ErrorMessage)
}

func TestDeleteProduct(t *testing.T) {
	productService := setupProductService(t)

	// 创建一个测试产品
	productId := createTestProduct(t, productService)

	// 删除产品
	deleteResp, err := productService.DeleteProduct(context.Background(), &productapi.DeleteProductRequest{
		ProductId: productId,
	})
	require.NoError(t, err)
	assert.True(t, deleteResp.Success)

	// 验证产品已被删除
	getResp, err := productService.GetProduct(context.Background(), &productapi.GetProductRequest{
		ProductId: productId,
	})
	require.NoError(t, err)
	assert.False(t, getResp.Success)
	assert.Equal(t, "商品不存在", getResp.ErrorMessage)

	// 测试删除不存在的产品
	nonExistentResp, err := productService.DeleteProduct(context.Background(), &productapi.DeleteProductRequest{
		ProductId: 9999,
	})
	require.NoError(t, err)
	assert.False(t, nonExistentResp.Success)
	assert.Equal(t, "商品不存在", nonExistentResp.ErrorMessage)
}
