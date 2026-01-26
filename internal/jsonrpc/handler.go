// Package jsonrpc implements JSON-RPC 2.0 handlers for mcp-memory.
package jsonrpc

import (
	"context"

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
	// TODO: implement
	return nil
}
