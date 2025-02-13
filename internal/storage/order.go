package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// OrderStorage описывает методы для работы с заказами.
type OrderStorage interface {
	// CreateOrder создаёт новый заказ в таблице orders с использованием транзакции.
	CreateOrder(ctx context.Context, tx *sql.Tx, userID int64, merchID int64, quantity int, totalPrice int) error
}

// orderRepository — конкретная реализация OrderStorage.
type orderRepository struct {
	db *sql.DB
}

// NewOrderRepository создаёт новый репозиторий заказов.
func NewOrderRepository(db *sql.DB) OrderStorage {
	return &orderRepository{db: db}
}

// CreateOrder вставляет новый заказ в таблицу orders.
func (r *orderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, userID int64, merchID int64, quantity int, totalPrice int) error {
	query := `INSERT INTO orders (user_id, merch_id, quantity, total_price, created_at) 
	          VALUES ($1, $2, $3, $4, NOW())`
	_, err := tx.ExecContext(ctx, query, userID, merchID, quantity, totalPrice)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	return nil
}
