package config

import (
	"os"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// 環境変数名の定数
const (
	EnvOpenAIAPIKey = "OPENAI_API_KEY"
)

// ApplyEnvOverrides は環境変数による設定上書きを適用する
// config を直接変更する
func ApplyEnvOverrides(config *model.Config) {
	// OpenAI APIキーの環境変数上書き
	if apiKey := os.Getenv(EnvOpenAIAPIKey); apiKey != "" {
		config.Embedder.APIKey = &apiKey
	}
}

// GetOpenAIAPIKey は環境変数からOpenAI APIキーを取得する
// 設定ファイルの値より環境変数を優先
func GetOpenAIAPIKey(config *model.Config) string {
	// 環境変数を優先
	if apiKey := os.Getenv(EnvOpenAIAPIKey); apiKey != "" {
		return apiKey
	}
	// 設定ファイルの値
	if config.Embedder.APIKey != nil {
		return *config.Embedder.APIKey
	}
	return ""
}
