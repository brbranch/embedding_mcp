package embedder

import (
	"context"
	"errors"
	"fmt"
)

// Embedder はテキストから埋め込みベクトルを生成するインターフェース
type Embedder interface {
	// Embed はテキストを埋め込みベクトルに変換する
	Embed(ctx context.Context, text string) ([]float32, error)

	// GetDimension はこのEmbedderが生成するベクトルの次元数を返す
	// 初回埋め込み前（dim未確定時）は 0 を返す
	GetDimension() int
}

// DimUpdater は次元数が確定した際に呼び出されるコールバック
type DimUpdater interface {
	UpdateDim(dim int) error
}

// エラー定義
var (
	ErrAPIKeyRequired   = errors.New("api key is required")
	ErrNotImplemented   = errors.New("embedder not implemented")
	ErrAPIRequestFailed = errors.New("API request failed")
	ErrInvalidResponse  = errors.New("invalid API response")
	ErrEmptyEmbedding   = errors.New("empty embedding returned")
	ErrUnknownProvider  = errors.New("unknown embedder provider")
)

// APIError は詳細なAPIエラー情報を保持
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

func (e *APIError) Is(target error) bool {
	return target == ErrAPIRequestFailed
}
