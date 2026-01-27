// Package jsonrpc implements JSON-RPC 2.0 handlers for mcp-memory.
package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/brbranch/embedding_mcp/internal/embedder"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/service"
)

// Handler はJSON-RPCリクエストを処理する
type Handler struct {
	noteService   service.NoteService
	configService service.ConfigService
	globalService service.GlobalService
}

// New は新しいHandlerを生成
func New(
	noteService service.NoteService,
	configService service.ConfigService,
	globalService service.GlobalService,
) *Handler {
	return &Handler{
		noteService:   noteService,
		configService: configService,
		globalService: globalService,
	}
}

// Handle はJSON-RPCリクエストをパースしてディスパッチ
// 戻り値は *model.Response または *model.ErrorResponse のJSON bytes
// 通知（idがnilまたは未設定）の場合はnilを返す
func (h *Handler) Handle(ctx context.Context, requestBytes []byte) []byte {
	// 1. パース（ID の存在を確認するため raw で）
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(requestBytes, &raw); err != nil {
		return h.encodeError(model.NewParseError(err.Error()))
	}

	// ID の有無と値を確認
	idRaw, hasID := raw["id"]
	isNotification := !hasID || (hasID && string(idRaw) == "null")

	// 構造体にパース
	var req model.Request
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		return h.encodeError(model.NewParseError(err.Error()))
	}

	// 2. バージョン確認
	if req.JSONRPC != "2.0" {
		if isNotification {
			return nil
		}
		return h.encodeError(model.NewInvalidRequest(req.ID, "jsonrpc must be 2.0"))
	}

	// 3. method確認
	if req.Method == "" {
		if isNotification {
			return nil
		}
		return h.encodeError(model.NewInvalidRequest(req.ID, "method is required"))
	}

	// 4. 通知の処理（レスポンスを返さない）
	if isNotification {
		// 通知は処理するがレスポンスは返さない
		// dispatch して結果は捨てる
		_, _ = h.dispatch(ctx, req.ID, req.Method, req.Params)
		return nil
	}

	// 5. ディスパッチ
	result, err := h.dispatch(ctx, req.ID, req.Method, req.Params)
	if err != nil {
		return h.encodeError(h.mapError(req.ID, err))
	}

	// 6. 成功レスポンス
	return h.encodeResponse(model.NewResponse(req.ID, result))
}

// dispatch はメソッドに応じて適切なハンドラーを呼び出す
func (h *Handler) dispatch(ctx context.Context, id any, method string, params any) (any, error) {
	switch method {
	// MCP 標準メソッド
	case "initialize":
		return h.handleInitialize(ctx, params)
	case "tools/list":
		return h.handleToolsList(ctx, params)
	case "tools/call":
		return h.handleToolsCall(ctx, id, params)
	// memory.* メソッド（後方互換性のため維持）
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

// mapError はサービスエラーをJSON-RPCエラーに変換
func (h *Handler) mapError(id any, err error) *model.ErrorResponse {
	// method not found
	var mnfErr *methodNotFoundError
	if errors.As(err, &mnfErr) {
		return model.NewMethodNotFound(id, mnfErr.method)
	}

	// invalid params
	if errors.Is(err, service.ErrProjectIDRequired) ||
		errors.Is(err, service.ErrGroupIDRequired) ||
		errors.Is(err, service.ErrInvalidGroupID) ||
		errors.Is(err, service.ErrTextRequired) ||
		errors.Is(err, service.ErrQueryRequired) ||
		errors.Is(err, service.ErrIDRequired) ||
		errors.Is(err, service.ErrInvalidTimeFormat) ||
		errors.Is(err, errKeyRequired) ||
		errors.Is(err, errIDRequired) {
		return model.NewInvalidParams(id, err.Error())
	}

	// not found
	if errors.Is(err, service.ErrNoteNotFound) {
		return model.NewErrorResponse(id, model.ErrCodeNotFound, "Note not found", nil)
	}
	if errors.Is(err, service.ErrGlobalConfigNotFound) {
		return model.NewErrorResponse(id, model.ErrCodeNotFound, "Global config not found", nil)
	}
	if errors.Is(err, errNotFound) {
		return model.NewErrorResponse(id, model.ErrCodeNotFound, "Not found", nil)
	}

	// invalid key prefix
	if errors.Is(err, service.ErrInvalidGlobalKey) {
		return model.NewErrorResponse(id, model.ErrCodeInvalidKeyPrefix, err.Error(), nil)
	}

	// API key missing
	if errors.Is(err, embedder.ErrAPIKeyRequired) {
		return model.NewErrorResponse(id, model.ErrCodeAPIKeyMissing, err.Error(), nil)
	}

	// internal error
	return model.NewInternalError(id, err.Error())
}

func (h *Handler) encodeResponse(resp *model.Response) []byte {
	b, _ := json.Marshal(resp)
	return b
}

func (h *Handler) encodeError(resp *model.ErrorResponse) []byte {
	b, _ := json.Marshal(resp)
	return b
}

// methodNotFoundError はメソッド未検出エラー
type methodNotFoundError struct {
	method string
}

func (e *methodNotFoundError) Error() string {
	return "method not found: " + e.method
}

// errKeyRequired はkey必須エラー
var errKeyRequired = errors.New("key is required")

// errIDRequired はID必須エラー
var errIDRequired = errors.New("id is required")

// errNotFound はNot Foundエラー（Note/GlobalConfig両方で見つからない場合）
var errNotFound = errors.New("not found")
