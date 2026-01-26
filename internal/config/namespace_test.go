package config

import (
	"testing"
)

// TestGenerateNamespace_WithDim は有効な次元数でnamespaceが生成されることをテスト
func TestGenerateNamespace_WithDim(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		dim      int
		expected string
	}{
		{
			name:     "openai with dim",
			provider: "openai",
			model:    "text-embedding-3-small",
			dim:      1536,
			expected: "openai:text-embedding-3-small:1536",
		},
		{
			name:     "ollama with dim",
			provider: "ollama",
			model:    "nomic-embed-text",
			dim:      768,
			expected: "ollama:nomic-embed-text:768",
		},
		{
			name:     "local with dim",
			provider: "local",
			model:    "custom-model",
			dim:      512,
			expected: "local:custom-model:512",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := GenerateNamespace(tt.provider, tt.model, tt.dim)
			if ns != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, ns)
			}
		})
	}
}

// TestGenerateNamespace_DimZero は次元数0の場合にdimを含まないnamespaceが生成されることをテスト
func TestGenerateNamespace_DimZero(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		expected string
	}{
		{
			name:     "openai without dim",
			provider: "openai",
			model:    "text-embedding-3-small",
			expected: "openai:text-embedding-3-small",
		},
		{
			name:     "ollama without dim",
			provider: "ollama",
			model:    "nomic-embed-text",
			expected: "ollama:nomic-embed-text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := GenerateNamespace(tt.provider, tt.model, 0)
			if ns != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, ns)
			}
		})
	}
}

// TestGenerateNamespace_Ollama はOllamaプロバイダで正しいnamespaceが生成されることをテスト
func TestGenerateNamespace_Ollama(t *testing.T) {
	ns := GenerateNamespace("ollama", "nomic-embed-text", 768)
	expected := "ollama:nomic-embed-text:768"

	if ns != expected {
		t.Errorf("expected %q, got %q", expected, ns)
	}
}

// TestParseNamespace_Valid は有効なnamespace文字列がパースできることをテスト
func TestParseNamespace_Valid(t *testing.T) {
	tests := []struct {
		name             string
		namespace        string
		expectedProvider string
		expectedModel    string
		expectedDim      int
	}{
		{
			name:             "with dim",
			namespace:        "openai:text-embedding-3-small:1536",
			expectedProvider: "openai",
			expectedModel:    "text-embedding-3-small",
			expectedDim:      1536,
		},
		{
			name:             "without dim",
			namespace:        "ollama:nomic-embed-text",
			expectedProvider: "ollama",
			expectedModel:    "nomic-embed-text",
			expectedDim:      0,
		},
		{
			name:             "local provider",
			namespace:        "local:custom-model:512",
			expectedProvider: "local",
			expectedModel:    "custom-model",
			expectedDim:      512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, dim, err := ParseNamespace(tt.namespace)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if provider != tt.expectedProvider {
				t.Errorf("provider: expected %q, got %q", tt.expectedProvider, provider)
			}

			if model != tt.expectedModel {
				t.Errorf("model: expected %q, got %q", tt.expectedModel, model)
			}

			if dim != tt.expectedDim {
				t.Errorf("dim: expected %d, got %d", tt.expectedDim, dim)
			}
		})
	}
}

// TestParseNamespace_InvalidFormat は不正なフォーマットでエラーになることをテスト
func TestParseNamespace_InvalidFormat(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{
			name:      "empty",
			namespace: "",
		},
		{
			name:      "only provider",
			namespace: "openai",
		},
		{
			name:      "too many parts",
			namespace: "openai:model:1536:extra",
		},
		{
			name:      "no separator",
			namespace: "openaimodel1536",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseNamespace(tt.namespace)
			if err == nil {
				t.Errorf("expected error for namespace %q, got nil", tt.namespace)
			}
		})
	}
}

// TestParseNamespace_DimZero は次元数0でもパース可能なことをテスト
func TestParseNamespace_DimZero(t *testing.T) {
	namespace := "openai:text-embedding-3-small:0"
	provider, model, dim, err := ParseNamespace(namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider != "openai" {
		t.Errorf("provider: expected %q, got %q", "openai", provider)
	}

	if model != "text-embedding-3-small" {
		t.Errorf("model: expected %q, got %q", "text-embedding-3-small", model)
	}

	if dim != 0 {
		t.Errorf("dim: expected 0, got %d", dim)
	}
}

// TestParseNamespace_NegativeDim は負の次元数でエラーになることをテスト
func TestParseNamespace_NegativeDim(t *testing.T) {
	namespace := "openai:text-embedding-3-small:-1536"
	_, _, _, err := ParseNamespace(namespace)
	if err == nil {
		t.Error("expected error for negative dim, got nil")
	}
}
