package embedder

import "context"

// LocalEmbedder はローカルモデルを使用するEmbedder実装（スタブ）
type LocalEmbedder struct{}

// NewLocalEmbedder は新しいLocalEmbedderを作成
func NewLocalEmbedder() *LocalEmbedder {
	return &LocalEmbedder{}
}

// Embed はテキストを埋め込みベクトルに変換（未実装）
func (e *LocalEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, ErrNotImplemented
}

// GetDimension は次元を返す
func (e *LocalEmbedder) GetDimension() int {
	return 0
}
