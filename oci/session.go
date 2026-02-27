package oci

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// UploadSession tracks an in-progress blob upload.
type UploadSession struct {
	UUID       string
	Repository string
	StartedAt  time.Time
	BytesWritten int64
}

// SessionManager manages upload sessions in memory.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*UploadSession
	timeout  time.Duration
}

// NewSessionManager creates a new session manager with the given timeout.
func NewSessionManager(timeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*UploadSession),
		timeout:  timeout,
	}
}

// Create creates a new upload session and returns the UUID.
func (sm *SessionManager) Create(repository string) (string, error) {
	uuid, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	session := &UploadSession{
		UUID:       uuid,
		Repository: repository,
		StartedAt:  time.Now(),
	}

	sm.mu.Lock()
	sm.sessions[uuid] = session
	sm.mu.Unlock()

	return uuid, nil
}

// Get retrieves a session by UUID. Returns ErrUploadNotFound if not found or expired.
func (sm *SessionManager) Get(uuid string) (*UploadSession, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[uuid]
	sm.mu.RUnlock()

	if !ok {
		return nil, ErrUploadNotFound
	}

	if time.Since(session.StartedAt) > sm.timeout {
		sm.Delete(uuid)
		return nil, ErrUploadNotFound
	}

	return session, nil
}

// UpdateBytes updates the bytes written count for a session.
func (sm *SessionManager) UpdateBytes(uuid string, bytesWritten int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[uuid]
	if !ok {
		return ErrUploadNotFound
	}

	session.BytesWritten = bytesWritten
	return nil
}

// Delete removes a session by UUID.
func (sm *SessionManager) Delete(uuid string) {
	sm.mu.Lock()
	delete(sm.sessions, uuid)
	sm.mu.Unlock()
}

// generateUUID generates a random UUID v4.
func generateUUID() (string, error) {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", err
	}
	// Set version 4 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}
