package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type ProductUseCase struct {
	productRepo ports.ProductRepository
}

func NewProductUseCase(productRepo ports.ProductRepository) *ProductUseCase {
	return &ProductUseCase{productRepo: productRepo}
}

type CreateProductInput struct {
	Name         string `json:"name"`
	PriceInCents int64  `json:"price_in_cents"`
	Stock        int    `json:"stock"`
}

func (uc *ProductUseCase) CreateProduct(ctx context.Context, input CreateProductInput) (*entity.Product, error) {
	tenantID, ok := kernel.GetTenantID(ctx)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	price := kernel.NewBRL(input.PriceInCents)
	product := entity.NewProduct(
		uuid.New().String(),
		tenantID,
		input.Name,
		price,
		input.Stock,
	)

	if err := uc.productRepo.Save(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context) ([]*entity.Product, error) {
	tenantID, ok := kernel.GetTenantID(ctx)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	return uc.productRepo.ListByTenant(ctx, tenantID)
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, id string) (*entity.Product, error) {
	return uc.productRepo.FindByID(ctx, id)
}
