package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *OrderRepository) Save(ctx context.Context, order *entity.Order) error {
	exec := r.getExecutor(ctx)
	
	tenantID, _ := kernel.GetTenantID(ctx)
	if tenantID != "" {
		order.TenantID = tenantID
	}

	query := `INSERT INTO orders (id, tenant_id, status, total, updated_at) 
			  VALUES (?, ?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE status = VALUES(status), total = VALUES(total), updated_at = VALUES(updated_at)`
	
	_, err := exec.ExecContext(ctx, query, order.ID, order.TenantID, order.Status, order.Total, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	// Save items
	_, _ = exec.ExecContext(ctx, "DELETE FROM order_items WHERE order_id = ?", order.ID)
	
	for _, item := range order.Items {
		_, err := exec.ExecContext(ctx, "INSERT INTO order_items (order_id, product_id, quantity, price) VALUES (?, ?, ?, ?)",
			order.ID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("failed to save order item: %w", err)
		}
	}

	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)
	
	query := "SELECT id, tenant_id, status, total, created_at, updated_at FROM orders WHERE id = ?"
	args := []interface{}{id}

	if tenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, tenantID)
	}

	order := &entity.Order{}
	err := exec.QueryRowContext(ctx, query, args...).
		Scan(&order.ID, &order.TenantID, &order.Status, &order.Total, &order.CreatedAt, &order.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	rows, err := exec.QueryContext(ctx, "SELECT product_id, quantity, price FROM order_items WHERE order_id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item entity.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}
