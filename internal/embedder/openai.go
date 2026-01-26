package embedder

import (
	"context"
	"net/http"
	"sync"
)

const (
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "text-embedding-3-small"
)

// OpenAIEmbedder はOpenAI APIを使用するEmbedder実装
type OpenAIEmbedder struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	dim        int
	dimOnce    sync.Once
	dimUpdater DimUpdater
}

// OpenAIOption はOpenAIEmbedderのオプション
type OpenAIOption func(*OpenAIEmbedder)

// WithBaseURL はベースURLを設定
func WithBaseURL(url string) OpenAIOption {
	return func(e *OpenAIEmbedder) {
		e.baseURL = url
	}
}

// WithModel はモデルを設定
func WithModel(model string) OpenAIOption {
	return func(e *OpenAIEmbedder) {
		e.model = model
	}
}

// WithDim は既知の次元を設定
func WithDim(dim int) OpenAIOption {
	return func(e *OpenAIEmbedder) {
		e.dim = dim
	}
}

// WithDimUpdater は次元更新コールバックを設定
func WithDimUpdater(updater DimUpdater) OpenAIOption {
	return func(e *OpenAIEmbedder) {
		e.dimUpdater = updater
	}
}

// WithHTTPClient はHTTPクライアントを設定
func WithHTTPClient(client *http.Client) OpenAIOption {
	return func(e *OpenAIEmbedder) {
		e.httpClient = client
	}
}

// NewOpenAIEmbedder は新しいOpenAIEmbedderを作成
func NewOpenAIEmbedder(apiKey string, opts ...OpenAIOption) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	e := &OpenAIEmbedder{
		httpClient: http.DefaultClient,
		baseURL:    DefaultOpenAIBaseURL,
		apiKey:     apiKey,
		model:      DefaultOpenAIModel,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Embed はテキストを埋め込みベクトルに変換（TODO: 実装）
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, ErrNotImplemented
}

// GetDimension は次元を返す
func (e *OpenAIEmbedder) GetDimension() int {
	return e.dim
}
