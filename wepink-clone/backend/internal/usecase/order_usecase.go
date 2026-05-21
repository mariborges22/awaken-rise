package usecase

import (
	"context"
	"fmt"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type OrderUseCase struct {
	orderRepo   ports.OrderRepository
	productRepo ports.ProductRepository
	dispatcher  ports.EventDispatcher
	txManager   ports.TransactionManager
}

func NewOrderUseCase(
	orderRepo ports.OrderRepository,
	productRepo ports.ProductRepository,
	dispatcher ports.EventDispatcher,
	txManager ports.TransactionManager,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		dispatcher:  dispatcher,
		txManager:   txManager,
	}
}

type CreateOrderInputItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, id string, items []CreateOrderInputItem) (*entity.Order, error) {
	tenantID, _ := kernel.GetTenantID(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("order must have at least one item")
	}

	// Pre-compute: collect unique product IDs and aggregate quantities
	idSet := make(map[string]int, len(items))
	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %s", item.ProductID)
		}
		idSet[item.ProductID] += item.Quantity
	}

	productIDs := make([]string, 0, len(idSet))
	for pid := range idSet {
		productIDs = append(productIDs, pid)
	}

	var orderItems []entity.OrderItem

	err := uc.txManager.Execute(ctx, func(txCtx context.Context) error {
		// Single SELECT ... WHERE id IN (...) FOR UPDATE — 1 query, pessimistic lock
		products, err := uc.productRepo.FindByIDs(txCtx, productIDs)
		if err != nil {
			return fmt.Errorf("failed to query products: %w", err)
		}

		// Index products by ID for O(1) lookup
		productMap := make(map[string]*entity.Product, len(products))
		for _, p := range products {
			productMap[p.ID] = p
		}

		// Validate all products in-memory (zero I/O)
		for _, pid := range productIDs {
			product, exists := productMap[pid]
			if !exists {
				return fmt.Errorf("product %s not found", pid)
			}
			if product.TenantID != tenantID {
				return fmt.Errorf("product %s does not belong to this tenant", pid)
			}
		}

		// Reserve stock and build order items (in-memory mutations)
		modifiedProducts := make([]*entity.Product, 0, len(idSet))
		for _, item := range items {
			product := productMap[item.ProductID]

			if err := product.ReserveStock(item.Quantity); err != nil {
				return fmt.Errorf("failed to reserve stock for %s: %w", product.Name, err)
			}

			orderItems = append(orderItems, entity.OrderItem{
				ProductID: product.ID,
				Quantity:  item.Quantity,
				Price:     product.Price,
			})
		}

		// Collect unique modified products for batch save
		seen := make(map[string]bool, len(idSet))
		for _, item := range items {
			if !seen[item.ProductID] {
				seen[item.ProductID] = true
				modifiedProducts = append(modifiedProducts, productMap[item.ProductID])
			}
		}

		// Single batch UPDATE — 1 query
		if err := uc.productRepo.SaveBatch(txCtx, modifiedProducts); err != nil {
			return fmt.Errorf("failed to update product stock: %w", err)
		}

		order := entity.NewOrder(id, tenantID, orderItems)

		if err := uc.orderRepo.Save(txCtx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		if len(order.Events()) > 0 {
			if err := uc.dispatcher.Dispatch(txCtx, order.Events()); err != nil {
				return fmt.Errorf("failed to dispatch events: %w", err)
			}
			order.ClearEvents()
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return uc.orderRepo.FindByID(ctx, id)
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}
	return order, nil
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) error {
	err := uc.txManager.Execute(ctx, func(txCtx context.Context) error {
		order, err := uc.orderRepo.FindByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to find order: %w", err)
		}
		if order == nil {
			return fmt.Errorf("order not found")
		}

		if err := order.Cancel(); err != nil {
			return err
		}

		if err := uc.orderRepo.Save(txCtx, order); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}

		// Batch restore: collect product IDs from order items
		productIDs := make([]string, 0, len(order.Items))
		qtyMap := make(map[string]int, len(order.Items))
		for _, item := range order.Items {
			if _, exists := qtyMap[item.ProductID]; !exists {
				productIDs = append(productIDs, item.ProductID)
			}
			qtyMap[item.ProductID] += item.Quantity
		}

		// Single SELECT ... FOR UPDATE
		products, err := uc.productRepo.FindByIDs(txCtx, productIDs)
		if err != nil {
			return fmt.Errorf("failed to find products for stock release: %w", err)
		}

		// Release stock in-memory
		for _, product := range products {
			if qty, ok := qtyMap[product.ID]; ok {
				product.ReleaseStock(qty)
			}
		}

		// Single batch UPDATE
		if err := uc.productRepo.SaveBatch(txCtx, products); err != nil {
			return fmt.Errorf("failed to restore stock: %w", err)
		}

		return nil
	})

	return err
}
