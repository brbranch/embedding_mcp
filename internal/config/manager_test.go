package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestManager_NewManager_DefaultPath はデフォルトパスでManagerが作成されることをテスト
func TestManager_NewManager_DefaultPath(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// デフォルトパスが設定されていることを確認
	cfg := mgr.GetConfig()
	if cfg.Paths.ConfigPath == "" {
		t.Error("expected non-empty config path")
	}

	if cfg.Paths.DataDir == "" {
		t.Error("expected non-empty data dir")
	}
}

// TestManager_NewManager_CustomPath はカスタムパスでManagerが作成されることをテスト
func TestManager_NewManager_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	cfg := mgr.GetConfig()
	if cfg.Paths.ConfigPath != configPath {
		t.Errorf("expected config path %q, got %q", configPath, cfg.Paths.ConfigPath)
	}
}

// TestManager_Load_NotExist は設定ファイルが存在しない場合にデフォルト設定が使われることをテスト
func TestManager_Load_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.Load()
	if err != nil {
		t.Fatalf("unexpected error on load: %v", err)
	}

	// デフォルト設定が使われることを確認
	cfg := mgr.GetConfig()
	if cfg.Embedder.Provider != "openai" {
		t.Errorf("expected default provider 'openai', got %q", cfg.Embedder.Provider)
	}
}

// TestManager_Load_Exist は既存の設定ファイルがロードされることをテスト
func TestManager_Load_Exist(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 設定ファイルを作成
	configJSON := `{
		"transportDefaults": {
			"defaultTransport": "http"
		},
		"embedder": {
			"provider": "ollama",
			"model": "nomic-embed-text",
			"dim": 768
		},
		"store": {
			"type": "sqlite"
		},
		"paths": {
			"configPath": "` + configPath + `",
			"dataDir": "` + filepath.Join(tmpDir, "data") + `"
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.Load()
	if err != nil {
		t.Fatalf("unexpected error on load: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg.TransportDefaults.DefaultTransport != "http" {
		t.Errorf("expected transport 'http', got %q", cfg.TransportDefaults.DefaultTransport)
	}

	if cfg.Embedder.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %q", cfg.Embedder.Provider)
	}

	if cfg.Embedder.Model != "nomic-embed-text" {
		t.Errorf("expected model 'nomic-embed-text', got %q", cfg.Embedder.Model)
	}

	if cfg.Embedder.Dim != 768 {
		t.Errorf("expected dim 768, got %d", cfg.Embedder.Dim)
	}

	if cfg.Store.Type != "sqlite" {
		t.Errorf("expected store type 'sqlite', got %q", cfg.Store.Type)
	}
}

// TestManager_Load_Invalid は不正なJSONでエラーになることをテスト
func TestManager_Load_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	// 不正なJSONを書き込み
	invalidJSON := `{"embedder": "not an object"}`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.Load()
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// TestManager_Save は設定が正しく保存されることをテスト
func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 設定を保存
	err = mgr.Save()
	if err != nil {
		t.Fatalf("unexpected error on save: %v", err)
	}

	// ファイルが作成されたことを確認
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

// TestManager_SaveAndLoad は保存した設定が正しくロードされることをテスト
func TestManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 最初のManagerで設定を保存
	mgr1, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %v", err)
	}

	// embedder設定を更新
	apiKey := "test-api-key"
	patch := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-large",
		APIKey:   &apiKey,
	}
	if err := mgr1.UpdateEmbedder(patch); err != nil {
		t.Fatalf("unexpected error updating embedder: %v", err)
	}

	if err := mgr1.Save(); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	// 新しいManagerでロード
	mgr2, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error creating second manager: %v", err)
	}

	if err := mgr2.Load(); err != nil {
		t.Fatalf("unexpected error loading: %v", err)
	}

	// 設定が一致することを確認
	cfg := mgr2.GetConfig()
	if cfg.Embedder.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", cfg.Embedder.Provider)
	}

	if cfg.Embedder.Model != "text-embedding-3-large" {
		t.Errorf("expected model 'text-embedding-3-large', got %q", cfg.Embedder.Model)
	}

	if cfg.Embedder.APIKey == nil || *cfg.Embedder.APIKey != "test-api-key" {
		t.Errorf("expected apiKey 'test-api-key', got %v", cfg.Embedder.APIKey)
	}
}

// TestManager_UpdateEmbedder はembedder設定が更新されることをテスト
func TestManager_UpdateEmbedder(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 初期設定を確認
	cfg := mgr.GetConfig()
	initialProvider := cfg.Embedder.Provider
	initialModel := cfg.Embedder.Model

	// 設定を更新
	newModel := "text-embedding-3-large"
	patch := &model.EmbedderConfig{
		Model: newModel,
	}

	if err := mgr.UpdateEmbedder(patch); err != nil {
		t.Fatalf("unexpected error updating embedder: %v", err)
	}

	// 更新が反映されていることを確認
	cfg = mgr.GetConfig()
	if cfg.Embedder.Provider != initialProvider {
		t.Errorf("provider should not change, expected %q, got %q", initialProvider, cfg.Embedder.Provider)
	}

	if cfg.Embedder.Model != newModel {
		t.Errorf("expected model %q, got %q", newModel, cfg.Embedder.Model)
	}
}

// TestManager_UpdateEmbedder_StorePathsUnchanged はembedder更新時にstoreとpathsが変更されないことをテスト
func TestManager_UpdateEmbedder_StorePathsUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := mgr.GetConfig()
	initialStoreType := cfg.Store.Type
	initialConfigPath := cfg.Paths.ConfigPath
	initialDataDir := cfg.Paths.DataDir

	// embedder設定を更新
	patch := &model.EmbedderConfig{
		Provider: "ollama",
		Model:    "nomic-embed-text",
	}

	if err := mgr.UpdateEmbedder(patch); err != nil {
		t.Fatalf("unexpected error updating embedder: %v", err)
	}

	// store/paths が変更されていないことを確認
	cfg = mgr.GetConfig()
	if cfg.Store.Type != initialStoreType {
		t.Errorf("store type should not change, expected %q, got %q", initialStoreType, cfg.Store.Type)
	}

	if cfg.Paths.ConfigPath != initialConfigPath {
		t.Errorf("config path should not change, expected %q, got %q", initialConfigPath, cfg.Paths.ConfigPath)
	}

	if cfg.Paths.DataDir != initialDataDir {
		t.Errorf("data dir should not change, expected %q, got %q", initialDataDir, cfg.Paths.DataDir)
	}
}

// TestManager_UpdateDim は次元数が更新されることをテスト
func TestManager_UpdateDim(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 初期dimを確認
	cfg := mgr.GetConfig()
	if cfg.Embedder.Dim != 0 {
		t.Errorf("expected initial dim 0, got %d", cfg.Embedder.Dim)
	}

	// dimを更新
	newDim := 1536
	if err := mgr.UpdateDim(newDim); err != nil {
		t.Fatalf("unexpected error updating dim: %v", err)
	}

	// 更新が反映されていることを確認
	cfg = mgr.GetConfig()
	if cfg.Embedder.Dim != newDim {
		t.Errorf("expected dim %d, got %d", newDim, cfg.Embedder.Dim)
	}
}

// TestDefaultConfig はデフォルト設定が正しく生成されることをテスト
func TestDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	dataDir := filepath.Join(tmpDir, "data")

	cfg := DefaultConfig(configPath, dataDir)

	if cfg.TransportDefaults.DefaultTransport != "stdio" {
		t.Errorf("expected default transport 'stdio', got %q", cfg.TransportDefaults.DefaultTransport)
	}

	if cfg.Embedder.Provider != "openai" {
		t.Errorf("expected default provider 'openai', got %q", cfg.Embedder.Provider)
	}

	if cfg.Embedder.Model != "text-embedding-3-small" {
		t.Errorf("expected default model 'text-embedding-3-small', got %q", cfg.Embedder.Model)
	}

	if cfg.Embedder.Dim != 0 {
		t.Errorf("expected default dim 0, got %d", cfg.Embedder.Dim)
	}

	if cfg.Store.Type != "chroma" {
		t.Errorf("expected default store type 'chroma', got %q", cfg.Store.Type)
	}

	if cfg.Paths.ConfigPath != configPath {
		t.Errorf("expected config path %q, got %q", configPath, cfg.Paths.ConfigPath)
	}

	if cfg.Paths.DataDir != dataDir {
		t.Errorf("expected data dir %q, got %q", dataDir, cfg.Paths.DataDir)
	}
}
