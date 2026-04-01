package tenant

import "context"

// Repository defines tenant persistence.
type Repository interface {
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*Tenant, error)
	Create(ctx context.Context, t *Tenant) error
	Update(ctx context.Context, t *Tenant) error
	List(ctx context.Context) ([]*Tenant, error)
	Delete(ctx context.Context, id string) error
}
