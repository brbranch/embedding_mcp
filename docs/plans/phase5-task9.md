# Phase 5 Task 9: HTTP transport 実装計画

## 概要

HTTP JSON-RPC transportの実装。`POST /rpc` エンドポイントでJSON-RPC 2.0リクエストを受け付け、CORS設定とgraceful shutdownをサポートする。

## 要件（TODO.mdより）

- POST /rpc エンドポイント
- CORS設定
  - 設定ファイルで許可オリジン指定可能
  - デフォルトはCORS無効（localhost直接アクセスのみ）
- graceful shutdown

完了条件: `curl -X POST http://localhost:8765/rpc` でJSON-RPCが動作すること

## 設計

### 1. 設定モデル追加 (`internal/model/config.go`)

HTTP transport用の設定構造体を追加:

```go
// HTTPConfig はHTTP transport設定
type HTTPConfig struct {
    Host        string   `json:"host"`                  // バインドホスト（デフォルト: "127.0.0.1"）
    Port        int      `json:"port"`                  // ポート（デフォルト: 8765）
    CORSOrigins []string `json:"corsOrigins,omitempty"` // 許可オリジン（空配列=CORS無効）
}

// Config に HTTPConfig を追加
type Config struct {
    // 既存フィールド...
    HTTP HTTPConfig `json:"http"`
}
```

### 2. HTTPサーバー実装 (`internal/transport/http/server.go`)

```go
// Server はHTTP JSON-RPCサーバー
type Server struct {
    handler Handler
    config  *HTTPConfig
    server  *http.Server
}

// Handler インターフェース（stdioと共通）
type Handler interface {
    Handle(ctx context.Context, requestBytes []byte) []byte
}

// New は新しいServerを生成
func New(handler Handler, config *HTTPConfig) *Server

// Run はサーバーを起動
func (s *Server) Run(ctx context.Context) error

// Shutdown はサーバーをシャットダウン
func (s *Server) Shutdown(ctx context.Context) error
```

### 3. エンドポイント設計

#### POST /rpc

- Content-Type: application/json
- リクエストボディ: JSON-RPC 2.0リクエスト
- レスポンス: JSON-RPC 2.0レスポンス

### 4. CORS設定

- `CORSOrigins` が空または未設定の場合: CORSヘッダーを付与しない
- `CORSOrigins` が設定されている場合:
  - `Access-Control-Allow-Origin`: リクエストのOriginが許可リストに含まれていれば、そのOriginを返す
  - `Access-Control-Allow-Methods`: POST, OPTIONS
  - `Access-Control-Allow-Headers`: Content-Type
  - OPTIONSリクエスト（preflight）にも対応

### 5. Graceful Shutdown

- contextキャンセルで `server.Shutdown()` を呼び出し
- SIGINT/SIGTERM シグナルで安全に停止

## ファイル構成

```
internal/transport/http/
├── server.go       # HTTPサーバー本体
├── server_test.go  # テスト
├── cors.go         # CORSミドルウェア
└── cors_test.go    # CORSテスト
```

## テストケース

### server_test.go

1. **基本的なJSON-RPC呼び出し**
   - POST /rpc で有効なJSON-RPCリクエストを送信
   - 正しいレスポンスが返ること

2. **不正なリクエスト**
   - 不正なJSONを送信
   - Parse Errorが返ること

3. **不正なHTTPメソッド**
   - GET /rpc を送信
   - 405 Method Not Allowedが返ること

4. **Graceful Shutdown**
   - contextをキャンセルしてサーバーが停止すること

### cors_test.go

1. **CORS無効（デフォルト）**
   - CORSOrigins未設定時、CORSヘッダーが付与されないこと

2. **CORS有効**
   - 許可オリジンからのリクエストに適切なヘッダーが付与されること

3. **CORS Preflight**
   - OPTIONSリクエストに適切なヘッダーが返ること

4. **許可されていないオリジン**
   - 許可リストにないオリジンからのリクエストにCORSヘッダーが付与されないこと

## 実装順序

1. `internal/model/config.go` に `HTTPConfig` 追加
2. `internal/config/manager.go` のデフォルト設定に `HTTP` 追加
3. `internal/transport/http/server.go` 実装
4. `internal/transport/http/cors.go` 実装
5. テスト作成・実行
6. README更新（動作確認手順を含む）

## README更新内容

### 追加する動作確認手順

```bash
# HTTPサーバー起動
go run ./cmd/mcp-memory serve --transport http --port 8765

# JSON-RPC呼び出し（別ターミナル）
curl -X POST http://localhost:8765/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}'

# レスポンス例
# {"jsonrpc":"2.0","id":1,"result":{"transportDefaults":...}}
```

## 注意事項

- HTTP transportはCLI完成後（Phase 6 Task 10）に本格的に使用可能
- 現時点では単体テストとcurlでの動作確認を行う
- CORS設定はセキュリティ上重要なので、デフォルトは無効
