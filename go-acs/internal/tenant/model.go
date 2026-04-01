package tenant

import "time"

// Tenant represents a multi-tenant organization.
type Tenant struct {
	ID                string    `db:"id"                    json:"id"`
	Slug              string    `db:"slug"                  json:"slug"`
	Name              string    `db:"name"                  json:"name"`
	Plan              string    `db:"plan"                  json:"plan"`
	MaxDevices        int       `db:"max_devices"           json:"max_devices"`
	APIKey            string    `db:"api_key"               json:"-"`
	Active            bool      `db:"active"                json:"active"`
	DefaultCRUsername string    `db:"default_cr_username"   json:"default_cr_username"`
	DefaultCRPassword string    `db:"default_cr_password"   json:"default_cr_password"`
	CreatedAt         time.Time `db:"created_at"            json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"            json:"updated_at"`
}
