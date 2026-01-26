package model

import (
	"encoding/json"
	"testing"
)

// TestConfig_JSONMarshal はConfigが正しくJSONシリアライズされることをテスト
func TestConfig_JSONMarshal(t *testing.T) {
	baseURL := "https://api.example.com"
	apiKey := "test-api-key"
	storePath := "/data/store.db"

	config := &Config{
		TransportDefaults: TransportDefaults{
			DefaultTransport: TransportStdio,
		},
		Embedder: EmbedderConfig{
			Provider: ProviderOpenAI,
			Model:    "text-embedding-3-small",
			Dim:      1536,
			BaseURL:  &baseURL,
			APIKey:   &apiKey,
		},
		Store: StoreConfig{
			Type: StoreTypeSQLite,
			Path: &storePath,
			URL:  nil,
		},
		Paths: PathsConfig{
			ConfigPath: "/config/mcp-memory.json",
			DataDir:    "/data",
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal Config: %v", err)
	}

	var unmarshaled Config
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Config: %v", err)
	}

	// 基本的な値の検証
	if unmarshaled.TransportDefaults.DefaultTransport != TransportStdio {
		t.Errorf("expected DefaultTransport %q, got %q", TransportStdio, unmarshaled.TransportDefaults.DefaultTransport)
	}
	if unmarshaled.Embedder.Provider != ProviderOpenAI {
		t.Errorf("expected Provider %q, got %q", ProviderOpenAI, unmarshaled.Embedder.Provider)
	}
	if unmarshaled.Embedder.Dim != 1536 {
		t.Errorf("expected Dim %d, got %d", 1536, unmarshaled.Embedder.Dim)
	}
	if unmarshaled.Store.Type != StoreTypeSQLite {
		t.Errorf("expected Store Type %q, got %q", StoreTypeSQLite, unmarshaled.Store.Type)
	}
}

// TestConfig_JSONUnmarshal はJSONからConfigが正しくデシリアライズされることをテスト
func TestConfig_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"transportDefaults": {
			"defaultTransport": "http"
		},
		"embedder": {
			"provider": "ollama",
			"model": "nomic-embed-text",
			"dim": 768,
			"baseUrl": "http://localhost:11434"
		},
		"store": {
			"type": "chroma",
			"url": "http://localhost:8000"
		},
		"paths": {
			"configPath": "/config/test.json",
			"dataDir": "/data/test"
		}
	}`

	var config Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// TransportDefaults
	if config.TransportDefaults.DefaultTransport != TransportHTTP {
		t.Errorf("expected DefaultTransport %q, got %q", TransportHTTP, config.TransportDefaults.DefaultTransport)
	}

	// Embedder
	if config.Embedder.Provider != ProviderOllama {
		t.Errorf("expected Provider %q, got %q", ProviderOllama, config.Embedder.Provider)
	}
	if config.Embedder.Model != "nomic-embed-text" {
		t.Errorf("expected Model %q, got %q", "nomic-embed-text", config.Embedder.Model)
	}
	if config.Embedder.Dim != 768 {
		t.Errorf("expected Dim %d, got %d", 768, config.Embedder.Dim)
	}
	if config.Embedder.BaseURL == nil {
		t.Error("expected BaseURL to be non-nil")
	} else if *config.Embedder.BaseURL != "http://localhost:11434" {
		t.Errorf("expected BaseURL %q, got %q", "http://localhost:11434", *config.Embedder.BaseURL)
	}

	// Store
	if config.Store.Type != StoreTypeChroma {
		t.Errorf("expected Store Type %q, got %q", StoreTypeChroma, config.Store.Type)
	}
	if config.Store.URL == nil {
		t.Error("expected URL to be non-nil")
	} else if *config.Store.URL != "http://localhost:8000" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8000", *config.Store.URL)
	}

	// Paths
	if config.Paths.ConfigPath != "/config/test.json" {
		t.Errorf("expected ConfigPath %q, got %q", "/config/test.json", config.Paths.ConfigPath)
	}
	if config.Paths.DataDir != "/data/test" {
		t.Errorf("expected DataDir %q, got %q", "/data/test", config.Paths.DataDir)
	}
}

// TestEmbedderConfig_OmitEmpty は空のBaseURL/APIKeyがJSON出力で省略されることをテスト
func TestEmbedderConfig_OmitEmpty(t *testing.T) {
	config := &EmbedderConfig{
		Provider: ProviderLocal,
		Model:    "all-MiniLM-L6-v2",
		Dim:      384,
		BaseURL:  nil,
		APIKey:   nil,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal EmbedderConfig: %v", err)
	}

	// baseUrlとapiKeyが出力されていないことを確認
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if _, exists := result["baseUrl"]; exists {
		t.Error("expected baseUrl to be omitted, but it exists in JSON")
	}
	if _, exists := result["apiKey"]; exists {
		t.Error("expected apiKey to be omitted, but it exists in JSON")
	}

	// 必須フィールドは存在すること
	if _, exists := result["provider"]; !exists {
		t.Error("expected provider to exist in JSON")
	}
	if _, exists := result["model"]; !exists {
		t.Error("expected model to exist in JSON")
	}
	if _, exists := result["dim"]; !exists {
		t.Error("expected dim to exist in JSON")
	}
}
