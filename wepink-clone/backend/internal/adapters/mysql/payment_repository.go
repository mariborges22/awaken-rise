package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/awaken-rise/backend/internal/domain/entity"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *PaymentRepository) Save(ctx context.Context, payment *entity.Payment) error {
	exec := r.getExecutor(ctx)

	query := `INSERT INTO payments (id, order_id, amount, status, idempotency_key, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE status = VALUES(status), updated_at = VALUES(updated_at)`
	
	_, err := exec.ExecContext(ctx, query, payment.ID, payment.OrderID, payment.Amount, payment.Status, payment.IdempotencyKey, payment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	
	payment := &entity.Payment{}
	err := exec.QueryRowContext(ctx, "SELECT id, order_id, amount, status, idempotency_key, created_at, updated_at FROM payments WHERE id = ?", id).
		Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Status, &payment.IdempotencyKey, &payment.CreatedAt, &payment.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment: %w", err)
	}

	return payment, nil
}

func (r *PaymentRepository) FindByOrderID(ctx context.Context, orderID string) ([]*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	
	rows, err := exec.QueryContext(ctx, "SELECT id, order_id, amount, status, idempotency_key, created_at, updated_at FROM payments WHERE order_id = ?", orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments for order: %w", err)
	}
	defer rows.Close()

	var payments []*entity.Payment
	for rows.Next() {
		p := &entity.Payment{}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Status, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	return payments, nil
}
func (r *PaymentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	
	payment := &entity.Payment{}
	err := exec.QueryRowContext(ctx, "SELECT id, order_id, amount, status, idempotency_key, created_at, updated_at FROM payments WHERE idempotency_key = ?", key).
		Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Status, &payment.IdempotencyKey, &payment.CreatedAt, &payment.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment by idempotency key: %w", err)
	}

	return payment, nil
}
