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
func (h *Handler) Handle(ctx context.Context, requestBytes []byte) []byte {
	// 1. パース
	var req model.Request
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		return h.encodeError(model.NewParseError(err.Error()))
	}

	// 2. バージョン確認
	if req.JSONRPC != "2.0" {
		return h.encodeError(model.NewInvalidRequest(req.ID, "jsonrpc must be 2.0"))
	}

	// 3. method確認
	if req.Method == "" {
		return h.encodeError(model.NewInvalidRequest(req.ID, "method is required"))
	}

	// 4. ディスパッチ
	result, err := h.dispatch(ctx, req.ID, req.Method, req.Params)
	if err != nil {
		return h.encodeError(h.mapError(req.ID, err))
	}

	// 5. 成功レスポンス
	return h.encodeResponse(model.NewResponse(req.ID, result))
}

// dispatch はメソッドに応じて適切なハンドラーを呼び出す
func (h *Handler) dispatch(ctx context.Context, id any, method string, params any) (any, error) {
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
		errors.Is(err, errKeyRequired) {
		return model.NewInvalidParams(id, err.Error())
	}

	// not found
	if errors.Is(err, service.ErrNoteNotFound) {
		return model.NewErrorResponse(id, model.ErrCodeNotFound, "Note not found", nil)
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
