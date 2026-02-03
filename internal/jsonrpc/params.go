package jsonrpc

import (
	"encoding/json"

	"github.com/brbranch/embedding_mcp/internal/service"
)

// AddNoteParams は memory.add_note のパラメータ
type AddNoteParams struct {
	ProjectID string         `json:"projectId"`
	GroupID   string         `json:"groupId"`
	Title     *string        `json:"title"`
	Text      string         `json:"text"`
	Tags      []string       `json:"tags"`
	Source    *string        `json:"source"`
	CreatedAt *string        `json:"createdAt"`
	Metadata  map[string]any `json:"metadata"`
}

// ToRequest はサービスリクエストに変換
func (p *AddNoteParams) ToRequest() *service.AddNoteRequest {
	return &service.AddNoteRequest{
		ProjectID: p.ProjectID,
		GroupID:   p.GroupID,
		Title:     p.Title,
		Text:      p.Text,
		Tags:      p.Tags,
		Source:    p.Source,
		CreatedAt: p.CreatedAt,
		Metadata:  p.Metadata,
	}
}

// SearchParams は memory.search のパラメータ
type SearchParams struct {
	ProjectID string   `json:"projectId"`
	GroupID   *string  `json:"groupId"`
	Query     string   `json:"query"`
	TopK      *int     `json:"topK"`
	Tags      []string `json:"tags"`
	Since     *string  `json:"since"`
	Until     *string  `json:"until"`
}

// ToRequest はサービスリクエストに変換
func (p *SearchParams) ToRequest() *service.SearchRequest {
	topK := p.TopK
	if topK == nil {
		defaultTopK := 5
		topK = &defaultTopK
	}
	return &service.SearchRequest{
		ProjectID: p.ProjectID,
		GroupID:   p.GroupID,
		Query:     p.Query,
		TopK:      topK,
		Tags:      p.Tags,
		Since:     p.Since,
		Until:     p.Until,
	}
}

// GetParams は memory.get のパラメータ
type GetParams struct {
	ID string `json:"id"`
}

// UpdateParams は memory.update のパラメータ
type UpdateParams struct {
	ID    string      `json:"id"`
	Patch PatchParams `json:"patch"`
}

// PatchParams は memory.update のパッチパラメータ
// json.RawMessageを使って「未指定」「null」「値あり」を区別
type PatchParams struct {
	Title    json.RawMessage `json:"title,omitempty"`
	Text     *string         `json:"text,omitempty"`
	Tags     *[]string       `json:"tags,omitempty"`
	Source   json.RawMessage `json:"source,omitempty"`
	GroupID  *string         `json:"groupId,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// ToRequest はサービスリクエストに変換
// 注意: JSON-RPCの「null」と「キー未指定」の区別について
// - キー未指定 → nilを渡す（変更なし）
// - null → 空文字列/空mapを渡す（クリア扱い）
// 将来的にはservice層でnullクリアを明示的にサポートする設計変更が望ましい
func (p *UpdateParams) ToRequest() (*service.UpdateRequest, error) {
	patch := service.NotePatch{
		Text:    p.Patch.Text,
		Tags:    p.Patch.Tags,
		GroupID: p.Patch.GroupID,
	}

	// Title: null か 値 か 未指定 かを判定
	if len(p.Patch.Title) > 0 {
		if string(p.Patch.Title) == "null" {
			// null でクリア（空文字列で表現）
			empty := ""
			patch.Title = &empty
		} else {
			var title string
			if err := json.Unmarshal(p.Patch.Title, &title); err == nil {
				patch.Title = &title
			}
			// 型不正の場合は無視（未指定扱い）
		}
	}

	// Source: null か 値 か 未指定 かを判定
	if len(p.Patch.Source) > 0 {
		if string(p.Patch.Source) == "null" {
			// null でクリア（空文字列で表現）
			empty := ""
			patch.Source = &empty
		} else {
			var source string
			if err := json.Unmarshal(p.Patch.Source, &source); err == nil {
				patch.Source = &source
			}
			// 型不正の場合は無視（未指定扱い）
		}
	}

	// Metadata: null か 値 か 未指定 かを判定
	if len(p.Patch.Metadata) > 0 {
		if string(p.Patch.Metadata) == "null" {
			// null でクリア（空mapで表現）
			emptyMap := map[string]any{}
			patch.Metadata = &emptyMap
		} else {
			var metadata map[string]any
			if err := json.Unmarshal(p.Patch.Metadata, &metadata); err == nil {
				patch.Metadata = &metadata
			}
			// 型不正の場合は無視（未指定扱い）
		}
	}

	return &service.UpdateRequest{
		ID:    p.ID,
		Patch: patch,
	}, nil
}

// ListRecentParams は memory.list_recent のパラメータ
type ListRecentParams struct {
	ProjectID string   `json:"projectId"`
	GroupID   *string  `json:"groupId"`
	Limit     *int     `json:"limit"`
	Tags      []string `json:"tags"`
}

// ToRequest はサービスリクエストに変換
func (p *ListRecentParams) ToRequest() *service.ListRecentRequest {
	return &service.ListRecentRequest{
		ProjectID: p.ProjectID,
		GroupID:   p.GroupID,
		Limit:     p.Limit,
		Tags:      p.Tags,
	}
}

// SetConfigParams は memory.set_config のパラメータ
type SetConfigParams struct {
	Embedder *EmbedderParams `json:"embedder"`
}

// EmbedderParams はembedder設定のパラメータ
type EmbedderParams struct {
	Provider *string `json:"provider"`
	Model    *string `json:"model"`
	BaseURL  *string `json:"baseUrl"`
	APIKey   *string `json:"apiKey"`
}

// ToRequest はサービスリクエストに変換
func (p *SetConfigParams) ToRequest() *service.SetConfigRequest {
	if p.Embedder == nil {
		return &service.SetConfigRequest{}
	}
	return &service.SetConfigRequest{
		Embedder: &service.EmbedderPatch{
			Provider: p.Embedder.Provider,
			Model:    p.Embedder.Model,
			BaseURL:  p.Embedder.BaseURL,
			APIKey:   p.Embedder.APIKey,
		},
	}
}

// UpsertGlobalParams は memory.upsert_global のパラメータ
type UpsertGlobalParams struct {
	ProjectID string  `json:"projectId"`
	Key       string  `json:"key"`
	Value     any     `json:"value"`
	UpdatedAt *string `json:"updatedAt"`
}

// ToRequest はサービスリクエストに変換
func (p *UpsertGlobalParams) ToRequest() *service.UpsertGlobalRequest {
	return &service.UpsertGlobalRequest{
		ProjectID: p.ProjectID,
		Key:       p.Key,
		Value:     p.Value,
		UpdatedAt: p.UpdatedAt,
	}
}

// GetGlobalParams は memory.get_global のパラメータ
type GetGlobalParams struct {
	ProjectID string `json:"projectId"`
	Key       string `json:"key"`
}

// DeleteParams は memory.delete のパラメータ
type DeleteParams struct {
	ID string `json:"id"`
}

// GroupCreateParams は memory.group_create のパラメータ
type GroupCreateParams struct {
	ProjectID   string `json:"projectId"`
	GroupKey    string `json:"groupKey"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ToRequest はサービスリクエストに変換
func (p *GroupCreateParams) ToRequest() *service.CreateGroupRequest {
	return &service.CreateGroupRequest{
		ProjectID:   p.ProjectID,
		GroupKey:    p.GroupKey,
		Title:       p.Title,
		Description: p.Description,
	}
}

// GroupGetParams は memory.group_get のパラメータ
type GroupGetParams struct {
	ID string `json:"id"`
}

// GroupUpdateParams は memory.group_update のパラメータ
type GroupUpdateParams struct {
	ID    string          `json:"id"`
	Patch GroupPatchParam `json:"patch"`
}

// GroupPatchParam はグループ更新のパッチパラメータ
type GroupPatchParam struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ToRequest はサービスリクエストに変換
func (p *GroupUpdateParams) ToRequest() *service.UpdateGroupRequest {
	return &service.UpdateGroupRequest{
		ID: p.ID,
		Patch: service.GroupPatch{
			Title:       p.Patch.Title,
			Description: p.Patch.Description,
		},
	}
}

// GroupDeleteParams は memory.group_delete のパラメータ
type GroupDeleteParams struct {
	ID string `json:"id"`
}

// GroupListParams は memory.group_list のパラメータ
type GroupListParams struct {
	ProjectID string `json:"projectId"`
}
