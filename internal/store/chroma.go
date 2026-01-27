package store

import (
	"context"
	"fmt"

	"github.com/brbranch/embedding_mcp/internal/model"
)

const (
	// DefaultChromaURL はデフォルトのChromaサーバーURL
	DefaultChromaURL = "http://localhost:8000"
)

// ChromaStore はChromaを使用したStore実装（スタブ）
// TODO: chroma-go v2 APIを使用した完全実装
type ChromaStore struct {
	baseURL   string
	namespace string
	// 実際の実装では chroma.Client などを保持
}

// NewChromaStore はChromaStoreを作成する
func NewChromaStore(url string) (*ChromaStore, error) {
	if url == "" {
		url = DefaultChromaURL
	}

	return &ChromaStore{
		baseURL: url,
	}, nil
}

// Initialize はストアを初期化する
func (s *ChromaStore) Initialize(ctx context.Context, namespace string) error {
	s.namespace = namespace
	// TODO: chroma-go v2 APIでコレクション作成
	return fmt.Errorf("ChromaStore is not yet implemented")
}

// Close はストアをクローズする
func (s *ChromaStore) Close() error {
	return nil
}

// AddNote はノートを追加する
func (s *ChromaStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
	return fmt.Errorf("ChromaStore is not yet implemented")
}

// Get はIDでノートを取得する
func (s *ChromaStore) Get(ctx context.Context, id string) (*model.Note, error) {
	return nil, fmt.Errorf("ChromaStore is not yet implemented")
}

// Update はノートを更新する
func (s *ChromaStore) Update(ctx context.Context, note *model.Note, embedding []float32) error {
	return fmt.Errorf("ChromaStore is not yet implemented")
}

// Delete はノートを削除する
func (s *ChromaStore) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("ChromaStore is not yet implemented")
}

// Search はベクトル検索を実行する
func (s *ChromaStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error) {
	return nil, fmt.Errorf("ChromaStore is not yet implemented")
}

// ListRecent は最新ノート一覧を取得する
func (s *ChromaStore) ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error) {
	return nil, fmt.Errorf("ChromaStore is not yet implemented")
}

// UpsertGlobal はグローバル設定を追加/更新する
func (s *ChromaStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error {
	return fmt.Errorf("ChromaStore is not yet implemented")
}

// GetGlobal はグローバル設定を取得する
func (s *ChromaStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error) {
	return nil, false, fmt.Errorf("ChromaStore is not yet implemented")
}

// GetGlobalByID はIDでグローバル設定を取得する
func (s *ChromaStore) GetGlobalByID(ctx context.Context, id string) (*model.GlobalConfig, error) {
	return nil, fmt.Errorf("ChromaStore is not yet implemented")
}

// DeleteGlobalByID はIDでグローバル設定を削除する
func (s *ChromaStore) DeleteGlobalByID(ctx context.Context, id string) error {
	return fmt.Errorf("ChromaStore is not yet implemented")
}
