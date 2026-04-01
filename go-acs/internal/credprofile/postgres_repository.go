package credprofile

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, log *zap.Logger) *PostgresRepository {
	return &PostgresRepository{pool: pool, log: log}
}

func (r *PostgresRepository) FindByOUI(ctx context.Context, tenantID, oui string) (*Profile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, name, oui, manufacturer, model_name,
		       cr_username, cr_password, cwmp_username, cwmp_password, active, notes, created_at
		FROM credential_profiles
		WHERE tenant_id::text = $1 AND oui = $2 AND active = true
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, oui)
	return r.scanProfile(row)
}

func (r *PostgresRepository) FindByManufacturer(ctx context.Context, tenantID, manufacturer string) (*Profile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, name, oui, manufacturer, model_name,
		       cr_username, cr_password, cwmp_username, cwmp_password, active, notes, created_at
		FROM credential_profiles
		WHERE tenant_id::text = $1 AND LOWER(manufacturer) = LOWER($2) AND active = true
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, manufacturer)
	return r.scanProfile(row)
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string) ([]*Profile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, name, oui, manufacturer, model_name,
		       cr_username, cr_password, cwmp_username, cwmp_password, active, notes, created_at
		FROM credential_profiles
		WHERE tenant_id::text = $1
		ORDER BY manufacturer, oui`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Profile
	for rows.Next() {
		p, err := r.scanProfile(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) Create(ctx context.Context, p *Profile) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO credential_profiles
		(tenant_id, name, oui, manufacturer, model_name, cr_username, cr_password, cwmp_username, cwmp_password, active, notes)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text, created_at`,
		p.TenantID, p.Name, p.OUI, p.Manufacturer, p.ModelName,
		p.CRUsername, p.CRPassword, p.CWMPUsername, p.CWMPPassword, p.Active, p.Notes).Scan(&p.ID, &p.CreatedAt)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, p *Profile) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE credential_profiles SET
			name = $1, oui = $2, manufacturer = $3, model_name = $4,
			cr_username = $5, cr_password = $6, cwmp_username = $7, cwmp_password = $8,
			active = $9, notes = $10
		WHERE tenant_id::text = $11 AND id::text = $12`,
		p.Name, p.OUI, p.Manufacturer, p.ModelName,
		p.CRUsername, p.CRPassword, p.CWMPUsername, p.CWMPPassword, p.Active, p.Notes,
		p.TenantID, p.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE credential_profiles SET active = false
		WHERE tenant_id::text = $1 AND id::text = $2`,
		tenantID, id)
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

func (r *PostgresRepository) scanProfile(s scanner) (*Profile, error) {
	var p Profile
	err := s.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.OUI, &p.Manufacturer, &p.ModelName,
		&p.CRUsername, &p.CRPassword, &p.CWMPUsername, &p.CWMPPassword,
		&p.Active, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}
