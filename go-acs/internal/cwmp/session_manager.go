package cwmp

import (
	"sync"
	"time"
)

// SessionManager stores and retrieves sessions by ID (e.g. cookie/session id).
type SessionManager struct {
	mu            sync.Map // map[string]*Session
	timeout       time.Duration
	maxConcurrent int
	current       int
	currentMu     sync.Mutex
}

// NewSessionManager creates a manager with session timeout and max concurrent limit.
func NewSessionManager(timeout time.Duration, maxConcurrent int) *SessionManager {
	return &SessionManager{
		timeout:       timeout,
		maxConcurrent: maxConcurrent,
	}
}

// Get returns a session by id, or nil.
func (m *SessionManager) Get(id string) *Session {
	v, _ := m.mu.Load(id)
	if v == nil {
		return nil
	}
	return v.(*Session)
}

// Put stores a session. Replaces existing. Does not enforce maxConcurrent; caller may check.
func (m *SessionManager) Put(id string, s *Session) {
	m.mu.Store(id, s)
}

// Delete removes a session and closes it.
func (m *SessionManager) Delete(id string) {
	if v, ok := m.mu.LoadAndDelete(id); ok {
		if s, _ := v.(*Session); s != nil {
			s.Close()
		}
	}
}

// NewSession creates a new session, stores it, and returns it.
func (m *SessionManager) NewSession(id string) *Session {
	s := NewSession(id, m.timeout)
	m.mu.Store(id, s)
	return s
}

// GetOrCreate returns existing session or creates a new one.
func (m *SessionManager) GetOrCreate(id string) *Session {
	if s := m.Get(id); s != nil && !s.Expired() {
		return s
	}
	m.Delete(id)
	return m.NewSession(id)
}

// CleanupExpired removes sessions that have passed their deadline (call periodically).
func (m *SessionManager) CleanupExpired() {
	m.mu.Range(func(key, value interface{}) bool {
		s, _ := value.(*Session)
		if s != nil && s.Expired() {
			m.mu.Delete(key)
			s.Close()
		}
		return true
	})
}
