package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	uniqueViolationCode = "23505"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, log *zap.Logger) *PostgresRepository {
	return &PostgresRepository{pool: pool, log: log}
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, tenantID, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, email, password_hash, role, active, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND email = $2 AND active = true`,
		tenantID, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, email, password_hash, role, active, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND id = $2`,
		tenantID, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *PostgresRepository) Create(ctx context.Context, u *User) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, role, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at`,
		u.TenantID, u.Email, u.PasswordHash, u.Role, u.Active)
	err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return ErrDuplicate
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string) ([]*User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, email, password_hash, role, active, created_at, updated_at
		FROM users WHERE tenant_id = $1 ORDER BY created_at`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, tenantID, id, hash string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3`, hash, tenantID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetActive(ctx context.Context, tenantID, id string, active bool) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE users SET active = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3`, active, tenantID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
