package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// mockHandler はテスト用のJSON-RPCハンドラー
type mockHandler struct {
	responses map[string]any
}

func newMockHandler() *mockHandler {
	return &mockHandler{
		responses: make(map[string]any),
	}
}

func (h *mockHandler) Handle(ctx context.Context, requestBytes []byte) []byte {
	var req model.Request
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		resp := model.NewParseError(err.Error())
		b, _ := json.Marshal(resp)
		return b
	}

	if req.JSONRPC != "2.0" {
		resp := model.NewInvalidRequest(req.ID, "jsonrpc must be 2.0")
		b, _ := json.Marshal(resp)
		return b
	}

	if req.Method == "" {
		resp := model.NewInvalidRequest(req.ID, "method is required")
		b, _ := json.Marshal(resp)
		return b
	}

	// メソッドに応じてレスポンスを返す
	if response, ok := h.responses[req.Method]; ok {
		resp := model.NewResponse(req.ID, response)
		b, _ := json.Marshal(resp)
		return b
	}

	// 未知のメソッド
	resp := model.NewMethodNotFound(req.ID, req.Method)
	b, _ := json.Marshal(resp)
	return b
}

func (h *mockHandler) SetResponse(method string, response any) {
	h.responses[method] = response
}

// TestServer_Run_SingleRequest は単一リクエスト/レスポンスをテスト
func TestServer_Run_SingleRequest(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{
		"transportDefaults": map[string]any{"defaultTransport": "stdio"},
	})

	input := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}` + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// 出力が1行であることを確認
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}

	// JSONとしてパース可能か確認
	var resp model.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if resp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", resp.ID)
	}
}

// TestServer_Run_MultipleRequests は複数リクエストの連続処理をテスト
func TestServer_Run_MultipleRequests(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	input := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"memory.get_config"}` + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// TestServer_Run_EmptyLines は空行のスキップ処理をテスト
func TestServer_Run_EmptyLines(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	input := "\n" +
		`{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}` + "\n" +
		"\n" +
		`{"jsonrpc":"2.0","id":2,"method":"memory.get_config"}` + "\n" +
		"\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// 空行に対するレスポンスは出力されない
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// TestServer_Run_MultilineText は改行を含むtextをテスト
func TestServer_Run_MultilineText(t *testing.T) {
	handler := newMockHandler()
	// 改行を含むレスポンス
	handler.SetResponse("memory.get", map[string]any{
		"id":   "123",
		"text": "line1\nline2\nline3",
	})

	input := `{"jsonrpc":"2.0","id":1,"method":"memory.get","params":{"id":"123"}}` + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// 出力が1行であることを確認（改行はエスケープされている）
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %q", len(lines), output.String())
	}

	// JSONとしてパース可能か確認
	var resp model.Response
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}
}

// TestServer_Run_InvalidJSON は不正JSONをテスト
func TestServer_Run_InvalidJSON(t *testing.T) {
	handler := newMockHandler()

	input := `{invalid json}` + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// ParseErrorが返ること
	var resp model.ErrorResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &resp); err != nil {
		t.Errorf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != model.ErrCodeParseError {
		t.Errorf("expected ParseError code %d, got %d", model.ErrCodeParseError, resp.Error.Code)
	}
}

// TestServer_Run_InvalidMethod は不正メソッドをテスト
func TestServer_Run_InvalidMethod(t *testing.T) {
	handler := newMockHandler()

	input := `{"jsonrpc":"2.0","id":1,"method":"unknown.method"}` + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// MethodNotFoundが返ること
	var resp model.ErrorResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &resp); err != nil {
		t.Errorf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != model.ErrCodeMethodNotFound {
		t.Errorf("expected MethodNotFound code %d, got %d", model.ErrCodeMethodNotFound, resp.Error.Code)
	}
}

// TestServer_Run_ContextCancel はコンテキストキャンセルをテスト
func TestServer_Run_ContextCancel(t *testing.T) {
	handler := newMockHandler()

	ctx, cancel := context.WithCancel(context.Background())

	// コンテキストキャンセルまでブロックするReader
	reader := &blockingReader{ctx: ctx}
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))

	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	// キャンセル
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for server to stop")
	}
}

// blockingReader はコンテキストキャンセルまでブロックするReader
type blockingReader struct {
	ctx context.Context
}

func (r *blockingReader) Read(p []byte) (n int, err error) {
	// コンテキストがキャンセルされるまでブロック
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

// TestServer_Run_EOF はEOFをテスト
func TestServer_Run_EOF(t *testing.T) {
	handler := newMockHandler()

	// 空のReader（即EOF）
	reader := strings.NewReader("")
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	// EOFはnil返却（正常終了）
	if err != nil {
		t.Errorf("expected nil error on EOF, got %v", err)
	}
}

// TestServer_Run_LargeJSON は大きなJSON（1MB未満）をテスト
func TestServer_Run_LargeJSON(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.add_note", map[string]any{"id": "123", "namespace": "test"})

	// 約900KBのテキスト（1MB境界に近い）
	largeText := strings.Repeat("a", 900*1024)
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "memory.add_note",
		"params": map[string]any{
			"projectId": "/test",
			"groupId":   "global",
			"text":      largeText,
		},
	}
	reqBytes, _ := json.Marshal(req)
	input := string(reqBytes) + "\n"

	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	if err != nil {
		t.Errorf("expected nil error for large JSON, got %v", err)
	}

	// 正常なレスポンスが返ること
	var resp model.Response
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}
}

// TestServer_Run_HugeJSON は巨大なJSON（1MB超過）をテスト
func TestServer_Run_HugeJSON(t *testing.T) {
	handler := newMockHandler()

	// 約1.1MBのテキスト（1MB制限をわずかに超える）
	hugeText := strings.Repeat("a", 1100*1024)
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "memory.add_note",
		"params": map[string]any{
			"projectId": "/test",
			"groupId":   "global",
			"text":      hugeText,
		},
	}
	reqBytes, _ := json.Marshal(req)
	input := string(reqBytes) + "\n"

	reader := strings.NewReader(input)
	var output bytes.Buffer

	server := New(handler, WithReader(reader), WithWriter(&output))
	err := server.Run(context.Background())

	// バッファ制限エラーが発生すること
	if err == nil {
		t.Error("expected error for huge JSON, got nil")
	}
	// エラーがbufio.ErrTooLong相当であること
	if err != nil && !strings.Contains(err.Error(), "token too long") {
		t.Logf("error type: %T, message: %v", err, err)
	}
}

// handlerInterface はテスト用のハンドラーインターフェース
type handlerInterface interface {
	Handle(ctx context.Context, requestBytes []byte) []byte
}

// mockWriter は書き込みエラーをシミュレートするWriter
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

// TestServer_Run_WriteError は書き込みエラーをテスト
func TestServer_Run_WriteError(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	input := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}` + "\n"
	reader := strings.NewReader(input)
	writer := &errorWriter{err: io.ErrClosedPipe}

	server := New(handler, WithReader(reader), WithWriter(writer))
	err := server.Run(context.Background())

	// 書き込みエラーが発生すること
	if err == nil {
		t.Error("expected write error, got nil")
	}
}
