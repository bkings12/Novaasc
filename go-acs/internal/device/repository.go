package device

import "context"

// Repository abstracts device persistence (MongoDB).
type Repository interface {
	// Upsert inserts or updates device scoped to tenant.
	// Composite unique key: (tenant_id, serial_number)
	Upsert(ctx context.Context, d *Device) error

	// GetBySerial returns device for a tenant by serial number.
	GetBySerial(ctx context.Context, tenantID, serial string) (*Device, error)

	// GetByID returns device by MongoDB _id.
	GetByID(ctx context.Context, tenantID, id string) (*Device, error)

	// List returns devices for a tenant with optional filters.
	List(ctx context.Context, tenantID string, filter DeviceFilter) ([]*Device, int64, error)

	// Delete removes a device (hard delete).
	Delete(ctx context.Context, tenantID, serial string) error

	// SetOnline updates the online status and last_inform timestamp.
	SetOnline(ctx context.Context, tenantID, serial string, online bool) error

	// UpdateParameters does a partial update of the parameter map.
	UpdateParameters(ctx context.Context, tenantID, serial string, params map[string]string) error

	// UpdateConnectionRequest updates connection request fields (e.g. connection_request_username, connection_request_password).
	UpdateConnectionRequest(ctx context.Context, tenantID, serial string, fields map[string]string) error
}

// DeviceFilter filters devices for List.
type DeviceFilter struct {
	Online       *bool
	Manufacturer string
	ProductClass string
	Tags         []string
	Search       string // matches serial, IP, model
	Limit        int64
	Offset       int64
}
