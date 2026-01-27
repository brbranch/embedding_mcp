package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/service"
)

// === MCP initialize テスト ===

func TestHandle_Initialize_Success(t *testing.T) {
	h := newTestHandler()
	req := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"clientInfo": {"name": "test-client", "version": "1.0.0"},
			"capabilities": {}
		}
	}`)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// protocolVersion を確認
	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion '2024-11-05', got %v", resultMap["protocolVersion"])
	}

	// serverInfo を確認
	serverInfo := resultMap["serverInfo"].(map[string]any)
	if serverInfo["name"] != "mcp-memory" {
		t.Errorf("expected serverInfo.name 'mcp-memory', got %v", serverInfo["name"])
	}
	if serverInfo["version"] == nil || serverInfo["version"] == "" {
		t.Error("expected serverInfo.version to be non-empty")
	}

	// capabilities を確認
	capabilities := resultMap["capabilities"].(map[string]any)
	if capabilities["tools"] == nil {
		t.Error("expected capabilities.tools to exist")
	}
}

// === MCP tools/list テスト ===

func TestHandle_ToolsList_Success(t *testing.T) {
	h := newTestHandler()
	req := makeRequest("tools/list", nil)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)
	tools := resultMap["tools"].([]any)

	// 10個のツールがあることを確認
	if len(tools) != 10 {
		t.Errorf("expected 10 tools, got %d", len(tools))
	}

	// ツール名を確認（ドットはアンダースコアに変換）
	expectedTools := []string{
		"memory_add_note",
		"memory_search",
		"memory_get",
		"memory_update",
		"memory_delete",
		"memory_list_recent",
		"memory_get_config",
		"memory_set_config",
		"memory_upsert_global",
		"memory_get_global",
	}

	toolNames := make(map[string]bool)
	for _, t := range tools {
		tool := t.(map[string]any)
		name := tool["name"].(string)
		toolNames[name] = true

		// inputSchema が存在することを確認
		if tool["inputSchema"] == nil {
			panic("expected inputSchema for tool: " + name)
		}
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool %q not found", expected)
		}
	}
}

// === MCP tools/call テスト ===

func TestHandle_ToolsCall_Success(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return &service.AddNoteResponse{ID: "new-note-id", Namespace: "test-ns"}, nil
		},
	}

	// tools/call で memory_add_note を呼び出す
	params := map[string]any{
		"name": "memory_add_note",
		"arguments": map[string]any{
			"projectId": "/test/project",
			"groupId":   "global",
			"text":      "test note content",
		},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// content を確認
	content := resultMap["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected content to have at least one item")
	}

	// 最初の content item を確認
	contentItem := content[0].(map[string]any)
	if contentItem["type"] != "text" {
		t.Errorf("expected content type 'text', got %v", contentItem["type"])
	}

	// text をパースして結果を確認
	text := contentItem["text"].(string)
	var innerResult map[string]any
	if err := json.Unmarshal([]byte(text), &innerResult); err != nil {
		t.Fatalf("failed to parse inner result: %v", err)
	}

	if innerResult["id"] != "new-note-id" {
		t.Errorf("expected id 'new-note-id', got %v", innerResult["id"])
	}
}

func TestHandle_ToolsCall_ToolNotFound(t *testing.T) {
	h := newTestHandler()

	params := map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	// MCP仕様では、ツールが見つからない場合もcontentでエラーを返す（JSON-RPCエラーではない）
	if resp["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// isError フラグを確認
	if resultMap["isError"] != true {
		t.Error("expected isError: true for unknown tool")
	}

	// content にエラーメッセージが含まれることを確認
	content := resultMap["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected content to have at least one item")
	}

	contentItem := content[0].(map[string]any)
	text := contentItem["text"].(string)
	if text == "" {
		t.Error("expected error message in content text")
	}
}

func TestHandle_ToolsCall_ToolError(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		addNoteFunc: func(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
			return nil, service.ErrProjectIDRequired
		},
	}

	params := map[string]any{
		"name": "memory_add_note",
		"arguments": map[string]any{
			"groupId": "global",
			"text":    "test note",
		},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	// MCP仕様では、ツール実行エラーもcontentで返す
	if resp["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// isError フラグを確認
	if resultMap["isError"] != true {
		t.Error("expected isError: true for tool execution error")
	}
}

// === 通知テスト ===

func TestHandle_Notification_Initialized(t *testing.T) {
	h := newTestHandler()

	// ID が nil の通知（notifications/initialized）
	req := []byte(`{
		"jsonrpc": "2.0",
		"method": "notifications/initialized"
	}`)
	result := h.Handle(context.Background(), req)

	// 通知にはレスポンスを返さない
	if result != nil {
		t.Errorf("expected nil response for notification, got %s", string(result))
	}
}

func TestHandle_Notification_ExplicitNull(t *testing.T) {
	h := newTestHandler()

	// ID が明示的に null の通知
	req := []byte(`{
		"jsonrpc": "2.0",
		"id": null,
		"method": "notifications/initialized"
	}`)
	result := h.Handle(context.Background(), req)

	// 通知にはレスポンスを返さない
	if result != nil {
		t.Errorf("expected nil response for notification with null id, got %s", string(result))
	}
}

// === バージョン情報テスト ===

func TestHandle_Initialize_VersionInfo(t *testing.T) {
	h := newTestHandler()
	req := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"clientInfo": {"name": "claude-code", "version": "1.0.0"}
		}
	}`)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	resultMap := resp["result"].(map[string]any)
	serverInfo := resultMap["serverInfo"].(map[string]any)

	// バージョンが設定されていることを確認（空でないこと）
	version := serverInfo["version"].(string)
	if version == "" {
		t.Error("expected non-empty server version")
	}
}

// === tools/call の各ツールテスト ===

func TestHandle_ToolsCall_Search(t *testing.T) {
	h := newTestHandler()
	h.noteService = &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			return &service.SearchResponse{
				Namespace: "test-ns",
				Results: []service.SearchResult{
					{ID: "note-1", Text: "found note", Score: 0.95},
				},
			}, nil
		},
	}

	params := map[string]any{
		"name": "memory_search",
		"arguments": map[string]any{
			"projectId": "/test/project",
			"query":     "test query",
		},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)
	content := resultMap["content"].([]any)

	if len(content) == 0 {
		t.Fatal("expected content")
	}

	// isError が false または存在しないことを確認
	if isError, ok := resultMap["isError"]; ok && isError == true {
		t.Error("expected isError to be false or absent for successful call")
	}
}

func TestHandle_ToolsCall_GetConfig(t *testing.T) {
	h := newTestHandler()

	params := map[string]any{
		"name":      "memory_get_config",
		"arguments": map[string]any{},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// isError が false または存在しないことを確認
	if isError, ok := resultMap["isError"]; ok && isError == true {
		t.Error("expected isError to be false or absent for successful call")
	}
}

// === tools/call パラメータ検証テスト ===

func TestHandle_ToolsCall_MissingName(t *testing.T) {
	h := newTestHandler()

	params := map[string]any{
		"arguments": map[string]any{},
	}
	req := makeRequest("tools/call", params)
	result := h.Handle(context.Background(), req)
	resp := parseResponse(t, result)

	// MCP仕様では、name欠落もcontentでエラーを返す
	if resp["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp["error"])
	}

	resultMap := resp["result"].(map[string]any)

	// isError フラグを確認
	if resultMap["isError"] != true {
		t.Error("expected isError: true for missing tool name")
	}
}
