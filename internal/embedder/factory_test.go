package embedder

import (
	"errors"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

func TestNewEmbedder_OpenAI(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		Dim:      0,
	}

	emb, err := NewEmbedder(cfg, "test-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := emb.(*OpenAIEmbedder); !ok {
		t.Errorf("expected *OpenAIEmbedder, got %T", emb)
	}
}

func TestNewEmbedder_OpenAI_CfgAPIKey(t *testing.T) {
	apiKey := "cfg-api-key"
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		APIKey:   &apiKey,
	}

	// cfg.APIKey takes priority over envAPIKey
	emb, err := NewEmbedder(cfg, "env-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	openaiEmb := emb.(*OpenAIEmbedder)
	if openaiEmb.apiKey != "cfg-api-key" {
		t.Errorf("expected cfg-api-key, got %s", openaiEmb.apiKey)
	}
}

func TestNewEmbedder_OpenAI_EnvAPIKey(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		APIKey:   nil, // nil means use envAPIKey
	}

	emb, err := NewEmbedder(cfg, "env-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	openaiEmb := emb.(*OpenAIEmbedder)
	if openaiEmb.apiKey != "env-api-key" {
		t.Errorf("expected env-api-key, got %s", openaiEmb.apiKey)
	}
}

func TestNewEmbedder_OpenAI_NoAPIKey(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
	}

	_, err := NewEmbedder(cfg, "", nil)
	if !errors.Is(err, ErrAPIKeyRequired) {
		t.Errorf("expected ErrAPIKeyRequired, got %v", err)
	}
}

func TestNewEmbedder_OpenAI_BaseURL(t *testing.T) {
	baseURL := "https://custom.openai.com/v1"
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		BaseURL:  &baseURL,
	}

	emb, err := NewEmbedder(cfg, "test-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	openaiEmb := emb.(*OpenAIEmbedder)
	if openaiEmb.baseURL != baseURL {
		t.Errorf("expected %s, got %s", baseURL, openaiEmb.baseURL)
	}
}

func TestNewEmbedder_OpenAI_Model(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-large",
	}

	emb, err := NewEmbedder(cfg, "test-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	openaiEmb := emb.(*OpenAIEmbedder)
	if openaiEmb.model != "text-embedding-3-large" {
		t.Errorf("expected text-embedding-3-large, got %s", openaiEmb.model)
	}
}

func TestNewEmbedder_OpenAI_Dim(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		Dim:      1536,
	}

	emb, err := NewEmbedder(cfg, "test-api-key", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dim := emb.GetDimension(); dim != 1536 {
		t.Errorf("expected 1536, got %d", dim)
	}
}

func TestNewEmbedder_Ollama(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "ollama",
		Model:    "nomic-embed-text",
	}

	emb, err := NewEmbedder(cfg, "", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := emb.(*OllamaEmbedder); !ok {
		t.Errorf("expected *OllamaEmbedder, got %T", emb)
	}
}

func TestNewEmbedder_Local(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "local",
	}

	emb, err := NewEmbedder(cfg, "", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, ok := emb.(*LocalEmbedder); !ok {
		t.Errorf("expected *LocalEmbedder, got %T", emb)
	}
}

func TestNewEmbedder_Unknown(t *testing.T) {
	cfg := &model.EmbedderConfig{
		Provider: "unknown-provider",
	}

	_, err := NewEmbedder(cfg, "", nil)
	if !errors.Is(err, ErrUnknownProvider) {
		t.Errorf("expected ErrUnknownProvider, got %v", err)
	}
}
