package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brbranch/embedding_mcp/internal/bootstrap"
	"github.com/brbranch/embedding_mcp/internal/jsonrpc"
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
	var err error

	// 引数なしの場合はserveをデフォルト実行
	if len(os.Args) < 2 {
		err = run([]string{})
	} else {
		switch os.Args[1] {
		case "serve":
			// 既存の run() フローを使用
			err = run(os.Args[1:])
		case "search":
			err = runSearchCmd(os.Args[2:])
		case "version", "-v", "--version":
			printVersion()
			return
		case "help", "-h", "--help":
			printUsage()
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// printUsage prints the usage information
func printUsage() {
	fmt.Println(`mcp-memory - Local MCP Memory Server

Usage:
  mcp-memory <command> [options]

Commands:
  serve     Start the MCP server (stdio or HTTP)
  search    Search notes (oneshot command)
  version   Print version information
  help      Print this help message

Serve Options:
  -t, --transport string   Transport type: stdio, http (default: stdio)
  --host string            HTTP host (default: 127.0.0.1)
  -p, --port int           HTTP port (default: 8765)
  -c, --config string      Config file path

Search Options:
  -p, --project string     Project ID/path (required)
  -g, --group string       Group ID (optional, search all groups if omitted)
  -k, --top-k int          Number of results (default: 5)
  --tags string            Tag filter (comma-separated)
  -f, --format string      Output format: text, json (default: text)
  -c, --config string      Config file path
  --stdin                  Read query from stdin

Examples:
  mcp-memory serve
  mcp-memory serve -t http -p 8080
  mcp-memory search -p /path/to/project "search query"
  mcp-memory search -p ~/project -g global -k 10 "query"
  echo "query" | mcp-memory search -p /path/to/project --stdin`)
}

// printVersion prints the version information
func printVersion() {
	fmt.Printf("mcp-memory version %s\n", version)
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

	// 空配列の場合はserveをデフォルトとして扱う
	// serveサブコマンド確認（引数なしまたは"serve"で始まる場合のみ許可）
	var flagArgs []string
	if len(args) == 0 {
		// 引数なし: デフォルトでserve
		flagArgs = []string{}
	} else if args[0] == "serve" {
		flagArgs = args[1:]
	} else {
		return nil, fmt.Errorf("usage: mcp-memory serve [options]")
	}

	if err := fs.Parse(flagArgs); err != nil {
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
	// bootstrap.Initializeを使用して共通初期化ロジックを実行
	services, cleanup, err := bootstrap.Initialize(ctx, opts.ConfigPath)
	if err != nil {
		return err
	}
	defer cleanup()

	// JSON-RPC Handler初期化
	handler := jsonrpc.New(services.NoteService, services.ConfigService, services.GlobalService, services.GroupService)

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
