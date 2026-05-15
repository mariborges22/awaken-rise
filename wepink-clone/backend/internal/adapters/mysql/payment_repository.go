package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
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

	query := `INSERT INTO payments (id, order_id, transaction_id, amount, status, idempotency_key, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?) 
			  ON DUPLICATE KEY UPDATE status = VALUES(status), transaction_id = VALUES(transaction_id), updated_at = VALUES(updated_at)`
	
	_, err := exec.ExecContext(ctx, query, payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status, payment.IdempotencyKey, payment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)
	
	query := `SELECT p.id, p.order_id, p.transaction_id, p.amount, p.status, p.idempotency_key, p.created_at, p.updated_at 
			  FROM payments p 
			  INNER JOIN orders o ON p.order_id = o.id 
			  WHERE p.id = ?`
	args := []interface{}{id}

	if tenantID != "" {
		query += " AND o.tenant_id = ?"
		args = append(args, tenantID)
	}

	payment := &entity.Payment{}
	err := exec.QueryRowContext(ctx, query, args...).
		Scan(&payment.ID, &payment.OrderID, &payment.TransactionID, &payment.Amount, &payment.Status, &payment.IdempotencyKey, &payment.CreatedAt, &payment.UpdatedAt)
	
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
	tenantID, _ := kernel.GetTenantID(ctx)
	
	query := `SELECT p.id, p.order_id, p.transaction_id, p.amount, p.status, p.idempotency_key, p.created_at, p.updated_at 
			  FROM payments p 
			  INNER JOIN orders o ON p.order_id = o.id 
			  WHERE p.order_id = ?`
	args := []interface{}{orderID}

	if tenantID != "" {
		query += " AND o.tenant_id = ?"
		args = append(args, tenantID)
	}

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments for order: %w", err)
	}
	defer rows.Close()

	var payments []*entity.Payment
	for rows.Next() {
		p := &entity.Payment{}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	return payments, nil
}

func (r *PaymentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)
	
	query := `SELECT p.id, p.order_id, p.transaction_id, p.amount, p.status, p.idempotency_key, p.created_at, p.updated_at 
			  FROM payments p 
			  INNER JOIN orders o ON p.order_id = o.id 
			  WHERE p.idempotency_key = ?`
	args := []interface{}{key}

	if tenantID != "" {
		query += " AND o.tenant_id = ?"
		args = append(args, tenantID)
	}

	payment := &entity.Payment{}
	err := exec.QueryRowContext(ctx, query, args...).
		Scan(&payment.ID, &payment.OrderID, &payment.TransactionID, &payment.Amount, &payment.Status, &payment.IdempotencyKey, &payment.CreatedAt, &payment.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment by idempotency key: %w", err)
	}

	return payment, nil
}

func (r *PaymentRepository) FindByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error) {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)
	
	query := `SELECT p.id, p.order_id, p.transaction_id, p.amount, p.status, p.idempotency_key, p.created_at, p.updated_at 
			  FROM payments p 
			  INNER JOIN orders o ON p.order_id = o.id 
			  WHERE p.transaction_id = ?`
	args := []interface{}{transactionID}

	if tenantID != "" {
		query += " AND o.tenant_id = ?"
		args = append(args, tenantID)
	}

	payment := &entity.Payment{}
	err := exec.QueryRowContext(ctx, query, args...).
		Scan(&payment.ID, &payment.OrderID, &payment.TransactionID, &payment.Amount, &payment.Status, &payment.IdempotencyKey, &payment.CreatedAt, &payment.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment by transaction_id: %w", err)
	}

	return payment, nil
}
