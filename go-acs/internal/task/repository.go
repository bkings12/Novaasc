package task

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("task not found")

// Repository defines task queue persistence.
type Repository interface {
	Enqueue(ctx context.Context, t *Task) error
	NextForDevice(ctx context.Context, tenantID, serial string) (*Task, error)
	Complete(ctx context.Context, tenantID, taskID string, result Result) error
	Fail(ctx context.Context, tenantID, taskID string, reason string) error
	Cancel(ctx context.Context, tenantID, taskID string) error
	GetByID(ctx context.Context, tenantID, taskID string) (*Task, error)
	ListForDevice(ctx context.Context, tenantID, serial string, limit int64) ([]*Task, error)
	ListForTenant(ctx context.Context, tenantID string, filter Filter) ([]*Task, int64, error)
	TimeoutStale(ctx context.Context, cutoff time.Time) (int64, error)
}

// Filter filters tasks for ListForTenant.
type Filter struct {
	Status       Status
	DeviceSerial string
	Limit        int64
	Offset       int64
}
