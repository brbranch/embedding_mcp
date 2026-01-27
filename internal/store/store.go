// Package store provides vector storage interfaces and implementations.
package store

import (
	"context"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// Store はベクトルストアの抽象インターフェース
type Store interface {
	// Note操作
	AddNote(ctx context.Context, note *model.Note, embedding []float32) error
	Get(ctx context.Context, id string) (*model.Note, error)
	Update(ctx context.Context, note *model.Note, embedding []float32) error
	Delete(ctx context.Context, id string) error

	// ベクトル検索
	Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error)

	// 最新一覧取得（createdAt降順）
	ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error)

	// GlobalConfig操作
	UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error
	GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error)
	GetGlobalByID(ctx context.Context, id string) (*model.GlobalConfig, error)
	DeleteGlobalByID(ctx context.Context, id string) error

	// 初期化・終了
	Initialize(ctx context.Context, namespace string) error
	Close() error
}
