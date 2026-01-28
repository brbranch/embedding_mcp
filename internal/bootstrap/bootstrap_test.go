package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInitialize_WithValidConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// テスト用の設定ファイルを作成（memory storeを使用）
	configContent := `{
		"embedder": {
			"provider": "local",
			"model": "mock"
		},
		"store": {
			"type": "memory"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	ctx := context.Background()
	services, cleanup, err := Initialize(ctx, configPath)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	defer cleanup()

	// サービスが正しく初期化されていることを確認
	if services.NoteService == nil {
		t.Error("expected NoteService to be non-nil")
	}
	if services.Config == nil {
		t.Error("expected Config to be non-nil")
	}
}

func TestInitialize_WithDefaultConfig(t *testing.T) {
	// 設定ファイルパスを空にした場合はデフォルト設定を使用
	// デフォルト設定ファイルが存在しない環境でも動作することを確認
	// ただし、この場合は環境に依存するためスキップ
	t.Skip("Skipping default config test - environment dependent")
}

func TestInitialize_WithNonExistentConfigPath(t *testing.T) {
	// 設定ファイルが存在しなくてもデフォルト設定で動作することを確認
	// ただし、OpenAI API keyが必要なためスキップ
	t.Skip("Skipping - requires OPENAI_API_KEY environment variable")
}
