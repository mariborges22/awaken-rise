package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *ProductRepository) FindByID(ctx context.Context, id string) (*entity.Product, error) {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)

	query := `SELECT id, tenant_id, name, price, stock, created_at, updated_at 
	          FROM products WHERE id = ?`
	args := []interface{}{id}

	if tenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, tenantID)
	}

	row := exec.QueryRowContext(ctx, query, args...)

	var p entity.Product
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find product: %w", err)
	}

	return &p, nil
}

func (r *ProductRepository) Save(ctx context.Context, p *entity.Product) error {
	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)
	if tenantID != "" {
		p.TenantID = tenantID
	}

	query := `INSERT INTO products (
		id, tenant_id, name, price, stock
	) VALUES (?, ?, ?, ?, ?) 
	ON DUPLICATE KEY UPDATE 
		name = VALUES(name), 
		price = VALUES(price),
		stock = VALUES(stock)`
	
	_, err := exec.ExecContext(ctx, query, 
		p.ID, p.TenantID, p.Name, p.Price, p.Stock,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}
	return nil
}

func (r *ProductRepository) FindByIDs(ctx context.Context, ids []string) ([]*entity.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	exec := r.getExecutor(ctx)
	tenantID, _ := kernel.GetTenantID(ctx)

	// Build placeholders: ?,?,?
	placeholders := ""
	args := make([]interface{}, 0, len(ids)+1)
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}

	query := `SELECT id, tenant_id, name, price, stock, created_at, updated_at 
	          FROM products WHERE id IN (` + placeholders + `)`

	if tenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, tenantID)
	}

	// FOR UPDATE: pessimistic lock within the transaction to prevent concurrent stock modification
	if GetTx(ctx) != nil {
		query += " FOR UPDATE"
	}

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find products by ids: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, &p)
	}

	return products, nil
}

func (r *ProductRepository) SaveBatch(ctx context.Context, products []*entity.Product) error {
	if len(products) == 0 {
		return nil
	}

	exec := r.getExecutor(ctx)

	// Build multi-row INSERT ... ON DUPLICATE KEY UPDATE
	query := `INSERT INTO products (id, tenant_id, name, price, stock) VALUES `
	args := make([]interface{}, 0, len(products)*5)

	for i, p := range products {
		if i > 0 {
			query += ","
		}
		query += "(?,?,?,?,?)"
		args = append(args, p.ID, p.TenantID, p.Name, p.Price, p.Stock)
	}

	query += ` ON DUPLICATE KEY UPDATE 
		name = VALUES(name), 
		price = VALUES(price),
		stock = VALUES(stock)`

	_, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to save products batch: %w", err)
	}

	return nil
}

func (r *ProductRepository) ListByTenant(ctx context.Context, tenantID string) ([]*entity.Product, error) {
	exec := r.getExecutor(ctx)

	query := `SELECT id, tenant_id, name, price, stock, created_at, updated_at 
	          FROM products WHERE tenant_id = ?`
	
	rows, err := exec.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, &p)
	}

	return products, nil
}
