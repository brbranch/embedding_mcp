# Phase 5 Task 8: stdio transport (internal/transport/stdio)

## 概要
標準入出力（stdin/stdout）を使用したJSON-RPC 2.0トランスポート層を実装。NDJSON形式で1行1リクエスト/レスポンスを処理する。

## 要件（TODO.mdより）
- NDJSON形式の入出力
  - 1リクエスト = 1行を厳守（改行で区切る）
  - JSON内のtext等に含まれる改行は `\n` でエスケープ
  - 複数行にまたがるJSONは不可
- graceful shutdown（SIGINT/SIGTERM対応）

## 完了条件
stdio経由でJSON-RPCリクエストを送り、正しいレスポンスが返ること

## ディレクトリ構成

```
internal/transport/stdio/
├── stdio.go       # Server構造体とRunメソッド
└── stdio_test.go  # テスト
```

## 設計

### Server構造体

```go
// Server はstdio JSON-RPCサーバー
type Server struct {
    handler *jsonrpc.Handler
    reader  io.Reader  // デフォルト: os.Stdin
    writer  io.Writer  // デフォルト: os.Stdout
}

// Option はサーバーオプション
type Option func(*Server)

// WithReader はreaderを設定（テスト用）
func WithReader(r io.Reader) Option

// WithWriter はwriterを設定（テスト用）
func WithWriter(w io.Writer) Option

// New は新しいServerを生成
func New(handler *jsonrpc.Handler, opts ...Option) *Server

// Run はサーバーを起動し、contextがキャンセルされるまで実行
func (s *Server) Run(ctx context.Context) error
```

### NDJSON仕様
- 入力: stdinから1行ずつ読み取り（bufio.Scanner使用）
- 出力: レスポンスを1行JSON + 改行で出力
- JSON内の改行は `\n` エスケープ（標準のJSONエンコーダが自動処理）
- **バッファサイズ**: bufio.Scannerのデフォルト64KB制限を1MBに拡張（長文textに対応）

### 処理フロー
1. bufio.Scannerで1行読み取り
2. 空行はスキップ
3. handler.Handle(ctx, bytes)呼び出し
4. stdoutに書き込み + 改行

### Graceful Shutdown
- contextのキャンセルを検知してループ終了
- EOFでも正常終了（nil返却）
- **注**: SIGINT/SIGTERM対応はTask 10（CLIエントリポイント）で実装。本パッケージはcontextのキャンセルを受けてループ終了するのみ

### エラーハンドリング・戻り値
| 状況 | 戻り値 | 理由 |
|------|--------|------|
| EOF（stdin終了） | `nil` | 正常終了 |
| contextキャンセル | `ctx.Err()` | キャンセル理由を伝播 |
| Scanner読み取りエラー | `scanner.Err()` | 読み取り失敗 |
| stdout書き込みエラー | エラー | 書き込み失敗 |
| JSON解析エラー | 継続（エラーレスポンス出力） | handler側でParseError返却 |

## 関数シグネチャ

```go
package stdio

import (
    "context"
    "io"

    "github.com/brbranch/embedding_mcp/internal/jsonrpc"
)

type Server struct {
    handler *jsonrpc.Handler
    reader  io.Reader
    writer  io.Writer
}

type Option func(*Server)

func WithReader(r io.Reader) Option
func WithWriter(w io.Writer) Option

func New(handler *jsonrpc.Handler, opts ...Option) *Server
func (s *Server) Run(ctx context.Context) error
```

## テストケース

### 正常系
1. 単一リクエスト/レスポンス
   - 入力: `{"jsonrpc":"2.0","id":1,"method":"memory.get_config"}`
   - 期待: 正しいJSON-RPCレスポンスが1行で返る

2. 複数リクエスト（連続処理）
   - 入力: 2行の有効なリクエスト
   - 期待: 2行のレスポンスが返る

3. 空行処理
   - 入力: 空行を含むリクエスト
   - 期待: 空行はスキップされ、有効なリクエストのみ処理

4. 改行を含むtext
   - 入力: textフィールドに改行を含むadd_noteリクエスト
   - 期待: JSON内の改行は `\n` でエスケープされ、1行で出力

### 異常系
5. 不正JSON
   - 入力: `{invalid json}`
   - 期待: JSON-RPC ParseErrorレスポンス

6. 不正メソッド
   - 入力: 存在しないメソッド
   - 期待: JSON-RPC MethodNotFoundレスポンス

### シャットダウン
7. コンテキストキャンセル
   - 入力: contextがキャンセルされた場合
   - 期待: Runがctx.Err()で終了

8. EOF
   - 入力: stdinがEOFに達した場合
   - 期待: Runがnilで正常終了

### バッファ制限
9. 大きなJSON（1MB未満）
   - 入力: 長文textを含むリクエスト
   - 期待: 正常に処理される

10. 巨大なJSON（1MB超過）
    - 入力: 1MBを超えるリクエスト
    - 期待: Scanner読み取りエラー

## 依存関係
- internal/jsonrpc: Handler

## 実装順序
1. Server構造体とNew関数
2. Run関数の基本ループ
3. NDJSON読み取り・書き込み
4. コンテキストキャンセル対応
5. テスト実装

## 備考
- SIGINT/SIGTERM対応はTask 10（cmd/mcp-memory CLIエントリポイント）で実装予定
- このパッケージはcontextのキャンセルを受けてループ終了するのみ

## レビュー対応履歴
- 2026-01-26: bufio.Scanner 64KB制限を1MBに拡張する方針を追記
- 2026-01-26: contextキャンセル時の戻り値をctx.Err()に統一（戻り値表を追加）
- 2026-01-26: graceful shutdownがTask 10の範囲であることを明記
