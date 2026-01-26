// Package stdio implements stdio transport for mcp-memory.
package stdio

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
)

// MaxBufferSize はScannerの最大バッファサイズ（1MB）
const MaxBufferSize = 1024 * 1024

// Handler はJSON-RPCリクエストを処理するインターフェース
type Handler interface {
	Handle(ctx context.Context, requestBytes []byte) []byte
}

// Server はstdio JSON-RPCサーバー
type Server struct {
	handler Handler
	reader  io.Reader
	writer  io.Writer
}

// Option はサーバーオプション
type Option func(*Server)

// WithReader はreaderを設定（テスト用）
func WithReader(r io.Reader) Option {
	return func(s *Server) {
		s.reader = r
	}
}

// WithWriter はwriterを設定（テスト用）
func WithWriter(w io.Writer) Option {
	return func(s *Server) {
		s.writer = w
	}
}

// New は新しいServerを生成
func New(handler Handler, opts ...Option) *Server {
	s := &Server{
		handler: handler,
		reader:  os.Stdin,
		writer:  os.Stdout,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run はサーバーを起動し、contextがキャンセルされるまで実行
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.reader)
	// バッファサイズを1MBに拡張
	buf := make([]byte, MaxBufferSize)
	scanner.Buffer(buf, MaxBufferSize)

	// コンテキストキャンセルをチェックするチャネル
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	for {
		// コンテキストキャンセルをチェック
		select {
		case <-done:
			return ctx.Err()
		default:
		}

		// 1行読み取り
		if !scanner.Scan() {
			// EOFまたはエラー
			if err := scanner.Err(); err != nil {
				return err
			}
			// EOF: 正常終了
			return nil
		}

		line := scanner.Text()

		// 空行はスキップ
		if strings.TrimSpace(line) == "" {
			continue
		}

		// ハンドラーでリクエストを処理
		response := s.handler.Handle(ctx, []byte(line))

		// レスポンスを書き込み（1行 + 改行）
		if _, err := s.writer.Write(response); err != nil {
			return err
		}
		if _, err := s.writer.Write([]byte("\n")); err != nil {
			return err
		}
	}
}
