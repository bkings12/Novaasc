package tenant

import "context"

type contextKey string

const tenantCtxKey contextKey = "tenant"

// WithTenant returns a new context with the tenant stored.
func WithTenant(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, tenantCtxKey, t)
}

// FromContext extracts the tenant from context.
// Returns nil, false if not present.
func FromContext(ctx context.Context) (*Tenant, bool) {
	t, ok := ctx.Value(tenantCtxKey).(*Tenant)
	return t, ok
}

// MustFromContext extracts tenant or panics — use only in middleware-guarded handlers.
func MustFromContext(ctx context.Context) *Tenant {
	t, ok := FromContext(ctx)
	if !ok {
		panic("tenant not found in context")
	}
	return t
}
