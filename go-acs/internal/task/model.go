package task

import "time"

// Type is the CWMP RPC / task type.
type Type string

const (
	TypeGetParameterValues Type = "GetParameterValues"
	TypeSetParameterValues Type = "SetParameterValues"
	TypeGetParameterNames  Type = "GetParameterNames"
	TypeAddObject          Type = "AddObject"
	TypeDeleteObject       Type = "DeleteObject"
	TypeReboot             Type = "Reboot"
	TypeFactoryReset       Type = "FactoryReset"
	TypeDownload           Type = "Download"
	TypeUpload             Type = "Upload"
	TypeScheduleInform     Type = "ScheduleInform"
)

// Status is the task lifecycle status.
type Status string

const (
	StatusPending    Status = "pending"
	StatusDispatched Status = "dispatched"
	StatusComplete   Status = "complete"
	StatusFailed     Status = "failed"
	StatusTimeout    Status = "timeout"
	StatusCancelled  Status = "cancelled"
)

// Task represents a single command to be sent to a device.
type Task struct {
	ID           string `bson:"_id"            json:"id"`
	TenantID     string `bson:"tenant_id"      json:"tenant_id"`
	DeviceSerial string `bson:"device_serial"  json:"device_serial"`
	Type         Type   `bson:"type"           json:"type"`
	Status       Status `bson:"status"         json:"status"`
	Priority     int    `bson:"priority"       json:"priority"`

	ParameterNames  []string          `bson:"parameter_names,omitempty"  json:"parameter_names,omitempty"`
	ParameterValues map[string]string `bson:"parameter_values,omitempty" json:"parameter_values,omitempty"`
	Download        *DownloadArgs     `bson:"download,omitempty"         json:"download,omitempty"`

	Result       *Result    `bson:"result,omitempty" json:"result,omitempty"`
	CreatedAt    time.Time  `bson:"created_at"              json:"created_at"`
	DispatchedAt *time.Time `bson:"dispatched_at,omitempty"   json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `bson:"completed_at,omitempty"    json:"completed_at,omitempty"`
	Timeout      int64      `bson:"timeout"                  json:"timeout"` // nanoseconds
	CreatedBy    string     `bson:"created_by"               json:"created_by"`

	ResultChan chan Result `bson:"-" json:"-"`
}

// DownloadArgs holds parameters for a Download RPC.
type DownloadArgs struct {
	FileType     string `bson:"file_type"     json:"file_type"`
	URL          string `bson:"url"           json:"url"`
	Username     string `bson:"username"      json:"username"`
	Password     string `bson:"password"      json:"password"`
	FileSize     int    `bson:"file_size"     json:"file_size"`
	TargetFile   string `bson:"target_file"   json:"target_file"`
	DelaySeconds int    `bson:"delay_seconds" json:"delay_seconds"`
	CommandKey   string `bson:"command_key"   json:"command_key"`
}

// Result is stored when a task completes.
type Result struct {
	Success     bool              `bson:"success"            json:"success"`
	Fault       *FaultResult      `bson:"fault,omitempty"    json:"fault,omitempty"`
	Values      map[string]string `bson:"values,omitempty"   json:"values,omitempty"`
	CompletedAt time.Time         `bson:"completed_at"       json:"completed_at"`
}

// FaultResult holds fault details when a task fails.
type FaultResult struct {
	Code    string `bson:"code"    json:"code"`
	Message string `bson:"message" json:"message"`
}
