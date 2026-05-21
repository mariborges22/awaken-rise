package kernel

import (
	"context"
)

type contextKey string

const tenantKey contextKey = "tenant_id"
const userKey contextKey = "user_id"
const roleKey contextKey = "user_role"

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
func MustGetTenantID(ctx context.Context) string {
	tenantID, _ := GetTenantID(ctx)
	return tenantID
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userKey, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userKey).(string)
	return userID, ok
}

func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}
