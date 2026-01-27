//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestE2E_MCP_FullFlow は MCP プロトコルの一連のフローをテスト
// initialize → tools/list → tools/call (memory_add_note) → tools/call (memory_search)
func TestE2E_MCP_FullFlow(t *testing.T) {
	h := setupTestHandler(t)
	ctx := context.Background()

	// 1. initialize
	t.Run("initialize", func(t *testing.T) {
		req := []byte(`{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "initialize",
			"params": {
				"protocolVersion": "2024-11-05",
				"clientInfo": {"name": "claude-code", "version": "1.0.0"},
				"capabilities": {}
			}
		}`)

		respBytes := h.Handle(ctx, req)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("initialize failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)

		// protocolVersion を確認
		if result["protocolVersion"] != "2024-11-05" {
			t.Errorf("expected protocolVersion '2024-11-05', got %v", result["protocolVersion"])
		}

		// serverInfo を確認
		serverInfo := result["serverInfo"].(map[string]any)
		if serverInfo["name"] != "mcp-memory" {
			t.Errorf("expected serverInfo.name 'mcp-memory', got %v", serverInfo["name"])
		}

		// capabilities を確認
		capabilities := result["capabilities"].(map[string]any)
		if capabilities["tools"] == nil {
			t.Error("expected capabilities.tools to exist")
		}
	})

	// 2. notifications/initialized（レスポンスなし）
	t.Run("notifications/initialized", func(t *testing.T) {
		req := []byte(`{
			"jsonrpc": "2.0",
			"method": "notifications/initialized"
		}`)

		respBytes := h.Handle(ctx, req)

		// 通知にはレスポンスを返さない
		if respBytes != nil {
			t.Errorf("expected nil response for notification, got %s", string(respBytes))
		}
	})

	// 3. tools/list
	t.Run("tools/list", func(t *testing.T) {
		req := []byte(`{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tools/list"
		}`)

		respBytes := h.Handle(ctx, req)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("tools/list failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)
		tools := result["tools"].([]any)

		// 10個のツールがあることを確認
		if len(tools) != 10 {
			t.Errorf("expected 10 tools, got %d", len(tools))
		}

		// 必要なツールが存在することを確認
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolMap := tool.(map[string]any)
			name := toolMap["name"].(string)
			toolNames[name] = true

			// inputSchema が存在することを確認
			if toolMap["inputSchema"] == nil {
				t.Errorf("expected inputSchema for tool: %s", name)
			}
		}

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

		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("expected tool %q not found", expected)
			}
		}
	})

	// 4. tools/call (memory_add_note)
	var noteID string
	t.Run("tools/call memory_add_note", func(t *testing.T) {
		reqBytes, _ := json.Marshal(model.Request{
			JSONRPC: "2.0",
			ID:      3,
			Method:  "tools/call",
			Params: map[string]any{
				"name": "memory_add_note",
				"arguments": map[string]any{
					"projectId": "/test/mcp-e2e",
					"groupId":   "global",
					"text":      "MCP E2E テストノート",
					"tags":      []string{"test", "e2e"},
				},
			},
		})

		respBytes := h.Handle(ctx, reqBytes)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("tools/call failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)

		// isError が存在しないか false であることを確認
		if isError, ok := result["isError"]; ok && isError.(bool) {
			t.Fatalf("expected isError to be false, got true")
		}

		// content を確認
		content := result["content"].([]any)
		if len(content) == 0 {
			t.Fatal("expected content to have at least one item")
		}

		contentItem := content[0].(map[string]any)
		if contentItem["type"] != "text" {
			t.Errorf("expected content type 'text', got %v", contentItem["type"])
		}

		// text をパースして ID を取得
		text := contentItem["text"].(string)
		var innerResult map[string]any
		if err := json.Unmarshal([]byte(text), &innerResult); err != nil {
			t.Fatalf("failed to parse inner result: %v", err)
		}

		noteID = innerResult["id"].(string)
		if noteID == "" {
			t.Error("expected note ID to be non-empty")
		}
	})

	// 5. tools/call (memory_search)
	t.Run("tools/call memory_search", func(t *testing.T) {
		reqBytes, _ := json.Marshal(model.Request{
			JSONRPC: "2.0",
			ID:      4,
			Method:  "tools/call",
			Params: map[string]any{
				"name": "memory_search",
				"arguments": map[string]any{
					"projectId": "/test/mcp-e2e",
					"query":     "E2E テスト",
				},
			},
		})

		respBytes := h.Handle(ctx, reqBytes)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("tools/call failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)

		// isError が存在しないか false であることを確認
		if isError, ok := result["isError"]; ok && isError.(bool) {
			content := result["content"].([]any)
			if len(content) > 0 {
				t.Fatalf("search failed: %v", content[0])
			}
		}

		// content を確認
		content := result["content"].([]any)
		if len(content) == 0 {
			t.Fatal("expected content to have at least one item")
		}

		contentItem := content[0].(map[string]any)
		text := contentItem["text"].(string)

		var innerResult map[string]any
		if err := json.Unmarshal([]byte(text), &innerResult); err != nil {
			t.Fatalf("failed to parse inner result: %v", err)
		}

		results := innerResult["results"].([]any)
		if len(results) == 0 {
			t.Error("expected at least 1 search result")
		}

		// 追加したノートが見つかることを確認
		found := false
		for _, r := range results {
			rMap := r.(map[string]any)
			if rMap["id"].(string) == noteID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find note with ID %s in search results", noteID)
		}
	})

	// 6. tools/call (memory_get)
	t.Run("tools/call memory_get", func(t *testing.T) {
		reqBytes, _ := json.Marshal(model.Request{
			JSONRPC: "2.0",
			ID:      5,
			Method:  "tools/call",
			Params: map[string]any{
				"name": "memory_get",
				"arguments": map[string]any{
					"id": noteID,
				},
			},
		})

		respBytes := h.Handle(ctx, reqBytes)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("tools/call failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)

		// isError が存在しないか false であることを確認
		if isError, ok := result["isError"]; ok && isError.(bool) {
			t.Fatalf("expected isError to be false")
		}

		// content を確認
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		text := contentItem["text"].(string)

		var innerResult map[string]any
		if err := json.Unmarshal([]byte(text), &innerResult); err != nil {
			t.Fatalf("failed to parse inner result: %v", err)
		}

		if innerResult["id"].(string) != noteID {
			t.Errorf("expected ID %s, got %s", noteID, innerResult["id"])
		}
		if innerResult["text"].(string) != "MCP E2E テストノート" {
			t.Errorf("expected text 'MCP E2E テストノート', got %v", innerResult["text"])
		}
	})

	// 7. tools/call (memory_delete)
	t.Run("tools/call memory_delete", func(t *testing.T) {
		reqBytes, _ := json.Marshal(model.Request{
			JSONRPC: "2.0",
			ID:      6,
			Method:  "tools/call",
			Params: map[string]any{
				"name": "memory_delete",
				"arguments": map[string]any{
					"id": noteID,
				},
			},
		})

		respBytes := h.Handle(ctx, reqBytes)
		var resp RawResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("tools/call failed: %v", resp.Error)
		}

		result := resp.Result.(map[string]any)

		// isError が存在しないか false であることを確認
		if isError, ok := result["isError"]; ok && isError.(bool) {
			t.Fatalf("expected isError to be false")
		}

		// content を確認
		content := result["content"].([]any)
		contentItem := content[0].(map[string]any)
		text := contentItem["text"].(string)

		var innerResult map[string]any
		if err := json.Unmarshal([]byte(text), &innerResult); err != nil {
			t.Fatalf("failed to parse inner result: %v", err)
		}

		if innerResult["ok"] != true {
			t.Error("expected ok: true")
		}
	})
}

// TestE2E_MCP_ToolError はツールエラーの処理をテスト
func TestE2E_MCP_ToolError(t *testing.T) {
	h := setupTestHandler(t)
	ctx := context.Background()

	// 存在しないノートを取得
	reqBytes, _ := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "memory_get",
			"arguments": map[string]any{
				"id": "nonexistent-id",
			},
		},
	})

	respBytes := h.Handle(ctx, reqBytes)
	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// JSON-RPC レベルではエラーにならない
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)

	// isError が true であることを確認
	if isError, ok := result["isError"]; !ok || !isError.(bool) {
		t.Error("expected isError: true for tool error")
	}

	// content にエラーメッセージが含まれることを確認
	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected content to have at least one item")
	}

	contentItem := content[0].(map[string]any)
	text := contentItem["text"].(string)
	if text == "" {
		t.Error("expected error message in content text")
	}
}

// TestE2E_MCP_UnknownTool は存在しないツールの処理をテスト
func TestE2E_MCP_UnknownTool(t *testing.T) {
	h := setupTestHandler(t)
	ctx := context.Background()

	reqBytes, _ := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "nonexistent_tool",
			"arguments": map[string]any{},
		},
	})

	respBytes := h.Handle(ctx, reqBytes)
	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// JSON-RPC レベルではエラーにならない
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)

	// isError が true であることを確認
	if isError, ok := result["isError"]; !ok || !isError.(bool) {
		t.Error("expected isError: true for unknown tool")
	}
}

// TestE2E_MCP_BackwardsCompatibility は memory.* メソッドの後方互換性をテスト
func TestE2E_MCP_BackwardsCompatibility(t *testing.T) {
	h := setupTestHandler(t)
	ctx := context.Background()

	// memory.add_note を直接呼び出し（MCP tools/call 経由ではなく）
	reqBytes, _ := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.add_note",
		Params: map[string]any{
			"projectId": "/test/backwards-compat",
			"groupId":   "global",
			"text":      "後方互換性テスト",
		},
	})

	respBytes := h.Handle(ctx, reqBytes)
	var resp RawResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("memory.add_note failed: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	if result["id"] == nil || result["id"] == "" {
		t.Error("expected id in result")
	}
}
