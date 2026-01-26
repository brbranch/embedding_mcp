package main

import (
	"bytes"
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestParseFlags_DefaultOptions はデフォルトオプション解析をテスト
func TestParseFlags_DefaultOptions(t *testing.T) {
	args := []string{"serve"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Transport != defaultTransport {
		t.Errorf("expected transport %s, got %s", defaultTransport, opts.Transport)
	}
	if opts.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", opts.Host)
	}
	if opts.Port != 8765 {
		t.Errorf("expected port 8765, got %d", opts.Port)
	}
}

// TestParseFlags_TransportStdio はtransport=stdioオプションをテスト
func TestParseFlags_TransportStdio(t *testing.T) {
	args := []string{"serve", "--transport", "stdio"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Transport != "stdio" {
		t.Errorf("expected transport stdio, got %s", opts.Transport)
	}
}

// TestParseFlags_TransportHTTP はtransport=httpオプションをテスト
func TestParseFlags_TransportHTTP(t *testing.T) {
	args := []string{"serve", "--transport", "http"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Transport != "http" {
		t.Errorf("expected transport http, got %s", opts.Transport)
	}
}

// TestParseFlags_HostPortOptions は--host, --portオプションをテスト
func TestParseFlags_HostPortOptions(t *testing.T) {
	args := []string{"serve", "--host", "0.0.0.0", "--port", "9999"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", opts.Host)
	}
	if opts.Port != 9999 {
		t.Errorf("expected port 9999, got %d", opts.Port)
	}
}

// TestParseFlags_ShortOptions は短縮オプションをテスト
func TestParseFlags_ShortOptions(t *testing.T) {
	args := []string{"serve", "-t", "http", "-p", "9999"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Transport != "http" {
		t.Errorf("expected transport http, got %s", opts.Transport)
	}
	if opts.Port != 9999 {
		t.Errorf("expected port 9999, got %d", opts.Port)
	}
}

// TestParseFlags_ConfigPath はconfig指定をテスト
func TestParseFlags_ConfigPath(t *testing.T) {
	args := []string{"serve", "--config", "/path/to/config.json"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.ConfigPath != "/path/to/config.json" {
		t.Errorf("expected config path /path/to/config.json, got %s", opts.ConfigPath)
	}
}

// TestParseFlags_InvalidTransport は不正なtransportでエラーを返すことをテスト
func TestParseFlags_InvalidTransport(t *testing.T) {
	args := []string{"serve", "--transport", "unknown"}
	_, err := parseFlags(args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "invalid transport: unknown (must be stdio or http)"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestParseFlags_InvalidPort は不正なportでエラーを返すことをテスト
func TestParseFlags_InvalidPort(t *testing.T) {
	testCases := []struct {
		name     string
		port     string
		expected string
	}{
		{"port 0", "0", "invalid port: 0 (must be 1-65535)"},
		{"port 70000", "70000", "invalid port: 70000 (must be 1-65535)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"serve", "--port", tc.port}
			_, err := parseFlags(args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Error() != tc.expected {
				t.Errorf("expected error message '%s', got '%s'", tc.expected, err.Error())
			}
		})
	}
}

// TestParseFlags_InvalidSubcommand は不正なサブコマンドでエラーを返すことをテスト
func TestParseFlags_InvalidSubcommand(t *testing.T) {
	args := []string{"unknown"}
	_, err := parseFlags(args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "usage: mcp-memory serve [options]"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestSetupSignalHandler_SIGINT はSIGINT受信でcontextがキャンセルされることをテスト
func TestSetupSignalHandler_SIGINT(t *testing.T) {
	ctx, cancel := setupSignalHandler()
	defer cancel()

	// SIGINTを送信
	go func() {
		time.Sleep(10 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGINT)
	}()

	// contextがキャンセルされるまで待機
	select {
	case <-ctx.Done():
		// 成功
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled after SIGINT")
	}
}

// TestSetupSignalHandler_SIGTERM はSIGTERM受信でcontextがキャンセルされることをテスト
func TestSetupSignalHandler_SIGTERM(t *testing.T) {
	ctx, cancel := setupSignalHandler()
	defer cancel()

	// SIGTERMを送信
	go func() {
		time.Sleep(10 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)
	}()

	// contextがキャンセルされるまで待機
	select {
	case <-ctx.Done():
		// 成功
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled after SIGTERM")
	}
}

// MockHandler はテスト用のJSON-RPCハンドラーモック
type MockHandler struct {
	HandleFunc func(ctx context.Context, requestBytes []byte) []byte
}

func (m *MockHandler) Handle(ctx context.Context, requestBytes []byte) []byte {
	if m.HandleFunc != nil {
		return m.HandleFunc(ctx, requestBytes)
	}
	return []byte(`{"jsonrpc":"2.0","result":null,"id":1}`)
}

// TestStdioServer_StartAndShutdown はstdio起動・終了をテスト
func TestStdioServer_StartAndShutdown(t *testing.T) {
	// NOTE: 実際のrunServe統合テストは依存が大きいため、
	// ここではstdio.Server単体の動作を確認する簡易テスト。
	// 完全な統合テストは依存関係のモック化が必要。

	t.Skip("統合テスト: 実装完了後に有効化")

	// 以下は統合テスト例（実装時に有効化）
	// ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	// defer cancel()
	//
	// handler := &MockHandler{}
	// server := stdio.New(handler, stdio.WithReader(bytes.NewReader([]byte(""))))
	//
	// err := server.Run(ctx)
	// if err != nil && err != context.DeadlineExceeded {
	//     t.Errorf("unexpected error: %v", err)
	// }
}

// TestHTTPServer_StartAndShutdown はHTTP起動・終了をテスト
func TestHTTPServer_StartAndShutdown(t *testing.T) {
	// NOTE: 実際のrunServe統合テストは依存が大きいため、
	// ここではhttp.Server単体の動作を確認する簡易テスト。

	t.Skip("統合テスト: 実装完了後に有効化")

	// 以下は統合テスト例（実装時に有効化）
	// ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	// defer cancel()
	//
	// handler := &MockHandler{}
	// server := http.New(handler, http.Config{Addr: "127.0.0.1:0"})
	//
	// go func() {
	//     server.Run(ctx)
	// }()
	//
	// time.Sleep(50 * time.Millisecond)
	// cancel()
}

// TestEnvAPIKey は環境変数によるapiKey上書きをテスト
func TestEnvAPIKey(t *testing.T) {
	// NOTE: initComponents内でos.Getenv("OPENAI_API_KEY")を使用している。
	// 環境変数が設定されている場合、embedder.NewEmbedderに渡される。
	// このテストは統合テストとして実装時に確認する。

	t.Skip("統合テスト: 実装完了後に環境変数設定して確認")

	// 以下は統合テスト例（実装時に有効化）
	// os.Setenv("OPENAI_API_KEY", "test-key")
	// defer os.Unsetenv("OPENAI_API_KEY")
	//
	// // initComponentsを呼び出して環境変数が使われることを確認
}

// TestDefaultConfig は設定ファイルが存在しない場合のデフォルト動作をテスト
func TestDefaultConfig(t *testing.T) {
	// NOTE: config.NewManager("")でデフォルト設定が使われる。
	// この動作はconfig.Managerのテストで確認済み。
	// ここではparseFlags経由でConfigPath=""が許容されることを確認。

	args := []string{"serve"}
	opts, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.ConfigPath != "" {
		t.Errorf("expected empty config path, got %s", opts.ConfigPath)
	}
}

// TestStoreInitializeAndClose はStore初期化・Close呼び出しをテスト
func TestStoreInitializeAndClose(t *testing.T) {
	// NOTE: initComponents内でst.Initialize()とdefer cleanup(st.Close())が呼ばれる。
	// この動作は統合テストで確認する。

	t.Skip("統合テスト: 実装完了後にモックStoreで確認")

	// 以下は統合テスト例（実装時に有効化）
	// mockStore := &MockStore{
	//     InitializeCalled: false,
	//     CloseCalled: false,
	// }
	//
	// // initComponentsを呼び出してInitialize/Closeが呼ばれることを確認
}

// TestHTTPCORSConfig はHTTP CORS設定の動作確認をテスト
func TestHTTPCORSConfig(t *testing.T) {
	// NOTE: 現時点では設定ファイルからCORSOriginsを読み込む実装はない。
	// 将来的にmodel.ConfigにCORSOrigins追加後、この動作を確認する。

	t.Skip("将来実装: model.ConfigにCORSOrigins追加後に有効化")

	// 以下は統合テスト例（実装時に有効化）
	// cfg := &model.Config{
	//     CORSOrigins: []string{"http://localhost:3000"},
	// }
	//
	// // runServeでHTTP transportに反映されることを確認
}

// TestRun_WithMockComponents はrun関数の基本動作をテスト
func TestRun_WithMockComponents(t *testing.T) {
	// NOTE: run関数は実際のコンポーネント初期化を行うため、
	// 完全なモック化が必要。ここではparseFlags/setupSignalHandlerの
	// 組み合わせ動作のみを確認する簡易テスト。

	t.Skip("統合テスト: 全依存関係のモック化後に有効化")

	// 以下は統合テスト例（実装時に有効化）
	// // 不正な引数でエラーが返ることを確認
	// err := run([]string{"unknown"})
	// if err == nil {
	//     t.Fatal("expected error, got nil")
	// }
}

// TestMain_ErrorHandling はmain関数のエラー処理をテスト
func TestMain_ErrorHandling(t *testing.T) {
	// NOTE: main関数はos.Exit(1)を呼ぶため、直接テストできない。
	// run関数のエラー処理を確認することで間接的にテストする。

	// 不正な引数でエラーが返ることを確認
	err := run([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "usage: mcp-memory serve [options]"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// ベンチマーク: parseFlags
func BenchmarkParseFlags(b *testing.B) {
	args := []string{"serve", "--transport", "http", "--port", "9999"}
	for i := 0; i < b.N; i++ {
		_, _ = parseFlags(args)
	}
}

// NOTE: Exampleテストは公開関数が必要だが、parseFlagsは非公開なので
// ここでは通常のテストケースとして実装済み

// テーブル駆動テスト: parseFlags バリデーション
func TestParseFlags_Validation_TableDriven(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid stdio",
			args:        []string{"serve", "--transport", "stdio"},
			expectError: false,
		},
		{
			name:        "valid http",
			args:        []string{"serve", "--transport", "http"},
			expectError: false,
		},
		{
			name:        "invalid transport",
			args:        []string{"serve", "--transport", "grpc"},
			expectError: true,
			errorMsg:    "invalid transport: grpc (must be stdio or http)",
		},
		{
			name:        "port too low",
			args:        []string{"serve", "--port", "0"},
			expectError: true,
			errorMsg:    "invalid port: 0 (must be 1-65535)",
		},
		{
			name:        "port too high",
			args:        []string{"serve", "--port", "99999"},
			expectError: true,
			errorMsg:    "invalid port: 99999 (must be 1-65535)",
		},
		{
			name:        "no subcommand",
			args:        []string{},
			expectError: true,
			errorMsg:    "usage: mcp-memory serve [options]",
		},
		{
			name:        "wrong subcommand",
			args:        []string{"start"},
			expectError: true,
			errorMsg:    "usage: mcp-memory serve [options]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseFlags(tc.args)
			if tc.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tc.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// サブテスト: setupSignalHandler複数シグナル
func TestSetupSignalHandler_MultipleSignals(t *testing.T) {
	tests := []struct {
		name   string
		signal os.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := setupSignalHandler()
			defer cancel()

			// シグナル送信
			go func() {
				time.Sleep(10 * time.Millisecond)
				p, _ := os.FindProcess(os.Getpid())
				p.Signal(tt.signal)
			}()

			// contextがキャンセルされるまで待機
			select {
			case <-ctx.Done():
				// 成功
			case <-time.After(1 * time.Second):
				t.Fatalf("context was not cancelled after %s", tt.name)
			}
		})
	}
}

// ヘルパー関数テスト用のモック
type mockWriter struct {
	buf bytes.Buffer
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

// 統合テスト用のヘルパー（将来実装）
func setupTestEnvironment(t *testing.T) (cleanup func()) {
	// テスト用の一時設定ファイル作成などを行う
	return func() {
		// クリーンアップ処理
	}
}
