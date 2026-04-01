package device

import "time"

// Device represents a CPE (TR-069/TR-181), scoped to a tenant.
type Device struct {
	// Identity
	ID           string `bson:"_id,omitempty"  json:"id"`
	TenantID     string `bson:"tenant_id"       json:"tenant_id"`
	SerialNumber string `bson:"serial_number"   json:"serial_number"`
	Manufacturer string `bson:"manufacturer"    json:"manufacturer"`
	OUI          string `bson:"oui"             json:"oui"`
	ProductClass string `bson:"product_class"   json:"product_class"`
	ModelName    string `bson:"model_name"      json:"model_name"`

	// Software / Hardware
	SoftwareVersion string `bson:"software_version" json:"software_version"`
	HardwareVersion string `bson:"hardware_version" json:"hardware_version"`

	// Network
	IPAddress  string `bson:"ip_address"   json:"ip_address"`
	MACAddress string `bson:"mac_address"  json:"mac_address"`
	CWMPURL    string `bson:"cwmp_url"     json:"cwmp_url"` // ManagementServer URL

	// Connection Request (so ACS can wake the device)
	ConnectionRequestURL      string `bson:"connection_request_url"      json:"connection_request_url"`
	ConnectionRequestUsername string `bson:"connection_request_username" json:"connection_request_username"`
	ConnectionRequestPassword string `bson:"connection_request_password" json:"connection_request_password"`

	// Status
	Online     bool      `bson:"online"        json:"online"`
	LastInform time.Time `bson:"last_inform"   json:"last_inform"`
	LastBoot   time.Time `bson:"last_boot"     json:"last_boot,omitempty"`
	FirstSeen  time.Time `bson:"first_seen"    json:"first_seen"`
	BootCount  int       `bson:"boot_count"    json:"boot_count"`

	// Events from last Inform
	LastEvents []string `bson:"last_events" json:"last_events"`

	// Full TR-181 / TR-098 parameter tree (key = path, value = string)
	Parameters map[string]string `bson:"parameters" json:"parameters"`

	// xPON / GPON (populated only for ONU/OLT devices)
	PON *PONInfo `bson:"pon,omitempty" json:"pon,omitempty"`

	// Tags (set by provisioning or operator)
	Tags []string `bson:"tags" json:"tags"`
}

// PONInfo holds xPON/GPON-specific data.
type PONInfo struct {
	PONType     string `bson:"pon_type"     json:"pon_type"` // GPON, XGPON, EPON
	ONUID       string `bson:"onu_id"       json:"onu_id"`
	OLTID       string `bson:"olt_id"       json:"olt_id"`
	SignalLevel string `bson:"signal_level" json:"signal_level"` // dBm rx power
	Distance    string `bson:"distance"     json:"distance"`
}

// GetParameter returns a parameter value by full path, empty string if not found.
func (d *Device) GetParameter(name string) string {
	if d.Parameters == nil {
		return ""
	}
	return d.Parameters[name]
}

// DeviceID returns a human-readable identifier.
func (d *Device) DeviceID() string {
	if d.SerialNumber != "" {
		return d.SerialNumber
	}
	return d.ID
}
