package provisioning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("provisioning rule not found")

const selectColumns = `id::text, tenant_id::text, name, description, priority, active, trigger,
	match_manufacturer, match_oui, match_product_class, match_model_name, match_sw_version,
	actions, created_at, updated_at`

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

// NewPostgresRepository returns a new provisioning rules repository.
func NewPostgresRepository(pool *pgxpool.Pool, log *zap.Logger) *PostgresRepository {
	return &PostgresRepository{pool: pool, log: log}
}

func (r *PostgresRepository) ListActive(ctx context.Context, tenantID string) ([]*Rule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectColumns+`
		FROM provisioning_rules
		WHERE tenant_id = $1 AND active = true
		ORDER BY priority DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list active rules: %w", err)
	}
	defer rows.Close()

	var list []*Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		if err := rule.ParseActions(); err != nil {
			r.log.Warn("parse rule actions", zap.String("rule_id", rule.ID), zap.Error(err))
			continue
		}
		list = append(list, rule)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id string) (*Rule, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+selectColumns+`
		FROM provisioning_rules
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	rule, err := scanRuleRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := rule.ParseActions(); err != nil {
		return nil, fmt.Errorf("parse actions: %w", err)
	}
	return rule, nil
}

func (r *PostgresRepository) Create(ctx context.Context, rule *Rule) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO provisioning_rules
		(tenant_id, name, description, priority, active, trigger,
		 match_manufacturer, match_oui, match_product_class, match_model_name, match_sw_version, actions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text, created_at, updated_at`,
		rule.TenantID, rule.Name, rule.Description, rule.Priority, rule.Active, rule.Trigger,
		rule.MatchManufacturer, rule.MatchOUI, rule.MatchProductClass, rule.MatchModelName, rule.MatchSWVersion,
		rule.ActionsRaw,
	)
	if err := row.Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		return fmt.Errorf("create rule: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, rule *Rule) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE provisioning_rules SET
			name = $1, description = $2, priority = $3, active = $4, trigger = $5,
			match_manufacturer = $6, match_oui = $7, match_product_class = $8,
			match_model_name = $9, match_sw_version = $10, actions = $11, updated_at = NOW()
		WHERE tenant_id = $12 AND id = $13`,
		rule.Name, rule.Description, rule.Priority, rule.Active, rule.Trigger,
		rule.MatchManufacturer, rule.MatchOUI, rule.MatchProductClass, rule.MatchModelName, rule.MatchSWVersion,
		rule.ActionsRaw, rule.TenantID, rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	rule.UpdatedAt = time.Now()
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE provisioning_rules SET active = false, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string) ([]*Rule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectColumns+`
		FROM provisioning_rules
		WHERE tenant_id = $1
		ORDER BY priority DESC, created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var list []*Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		if err := rule.ParseActions(); err != nil {
			r.log.Warn("parse rule actions", zap.String("rule_id", rule.ID), zap.Error(err))
		}
		list = append(list, rule)
	}
	return list, rows.Err()
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(s rowScanner) (*Rule, error) {
	var r Rule
	err := s.Scan(
		&r.ID, &r.TenantID, &r.Name, &r.Description, &r.Priority, &r.Active, &r.Trigger,
		&r.MatchManufacturer, &r.MatchOUI, &r.MatchProductClass, &r.MatchModelName, &r.MatchSWVersion,
		&r.ActionsRaw, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanRuleRow(row pgx.Row) (*Rule, error) {
	return scanRule(row)
}
