package service

import (
	"context"
	"errors"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/store"
)

func newTestGlobalService(s store.Store, namespace string) *globalService {
	// Initialize store if needed
	s.Initialize(context.Background(), namespace)
	return &globalService{
		store:     s,
		namespace: namespace,
	}
}

func TestGlobalService_UpsertGlobal_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	req := &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "global.persona",
		Value:     "helpful assistant",
	}

	resp, err := svc.UpsertGlobal(context.Background(), req)
	if err != nil {
		t.Fatalf("UpsertGlobal failed: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK to be true")
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Namespace != "openai:test:3" {
		t.Errorf("expected namespace openai:test:3, got %s", resp.Namespace)
	}
}

func TestGlobalService_UpsertGlobal_ProjectIDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	req := &UpsertGlobalRequest{
		Key:   "global.persona",
		Value: "helper",
	}

	_, err := svc.UpsertGlobal(context.Background(), req)
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestGlobalService_UpsertGlobal_InvalidKey(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	req := &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "local.setting", // missing "global." prefix
		Value:     "value",
	}

	_, err := svc.UpsertGlobal(context.Background(), req)
	if !errors.Is(err, ErrInvalidGlobalKey) {
		t.Errorf("expected ErrInvalidGlobalKey, got %v", err)
	}
}

func TestGlobalService_UpsertGlobal_UpdatedAtDefault(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	req := &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "global.test",
		Value:     "value",
		// UpdatedAt not set
	}

	resp, err := svc.UpsertGlobal(context.Background(), req)
	if err != nil {
		t.Fatalf("UpsertGlobal failed: %v", err)
	}

	// Verify updatedAt was set
	getResp, err := svc.GetGlobal(context.Background(), "/test/project", "global.test")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}

	if !getResp.Found {
		t.Fatal("expected to find global config")
	}
	if getResp.UpdatedAt == nil {
		t.Error("expected updatedAt to be set")
	}

	_ = resp // silence unused warning
}

func TestGlobalService_GetGlobal_Found(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	// Upsert first
	_, _ = svc.UpsertGlobal(context.Background(), &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "global.persona",
		Value:     "helpful",
	})

	// Get
	resp, err := svc.GetGlobal(context.Background(), "/test/project", "global.persona")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}

	if !resp.Found {
		t.Error("expected found to be true")
	}
	if resp.Value != "helpful" {
		t.Errorf("expected value 'helpful', got %v", resp.Value)
	}
	if resp.Namespace != "openai:test:3" {
		t.Errorf("expected namespace openai:test:3, got %s", resp.Namespace)
	}
}

func TestGlobalService_GetGlobal_NotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	resp, err := svc.GetGlobal(context.Background(), "/test/project", "global.nonexistent")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}

	if resp.Found {
		t.Error("expected found to be false")
	}
	if resp.ID != nil {
		t.Error("expected ID to be nil")
	}
}

func TestGlobalService_GetGlobal_ProjectIDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	_, err := svc.GetGlobal(context.Background(), "", "global.test")
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestGlobalService_UpsertGlobal_EmptyKey(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	req := &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "", // empty key
		Value:     "value",
	}

	_, err := svc.UpsertGlobal(context.Background(), req)
	// Empty key should fail (doesn't have "global." prefix)
	if !errors.Is(err, ErrInvalidGlobalKey) {
		t.Errorf("expected ErrInvalidGlobalKey for empty key, got %v", err)
	}
}

func TestGlobalService_UpsertGlobal_Overwrite(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	// First upsert
	_, err := svc.UpsertGlobal(context.Background(), &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "global.test",
		Value:     "first",
	})
	if err != nil {
		t.Fatalf("first UpsertGlobal failed: %v", err)
	}

	// Overwrite with second upsert
	_, err = svc.UpsertGlobal(context.Background(), &UpsertGlobalRequest{
		ProjectID: "/test/project",
		Key:       "global.test",
		Value:     "second",
	})
	if err != nil {
		t.Fatalf("second UpsertGlobal failed: %v", err)
	}

	// Verify new value
	resp, err := svc.GetGlobal(context.Background(), "/test/project", "global.test")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}

	if resp.Value != "second" {
		t.Errorf("expected value 'second', got %v", resp.Value)
	}
}

func TestGlobalService_GetGlobal_EmptyKey(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	resp, err := svc.GetGlobal(context.Background(), "/test/project", "")
	if err != nil {
		// Empty key might return not found or error
		t.Logf("GetGlobal with empty key returned error: %v", err)
		return
	}

	// If no error, should not be found
	if resp.Found {
		t.Error("expected found to be false for empty key")
	}
}

func TestGlobalService_UpsertGlobal_StandardKeys(t *testing.T) {
	// Test standard keys defined in spec
	standardKeys := []string{
		"global.memory.embedder.provider",
		"global.memory.embedder.model",
		"global.memory.groupDefaults",
		"global.project.conventions",
	}

	memStore := store.NewMemoryStore()
	svc := newTestGlobalService(memStore, "openai:test:3")

	for _, key := range standardKeys {
		_, err := svc.UpsertGlobal(context.Background(), &UpsertGlobalRequest{
			ProjectID: "/test/project",
			Key:       key,
			Value:     "test-value",
		})
		if err != nil {
			t.Errorf("UpsertGlobal failed for key %s: %v", key, err)
		}
	}
}