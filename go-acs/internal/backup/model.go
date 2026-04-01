package backup

import "time"

// Backup is a full parameter snapshot of a device.
type Backup struct {
	ID           string `bson:"_id"            json:"id"`
	TenantID     string `bson:"tenant_id"      json:"tenant_id"`
	DeviceSerial string `bson:"device_serial"  json:"device_serial"`
	Label        string `bson:"label"          json:"label"`
	// "manual" | "auto-bootstrap" | "pre-firmware" | "scheduled"
	Trigger         string            `bson:"trigger"        json:"trigger"`
	ParameterCount  int               `bson:"parameter_count" json:"parameter_count"`
	Parameters      map[string]string `bson:"parameters"     json:"parameters"`
	SoftwareVersion string            `bson:"software_version" json:"software_version"`
	IPAddress       string            `bson:"ip_address"     json:"ip_address"`
	CreatedAt       time.Time         `bson:"created_at"     json:"created_at"`
	CreatedBy       string            `bson:"created_by"     json:"created_by"`
	RestoredAt      *time.Time        `bson:"restored_at,omitempty"  json:"restored_at,omitempty"`
	RestoredBy      string            `bson:"restored_by,omitempty"  json:"restored_by,omitempty"`
	RestoreTaskID   string            `bson:"restore_task_id,omitempty" json:"restore_task_id,omitempty"`
}

// RestoreJob tracks a restore operation.
type RestoreJob struct {
	ID           string `bson:"_id"          json:"id"`
	TenantID     string `bson:"tenant_id"    json:"tenant_id"`
	BackupID     string `bson:"backup_id"    json:"backup_id"`
	DeviceSerial string `bson:"device_serial" json:"device_serial"`
	Status       string `bson:"status"       json:"status"`
	// "pending" | "running" | "complete" | "failed"
	TaskIDs     []string   `bson:"task_ids"     json:"task_ids"`
	TotalChunks int        `bson:"total_chunks" json:"total_chunks"`
	DoneChunks  int        `bson:"done_chunks"  json:"done_chunks"`
	Error       string     `bson:"error,omitempty" json:"error,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"   json:"created_at"`
	CompletedAt *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	CreatedBy   string     `bson:"created_by"   json:"created_by"`
}
