package backup

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("backup not found")

type Repository interface {
	Create(ctx context.Context, b *Backup) error
	GetByID(ctx context.Context, tenantID, id string) (*Backup, error)
	ListForDevice(ctx context.Context, tenantID, serial string, limit int64) ([]*Backup, error)
	Delete(ctx context.Context, tenantID, id string) error
	CreateRestoreJob(ctx context.Context, job *RestoreJob) error
	GetRestoreJob(ctx context.Context, tenantID, id string) (*RestoreJob, error)
	UpdateRestoreJob(ctx context.Context, job *RestoreJob) error
}
