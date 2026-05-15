package kernel

import (
	"context"
)

type contextKey string

const tenantKey contextKey = "tenant_id"

// WithTenantID returns a new context with the tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// GetTenantID extracts the tenant ID from the context.
func GetTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantKey).(string)
	return tenantID, ok
}

// MustGetTenantID extracts the tenant ID or returns an empty string if not found.
// In a stricter implementation, this could panic or return an error.
func MustGetTenantID(ctx context.Context) string {
	tenantID, _ := GetTenantID(ctx)
	return tenantID
}
