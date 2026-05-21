package entity

import (
	"errors"
	"time"

	"github.com/awaken-rise/backend/internal/domain/kernel"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock for this product")
)

type Product struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Name      string       `json:"name"`
	Price     kernel.Money `json:"price"`
	Stock     int          `json:"stock"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func NewProduct(id string, tenantID string, name string, price kernel.Money, stock int) *Product {
	return &Product{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		Price:     price,
		Stock:     stock,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (p *Product) ReserveStock(qty int) error {
	if p.Stock < qty {
		return ErrInsufficientStock
	}
	p.Stock -= qty
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Product) ReleaseStock(qty int) {
	p.Stock += qty
	p.UpdatedAt = time.Now()
}
