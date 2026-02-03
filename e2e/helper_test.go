//go:build e2e || qdrant_e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/jsonrpc"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/service"
	"github.com/brbranch/embedding_mcp/internal/store"
)

// mockEmbedder はテスト用のモックEmbedder
// 決定論的な埋め込みベクトルを生成（テキストのハッシュから）
type mockEmbedder struct {
	dim int
}

// Embed はテキストから決定論的なベクトルを生成
func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// テキストのSHA256ハッシュを計算
	hash := sha256.Sum256([]byte(text))

	// ハッシュから決定論的にベクトルを生成
	vec := make([]float32, m.dim)
	for i := 0; i < m.dim; i++ {
		// 4バイトずつ読み込んでfloat32に変換
		offset := (i * 4) % len(hash)
		bytes := hash[offset : offset+4]
		val := binary.BigEndian.Uint32(bytes)
		// 0-1の範囲に正規化
		vec[i] = float32(val) / float32(0xFFFFFFFF)
	}

	return vec, nil
}

// GetDimension はベクトルの次元数を返す
func (m *mockEmbedder) GetDimension() int {
	return m.dim
}

// mockConfigService はテスト用のモックConfigService
type mockConfigService struct{}

func (m *mockConfigService) GetConfig(ctx context.Context) (*service.GetConfigResponse, error) {
	return &service.GetConfigResponse{
		TransportDefaults: model.TransportDefaults{
			DefaultTransport: "stdio",
		},
		Embedder: model.EmbedderConfig{
			Provider: "mock",
			Model:    "mock-128",
			Dim:      128,
			BaseURL:  (*string)(nil),
			APIKey:   (*string)(nil),
		},
		Store: model.StoreConfig{
			Type: "memory",
			Path: (*string)(nil),
			URL:  (*string)(nil),
		},
		Paths: model.PathsConfig{
			ConfigPath: "",
			DataDir:    "",
		},
	}, nil
}

func (m *mockConfigService) SetConfig(ctx context.Context, req *service.SetConfigRequest) (*service.SetConfigResponse, error) {
	return &service.SetConfigResponse{
		OK:                 true,
		EffectiveNamespace: "mock:mock-128:128",
	}, nil
}

// setupTestHandler はテスト用のHandlerを構築
func setupTestHandler(t *testing.T) *jsonrpc.Handler {
	t.Helper()

	// 1. MockEmbedder作成
	emb := &mockEmbedder{dim: 128}

	// 2. MemoryStore作成・初期化
	st := store.NewMemoryStore()
	namespace := "test:mock:128"
	if err := st.Initialize(context.Background(), namespace); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// 3. Services作成
	noteService := service.NewNoteService(emb, st, namespace)
	globalService := service.NewGlobalService(st, namespace)
	groupService := service.NewGroupService(st, namespace)
	configService := &mockConfigService{}

	// 4. Handler作成
	return jsonrpc.New(noteService, configService, globalService, groupService)
}

// callAddNote はmemory.add_noteを呼び出す
func callAddNote(t *testing.T, h *jsonrpc.Handler, projectID, groupID, text string) *AddNoteResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.add_note",
		Params: map[string]any{
			"projectId": projectID,
			"groupId":   groupID,
			"text":      text,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("add_note failed: %v", rawResp.Error)
	}

	result := &AddNoteResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callSearch はmemory.searchを呼び出す
func callSearch(t *testing.T, h *jsonrpc.Handler, projectID string, groupID *string, query string) *SearchResult {
	t.Helper()

	params := map[string]any{
		"projectId": projectID,
		"query":     query,
	}
	if groupID != nil {
		params["groupId"] = *groupID
	}

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.search",
		Params:  params,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("search failed: %v", rawResp.Error)
	}

	result := &SearchResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callSearchRaw はmemory.searchを呼び出して生のレスポンスを返す
func callSearchRaw(t *testing.T, h *jsonrpc.Handler, projectID string, groupID *string, query string) *RawResponse {
	t.Helper()

	params := map[string]any{
		"query": query,
	}
	if projectID != "" {
		params["projectId"] = projectID
	}
	if groupID != nil {
		params["groupId"] = *groupID
	}

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.search",
		Params:  params,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	return &resp
}

// callUpsertGlobal はmemory.upsert_globalを呼び出す
func callUpsertGlobal(t *testing.T, h *jsonrpc.Handler, projectID, key string, value any) *UpsertGlobalResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.upsert_global",
		Params: map[string]any{
			"projectId": projectID,
			"key":       key,
			"value":     value,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("upsert_global failed: %v", rawResp.Error)
	}

	result := &UpsertGlobalResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callUpsertGlobalRaw はmemory.upsert_globalを呼び出して生のレスポンスを返す
func callUpsertGlobalRaw(t *testing.T, h *jsonrpc.Handler, projectID, key string, value any) *RawResponse {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.upsert_global",
		Params: map[string]any{
			"projectId": projectID,
			"key":       key,
			"value":     value,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	return &resp
}

// callGetGlobal はmemory.get_globalを呼び出す
func callGetGlobal(t *testing.T, h *jsonrpc.Handler, projectID, key string) *GetGlobalResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.get_global",
		Params: map[string]any{
			"projectId": projectID,
			"key":       key,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("get_global failed: %v", rawResp.Error)
	}

	result := &GetGlobalResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// ptr は文字列のポインタを返すヘルパー
func ptr(s string) *string {
	return &s
}

// レスポンス型定義

type AddNoteResult struct {
	ID                  string `json:"id"`
	Namespace           string `json:"namespace"`
	CanonicalProjectID  string `json:"canonicalProjectId,omitempty"`
}

type SearchResult struct {
	Namespace string              `json:"namespace"`
	Results   []SearchResultItem  `json:"results"`
}

type SearchResultItem struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	GroupID   string         `json:"groupId"`
	Title     *string        `json:"title"`
	Text      string         `json:"text"`
	Tags      []string       `json:"tags"`
	Source    *string        `json:"source"`
	CreatedAt string         `json:"createdAt"`
	Score     float64        `json:"score"`
	Metadata  map[string]any `json:"metadata"`
}

type UpsertGlobalResult struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
}

type GetGlobalResult struct {
	Namespace string  `json:"namespace"`
	Found     bool    `json:"found"`
	ID        *string `json:"id"`
	Value     any     `json:"value"`
	UpdatedAt *string `json:"updatedAt"`
}

// callGroupCreate はmemory.group_createを呼び出す
func callGroupCreate(t *testing.T, h *jsonrpc.Handler, projectID, groupKey, title, description string) *GroupCreateResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_create",
		Params: map[string]any{
			"projectId":   projectID,
			"groupKey":    groupKey,
			"title":       title,
			"description": description,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("group_create failed: %v", rawResp.Error)
	}

	result := &GroupCreateResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callGroupCreateRaw はmemory.group_createを呼び出して生のレスポンスを返す
func callGroupCreateRaw(t *testing.T, h *jsonrpc.Handler, projectID, groupKey, title, description string) *RawResponse {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_create",
		Params: map[string]any{
			"projectId":   projectID,
			"groupKey":    groupKey,
			"title":       title,
			"description": description,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	return &resp
}

// callGroupGet はmemory.group_getを呼び出す
func callGroupGet(t *testing.T, h *jsonrpc.Handler, id string) *GroupGetResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_get",
		Params: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("group_get failed: %v", rawResp.Error)
	}

	result := &GroupGetResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callGroupGetRaw はmemory.group_getを呼び出して生のレスポンスを返す
func callGroupGetRaw(t *testing.T, h *jsonrpc.Handler, id string) *RawResponse {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_get",
		Params: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	return &resp
}

// callGroupUpdate はmemory.group_updateを呼び出す
func callGroupUpdate(t *testing.T, h *jsonrpc.Handler, id string, title, description *string) *GroupUpdateResult {
	t.Helper()

	params := map[string]any{
		"id": id,
	}
	patchFields := map[string]any{}
	if title != nil {
		patchFields["title"] = *title
	}
	if description != nil {
		patchFields["description"] = *description
	}
	if len(patchFields) > 0 {
		params["patch"] = patchFields
	}

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_update",
		Params:  params,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("group_update failed: %v", rawResp.Error)
	}

	result := &GroupUpdateResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callGroupDelete はmemory.group_deleteを呼び出す
func callGroupDelete(t *testing.T, h *jsonrpc.Handler, id string) *GroupDeleteResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_delete",
		Params: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("group_delete failed: %v", rawResp.Error)
	}

	result := &GroupDeleteResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// callGroupList はmemory.group_listを呼び出す
func callGroupList(t *testing.T, h *jsonrpc.Handler, projectID string) *GroupListResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.group_list",
		Params: map[string]any{
			"projectId": projectID,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(context.Background(), reqBytes)

	// エラーチェック用に一旦RawResponseにUnmarshal
	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("group_list failed: %v", rawResp.Error)
	}

	result := &GroupListResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

type GroupCreateResult struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
}

type GroupGetResult struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	GroupKey    string `json:"groupKey"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Namespace   string `json:"namespace"`
}

type GroupUpdateResult struct {
	OK bool `json:"ok"`
}

type GroupDeleteResult struct {
	OK bool `json:"ok"`
}

type GroupListResult struct {
	Namespace string          `json:"namespace"`
	Groups    []GroupListItem `json:"groups"`
}

type GroupListItem struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	GroupKey    string `json:"groupKey"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type RawResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  any            `json:"result,omitempty"`
	Error   *model.RPCError `json:"error,omitempty"`
}
