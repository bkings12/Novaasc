package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("tenant not found")

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository returns a new tenant repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, slug, name, plan, max_devices, api_key, active, default_cr_username, default_cr_password, created_at, updated_at
		FROM tenants WHERE id = $1 AND active = true`, id)
	return scanTenant(row)
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, slug, name, plan, max_devices, api_key, active, default_cr_username, default_cr_password, created_at, updated_at
		FROM tenants WHERE slug = $1 AND active = true`, slug)
	return scanTenant(row)
}

func (r *PostgresRepository) GetByAPIKey(ctx context.Context, apiKey string) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, slug, name, plan, max_devices, api_key, active, default_cr_username, default_cr_password, created_at, updated_at
		FROM tenants WHERE api_key = $1 AND active = true`, apiKey)
	return scanTenant(row)
}

func (r *PostgresRepository) Create(ctx context.Context, t *Tenant) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tenants (slug, name, plan, max_devices, api_key, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text`,
		t.Slug, t.Name, t.Plan, t.MaxDevices, t.APIKey, t.Active)
	if err := row.Scan(&t.ID); err != nil {
		return fmt.Errorf("tenant create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, t *Tenant) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE tenants SET name = $1, plan = $2, max_devices = $3, active = $4,
		       default_cr_username = $5, default_cr_password = $6, updated_at = NOW()
		WHERE id = $7`,
		t.Name, t.Plan, t.MaxDevices, t.Active, t.DefaultCRUsername, t.DefaultCRPassword, t.ID)
	if err != nil {
		return fmt.Errorf("tenant update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]*Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, slug, name, plan, max_devices, api_key, active, default_cr_username, default_cr_password, created_at, updated_at
		FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, `UPDATE tenants SET active = false WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTenant(s scanner) (*Tenant, error) {
	var t Tenant
	err := s.Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.MaxDevices, &t.APIKey, &t.Active, &t.DefaultCRUsername, &t.DefaultCRPassword, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}
