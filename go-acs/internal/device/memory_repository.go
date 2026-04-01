package device

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryRepository is an in-memory device store for development/testing.
type MemoryRepository struct {
	mu      sync.RWMutex
	byKey   map[string]*Device // key = tenantID + ":" + serial
	byID    map[string]*Device // key = device ID
	counter int
}

// NewMemoryRepository returns a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byKey: make(map[string]*Device),
		byID:  make(map[string]*Device),
	}
}

func key(tenantID, serial string) string { return tenantID + ":" + serial }

// Upsert inserts or updates a device (composite key: tenant_id, serial_number).
func (r *MemoryRepository) Upsert(ctx context.Context, d *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(d.TenantID, d.SerialNumber)
	if existing, ok := r.byKey[k]; ok && d.ID == "" {
		d.ID = existing.ID
		d.FirstSeen = existing.FirstSeen
	}
	if d.ID == "" {
		r.counter++
		d.ID = fmt.Sprintf("mem-%d", r.counter)
		d.FirstSeen = time.Now()
	}
	r.byKey[k] = d
	r.byID[d.ID] = d
	return nil
}

// GetBySerial returns device for a tenant by serial number.
func (r *MemoryRepository) GetBySerial(ctx context.Context, tenantID, serial string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byKey[key(tenantID, serial)]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

// GetByID returns device by ID (and tenant).
func (r *MemoryRepository) GetByID(ctx context.Context, tenantID, id string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.byID[id]
	if !ok || d.TenantID != tenantID {
		return nil, ErrNotFound
	}
	return d, nil
}

// List returns devices for a tenant with optional filters (simplified: no filter).
func (r *MemoryRepository) List(ctx context.Context, tenantID string, filter DeviceFilter) ([]*Device, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*Device
	for _, d := range r.byKey {
		if d.TenantID != tenantID {
			continue
		}
		if filter.Online != nil && d.Online != *filter.Online {
			continue
		}
		if filter.Manufacturer != "" && d.Manufacturer != filter.Manufacturer {
			continue
		}
		if filter.ProductClass != "" && d.ProductClass != filter.ProductClass {
			continue
		}
		list = append(list, d)
	}
	total := int64(len(list))
	offset, limit := filter.Offset, filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if offset >= int64(len(list)) {
		return nil, total, nil
	}
	end := offset + limit
	if end > int64(len(list)) {
		end = int64(len(list))
	}
	return list[offset:end], total, nil
}

// Delete removes a device.
func (r *MemoryRepository) Delete(ctx context.Context, tenantID, serial string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(tenantID, serial)
	d, ok := r.byKey[k]
	if !ok {
		return ErrNotFound
	}
	delete(r.byKey, k)
	delete(r.byID, d.ID)
	return nil
}

// SetOnline updates online status and last_inform.
func (r *MemoryRepository) SetOnline(ctx context.Context, tenantID, serial string, online bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.byKey[key(tenantID, serial)]
	if !ok {
		return ErrNotFound
	}
	d.Online = online
	d.LastInform = time.Now()
	return nil
}

// UpdateParameters does a partial update of the parameter map.
func (r *MemoryRepository) UpdateParameters(ctx context.Context, tenantID, serial string, params map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.byKey[key(tenantID, serial)]
	if !ok {
		return ErrNotFound
	}
	if d.Parameters == nil {
		d.Parameters = make(map[string]string)
	}
	for k, v := range params {
		d.Parameters[k] = v
	}
	return nil
}
