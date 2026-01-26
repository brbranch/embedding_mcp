package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestCORS_Disabled はCORS無効時のテスト
func TestCORS_Disabled(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	// CORSOrigins未設定（デフォルト）
	server := New(handler, Config{
		Addr: "127.0.0.1:0",
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// CORSヘッダーが付与されないこと
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// TestCORS_Enabled はCORS有効時のテスト
func TestCORS_Enabled(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	server := New(handler, Config{
		Addr:        "127.0.0.1:0",
		CORSOrigins: []string{"http://example.com"},
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// 適切なCORSヘッダーが付与されること
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin http://example.com, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Errorf("expected methods POST, OPTIONS, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("expected headers Content-Type, got %q", w.Header().Get("Access-Control-Allow-Headers"))
	}
}

// TestCORS_Preflight はOPTIONSリクエスト（Preflight）をテスト
func TestCORS_Preflight(t *testing.T) {
	handler := newMockHandler()

	server := New(handler, Config{
		Addr:        "127.0.0.1:0",
		CORSOrigins: []string{"http://example.com"},
	})

	req := httptest.NewRequest("OPTIONS", "/rpc", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// 適切なCORSヘッダーが返ること
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin http://example.com, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Errorf("expected methods POST, OPTIONS, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("expected headers Content-Type, got %q", w.Header().Get("Access-Control-Allow-Headers"))
	}

	// レスポンスボディは空であること
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

// TestCORS_UnallowedOrigin は許可されていないオリジンをテスト
func TestCORS_UnallowedOrigin(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	server := New(handler, Config{
		Addr:        "127.0.0.1:0",
		CORSOrigins: []string{"http://example.com"},
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// CORSヘッダーが付与されないこと
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// 正常なレスポンスは返ること（CORSはブラウザ側でブロック）
	var resp model.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse response: %v", err)
	}
}

// TestCORS_MultipleOrigins は複数オリジンをテスト
func TestCORS_MultipleOrigins(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	server := New(handler, Config{
		Addr:        "127.0.0.1:0",
		CORSOrigins: []string{"http://example.com", "http://localhost:3000"},
	})

	// 最初のオリジン
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin http://example.com, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// 2番目のオリジン
	req = httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	w = httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected CORS origin http://localhost:3000, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// TestCORS_VaryHeader はVaryヘッダーをテスト
func TestCORS_VaryHeader(t *testing.T) {
	handler := newMockHandler()
	handler.SetResponse("memory.get_config", map[string]any{"ok": true})

	server := New(handler, Config{
		Addr:        "127.0.0.1:0",
		CORSOrigins: []string{"http://example.com"},
	})

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	server.handleRPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Vary: Origin が付与されること
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", w.Header().Get("Vary"))
	}
}
