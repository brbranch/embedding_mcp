package store

import (
	"errors"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// SearchOptions はSearch操作のオプション
type SearchOptions struct {
	ProjectID string     // 必須
	GroupID   *string    // nullable（nilの場合はフィルタなし）
	TopK      int        // default: 5
	Tags      []string   // AND検索、空/nilはフィルタなし、大小文字区別
	Since     *time.Time // UTC、境界条件: since <= createdAt
	Until     *time.Time // UTC、境界条件: createdAt < until
}

// ListOptions はListRecent操作のオプション
type ListOptions struct {
	ProjectID string   // 必須
	GroupID   *string  // nullable（nilの場合は全group）
	Limit     int      // default: 10
	Tags      []string // AND検索、空/nilはフィルタなし
}

// SearchResult はベクトル検索結果の1件を表す
type SearchResult struct {
	Note  *model.Note
	Score float64 // 0-1に正規化（1が最も類似）
}

// エラー定義
var (
	ErrNotFound         = errors.New("resource not found")
	ErrNotInitialized   = errors.New("store not initialized")
	ErrConnectionFailed = errors.New("failed to connect to store")
)

// DefaultSearchOptions はSearchOptionsのデフォルト値を返す
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{TopK: 5}
}

// DefaultListOptions はListOptionsのデフォルト値を返す
func DefaultListOptions() ListOptions {
	return ListOptions{Limit: 10}
}
