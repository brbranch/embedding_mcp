package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// embeddingRequest はOpenAI APIリクエストの構造
type embeddingRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	EncodingFormat string `json:"encoding_format"`
}

// embeddingResponse はOpenAI APIレスポンスの構造
type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Model string          `json:"model"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// Embed はテキストを埋め込みベクトルに変換
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// リクエストボディ作成
	reqBody := embeddingRequest{
		Model:          e.model,
		Input:          text,
		EncodingFormat: "float",
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	// HTTPリクエスト作成
	url := e.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	// リクエスト実行
	resp, err := e.httpClient.Do(req)
	if err != nil {
		// context.Canceledやcontext.DeadlineExceededはそのまま返す
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}
	defer resp.Body.Close()

	// レスポンスボディ読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrAPIRequestFailed, err)
	}

	// HTTPステータスチェック
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	// レスポンスパース
	var embResp embeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	// dataが空でないかチェック
	if len(embResp.Data) == 0 {
		return nil, ErrEmptyEmbedding
	}

	// embeddingが空でないかチェック
	embedding := embResp.Data[0].Embedding
	if len(embedding) == 0 {
		return nil, ErrEmptyEmbedding
	}

	// 次元を更新（初回のみ、かつdimが未設定の場合）
	if e.dim == 0 {
		e.dimOnce.Do(func() {
			e.dim = len(embedding)
			if e.dimUpdater != nil {
				if err := e.dimUpdater.UpdateDim(e.dim); err != nil {
					log.Printf("[WARN] failed to update dim: %v", err)
				}
			}
		})
	}

	return embedding, nil
}

// GetDimension は次元を返す
func (e *OpenAIEmbedder) GetDimension() int {
	return e.dim
}
