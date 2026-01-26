package service

import (
	"context"
	"errors"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/model"
)

func newTestConfigService(mgr *config.Manager) *configService {
	return &configService{
		manager: mgr,
	}
}

func TestConfigService_GetConfig_Success(t *testing.T) {
	// Create temp config for testing
	cfg := &model.Config{
		TransportDefaults: model.TransportDefaults{
			DefaultTransport: "stdio",
		},
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
		Store: model.StoreConfig{
			Type: "chroma",
		},
		Paths: model.PathsConfig{
			ConfigPath: "/tmp/test/config.json",
			DataDir:    "/tmp/test/data",
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	resp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if resp.TransportDefaults.DefaultTransport != "stdio" {
		t.Errorf("expected defaultTransport stdio, got %s", resp.TransportDefaults.DefaultTransport)
	}
	if resp.Embedder.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", resp.Embedder.Provider)
	}
	if resp.Embedder.Dim != 1536 {
		t.Errorf("expected dim 1536, got %d", resp.Embedder.Dim)
	}
}

func TestConfigService_SetConfig_Provider(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	newProvider := "ollama"
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			Provider: &newProvider,
		},
	}

	resp, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK to be true")
	}

	// Verify change
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Provider != "ollama" {
		t.Errorf("expected provider ollama, got %s", getResp.Embedder.Provider)
	}
}

func TestConfigService_SetConfig_Model(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	newModel := "text-embedding-3-large"
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			Model: &newModel,
		},
	}

	resp, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK to be true")
	}

	// Verify change
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Model != "text-embedding-3-large" {
		t.Errorf("expected model text-embedding-3-large, got %s", getResp.Embedder.Model)
	}
}

func TestConfigService_SetConfig_DimReset(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536, // has dim
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	// Change provider - should reset dim
	newProvider := "ollama"
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			Provider: &newProvider,
		},
	}

	_, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Verify dim reset
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Dim != 0 {
		t.Errorf("expected dim to be reset to 0, got %d", getResp.Embedder.Dim)
	}
}

func TestConfigService_SetConfig_NamespaceChange(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	newProvider := "ollama"
	newModel := "nomic-embed-text"
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			Provider: &newProvider,
			Model:    &newModel,
		},
	}

	resp, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Namespace should reflect new provider/model with dim=0
	expectedNS := "ollama:nomic-embed-text:0"
	if resp.EffectiveNamespace != expectedNS {
		t.Errorf("expected namespace %s, got %s", expectedNS, resp.EffectiveNamespace)
	}
}

func TestConfigService_SetConfig_NilPatch(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	// nil Embedder patch - should be no-op
	req := &SetConfigRequest{
		Embedder: nil,
	}

	resp, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK to be true")
	}

	// Verify no change
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Provider != "openai" {
		t.Errorf("expected provider unchanged, got %s", getResp.Embedder.Provider)
	}
}

func TestConfigService_SetConfig_EmptyProvider(t *testing.T) {
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	// Empty string provider - should be ignored (keep original)
	emptyProvider := ""
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			Provider: &emptyProvider,
		},
	}

	_, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Empty provider should be ignored, original value preserved
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Provider != "openai" {
		t.Errorf("expected provider unchanged when empty, got %s", getResp.Embedder.Provider)
	}
}

func TestConfigService_SetConfig_PartialPatch(t *testing.T) {
	baseURL := "https://api.openai.com"
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
			BaseURL:  &baseURL,
		},
	}

	mgr := config.NewManagerWithConfig(cfg)
	svc := newTestConfigService(mgr)

	// Only update APIKey, other fields should remain
	apiKey := "sk-test-key"
	req := &SetConfigRequest{
		Embedder: &EmbedderPatch{
			APIKey: &apiKey,
		},
	}

	_, err := svc.SetConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Verify other fields unchanged
	getResp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if getResp.Embedder.Provider != "openai" {
		t.Errorf("expected provider unchanged, got %s", getResp.Embedder.Provider)
	}
	if getResp.Embedder.Model != "text-embedding-3-small" {
		t.Errorf("expected model unchanged, got %s", getResp.Embedder.Model)
	}
	if getResp.Embedder.BaseURL == nil || *getResp.Embedder.BaseURL != "https://api.openai.com" {
		t.Errorf("expected baseURL unchanged, got %v", getResp.Embedder.BaseURL)
	}
}

// Stub implementation for tests to compile
type configService struct {
	manager *config.Manager
}

func (s *configService) GetConfig(ctx context.Context) (*GetConfigResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *configService) SetConfig(ctx context.Context, req *SetConfigRequest) (*SetConfigResponse, error) {
	return nil, errors.New("not implemented")
}
