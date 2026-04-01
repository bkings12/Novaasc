package credprofile

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("credential profile not found")

type Repository interface {
	FindByOUI(ctx context.Context, tenantID, oui string) (*Profile, error)
	FindByManufacturer(ctx context.Context, tenantID, manufacturer string) (*Profile, error)
	List(ctx context.Context, tenantID string) ([]*Profile, error)
	Create(ctx context.Context, p *Profile) error
	Update(ctx context.Context, p *Profile) error
	Delete(ctx context.Context, tenantID, id string) error
}
