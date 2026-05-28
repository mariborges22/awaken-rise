package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type ProductUseCase struct {
	productRepo ports.ProductRepository
	cache       ports.ProductCacheStore
}

func NewProductUseCase(productRepo ports.ProductRepository, cache ports.ProductCacheStore) *ProductUseCase {
	return &ProductUseCase{productRepo: productRepo, cache: cache}
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

	// Invalida o cache do catálogo deste tenant ao criar um produto novo.
	// Erro de cache não deve falhar a operação principal.
	if uc.cache != nil {
		if err := uc.cache.Invalidate(ctx, tenantID); err != nil {
			slog.Warn("Failed to invalidate product cache", "tenant_id", tenantID, "error", err)
		}
	}

	return product, nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context) ([]*entity.Product, error) {
	tenantID, ok := kernel.GetTenantID(ctx)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	// 1. Tenta buscar no cache (Cache-Aside)
	if uc.cache != nil {
		if cached, hit, err := uc.cache.Get(ctx, tenantID); hit {
			slog.Debug("Cache HIT for product catalog", "tenant_id", tenantID)
			return cached, nil
		} else if err != nil {
			// Redis fora do ar: apenas loga e cai no banco (degradação graciosa)
			slog.Warn("Cache read failed, falling back to DB", "tenant_id", tenantID, "error", err)
		}
	}

	// 2. Cache MISS — busca no banco de dados
	products, err := uc.productRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. Popula o cache para as próximas requisições (fire-and-forget)
	if uc.cache != nil {
		if err := uc.cache.Set(ctx, tenantID, products); err != nil {
			slog.Warn("Failed to populate product cache", "tenant_id", tenantID, "error", err)
		}
	}

	return products, nil
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, id string) (*entity.Product, error) {
	return uc.productRepo.FindByID(ctx, id)
}
