package oci

import (
	"testing"
	"time"
)

func TestSessionManager_CreateAndGet(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	uuid, err := sm.Create("myrepo")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if uuid == "" {
		t.Fatal("UUID should not be empty")
	}

	session, err := sm.Get(uuid)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session.UUID != uuid {
		t.Errorf("UUID = %q, want %q", session.UUID, uuid)
	}
	if session.Repository != "myrepo" {
		t.Errorf("Repository = %q, want %q", session.Repository, "myrepo")
	}
}

func TestSessionManager_GetNotFound(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	_, err := sm.Get("nonexistent")
	if err != ErrUploadNotFound {
		t.Errorf("expected ErrUploadNotFound, got %v", err)
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	uuid, _ := sm.Create("myrepo")
	sm.Delete(uuid)

	_, err := sm.Get(uuid)
	if err != ErrUploadNotFound {
		t.Errorf("expected ErrUploadNotFound after delete, got %v", err)
	}
}

func TestSessionManager_UpdateBytes(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	uuid, _ := sm.Create("myrepo")
	err := sm.UpdateBytes(uuid, 1024)
	if err != nil {
		t.Fatalf("failed to update bytes: %v", err)
	}

	session, _ := sm.Get(uuid)
	if session.BytesWritten != 1024 {
		t.Errorf("BytesWritten = %d, want 1024", session.BytesWritten)
	}
}

func TestSessionManager_UpdateBytesNotFound(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	err := sm.UpdateBytes("nonexistent", 1024)
	if err != ErrUploadNotFound {
		t.Errorf("expected ErrUploadNotFound, got %v", err)
	}
}

func TestSessionManager_Expiry(t *testing.T) {
	// Create with very short timeout
	sm := NewSessionManager(1 * time.Millisecond)

	uuid, _ := sm.Create("myrepo")

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	_, err := sm.Get(uuid)
	if err != ErrUploadNotFound {
		t.Errorf("expected ErrUploadNotFound for expired session, got %v", err)
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid1, err := generateUUID()
	if err != nil {
		t.Fatalf("failed to generate UUID: %v", err)
	}
	uuid2, err := generateUUID()
	if err != nil {
		t.Fatalf("failed to generate UUID: %v", err)
	}

	if uuid1 == uuid2 {
		t.Error("two generated UUIDs should be different")
	}
	if len(uuid1) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid1))
	}
}
