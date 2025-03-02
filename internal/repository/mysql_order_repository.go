package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLOrderRepository struct {
	db *sql.DB
}

func NewMySQLOrderRepository(db *sql.DB) *MySQLOrderRepository {
	return &MySQLOrderRepository{db: db}
}

// Create inserts a new order into the database
func (r *MySQLOrderRepository) Create(ctx context.Context, order *Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Insert order
	var orderID int64
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to marshal order items: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO orders (user_id, total_amount, status, shipping_address, items, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		order.UserID,
		order.TotalAmount,
		order.Status,
		order.ShippingAddress,
		itemsJSON,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert order: %w", err)
	}

	orderID, err = result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Set the order ID
	order.ID = fmt.Sprintf("%d", orderID)
	return nil
}

// Get retrieves an order by ID
func (r *MySQLOrderRepository) Get(ctx context.Context, orderID string) (*Order, error) {
	// Get order details
	var order Order
	var statusInt int
	var itemsJSON []byte

	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, user_id, total_amount, status, shipping_address, items, created_at, updated_at FROM orders WHERE id = ?",
		orderID,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.TotalAmount,
		&statusInt,
		&order.ShippingAddress,
		&itemsJSON,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	order.Status = OrderStatus(statusInt)

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order items: %w", err)
	}

	return &order, nil
}

// Update updates an existing order
func (r *MySQLOrderRepository) Update(ctx context.Context, order *Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Update order
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to marshal order items: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE orders SET status = ?, shipping_address = ?, items = ?, updated_at = ? WHERE id = ?",
		order.Status,
		order.ShippingAddress,
		itemsJSON,
		order.UpdatedAt,
		order.ID,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// List retrieves orders for a user with optional status filter
func (r *MySQLOrderRepository) List(ctx context.Context, userID string, status *OrderStatus) ([]*Order, error) {
	query := "SELECT id FROM orders WHERE user_id = ?"
	args := []interface{}{userID}

	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orderIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan order ID: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order IDs: %w", err)
	}

	orders := make([]*Order, 0, len(orderIDs))
	for _, id := range orderIDs {
		order, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// ListPendingOrdersOlderThan retrieves pending orders older than the specified duration
func (r *MySQLOrderRepository) ListPendingOrdersOlderThan(ctx context.Context, duration time.Duration) ([]*Order, error) {
	cutoffTime := time.Now().Add(-duration)

	rows, err := r.db.QueryContext(
		ctx,
		"SELECT id FROM orders WHERE status = ? AND created_at < ?",
		OrderStatusPending,
		cutoffTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending orders: %w", err)
	}
	defer rows.Close()

	var orderIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan order ID: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order IDs: %w", err)
	}

	orders := make([]*Order, 0, len(orderIDs))
	for _, id := range orderIDs {
		order, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetUserOrders retrieves paginated orders for a user with status filter
func (r *MySQLOrderRepository) GetUserOrders(ctx context.Context, userID string, page, pageSize int, status OrderStatus) ([]*Order, int, error) {
	// Calculate total count
	var total int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM orders WHERE user_id = ? AND status = ?",
		userID,
		status,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// If no orders, return early
	if total == 0 {
		return []*Order{}, 0, nil
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get paginated order IDs
	rows, err := r.db.QueryContext(
		ctx,
		"SELECT id FROM orders WHERE user_id = ? AND status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		userID,
		status,
		pageSize,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list paginated orders: %w", err)
	}
	defer rows.Close()

	var orderIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order ID: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating order IDs: %w", err)
	}

	orders := make([]*Order, 0, len(orderIDs))
	for _, id := range orderIDs {
		order, err := r.Get(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}

	return orders, total, nil
}