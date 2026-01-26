package embedder

import "context"

const (
	DefaultOllamaBaseURL = "http://localhost:11434"
)

// OllamaEmbedder はOllama APIを使用するEmbedder実装（スタブ）
type OllamaEmbedder struct {
	baseURL string
	model   string
}

// NewOllamaEmbedder は新しいOllamaEmbedderを作成
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}
	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
	}
}

// Embed はテキストを埋め込みベクトルに変換（未実装）
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, ErrNotImplemented
}

// GetDimension は次元を返す
func (e *OllamaEmbedder) GetDimension() int {
	return 0
}
