package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/awaken-rise/backend/internal/domain/entity"
)

const productCacheTTL = 5 * time.Minute

// ProductCacheStore armazena a vitrine de produtos no Redis por tenant.
// TTL de 5 minutos: equilíbrio entre consistência e performance.
// Se o Redis estiver fora do ar, o UseCase cai no banco sem propagar o erro.
type ProductCacheStore struct {
	client *redis.Client
}

func NewProductCacheStore(client *redis.Client) *ProductCacheStore {
	return &ProductCacheStore{client: client}
}

func (s *ProductCacheStore) cacheKey(tenantID string) string {
	return fmt.Sprintf("catalog:%s:products", tenantID)
}

func (s *ProductCacheStore) Get(ctx context.Context, tenantID string) ([]*entity.Product, bool, error) {
	val, err := s.client.Get(ctx, s.cacheKey(tenantID)).Result()
	if err == redis.Nil {
		// Cache miss — comportamento normal, sem erro
		return nil, false, nil
	}
	if err != nil {
		// Redis fora do ar — retorna miss sem propagar erro (degradação graciosa)
		return nil, false, fmt.Errorf("cache read error: %w", err)
	}

	var products []*entity.Product
	if err := json.Unmarshal([]byte(val), &products); err != nil {
		return nil, false, fmt.Errorf("cache unmarshal error: %w", err)
	}

	return products, true, nil
}

func (s *ProductCacheStore) Set(ctx context.Context, tenantID string, products []*entity.Product) error {
	data, err := json.Marshal(products)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := s.client.Set(ctx, s.cacheKey(tenantID), data, productCacheTTL).Err(); err != nil {
		return fmt.Errorf("cache write error: %w", err)
	}

	return nil
}

// Invalidate remove o cache de um tenant específico.
// Deve ser chamado sempre que um produto for criado, atualizado ou removido.
func (s *ProductCacheStore) Invalidate(ctx context.Context, tenantID string) error {
	if err := s.client.Del(ctx, s.cacheKey(tenantID)).Err(); err != nil {
		return fmt.Errorf("cache invalidation error: %w", err)
	}
	return nil
}
