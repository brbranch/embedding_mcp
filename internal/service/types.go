package service

import "github.com/brbranch/embedding_mcp/internal/model"

// AddNoteRequest はノート追加リクエスト
type AddNoteRequest struct {
	ProjectID string
	GroupID   string
	Title     *string
	Text      string
	Tags      []string
	Source    *string
	CreatedAt *string // nullならサーバー側で設定
	Metadata  map[string]any
}

// AddNoteResponse はノート追加レスポンス
type AddNoteResponse struct {
	ID                 string
	Namespace          string
	CanonicalProjectID string
}

// SearchRequest は検索リクエスト
type SearchRequest struct {
	ProjectID string
	GroupID   *string // nilなら全group
	Query     string
	TopK      *int     // default 5
	Tags      []string // AND検索
	Since     *string  // UTC ISO8601
	Until     *string  // UTC ISO8601
}

// SearchResponse は検索レスポンス
type SearchResponse struct {
	Namespace string
	Results   []SearchResult
}

// SearchResult は検索結果の1件
type SearchResult struct {
	ID        string
	ProjectID string
	GroupID   string
	Title     *string
	Text      string
	Tags      []string
	Source    *string
	CreatedAt string
	Score     float64 // 0-1正規化
	Metadata  map[string]any
}

// GetResponse はノート取得レスポンス
type GetResponse struct {
	ID        string
	ProjectID string
	GroupID   string
	Title     *string
	Text      string
	Tags      []string
	Source    *string
	CreatedAt string
	Namespace string
	Metadata  map[string]any
}

// UpdateRequest はノート更新リクエスト
type UpdateRequest struct {
	ID    string
	Patch NotePatch
}

// NotePatch はノート更新パッチ
type NotePatch struct {
	Title    *string         // nilは変更なし
	Text     *string         // 変更時のみ再埋め込み
	Tags     *[]string
	Source   *string
	GroupID  *string // 再埋め込み不要
	Metadata *map[string]any
}

// ListRecentRequest は最近のノート取得リクエスト
type ListRecentRequest struct {
	ProjectID string
	GroupID   *string
	Limit     *int // default 10
	Tags      []string
}

// ListRecentResponse は最近のノート取得レスポンス
type ListRecentResponse struct {
	Namespace string
	Items     []ListRecentItem
}

// ListRecentItem は最近のノートの1件
type ListRecentItem struct {
	ID        string
	ProjectID string
	GroupID   string
	Title     *string
	Text      string
	Tags      []string
	Source    *string
	CreatedAt string
	Namespace string
	Metadata  map[string]any
}

// GetConfigResponse は設定取得レスポンス
type GetConfigResponse struct {
	TransportDefaults model.TransportDefaults
	Embedder          model.EmbedderConfig
	Store             model.StoreConfig
	Paths             model.PathsConfig
}

// SetConfigRequest は設定変更リクエスト
type SetConfigRequest struct {
	Embedder *EmbedderPatch // embedderのみ変更可能
}

// EmbedderPatch はEmbedder設定パッチ
type EmbedderPatch struct {
	Provider *string
	Model    *string
	BaseURL  *string
	APIKey   *string
}

// SetConfigResponse は設定変更レスポンス
type SetConfigResponse struct {
	OK                 bool
	EffectiveNamespace string
}

// UpsertGlobalRequest はグローバル設定upsertリクエスト
type UpsertGlobalRequest struct {
	ProjectID string
	Key       string // "global." プレフィックス必須
	Value     any
	UpdatedAt *string
}

// UpsertGlobalResponse はグローバル設定upsertレスポンス
type UpsertGlobalResponse struct {
	OK        bool
	ID        string
	Namespace string
}

// GetGlobalResponse はグローバル設定取得レスポンス
type GetGlobalResponse struct {
	Namespace string
	Found     bool
	ID        *string
	Value     any
	UpdatedAt *string
}

// CreateGroupRequest はグループ作成リクエスト
type CreateGroupRequest struct {
	ProjectID   string
	GroupKey    string
	Title       string
	Description string
}

// CreateGroupResponse はグループ作成レスポンス
type CreateGroupResponse struct {
	ID        string
	Namespace string
}

// GetGroupResponse はグループ取得レスポンス
type GetGroupResponse struct {
	ID          string
	ProjectID   string
	GroupKey    string
	Title       string
	Description string
	CreatedAt   string
	UpdatedAt   string
	Namespace   string
}

// UpdateGroupRequest はグループ更新リクエスト
type UpdateGroupRequest struct {
	ID    string
	Patch GroupPatch
}

// GroupPatch はグループ更新パッチ
type GroupPatch struct {
	Title       *string // nilは変更なし
	Description *string // nilは変更なし
}

// ListGroupsResponse はグループ一覧レスポンス
type ListGroupsResponse struct {
	Namespace string
	Groups    []ListGroupItem
}

// ListGroupItem はグループ一覧の1件
type ListGroupItem struct {
	ID          string
	ProjectID   string
	GroupKey    string
	Title       string
	Description string
	CreatedAt   string
	UpdatedAt   string
}
