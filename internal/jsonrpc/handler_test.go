package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/service"
)

// === モックサービス ===

type mockNoteService struct {
	addNoteFunc    func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error)
	searchFunc     func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error)
	getFunc        func(ctx context.Context, id string) (*service.GetResponse, error)
	updateFunc     func(ctx context.Context, req *service.UpdateRequest) error
	listRecentFunc func(ctx context.Context, req *service.ListRecentRequest) (*service.ListRecentResponse, error)
}

func (m *mockNoteService) AddNote(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
	if m.addNoteFunc != nil {
		return m.addNoteFunc(ctx, req)
	}
	return &service.AddNoteResponse{ID: "test-id", Namespace: "test-ns"}, nil
}

func (m *mockNoteService) Search(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	return &service.SearchResponse{Namespace: "test-ns", Results: []service.SearchResult{}}, nil
}

func (m *mockNoteService) Get(ctx context.Context, id string) (*service.GetResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return &service.GetResponse{ID: id, ProjectID: "/test", GroupID: "global"}, nil
}

func (m *mockNoteService) Update(ctx context.Context, req *service.UpdateRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, req)
	}
	return nil
}

func (m *mockNoteService) ListRecent(ctx context.Context, req *service.ListRecentRequest) (*service.ListRecentResponse, error) {
	if m.listRecentFunc != nil {
		return m.listRecentFunc(ctx, req)
	}
	return &service.ListRecentResponse{Namespace: "test-ns", Items: []service.ListRecentItem{}}, nil
}

type mockConfigService struct {
	getConfigFunc func(ctx context.Context) (*service.GetConfigResponse, error)
	setConfigFunc func(ctx context.Context, req *service.SetConfigRequest) (*service.SetConfigResponse, error)
}

func (m *mockConfigService) GetConfig(ctx context.Context) (*service.GetConfigResponse, error) {
	if m.getConfigFunc != nil {
		return m.getConfigFunc(ctx)
	}
	return &service.GetConfigResponse{
		TransportDefaults: model.TransportDefaults{DefaultTransport: "stdio"},
		Embedder:          model.EmbedderConfig{Provider: "openai", Model: "text-embedding-3-small", Dim: 1536},
		Store:             model.StoreConfig{Type: "chroma"},
		Paths:             model.PathsConfig{ConfigPath: "/test/config.json", DataDir: "/test/data"},
	}, nil
}

func (m *mockConfigService) SetConfig(ctx context.Context, req *service.SetConfigRequest) (*service.SetConfigResponse, error) {
	if m.setConfigFunc != nil {
		return m.setConfigFunc(ctx, req)
	}
	return &service.SetConfigResponse{OK: true, EffectiveNamespace: "openai:text-embedding-3-small:1536"}, nil
}

type mockGlobalService struct {
	upsertGlobalFunc func(ctx context.Context, req *service.UpsertGlobalRequest) (*service.UpsertGlobalResponse, error)
	getGlobalFunc    func(ctx context.Context, projectID, key string) (*service.GetGlobalResponse, error)
}

func (m *mockGlobalService) UpsertGlobal(ctx context.Context, req *service.UpsertGlobalRequest) (*service.UpsertGlobalResponse, error) {
	if m.upsertGlobalFunc != nil {
		return m.upsertGlobalFunc(ctx, req)
	}
	return &service.UpsertGlobalResponse{OK: true, ID: "global-id", Namespace: "test-ns"}, nil
}

func (m *mockGlobalService) GetGlobal(ctx context.Context, projectID, key string) (*service.GetGlobalResponse, error) {
	if m.getGlobalFunc != nil {
		return m.getGlobalFunc(ctx, projectID, key)
	}
	return &service.GetGlobalResponse{Namespace: "test-ns", Found: false}, nil
}

// === ヘルパー関数 ===

func makeRequest(method string, params any) []byte {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	return b
}

func parseResponse(t *testing.T, data []byte) map[string]any {
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func parseErrorResponse(t *testing.T, data []byte) *model.ErrorResponse {
	var resp model.ErrorResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	return &resp
}

func newTestHandler() *Handler {
	return New(
		&mockNoteService{},
		&mockConfigService{},
		&mockGlobalService{},
	)
}

// === 1. パース系テスト ===

func TestHandle_ParseError_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	result := h.Handle(context.Background(), []byte("not json"))
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeParseError {
		t.Errorf("expected code %d, got %d", model.ErrCodeParseError, resp.Error.Code)
	}
}

func TestHandle_InvalidRequest_WrongVersion(t *testing.T) {
	h := newTestHandler()
	req := []byte(`{"jsonrpc":"1.0","id":1,"method":"memory.get_config"}`)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidRequest {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidRequest, resp.Error.Code)
	}
}

func TestHandle_InvalidRequest_NoMethod(t *testing.T) {
	h := newTestHandler()
	req := []byte(`{"jsonrpc":"2.0","id":1}`)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidRequest {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidRequest, resp.Error.Code)
	}
}

// === 2. ディスパッチ系テスト ===

func TestHandle_MethodNotFound(t *testing.T) {
	h := newTestHandler()
	req := makeRequest("unknown.method", nil)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeMethodNotFound {
		t.Errorf("expected code %d, got %d", model.ErrCodeMethodNotFound, resp.Error.Code)
	}
}

// === 3. memory.add_note テスト ===

func TestHandle_AddNote_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
		"groupId":   "global",
		"text":      "test note",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["id"] == nil {
		t.Error("expected id in result")
	}
	if resultMap["namespace"] == nil {
		t.Error("expected namespace in result")
	}
}

func TestHandle_AddNote_MissingProjectId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}
	params := map[string]any{
		"groupId": "global",
		"text":    "test note",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_AddNote_MissingGroupId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrGroupIDRequired
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"text":      "test note",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_AddNote_MissingText(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrTextRequired
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"groupId":   "global",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_AddNote_InvalidGroupId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrInvalidGroupID
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"groupId":   "invalid group!",
		"text":      "test note",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_AddNote_InvalidCreatedAt(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrInvalidTimeFormat
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"groupId":   "global",
		"text":      "test note",
		"createdAt": "invalid-time",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 4. memory.search テスト ===

func TestHandle_Search_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
		"query":     "test query",
	}
	req := makeRequest("memory.search", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["namespace"] == nil {
		t.Error("expected namespace in result")
	}
	if resultMap["results"] == nil {
		t.Error("expected results in result")
	}
}

func TestHandle_Search_TopKDefault(t *testing.T) {
	var capturedReq *service.SearchRequest
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			capturedReq = req
			return &service.SearchResponse{Namespace: "test-ns", Results: []service.SearchResult{}}, nil
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"query":     "test query",
	}
	req := makeRequest("memory.search", params)
	h.Handle(context.Background(), req)

	if capturedReq.TopK == nil || *capturedReq.TopK != 5 {
		t.Errorf("expected topK default 5, got %v", capturedReq.TopK)
	}
}

func TestHandle_Search_MissingProjectId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}
	params := map[string]any{
		"query": "test query",
	}
	req := makeRequest("memory.search", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_Search_MissingQuery(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			return nil, service.ErrQueryRequired
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
	}
	req := makeRequest("memory.search", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_Search_InvalidSince(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			return nil, service.ErrInvalidTimeFormat
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"query":     "test",
		"since":     "invalid-time",
	}
	req := makeRequest("memory.search", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_Search_InvalidUntil(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			return nil, service.ErrInvalidTimeFormat
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"query":     "test",
		"until":     "invalid-time",
	}
	req := makeRequest("memory.search", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 5. memory.get テスト ===

func TestHandle_Get_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"id": "test-id",
	}
	req := makeRequest("memory.get", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["id"] == nil {
		t.Error("expected id in result")
	}
}

func TestHandle_Get_MissingId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		getFunc: func(ctx context.Context, id string) (*service.GetResponse, error) {
			return nil, service.ErrIDRequired
		},
	}
	params := map[string]any{}
	req := makeRequest("memory.get", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_Get_NotFound(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		getFunc: func(ctx context.Context, id string) (*service.GetResponse, error) {
			return nil, service.ErrNoteNotFound
		},
	}
	params := map[string]any{
		"id": "nonexistent",
	}
	req := makeRequest("memory.get", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeNotFound {
		t.Errorf("expected code %d, got %d", model.ErrCodeNotFound, resp.Error.Code)
	}
}

// === 6. memory.update テスト ===

func TestHandle_Update_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"id": "test-id",
		"patch": map[string]any{
			"text": "updated text",
		},
	}
	req := makeRequest("memory.update", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["ok"] != true {
		t.Error("expected ok: true in result")
	}
}

func TestHandle_Update_MissingId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		updateFunc: func(ctx context.Context, req *service.UpdateRequest) error {
			return service.ErrIDRequired
		},
	}
	params := map[string]any{
		"patch": map[string]any{
			"text": "updated text",
		},
	}
	req := makeRequest("memory.update", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_Update_NotFound(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		updateFunc: func(ctx context.Context, req *service.UpdateRequest) error {
			return service.ErrNoteNotFound
		},
	}
	params := map[string]any{
		"id": "nonexistent",
		"patch": map[string]any{
			"text": "updated text",
		},
	}
	req := makeRequest("memory.update", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeNotFound {
		t.Errorf("expected code %d, got %d", model.ErrCodeNotFound, resp.Error.Code)
	}
}

func TestHandle_Update_NullClear(t *testing.T) {
	var capturedReq *service.UpdateRequest
	h := newTestHandler()
	h.noteService = &mockNoteService{
		updateFunc: func(ctx context.Context, req *service.UpdateRequest) error {
			capturedReq = req
			return nil
		},
	}
	// title, source, metadata を null でクリア
	params := map[string]any{
		"id": "test-id",
		"patch": map[string]any{
			"title":    nil,
			"source":   nil,
			"metadata": nil,
		},
	}
	req := makeRequest("memory.update", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	// Patchのフィールドがnullクリアとして設定されていることを確認
	// 実際の実装ではNotePatchにnullクリアを示すフラグが必要
	if capturedReq == nil {
		t.Error("expected update to be called")
	}
}

func TestHandle_Update_NullGroupId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		updateFunc: func(ctx context.Context, req *service.UpdateRequest) error {
			return service.ErrGroupIDRequired
		},
	}
	params := map[string]any{
		"id": "test-id",
		"patch": map[string]any{
			"groupId": nil,
		},
	}
	req := makeRequest("memory.update", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 7. memory.list_recent テスト ===

func TestHandle_ListRecent_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
	}
	req := makeRequest("memory.list_recent", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["namespace"] == nil {
		t.Error("expected namespace in result")
	}
	if resultMap["items"] == nil {
		t.Error("expected items in result")
	}
}

func TestHandle_ListRecent_MissingProjectId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		listRecentFunc: func(ctx context.Context, req *service.ListRecentRequest) (*service.ListRecentResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}
	params := map[string]any{}
	req := makeRequest("memory.list_recent", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 8. memory.get_config テスト ===

func TestHandle_GetConfig_Success(t *testing.T) {
	h := newTestHandler()
	req := makeRequest("memory.get_config", nil)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["transportDefaults"] == nil {
		t.Error("expected transportDefaults in result")
	}
	if resultMap["embedder"] == nil {
		t.Error("expected embedder in result")
	}
	if resultMap["store"] == nil {
		t.Error("expected store in result")
	}
	if resultMap["paths"] == nil {
		t.Error("expected paths in result")
	}
}

// === 9. memory.set_config テスト ===

func TestHandle_SetConfig_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"embedder": map[string]any{
			"provider": "openai",
			"model":    "text-embedding-3-small",
		},
	}
	req := makeRequest("memory.set_config", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["ok"] != true {
		t.Error("expected ok: true in result")
	}
	if resultMap["effectiveNamespace"] == nil {
		t.Error("expected effectiveNamespace in result")
	}
}

func TestHandle_SetConfig_EmptyParams(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{}
	req := makeRequest("memory.set_config", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["ok"] != true {
		t.Error("expected ok: true in result (no change)")
	}
}

// === 10. memory.upsert_global テスト ===

func TestHandle_UpsertGlobal_Success(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "global.test.key",
		"value":     "test value",
	}
	req := makeRequest("memory.upsert_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["ok"] != true {
		t.Error("expected ok: true in result")
	}
	if resultMap["id"] == nil {
		t.Error("expected id in result")
	}
	if resultMap["namespace"] == nil {
		t.Error("expected namespace in result")
	}
}

func TestHandle_UpsertGlobal_InvalidKeyPrefix(t *testing.T) {
	h := newTestHandler()
	h.globalService = &mockGlobalService{
		upsertGlobalFunc: func(ctx context.Context, req *service.UpsertGlobalRequest) (*service.UpsertGlobalResponse, error) {
			return nil, service.ErrInvalidGlobalKey
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "invalid.key",
		"value":     "test value",
	}
	req := makeRequest("memory.upsert_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidKeyPrefix {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidKeyPrefix, resp.Error.Code)
	}
}

func TestHandle_UpsertGlobal_MissingProjectId(t *testing.T) {
	h := newTestHandler()
	h.globalService = &mockGlobalService{
		upsertGlobalFunc: func(ctx context.Context, req *service.UpsertGlobalRequest) (*service.UpsertGlobalResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}
	params := map[string]any{
		"key":   "global.test.key",
		"value": "test value",
	}
	req := makeRequest("memory.upsert_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_UpsertGlobal_InvalidUpdatedAt(t *testing.T) {
	h := newTestHandler()
	h.globalService = &mockGlobalService{
		upsertGlobalFunc: func(ctx context.Context, req *service.UpsertGlobalRequest) (*service.UpsertGlobalResponse, error) {
			return nil, service.ErrInvalidTimeFormat
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "global.test.key",
		"value":     "test value",
		"updatedAt": "invalid-time",
	}
	req := makeRequest("memory.upsert_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 11. memory.get_global テスト ===

func TestHandle_GetGlobal_Success_Found(t *testing.T) {
	id := "global-id"
	value := "test value"
	updatedAt := "2024-01-15T10:30:00Z"
	h := newTestHandler()
	h.globalService = &mockGlobalService{
		getGlobalFunc: func(ctx context.Context, projectID, key string) (*service.GetGlobalResponse, error) {
			return &service.GetGlobalResponse{
				Namespace: "test-ns",
				Found:     true,
				ID:        &id,
				Value:     value,
				UpdatedAt: &updatedAt,
			}, nil
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "global.test.key",
	}
	req := makeRequest("memory.get_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["found"] != true {
		t.Error("expected found: true in result")
	}
	if resultMap["value"] != "test value" {
		t.Errorf("expected value 'test value', got %v", resultMap["value"])
	}
}

func TestHandle_GetGlobal_Success_NotFound(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "global.nonexistent",
	}
	req := makeRequest("memory.get_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	resultMap := resp["result"].(map[string]any)
	if resultMap["found"] != false {
		t.Error("expected found: false in result")
	}
}

func TestHandle_GetGlobal_MissingProjectId(t *testing.T) {
	h := newTestHandler()
	h.globalService = &mockGlobalService{
		getGlobalFunc: func(ctx context.Context, projectID, key string) (*service.GetGlobalResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}
	params := map[string]any{
		"key": "global.test.key",
	}
	req := makeRequest("memory.get_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_GetGlobal_MissingKey(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
	}
	req := makeRequest("memory.get_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_GetGlobal_EmptyKey(t *testing.T) {
	h := newTestHandler()
	params := map[string]any{
		"projectId": "/test/project",
		"key":       "",
	}
	req := makeRequest("memory.get_global", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// === 12. 境界値テスト（空文字） ===

func TestHandle_Get_EmptyId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		getFunc: func(ctx context.Context, id string) (*service.GetResponse, error) {
			return nil, service.ErrIDRequired
		},
	}
	params := map[string]any{
		"id": "",
	}
	req := makeRequest("memory.get", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandle_AddNote_EmptyGroupId(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrGroupIDRequired
		},
	}
	params := map[string]any{
		"projectId": "/test/project",
		"groupId":   "",
		"text":      "test note",
	}
	req := makeRequest("memory.add_note", params)
	result := h.Handle(context.Background(), req)
	resp := parseErrorResponse(t, result)

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}
