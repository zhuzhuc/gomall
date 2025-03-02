package order

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// SetupMySQLDatabase initializes the MySQL database for the order service
func SetupMySQLDatabase(dsn string) (*sql.DB, error) {
	// Connect to MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	// Create tables
	if err := createOrderTables(db); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("MySQL database setup completed successfully")
	return db, nil
}

// createOrderTables creates the necessary tables for the order service
func createOrderTables(db *sql.DB) error {
	// Create orders table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL,
			total_amount DECIMAL(10, 2) NOT NULL,
			status INT NOT NULL,
			shipping_address TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	// Create order_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS order_items (
			id INT AUTO_INCREMENT PRIMARY KEY,
			order_id INT NOT NULL,
			product_id VARCHAR(50) NOT NULL,
			product_name VARCHAR(255) NOT NULL,
			quantity INT NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create order_items table: %w", err)
	}

	return nil
}