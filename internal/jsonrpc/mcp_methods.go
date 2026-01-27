package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// ServerVersion はサーバーのバージョン（ビルド時に設定可能）
var ServerVersion = "0.1.0"

// handleInitialize は initialize メソッドを処理
func (h *Handler) handleInitialize(ctx context.Context, params any) (any, error) {
	// パラメータをパース（検証は最小限）
	var p model.InitializeParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	return &model.InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: model.ServerInfo{
			Name:    "mcp-memory",
			Version: ServerVersion,
		},
		Capabilities: model.Capabilities{
			Tools: &model.ToolsCapability{},
		},
	}, nil
}

// handleToolsList は tools/list メソッドを処理
func (h *Handler) handleToolsList(ctx context.Context, params any) (any, error) {
	return &model.ToolsListResult{
		Tools: mcpTools,
	}, nil
}

// handleToolsCall は tools/call メソッドを処理
func (h *Handler) handleToolsCall(ctx context.Context, id any, params any) (any, error) {
	var p model.ToolsCallParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	// ツール名必須チェック
	if p.Name == "" {
		return &model.ToolsCallResult{
			Content: []model.ContentItem{
				model.NewTextContent("Error: tool name is required"),
			},
			IsError: true,
		}, nil
	}

	// ツール名から内部メソッド名を取得
	internalMethod, ok := toolNameToMethod[p.Name]
	if !ok {
		// ツールが見つからない場合はエラーをcontentに含める
		return &model.ToolsCallResult{
			Content: []model.ContentItem{
				model.NewTextContent(fmt.Sprintf("Tool not found: %s", p.Name)),
			},
			IsError: true,
		}, nil
	}

	// 内部メソッドを呼び出す
	result, err := h.dispatchInternal(ctx, id, internalMethod, p.Arguments)
	if err != nil {
		// エラーをcontentに含める（MCP仕様）
		return &model.ToolsCallResult{
			Content: []model.ContentItem{
				model.NewTextContent(fmt.Sprintf("Error: %s", err.Error())),
			},
			IsError: true,
		}, nil
	}

	// 結果をJSON文字列に変換してcontentに含める
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &model.ToolsCallResult{
			Content: []model.ContentItem{
				model.NewTextContent(fmt.Sprintf("Error serializing result: %s", err.Error())),
			},
			IsError: true,
		}, nil
	}

	return &model.ToolsCallResult{
		Content: []model.ContentItem{
			model.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// dispatchInternal は内部メソッドを直接呼び出す（tools/call用）
func (h *Handler) dispatchInternal(ctx context.Context, id any, method string, params any) (any, error) {
	switch method {
	case "memory.add_note":
		return h.handleAddNote(ctx, params)
	case "memory.search":
		return h.handleSearch(ctx, params)
	case "memory.get":
		return h.handleGet(ctx, params)
	case "memory.update":
		return h.handleUpdate(ctx, params)
	case "memory.list_recent":
		return h.handleListRecent(ctx, params)
	case "memory.get_config":
		return h.handleGetConfig(ctx)
	case "memory.set_config":
		return h.handleSetConfig(ctx, params)
	case "memory.upsert_global":
		return h.handleUpsertGlobal(ctx, params)
	case "memory.get_global":
		return h.handleGetGlobal(ctx, params)
	case "memory.delete":
		return h.handleDelete(ctx, params)
	default:
		return nil, &methodNotFoundError{method: method}
	}
}

