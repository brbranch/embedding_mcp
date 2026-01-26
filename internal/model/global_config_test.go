package model

import (
	"testing"
)

// TestGlobalConfig_Validate_Valid は有効なGlobalConfigがバリデーションを通過することをテスト
func TestGlobalConfig_Validate_Valid(t *testing.T) {
	updatedAt := "2024-01-15T10:30:00Z"

	tests := []struct {
		name   string
		config *GlobalConfig
	}{
		{
			name: "standard key: embedder.provider",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				ProjectID: "/path/to/project",
				Key:       GlobalKeyEmbedderProvider,
				Value:     "openai",
				UpdatedAt: &updatedAt,
			},
		},
		{
			name: "standard key: embedder.model",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440001",
				ProjectID: "/path/to/project",
				Key:       GlobalKeyEmbedderModel,
				Value:     "text-embedding-3-small",
				UpdatedAt: &updatedAt,
			},
		},
		{
			name: "standard key: groupDefaults",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440002",
				ProjectID: "/path/to/project",
				Key:       GlobalKeyGroupDefaults,
				Value:     map[string]any{"default": true},
				UpdatedAt: &updatedAt,
			},
		},
		{
			name: "standard key: project.conventions",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440003",
				ProjectID: "/path/to/project",
				Key:       GlobalKeyProjectConventions,
				Value:     map[string]any{"style": "camelCase"},
				UpdatedAt: &updatedAt,
			},
		},
		{
			name: "custom global key",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440004",
				ProjectID: "/path/to/project",
				Key:       "global.custom.key",
				Value:     "custom-value",
				UpdatedAt: &updatedAt,
			},
		},
		{
			name: "null UpdatedAt",
			config: &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440005",
				ProjectID: "/path/to/project",
				Key:       "global.test.key",
				Value:     "test-value",
				UpdatedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// TestGlobalConfig_Validate_InvalidKey は"global."プレフィックスなしでエラーになることをテスト
func TestGlobalConfig_Validate_InvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"no prefix", "memory.embedder.provider"},
		{"empty", ""},
		{"only global", "global"},
		{"wrong prefix", "Global.test.key"},
		{"no dot after global", "globaltest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GlobalConfig{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				ProjectID: "/path/to/project",
				Key:       tt.key,
				Value:     "test-value",
			}

			if err := config.Validate(); err == nil {
				t.Errorf("expected error for invalid Key %q, got nil", tt.key)
			}
		})
	}
}

// TestValidateGlobalKey_Valid は有効なキー（標準キー含む）が通過することをテスト
func TestValidateGlobalKey_Valid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"embedder.provider", GlobalKeyEmbedderProvider},
		{"embedder.model", GlobalKeyEmbedderModel},
		{"groupDefaults", GlobalKeyGroupDefaults},
		{"project.conventions", GlobalKeyProjectConventions},
		{"custom key", "global.custom.key"},
		{"deep nested", "global.a.b.c.d"},
		{"minimal", "global.x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGlobalKey(tt.key); err != nil {
				t.Errorf("expected no error for Key %q, got %v", tt.key, err)
			}
		})
	}
}

// TestValidateGlobalKey_Invalid はプレフィックスなし/不正プレフィックスでエラーになることをテスト
func TestValidateGlobalKey_Invalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"no prefix", "memory.embedder.provider"},
		{"only global", "global"},
		{"only global with dot", "global."},
		{"wrong case", "Global.test.key"},
		{"no dot after global", "globaltest"},
		{"space", "global. test"},
		{"special char", "global.test@key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGlobalKey(tt.key); err == nil {
				t.Errorf("expected error for invalid Key %q, got nil", tt.key)
			}
		})
	}
}
