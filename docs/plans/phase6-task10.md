# Phase 6 Task 10: CLIエントリポイント (cmd/mcp-memory)

## 概要

mcp-memoryサーバーのCLIエントリポイントを実装する。`serve`コマンドで stdio/HTTP transportを起動し、シグナルハンドリングによるgraceful shutdownを提供する。

## 要件（TODO.mdより）

- serveコマンド実装
- --transport オプション（stdio/http）
- --host, --port オプション（HTTP用）
- -ldflags でデフォルトtransport切替対応
  - 例: `go build -ldflags "-X main.defaultTransport=http"`
- シグナルハンドリング（SIGINT/SIGTERM）

## 完了条件

- `go run ./cmd/mcp-memory serve` でstdio起動
- `go run ./cmd/mcp-memory serve --transport http --port 8765` でHTTP起動

## ディレクトリ構成

```
cmd/mcp-memory/
├── main.go          # エントリポイント、flagパース、serve実行
└── main_test.go     # テスト
```

## 設計

### 1. ビルド時変数（-ldflags対応）

```go
// ビルド時に -ldflags で変更可能な変数
var (
    // デフォルトtransport（"stdio" or "http"）
    // go build -ldflags "-X main.defaultTransport=http" で変更可能
    defaultTransport = "stdio"

    // バージョン情報（オプション）
    version = "dev"
)
```

### 2. コマンドライン引数

```
mcp-memory serve [options]

Options:
  --transport, -t    Transport type: stdio, http (default: stdio or -ldflags value)
  --host             HTTP host (default: 127.0.0.1)
  --port, -p         HTTP port (default: 8765)
  --config, -c       Config file path (default: ~/.local-mcp-memory/config.json)
  --help, -h         Show help message
  --version, -v      Show version
```

### 3. 処理フロー

```
1. flagパース
2. 設定ファイルロード（config.Manager）
3. 依存コンポーネント初期化
   - Embedder（OpenAI/Ollama/Local）
   - Store（Chroma）
   - Services（NoteService, ConfigService, GlobalService）
   - JSON-RPC Handler
4. transport選択・起動
   - stdio: stdio.Server.Run(ctx)
   - http: http.Server.Run(ctx)
5. シグナルハンドリング
   - SIGINT/SIGTERM → contextキャンセル → graceful shutdown
```

### 4. シグナルハンドリング

```go
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
```

### 5. 依存コンポーネント初期化

```go
// initComponents は各コンポーネントを初期化する
func initComponents(cfg *model.Config) (*jsonrpc.Handler, error) {
    // 1. Embedder初期化
    embedder, err := embedder.New(cfg.Embedder)
    if err != nil {
        return nil, fmt.Errorf("failed to create embedder: %w", err)
    }

    // 2. Store初期化
    store, err := store.New(cfg.Store)
    if err != nil {
        return nil, fmt.Errorf("failed to create store: %w", err)
    }

    // 3. Services初期化
    noteService := service.NewNoteService(store, embedder, configManager)
    configService := service.NewConfigService(configManager)
    globalService := service.NewGlobalService(store, configManager)

    // 4. JSON-RPC Handler初期化
    handler := jsonrpc.New(noteService, configService, globalService)

    return handler, nil
}
```

## 関数/構造体シグネチャ

```go
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

// ビルド時変数
var (
    defaultTransport = "stdio"
    version          = "dev"
)

// CLI引数
type Options struct {
    Transport  string
    Host       string
    Port       int
    ConfigPath string
}

// main はエントリポイント
func main()

// run は実際の処理を行う（テスト容易性のため分離）
func run(args []string) error

// parseFlags は引数をパースしてOptionsを返す
func parseFlags(args []string) (*Options, error)

// setupSignalHandler はシグナルハンドラーを設定
func setupSignalHandler() (context.Context, context.CancelFunc)

// runServe はserveコマンドを実行
func runServe(ctx context.Context, opts *Options) error

// initComponents は依存コンポーネントを初期化
func initComponents(cfg *model.Config, configManager *config.Manager) (*jsonrpc.Handler, error)
```

## テストケース

### main_test.go

#### 正常系

1. **デフォルトオプション解析**
   - 引数なしでserve実行時、defaultTransportが使用されること
   - hostは127.0.0.1、portは8765がデフォルトであること

2. **transport=stdio オプション**
   - `--transport stdio` でstdio transportが選択されること

3. **transport=http オプション**
   - `--transport http` でHTTP transportが選択されること

4. **--host, --port オプション（HTTP用）**
   - `--host 0.0.0.0 --port 9999` で指定値が使用されること

5. **短縮オプション**
   - `-t http -p 9999` で正しくパースされること

6. **config指定**
   - `--config /path/to/config.json` で指定パスが使用されること

#### 異常系

7. **不正なtransport**
   - `--transport unknown` でエラーが返ること

8. **不正なport（範囲外）**
   - `--port 0` や `--port 70000` でエラーが返ること

9. **不正なサブコマンド**
   - `mcp-memory unknown` でエラーが返ること

#### シグナルハンドリング

10. **SIGINT受信**
    - SIGINTでcontextがキャンセルされ、サーバーが正常終了すること

11. **SIGTERM受信**
    - SIGTERMでcontextがキャンセルされ、サーバーが正常終了すること

#### 統合テスト（短縮版）

12. **stdio起動・終了**
    - stdin/stdoutをモックしてserve起動
    - EOFで正常終了すること

13. **HTTP起動・終了**
    - HTTP transport起動後、contextキャンセルで正常終了すること

## 依存関係

- internal/config: 設定管理
- internal/model: データモデル
- internal/embedder: Embedding provider
- internal/store: Vector store
- internal/service: ビジネスロジック
- internal/jsonrpc: JSON-RPCハンドラー
- internal/transport/stdio: stdio transport
- internal/transport/http: HTTP transport

## 実装順序

1. ビルド時変数定義（defaultTransport, version）
2. Options構造体とparseFlags関数
3. setupSignalHandler関数
4. initComponents関数
5. runServe関数（transport分岐）
6. main関数とrun関数
7. テスト実装
8. README更新

## コード例

### main.go（概要）

```go
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
    "github.com/brbranch/embedding_mcp/internal/service"
    "github.com/brbranch/embedding_mcp/internal/store"
    "github.com/brbranch/embedding_mcp/internal/transport/http"
    "github.com/brbranch/embedding_mcp/internal/transport/stdio"
)

var (
    defaultTransport = "stdio"
    version          = "dev"
)

func main() {
    if err := run(os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run(args []string) error {
    opts, err := parseFlags(args)
    if err != nil {
        return err
    }

    ctx, cancel := setupSignalHandler()
    defer cancel()

    return runServe(ctx, opts)
}

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
    handler, cleanup, err := initComponents(cfg, configManager)
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
        server := http.New(handler, http.Config{
            Addr: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
        })
        return server.Run(ctx)
    default:
        return fmt.Errorf("unknown transport: %s", opts.Transport)
    }
}

func initComponents(cfg *model.Config, configManager *config.Manager) (*jsonrpc.Handler, func(), error) {
    // Embedder
    emb, err := embedder.New(cfg.Embedder, configManager)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
    }

    // Store
    st, err := store.New(cfg.Store)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create store: %w", err)
    }

    // Services
    noteService := service.NewNoteService(st, emb, configManager)
    configService := service.NewConfigService(configManager, defaultTransport)
    globalService := service.NewGlobalService(st, configManager)

    // Handler
    handler := jsonrpc.New(noteService, configService, globalService)

    cleanup := func() {
        // Store等のクリーンアップ（必要に応じて）
    }

    return handler, cleanup, nil
}
```

## README更新内容

以下の内容をREADMEに追加:

```markdown
## Usage

### stdio Transport (Default)

```bash
# ビルド
go build -o mcp-memory ./cmd/mcp-memory

# 起動
./mcp-memory serve

# または直接実行
go run ./cmd/mcp-memory serve
```

### HTTP Transport

```bash
# デフォルトポート(8765)で起動
go run ./cmd/mcp-memory serve --transport http

# カスタムポートで起動
go run ./cmd/mcp-memory serve --transport http --port 9999

# ホストも指定（セキュリティ上127.0.0.1推奨）
go run ./cmd/mcp-memory serve --transport http --host 127.0.0.1 --port 8765
```

### Build-time Default Transport

```bash
# HTTP をデフォルトにしてビルド
go build -ldflags "-X main.defaultTransport=http" -o mcp-memory ./cmd/mcp-memory

# このバイナリは --transport 指定なしでHTTPで起動
./mcp-memory serve
```

### Options

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| --transport | -t | stdio | Transport type: stdio, http |
| --host | | 127.0.0.1 | HTTP bind host |
| --port | -p | 8765 | HTTP bind port |
| --config | -c | ~/.local-mcp-memory/config.json | Config file path |

### Signal Handling

- `SIGINT` (Ctrl+C): Graceful shutdown
- `SIGTERM`: Graceful shutdown
```

## 備考

- Embedder/Storeの初期化は既存の実装に依存。未実装の場合はstub/mockを使用してテスト可能にする
- HTTP transportではセキュリティ上、デフォルトで127.0.0.1にバインド
- `--version` フラグは将来的に追加可能（現時点ではオプション）
- 設定ファイルが存在しない場合はデフォルト設定で動作

## 実装上の注意点

1. **flag短縮形の扱い**: Go標準のflagパッケージでは短縮形（-t）と長形（--transport）の両方を同一変数にバインドする場合、2つのフラグを定義する必要がある

2. **サブコマンド対応**: 現時点では`serve`のみだが、将来的に`version`、`config`等のサブコマンドを追加可能な構造にする

3. **エラー出力**: エラーはstderrに出力（stdoutはstdio transportで使用するため）

4. **終了コード**: 正常終了は0、エラー終了は1
