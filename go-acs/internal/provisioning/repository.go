package provisioning

import "context"

// Repository defines persistence for provisioning rules.
type Repository interface {
	ListActive(ctx context.Context, tenantID string) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id string) (*Rule, error)
	Create(ctx context.Context, r *Rule) error
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string) ([]*Rule, error)
}
