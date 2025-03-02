package cart

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	cartapi "github.com/bytedance-youthcamp/demo/api/cart"
	productapi "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/bytedance-youthcamp/demo/internal/config"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
)

type CartServiceOption func(*CartService) error

func WithTestDatabase(db *sql.DB) CartServiceOption {
	return func(cs *CartService) error {
		cs.db = db
		return nil
	}
}

func NewCartService(opts ...CartServiceOption) (*CartService, error) {
	// 加载购物车配置
	configPath := "/Users/Apple/Desktop/demo/configs/cart.yaml"
	cartConfig, err := config.LoadCartConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart config: %v", err)
	}

	// 打开数据库连接
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cartConfig.Database.User,
		cartConfig.Database.Password,
		cartConfig.Database.Host,
		cartConfig.Database.Port,
		cartConfig.Database.Name)
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

	// 连接到产品服务
	// 在实际环境中，这里应该使用服务发现来获取产品服务的地址
	// 为简化实现，这里直接连接到本地的产品服务
	productConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to product service: %v", err)
	}
	productClient := productapi.NewProductServiceClient(productConn)

	cartService := &CartService{
		db:            db,
		config:        cartConfig,
		productClient: productClient,
		productConn:   productConn,
	}

	// 应用测试选项
	for _, opt := range opts {
		if err := opt(cartService); err != nil {
			cartService.Close()
			return nil, err
		}
	}

	return cartService, nil
}

func createTestTables(db *sql.DB) error {
	// 创建购物车表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS carts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			total_price REAL DEFAULT 0,
			total_quantity INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// 创建购物车商品表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cart_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cart_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			product_name TEXT NOT NULL,
			price REAL NOT NULL,
			quantity INTEGER NOT NULL,
			image_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE
		);
	`)
	return err
}

type CartService struct {
	cartapi.UnimplementedCartServiceServer
	db            *sql.DB
	config        *config.CartConfig
	productClient productapi.ProductServiceClient
	productConn   *grpc.ClientConn
}

func (s *CartService) Close() {
	if s.db != nil {
		// 检查是否为测试环境
		if os.Getenv("GO_TEST_ENV") != "true" {
			s.db.Close()
		}
	}
	if s.productConn != nil {
		s.productConn.Close()
	}
}

// CreateCart 创建购物车
func (s *CartService) CreateCart(ctx context.Context, req *cartapi.CreateCartRequest) (*cartapi.CreateCartResponse, error) {
	if req.UserId <= 0 {
		return &cartapi.CreateCartResponse{
			Success:      false,
			ErrorMessage: "用户ID无效",
		}, nil
	}

	// 检查用户是否已有购物车
	var cartId int32
	query := "SELECT id FROM carts WHERE user_id = $1 LIMIT 1"
	err := s.db.QueryRowContext(ctx, query, req.UserId).Scan(&cartId)
	if err == nil {
		// 用户已有购物车，直接返回
		return &cartapi.CreateCartResponse{
			CartId:  cartId,
			Success: true,
		}, nil
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing cart: %w", err)
	}

	// 创建新购物车
	query = "INSERT INTO carts (user_id, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id"
	now := time.Now()
	err = s.db.QueryRowContext(ctx, query, req.UserId, now, now).Scan(&cartId)
	if err != nil {
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	return &cartapi.CreateCartResponse{
		CartId:  cartId,
		Success: true,
	}, nil
}

// ClearCart 清空购物车
func (s *CartService) ClearCart(ctx context.Context, req *cartapi.ClearCartRequest) (*cartapi.ClearCartResponse, error) {
	if req.CartId <= 0 {
		return &cartapi.ClearCartResponse{
			Success:      false,
			ErrorMessage: "购物车ID无效",
		}, nil
	}

	// 检查购物车是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM carts WHERE id = $1)"
	err := s.db.QueryRowContext(ctx, query, req.CartId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check cart existence: %w", err)
	}
	if !exists {
		return &cartapi.ClearCartResponse{
			Success:      false,
			ErrorMessage: "购物车不存在",
		}, nil
	}

	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 删除购物车中的所有商品
	_, err = tx.ExecContext(ctx, "DELETE FROM cart_items WHERE cart_id = $1", req.CartId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete cart items: %w", err)
	}

	// 更新购物车总价和总数量
	_, err = tx.ExecContext(ctx, 
		"UPDATE carts SET total_price = 0, total_quantity = 0, updated_at = $1 WHERE id = $2", 
		time.Now(), req.CartId)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &cartapi.ClearCartResponse{
		Success: true,
	}, nil
}

// GetCart 获取购物车信息
func (s *CartService) GetCart(ctx context.Context, req *cartapi.GetCartRequest) (*cartapi.GetCartResponse, error) {
	if req.CartId <= 0 {
		return &cartapi.GetCartResponse{
			Success:      false,
			ErrorMessage: "购物车ID无效",
		}, nil
	}

	// 查询购物车基本信息
	var cart cartapi.Cart
	var createdAt, updatedAt time.Time
	query := `
		SELECT id, user_id, total_price, total_quantity, created_at, updated_at 
		FROM carts 
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, req.CartId).Scan(
		&cart.Id,
		&cart.UserId,
		&cart.TotalPrice,
		&cart.TotalQuantity,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return &cartapi.GetCartResponse{
			Success:      false,
			ErrorMessage: "购物车不存在",
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	cart.CreatedAt = createdAt.Format(time.RFC3339)
	cart.UpdatedAt = updatedAt.Format(time.RFC3339)

	// 查询购物车商品
	query = `
		SELECT id, product_id, product_name, price, quantity, image_url, created_at 
		FROM cart_items 
		WHERE cart_id = $1
	`
	rows, err := s.db.QueryContext(ctx, query, req.CartId)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item cartapi.CartItem
		var itemCreatedAt time.Time
		err := rows.Scan(
			&item.Id,
			&item.ProductId,
			&item.ProductName,
			&item.Price,
			&item.Quantity,
			&item.ImageUrl,
			&itemCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cart item: %w", err)
		}
		item.CreatedAt = itemCreatedAt.Format(time.RFC3339)
		cart.Items = append(cart.Items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cart items: %w", err)
	}

	return &cartapi.GetCartResponse{
		Cart:    &cart,
		Success: true,
	}, nil
}

// AddToCart 添加商品到购物车
func (s *CartService) AddToCart(ctx context.Context, req *cartapi.AddToCartRequest) (*cartapi.AddToCartResponse, error) {
	if req.CartId <= 0 {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "购物车ID无效",
		}, nil
	}

	if req.ProductId <= 0 {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "商品ID无效",
		}, nil
	}

	if req.Quantity <= 0 {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "商品数量必须大于0",
		}, nil
	}

	// 检查购物车是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM carts WHERE id = $1)"
	err := s.db.QueryRowContext(ctx, query, req.CartId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check cart existence: %w", err)
	}
	if !exists {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "购物车不存在",
		}, nil
	}

	// 获取商品信息
	productResp, err := s.productClient.GetProduct(ctx, &productapi.GetProductRequest{
		ProductId: req.ProductId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get product info: %w", err)
	}
	if !productResp.Success {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "商品不存在",
		}, nil
	}

	// 检查库存
	if productResp.Product.Stock < req.Quantity {
		return &cartapi.AddToCartResponse{
			Success:      false,
			ErrorMessage: "商品库存不足",
		}, nil
	}

	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 检查购物车中是否已有该商品
	var existingItemId int32
	var existingQuantity int32
	query = "SELECT id, quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2"
	err = tx.QueryRowContext(ctx, query, req.CartId, req.ProductId).Scan(&existingItemId, &existingQuantity)

	var itemId int32
	if err == sql.ErrNoRows {
		// 商品不在购物车中，添加新商品
	query = `
		INSERT INTO cart_items (cart_id, product_id, product_name, price, quantity, image_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
		err = tx.QueryRowContext(ctx, query,
			req.CartId,
			req.ProductId,
			productResp.Product.Name,
			productResp.Product.Price,
			req.Quantity,
			productResp.Product.ImageUrl,
			time.Now(),
		).Scan(&itemId)
		if err != nil {
			return nil, fmt.Errorf("failed to add item to cart: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing item: %w", err)
	} else {
		// 更新已有商品的数量
		newQuantity := existingQuantity + req.Quantity
		_, err = tx.ExecContext(ctx,
			"UPDATE cart_items SET quantity = $1 WHERE id = $2",
			newQuantity, existingItemId)
		if err != nil {
			return nil, fmt.Errorf("failed to update item quantity: %w", err)
		}
		itemId = existingItemId
	}

	// 更新购物车总价和总数量
	query = `
		UPDATE carts
		SET total_price = (SELECT SUM(price * quantity) FROM cart_items WHERE cart_id = $1),
		    total_quantity = (SELECT SUM(quantity) FROM cart_items WHERE cart_id = $1),
		    updated_at = $2
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, query, req.CartId, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &cartapi.AddToCartResponse{
		Success: true,
	}, nil
}
// RemoveFromCart 从购物车中移除商品
func (s *CartService) RemoveFromCart(ctx context.Context, req *cartapi.RemoveFromCartRequest) (*cartapi.RemoveFromCartResponse, error) {
	if req.CartId <= 0 {
		return &cartapi.RemoveFromCartResponse{
			Success:      false,
			ErrorMessage: "购物车ID无效",
		}, nil
	}

	if req.CartItemId <= 0 {
		return &cartapi.RemoveFromCartResponse{
			Success:      false,
			ErrorMessage: "商品项ID无效",
		}, nil
	}

	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 检查商品项是否存在于购物车中
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM cart_items WHERE id = $1 AND cart_id = $2)"
	err = tx.QueryRowContext(ctx, query, req.CartItemId, req.CartId).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check cart item existence: %w", err)
	}
	if !exists {
		return &cartapi.RemoveFromCartResponse{
			Success:      false,
			ErrorMessage: "商品项不存在于购物车中",
		}, nil
	}

	// 删除商品项
	_, err = tx.ExecContext(ctx, "DELETE FROM cart_items WHERE id = $1", req.CartItemId)
	if err != nil {
		return nil, fmt.Errorf("failed to remove item from cart: %w", err)
	}

	// 更新购物车总价和总数量
	query = `
		UPDATE carts
		SET total_price = (SELECT COALESCE(SUM(price * quantity), 0) FROM cart_items WHERE cart_id = $1),
		    total_quantity = (SELECT COALESCE(SUM(quantity), 0) FROM cart_items WHERE cart_id = $1),
		    updated_at = $2
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, query, req.CartId, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &cartapi.RemoveFromCartResponse{
		Success: true,
	}, nil
}