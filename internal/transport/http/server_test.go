package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

// TestServer_BasicJSONRPCCall は基本的なJSON-RPC呼び出しをテスト
func TestServer_BasicJSONRPCCall(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{
		"transportDefaults": map[string]any{"defaultTransport": "http"},
	})

	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// レスポンスをパース
	var resp model.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}

	if resp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", resp.ID)
	}
}

// TestServer_InvalidJSON は不正なJSONをテスト
func TestServer_InvalidJSON(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// ParseErrorが返ること
	var resp model.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != model.ErrCodeParseError {
		t.Errorf("expected ParseError code %d, got %d", model.ErrCodeParseError, resp.Error.Code)
	}
}

// TestServer_InvalidHTTPMethod は不正なHTTPメソッドをテスト
func TestServer_InvalidHTTPMethod(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	req := httptest.NewRequest("GET", "/rpc", nil)
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestServer_InvalidContentType は不正なContent-Typeをテスト
func TestServer_InvalidContentType(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected status 415, got %d", w.Code)
	}
}

// TestServer_EmptyBody は空ボディをテスト
func TestServer_EmptyBody(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// ParseErrorが返ること
	var resp model.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != model.ErrCodeParseError {
		t.Errorf("expected ParseError code %d, got %d", model.ErrCodeParseError, resp.Error.Code)
	}
}

// TestServer_GracefulShutdown はGraceful Shutdownをテスト
func TestServer_GracefulShutdown(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// サーバーが起動するまで待機
	time.Sleep(100 * time.Millisecond)

	// キャンセル
	cancel()

	// サーバーが停止するまで待機
	select {
	case err := <-errCh:
		// http.ErrServerClosed はエラーとして返らないこと
		if err == http.ErrServerClosed {
			t.Errorf("expected nil, got http.ErrServerClosed")
		}
		if err != nil && err != context.Canceled {
			t.Errorf("expected nil or context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for server to stop")
	}
}

// TestServer_LargeJSON は大きなJSONをテスト
func TestServer_LargeJSON(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.add_note", map[string]any{"id": "123", "namespace": "test"})

	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	// 約900KBのテキスト
	largeText := strings.Repeat("a", 900*1024)
	reqMap := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "memory.add_note",
		"params": map[string]any{
			"projectId": "/test",
			"groupId":   "global",
			"text":      largeText,
		},
	}
	reqBytes, _ := json.Marshal(reqMap)

	req := httptest.NewRequest("POST", "/rpc", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// 正常なレスポンスが返ること
	var resp model.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}
}

// TestServer_ReadBodyError は本体読み取りエラーをテスト
func TestServer_ReadBodyError(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	// エラーを返すReader
	req := httptest.NewRequest("POST", "/rpc", &errorReader{err: io.ErrUnexpectedEOF})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// errorReader はエラーを返すReader
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// TestServer_TooLargeBody はサイズ制限を超えるボディをテスト
func TestServer_TooLargeBody(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	// MaxBodySize (1MB) を超えるボディ
	largeBody := strings.Repeat("a", MaxBodySize+1)
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", w.Code)
	}
}

// TestServer_DefaultAddr はAddr未設定時のデフォルト値をテスト
func TestServer_DefaultAddr(t *testing.T) {
	handler := newMockHandler()

	// Addr未設定
	server := New(handler, Config{})

	if server.srv.Addr != DefaultAddr {
		t.Errorf("expected default addr %s, got %s", DefaultAddr, server.srv.Addr)
	}
}

// TestServer_ReadHeaderTimeout はReadHeaderTimeoutが設定されていることをテスト
func TestServer_ReadHeaderTimeout(t *testing.T) {
	handler := newMockHandler()
	server := New(handler, Config{})

	if server.srv.ReadHeaderTimeout == 0 {
		t.Error("expected ReadHeaderTimeout to be set")
	}
}
