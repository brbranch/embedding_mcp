package config

import (
	"os"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestApplyEnvOverrides_OpenAIAPIKey は環境変数からOpenAI APIキーが設定されることをテスト
func TestApplyEnvOverrides_OpenAIAPIKey(t *testing.T) {
	// 環境変数を設定
	t.Setenv("OPENAI_API_KEY", "env-test-key")

	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
		},
	}

	ApplyEnvOverrides(cfg)

	if cfg.Embedder.APIKey == nil {
		t.Fatal("expected APIKey to be set from environment")
	}

	if *cfg.Embedder.APIKey != "env-test-key" {
		t.Errorf("expected apiKey 'env-test-key', got %q", *cfg.Embedder.APIKey)
	}
}

// TestApplyEnvOverrides_NoEnv は環境変数が設定されていない場合に設定が変更されないことをテスト
func TestApplyEnvOverrides_NoEnv(t *testing.T) {
	// 環境変数が設定されていないことを確認
	os.Unsetenv("OPENAI_API_KEY")

	existingKey := "config-key"
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			APIKey:   &existingKey,
		},
	}

	ApplyEnvOverrides(cfg)

	// 既存の設定が保持されることを確認
	if cfg.Embedder.APIKey == nil {
		t.Fatal("expected APIKey to remain set")
	}

	if *cfg.Embedder.APIKey != "config-key" {
		t.Errorf("expected apiKey 'config-key', got %q", *cfg.Embedder.APIKey)
	}
}

// TestGetOpenAIAPIKey_EnvPriority は環境変数が優先されることをテスト
func TestGetOpenAIAPIKey_EnvPriority(t *testing.T) {
	// 環境変数を設定
	t.Setenv("OPENAI_API_KEY", "env-key")

	configKey := "config-key"
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			APIKey:   &configKey,
		},
	}

	key := GetOpenAIAPIKey(cfg)

	if key != "env-key" {
		t.Errorf("expected env key 'env-key' to take priority, got %q", key)
	}
}

// TestGetOpenAIAPIKey_ConfigFallback は環境変数がない場合に設定ファイルの値が使われることをテスト
func TestGetOpenAIAPIKey_ConfigFallback(t *testing.T) {
	// 環境変数が設定されていないことを確認
	os.Unsetenv("OPENAI_API_KEY")

	configKey := "config-key"
	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			APIKey:   &configKey,
		},
	}

	key := GetOpenAIAPIKey(cfg)

	if key != "config-key" {
		t.Errorf("expected config key 'config-key', got %q", key)
	}
}

// TestGetOpenAIAPIKey_Empty は環境変数も設定もない場合に空文字列が返されることをテスト
func TestGetOpenAIAPIKey_Empty(t *testing.T) {
	// 環境変数が設定されていないことを確認
	os.Unsetenv("OPENAI_API_KEY")

	cfg := &model.Config{
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			APIKey:   nil,
		},
	}

	key := GetOpenAIAPIKey(cfg)

	if key != "" {
		t.Errorf("expected empty string, got %q", key)
	}
}
