package provisioning

import (
	"encoding/json"
	"time"

	"github.com/novaacs/go-acs/internal/task"
)

// Rule is a provisioning rule stored in PostgreSQL.
type Rule struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Active      bool   `json:"active"`
	Trigger     string `json:"trigger"` // "0 BOOTSTRAP" | "1 BOOT" | "2 PERIODIC" | "ANY"

	MatchManufacturer string `json:"match_manufacturer"`
	MatchOUI          string `json:"match_oui"`
	MatchProductClass string `json:"match_product_class"`
	MatchModelName    string `json:"match_model_name"`
	MatchSWVersion    string `json:"match_sw_version"` // regex pattern

	ActionsRaw json.RawMessage `json:"actions"`
	Actions    []RuleAction    `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParseActions deserializes ActionsRaw into Actions.
func (r *Rule) ParseActions() error {
	return json.Unmarshal(r.ActionsRaw, &r.Actions)
}

// RuleAction defines one task to enqueue when a rule matches.
type RuleAction struct {
	Type            task.Type          `json:"type"`
	ParameterNames  []string           `json:"parameter_names,omitempty"`
	ParameterValues map[string]string  `json:"parameter_values,omitempty"`
	Download        *task.DownloadArgs `json:"download,omitempty"`
	Priority        int                `json:"priority"`
}
