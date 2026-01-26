// Package http implements HTTP transport for mcp-memory.
package http

import (
	"context"
	"io"
	"net/http"
	"strings"
)

// Handler はJSON-RPCリクエストを処理する
type Handler interface {
	Handle(ctx context.Context, requestBytes []byte) []byte
}

// Config はHTTPサーバー設定
type Config struct {
	Addr        string   // listen address (例: "127.0.0.1:8765")
	CORSOrigins []string // 許可するオリジンリスト、空ならCORS無効
}

// Server はHTTP JSON-RPCサーバー
type Server struct {
	handler Handler
	config  Config
	srv     *http.Server
}

// New は新しいServerを生成
func New(handler Handler, config Config) *Server {
	s := &Server{
		handler: handler,
		config:  config,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)

	s.srv = &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	return s
}

// Run はサーバーを起動し、contextがキャンセルされるまで実行
func (s *Server) Run(ctx context.Context) error {
	// contextキャンセル時にShutdownを呼ぶ
	go func() {
		<-ctx.Done()
		s.srv.Shutdown(context.Background())
	}()

	err := s.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		// Graceful shutdownはエラーではない
		return nil
	}
	return err
}

// handleRPC はJSON-RPCリクエストを処理
func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	// CORS処理
	s.handleCORS(w, r)

	// Preflightリクエスト
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// POSTのみ許可
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content-Type確認
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}

	// リクエストボディ読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// JSON-RPC処理
	respBytes := s.handler.Handle(r.Context(), body)

	// レスポンス送信
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

// handleCORS はCORSヘッダーを設定
func (s *Server) handleCORS(w http.ResponseWriter, r *http.Request) {
	// CORS無効ならスキップ
	if len(s.config.CORSOrigins) == 0 {
		return
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	// 許可オリジンをチェック
	allowed := false
	for _, allowedOrigin := range s.config.CORSOrigins {
		if origin == allowedOrigin {
			allowed = true
			break
		}
	}

	if !allowed {
		return
	}

	// CORSヘッダーを設定
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")
}
