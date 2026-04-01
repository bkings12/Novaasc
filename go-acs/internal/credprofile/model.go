package credprofile

import "time"

// Profile is a per-OUI or per-manufacturer credential profile for connection request / CWMP.
type Profile struct {
	ID           string    `db:"id"             json:"id"`
	TenantID     string    `db:"tenant_id"      json:"tenant_id"`
	Name         string    `db:"name"           json:"name"`
	OUI          string    `db:"oui"            json:"oui"`
	Manufacturer string    `db:"manufacturer"   json:"manufacturer"`
	ModelName    string    `db:"model_name"     json:"model_name"`
	CRUsername   string    `db:"cr_username"    json:"cr_username"`
	CRPassword   string    `db:"cr_password"    json:"cr_password"`
	CWMPUsername string    `db:"cwmp_username" json:"cwmp_username"`
	CWMPPassword string    `db:"cwmp_password" json:"-"`
	Active       bool      `db:"active"        json:"active"`
	Notes        string    `db:"notes"          json:"notes"`
	CreatedAt    time.Time `db:"created_at"     json:"created_at"`
}

// ResolvedCredentials is the result of the credential resolution priority chain.
type ResolvedCredentials struct {
	Username string
	Password string
	Source   string // "body" | "device" | "oui_profile" | "manufacturer_profile" | "tenant_default" | "serial_fallback"
}
