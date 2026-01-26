package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/embedder"
	"github.com/brbranch/embedding_mcp/internal/jsonrpc"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/service"
	"github.com/brbranch/embedding_mcp/internal/store"
	"github.com/brbranch/embedding_mcp/internal/transport/http"
	"github.com/brbranch/embedding_mcp/internal/transport/stdio"
)

// ビルド時変数（-ldflags で変更可能）
var (
	defaultTransport = "stdio"
	version          = "dev"
)

// Options はCLI引数オプション
type Options struct {
	Transport  string
	Host       string
	Port       int
	ConfigPath string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run は実際の処理を行う（テスト容易性のため分離）
func run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	ctx, cancel := setupSignalHandler()
	defer cancel()

	return runServe(ctx, opts)
}

// parseFlags は引数をパースしてOptionsを返す
func parseFlags(args []string) (*Options, error) {
	fs := flag.NewFlagSet("mcp-memory", flag.ContinueOnError)

	opts := &Options{}
	fs.StringVar(&opts.Transport, "transport", defaultTransport, "Transport type: stdio, http")
	fs.StringVar(&opts.Transport, "t", defaultTransport, "Transport type (shorthand)")
	fs.StringVar(&opts.Host, "host", "127.0.0.1", "HTTP host")
	fs.IntVar(&opts.Port, "port", 8765, "HTTP port")
	fs.IntVar(&opts.Port, "p", 8765, "HTTP port (shorthand)")
	fs.StringVar(&opts.ConfigPath, "config", "", "Config file path")
	fs.StringVar(&opts.ConfigPath, "c", "", "Config file path (shorthand)")

	// serveサブコマンド確認
	if len(args) == 0 || args[0] != "serve" {
		return nil, fmt.Errorf("usage: mcp-memory serve [options]")
	}

	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}

	// バリデーション
	if opts.Transport != "stdio" && opts.Transport != "http" {
		return nil, fmt.Errorf("invalid transport: %s (must be stdio or http)", opts.Transport)
	}
	if opts.Port < 1 || opts.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d (must be 1-65535)", opts.Port)
	}

	return opts, nil
}

// setupSignalHandler はSIGINT/SIGTERMを受けてcontextをキャンセルする
func setupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	return ctx, cancel
}

// runServe はserveコマンドを実行
func runServe(ctx context.Context, opts *Options) error {
	// 設定ロード
	configManager, err := config.NewManager(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := configManager.GetConfig()

	// コンポーネント初期化
	handler, cleanup, err := initComponents(ctx, cfg, configManager)
	if err != nil {
		return err
	}
	defer cleanup()

	// transport起動
	switch opts.Transport {
	case "stdio":
		server := stdio.New(handler)
		return server.Run(ctx)
	case "http":
		// HTTP設定（CORS含む）
		httpConfig := http.Config{
			Addr: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		}
		// 将来的に設定ファイルからCORSOrigins読み込み予定
		server := http.New(handler, httpConfig)
		return server.Run(ctx)
	default:
		return fmt.Errorf("unknown transport: %s", opts.Transport)
	}
}

// initComponents は依存コンポーネントを初期化
func initComponents(ctx context.Context, cfg *model.Config, configManager *config.Manager) (*jsonrpc.Handler, func(), error) {
	// namespace生成
	namespace := config.GenerateNamespace(cfg.Embedder.Provider, cfg.Embedder.Model, cfg.Embedder.Dim)

	// 1. Embedder初期化（dimUpdater経由でManager更新）
	emb, err := embedder.NewEmbedder(&cfg.Embedder, os.Getenv("OPENAI_API_KEY"), configManager)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 2. Store初期化（タイプに応じて）
	var st store.Store
	switch cfg.Store.Type {
	case "chroma":
		url := "http://localhost:8000"
		if cfg.Store.URL != nil && *cfg.Store.URL != "" {
			url = *cfg.Store.URL
		}
		st, err = store.NewChromaStore(url)
	default:
		st = store.NewMemoryStore() // テスト・フォールバック用
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create store: %w", err)
	}

	// 3. Store初期化（namespace設定）
	if err := st.Initialize(ctx, namespace); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	// 4. Services初期化
	noteService := service.NewNoteService(emb, st, namespace)
	configService := service.NewConfigService(configManager)
	globalService := service.NewGlobalService(st, namespace)

	// 5. JSON-RPC Handler初期化
	handler := jsonrpc.New(noteService, configService, globalService)

	cleanup := func() {
		st.Close()
	}

	return handler, cleanup, nil
}
