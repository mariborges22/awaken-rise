package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/awaken-rise/backend/internal/domain/entity"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	query := `SELECT id, tenant_id, name, email, password_hash, role, created_at, updated_at 
	          FROM users WHERE id = ?`
	
	row := r.db.QueryRowContext(ctx, query, id)

	var u entity.User
	err := row.Scan(
		&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `SELECT id, tenant_id, name, email, password_hash, role, created_at, updated_at 
	          FROM users WHERE email = ?`
	
	row := r.db.QueryRowContext(ctx, query, email)

	var u entity.User
	err := row.Scan(
		&u.ID, &u.TenantID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) Save(ctx context.Context, u *entity.User) error {
	query := `INSERT INTO users (
		id, tenant_id, name, email, password_hash, role
	) VALUES (?, ?, ?, ?, ?, ?) 
	ON DUPLICATE KEY UPDATE 
		name = VALUES(name), 
		email = VALUES(email),
		password_hash = VALUES(password_hash),
		role = VALUES(role)`
	
	_, err := r.db.ExecContext(ctx, query, 
		u.ID, u.TenantID, u.Name, u.Email, u.PasswordHash, u.Role,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}
