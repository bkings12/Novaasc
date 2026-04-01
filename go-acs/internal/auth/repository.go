package auth

import (
	"context"
	"errors"
)

var (
	ErrNotFound  = errors.New("user not found")
	ErrDuplicate = errors.New("email already exists")
)

type Repository interface {
	GetByEmail(ctx context.Context, tenantID, email string) (*User, error)
	GetByID(ctx context.Context, tenantID, id string) (*User, error)
	Create(ctx context.Context, u *User) error
	List(ctx context.Context, tenantID string) ([]*User, error)
	UpdatePassword(ctx context.Context, tenantID, id, hash string) error
	SetActive(ctx context.Context, tenantID, id string, active bool) error
}
