package product

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	productapi "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/bytedance-youthcamp/demo/internal/config"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type ProductServiceOption func(*ProductService) error

func WithTestDatabase(db *sql.DB) ProductServiceOption {
	return func(ps *ProductService) error {
		ps.db = db
		return nil
	}
}

func NewProductService(opts ...ProductServiceOption) (*ProductService, error) {
	// 加载产品配置
	configPath := "/Users/Apple/Desktop/demo/configs/product.yaml"
	productConfig, err := config.LoadProductConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load product config: %v", err)
	}

	// 打开数据库连接
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		productConfig.Database.User,
		productConfig.Database.Password,
		productConfig.Database.Host,
		productConfig.Database.Port,
		productConfig.Database.Name)
	driver := "postgres"

	// 如果是测试环境，使用SQLite
	if os.Getenv("GO_TEST_ENV") == "true" {
		dbURL = ":memory:"
		driver = "sqlite3"
	} else if os.Getenv("TEST_DB_URL") != "" {
		dbURL = os.Getenv("TEST_DB_URL")
	}

	db, err := sql.Open(driver, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 如果是测试环境，创建表
	if os.Getenv("GO_TEST_ENV") == "true" {
		err = createTestTables(db)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create test tables: %v", err)
		}
	}

	productService := &ProductService{
		db:     db,
		config: productConfig,
	}

	// 应用测试选项
	for _, opt := range opts {
		if err := opt(productService); err != nil {
			productService.Close()
			return nil, err
		}
	}

	return productService, nil
}

func createTestTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
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
	return err
}

type ProductService struct {
	db     *sql.DB
	config *config.ProductConfig
}

func (s *ProductService) Close() {
	if s.db != nil {
		// 检查是否为测试环境
		if os.Getenv("GO_TEST_ENV") != "true" {
			s.db.Close()
		}
	}
}

// GetProduct 获取单个商品信息
func (s *ProductService) GetProduct(ctx context.Context, req *productapi.GetProductRequest) (*productapi.GetProductResponse, error) {
	query := `
		SELECT id, name, description, price, stock, category, image_url, 
		       created_at, updated_at 
		FROM products 
		WHERE id = $1
	`

	var product productapi.Product
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, req.ProductId).Scan(
		&product.Id,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Stock,
		&product.Category,
		&product.ImageUrl,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return &productapi.GetProductResponse{
			Success:      false,
			ErrorMessage: "商品不存在",
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	product.CreatedAt = createdAt.Format(time.RFC3339)
	product.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &productapi.GetProductResponse{
		Product: &product,
		Success: true,
	}, nil
}

// GetProducts 批量获取商品信息
func (s *ProductService) GetProducts(ctx context.Context, req *productapi.GetProductsRequest) (*productapi.GetProductsResponse, error) {
	// 设置默认分页参数
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = int32(s.config.Product.DefaultPageSize)
	}
	if pageSize > int32(s.config.Product.MaxQueryLimit) {
		pageSize = int32(s.config.Product.MaxQueryLimit)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * pageSize

	// 构建查询
	var args []interface{}
	var whereClause string

	// 按ID列表查询
	if len(req.ProductIds) > 0 {
		whereClause = "WHERE id IN ("
		for i, id := range req.ProductIds {
			if i > 0 {
				whereClause += ","
			}
			whereClause += fmt.Sprintf("$%d", len(args)+1)
			args = append(args, id)
		}
		whereClause += ")"
	} else if req.Category != "" {
		// 按分类查询
		whereClause = "WHERE category = $1"
		args = append(args, req.Category)
	}

	// 计算总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", whereClause)
	var total int32
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// 查询商品
	query := fmt.Sprintf(`
		SELECT id, name, description, price, stock, category, image_url, 
		       created_at, updated_at 
		FROM products 
		%s 
		ORDER BY id 
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []*productapi.Product
	for rows.Next() {
		var product productapi.Product
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&product.Id,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Stock,
			&product.Category,
			&product.ImageUrl,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		product.CreatedAt = createdAt.Format(time.RFC3339)
		product.UpdatedAt = updatedAt.Format(time.RFC3339)
		products = append(products, &product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return &productapi.GetProductsResponse{
		Products: products,
		Total:    total,
		Success:  true,
	}, nil
}

// CreateProduct 创建商品
func (s *ProductService) CreateProduct(ctx context.Context, req *productapi.CreateProductRequest) (*productapi.CreateProductResponse, error) {
	// 验证商品信息
	if req.Name == "" {
		return &productapi.CreateProductResponse{
			Success:      false,
			ErrorMessage: "商品名称不能为空",
		}, nil
	}

	if req.Price < 0 {
		return &productapi.CreateProductResponse{
			Success:      false,
			ErrorMessage: "商品价格不能为负数",
		}, nil
	}

	if req.Stock < 0 {
		return &productapi.CreateProductResponse{
			Success:      false,
			ErrorMessage: "商品库存不能为负数",
		}, nil
	}

	// 插入商品到数据库
	query := `
		INSERT INTO products (name, description, price, stock, category, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	var productID int32

	err := s.db.QueryRowContext(ctx, query,
		req.Name,
		req.Description,
		req.Price,
		req.Stock,
		req.Category,
		req.ImageUrl,
		now,
		now,
	).Scan(&productID)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &productapi.CreateProductResponse{
		ProductId: productID,
		Success:   true,
	}, nil
}

// UpdateProduct 更新商品信息
func (s *ProductService) UpdateProduct(ctx context.Context, req *productapi.UpdateProductRequest) (*productapi.UpdateProductResponse, error) {
	// 验证商品是否存在
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", req.ProductId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check product existence: %w", err)
	}

	if !exists {
		return &productapi.UpdateProductResponse{
			Success:      false,
			ErrorMessage: "商品不存在",
		}, nil
	}

	// 验证商品信息
	if req.Price < 0 {
		return &productapi.UpdateProductResponse{
			Success:      false,
			ErrorMessage: "商品价格不能为负数",
		}, nil
	}

	if req.Stock < 0 {
		return &productapi.UpdateProductResponse{
			Success:      false,
			ErrorMessage: "商品库存不能为负数",
		}, nil
	}

	// 更新商品信息
	query := `
		UPDATE products
		SET name = $1, description = $2, price = $3, stock = $4, category = $5, image_url = $6, updated_at = $7
		WHERE id = $8
	`

	now := time.Now()
	_, err = s.db.ExecContext(ctx, query,
		req.Name,
		req.Description,
		req.Price,
		req.Stock,
		req.Category,
		req.ImageUrl,
		now,
		req.ProductId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return &productapi.UpdateProductResponse{
		Success: true,
	}, nil
}

// DeleteProduct 删除商品
func (s *ProductService) DeleteProduct(ctx context.Context, req *productapi.DeleteProductRequest) (*productapi.DeleteProductResponse, error) {
	// 验证商品是否存在
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", req.ProductId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check product existence: %w", err)
	}

	if !exists {
		return &productapi.DeleteProductResponse{
			Success:      false,
			ErrorMessage: "商品不存在",
		}, nil
	}

	// 删除商品
	_, err = s.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", req.ProductId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	return &productapi.DeleteProductResponse{
		Success: true,
	}, nil
}
