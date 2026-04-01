package cwmp

import (
	"context"
	"sync"
	"time"

	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
)

// SessionState represents CWMP session state.
type SessionState int

const (
	StateNew         SessionState = iota // Just connected, awaiting Inform
	StateInformed                        // Inform received, dispatching tasks
	StateWaitingTask                     // Sent a task, awaiting response
	StateIdle                            // No pending tasks, session can end
	StateDone                            // Session complete
	StateFailed                          // Session failed
)

func (s SessionState) String() string {
	switch s {
	case StateNew:
		return "new"
	case StateInformed:
		return "informed"
	case StateWaitingTask:
		return "waiting_task"
	case StateIdle:
		return "idle"
	case StateDone:
		return "done"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Session holds state for one CWMP session (one CPE connection).
type Session struct {
	mu sync.RWMutex

	ID              string
	DeviceID        string
	DeviceSerial    string
	TenantID        string
	TenantSlug      string
	State           SessionState
	CWMPID          string
	Device          *device.Device
	ParameterTree   map[string]string // flat dot-notation from Inform
	TaskQueue       []interface{}     // queued tasks (e.g. *task.Task)
	CurrentTask     interface{}       // task awaiting response
	CurrentTaskID   string
	CurrentTaskType task.Type
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Deadline        time.Time
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewSession creates a session with timeout derived from ctx and timeout duration.
func NewSession(id string, timeout time.Duration) *Session {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s := &Session{
		ID:            id,
		State:         StateNew,
		ParameterTree: make(map[string]string),
		TaskQueue:     nil,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Deadline:      time.Now().Add(timeout),
		ctx:           ctx,
		cancel:        cancel,
	}
	return s
}

// Transition sets state and updates timestamp.
func (s *Session) Transition(st SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = st
	s.UpdatedAt = time.Now()
	s.Deadline = time.Now().Add(30 * time.Second) // refresh deadline on activity
}

// GetState returns current state.
func (s *Session) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// SetDevice sets device and device id.
func (s *Session) SetDevice(d *device.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Device = d
	if d != nil {
		s.DeviceID = d.ID
	}
	s.UpdatedAt = time.Now()
}

// SetTenant stores tenant id and slug on the session for downstream scoping.
func (s *Session) SetTenant(t *tenant.Tenant) {
	if t == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TenantID = t.ID
	s.TenantSlug = t.Slug
	s.UpdatedAt = time.Now()
}

// GetTenantID returns the tenant ID for this session.
func (s *Session) GetTenantID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TenantID
}

// SetDeviceSerial stores the device serial number for this session.
func (s *Session) SetDeviceSerial(serial string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DeviceSerial = serial
	s.UpdatedAt = time.Now()
}

// GetDeviceSerial returns the device serial number for this session.
func (s *Session) GetDeviceSerial() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DeviceSerial
}

// SetParameterTree stores full parameter tree from Inform (TR-181 / TR-098).
func (s *Session) SetParameterTree(params map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ParameterTree = params
	s.UpdatedAt = time.Now()
}

// GetParameterTree returns a copy of the parameter tree.
func (s *Session) GetParameterTree() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.ParameterTree))
	for k, v := range s.ParameterTree {
		out[k] = v
	}
	return out
}

// EnqueueTask appends a task to the queue.
func (s *Session) EnqueueTask(t interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TaskQueue = append(s.TaskQueue, t)
	s.UpdatedAt = time.Now()
}

// NextTask pops and sets current task; returns nil if queue empty.
func (s *Session) NextTask() interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.TaskQueue) == 0 {
		s.CurrentTask = nil
		return nil
	}
	t := s.TaskQueue[0]
	s.TaskQueue = s.TaskQueue[1:]
	s.CurrentTask = t
	s.UpdatedAt = time.Now()
	return t
}

// ClearCurrentTask clears the task awaiting response.
func (s *Session) ClearCurrentTask() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentTask = nil
	s.CurrentTaskID = ""
	s.CurrentTaskType = ""
	s.UpdatedAt = time.Now()
}

// SetCurrentTask sets the task currently being executed.
func (s *Session) SetCurrentTask(t *task.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentTask = t
	if t != nil {
		s.CurrentTaskID = t.ID
		s.CurrentTaskType = t.Type
	} else {
		s.CurrentTaskID = ""
		s.CurrentTaskType = ""
	}
	s.UpdatedAt = time.Now()
}

// GetCurrentTaskID returns the ID of the task awaiting response.
func (s *Session) GetCurrentTaskID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentTaskID
}

// GetCurrentTaskType returns the type of the task awaiting response.
func (s *Session) GetCurrentTaskType() task.Type {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentTaskType
}

// GetID returns the session ID (for SOAP cwmp:ID correlation).
func (s *Session) GetID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ID
}

// HasPendingTasks returns true if queue is non-empty or current task is set.
func (s *Session) HasPendingTasks() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.TaskQueue) > 0 || s.CurrentTask != nil
}

// SetCWMPID sets the last CWMP message ID (for response correlation).
func (s *Session) SetCWMPID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CWMPID = id
}

// GetCWMPID returns last CWMP ID.
func (s *Session) GetCWMPID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CWMPID
}

// Done returns the session's context done channel.
func (s *Session) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Close cancels the session context.
func (s *Session) Close() {
	s.cancel()
}

// Expired returns true if session has passed its deadline.
func (s *Session) Expired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.Deadline)
}
