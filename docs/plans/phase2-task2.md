# Phase 2 Task 2: データモデル定義 (internal/model)

## 1. 概要

mcp-memory サーバーで使用するデータモデルを定義する。
以下の4カテゴリの構造体を実装する:

1. **Note**: メモリノートのデータモデル
2. **GlobalConfig**: グローバル設定KVストア用モデル
3. **Config**: サーバー設定モデル
4. **JSON-RPC 2.0**: リクエスト/レスポンス/エラー構造体

## 2. 作成するファイル一覧

```
internal/model/
├── model.go         # 既存ファイル（パッケージ宣言のみ → 削除または統合）
├── note.go          # Note構造体とバリデーション
├── global_config.go # GlobalConfig構造体
├── config.go        # Config関連構造体
├── jsonrpc.go       # JSON-RPC 2.0 構造体
└── model_test.go    # 全構造体のテスト
```

## 3. 各構造体の詳細設計

### 3.1 Note構造体 (note.go)

```go
// Note はメモリノートを表す（内部データモデル）
// 注: レスポンス用のDTO（namespaceを含む）はjsonrpc層で別途定義する
type Note struct {
    ID        string         `json:"id"`                  // UUID形式
    ProjectID string         `json:"projectId"`           // 正規化済みパス
    GroupID   string         `json:"groupId"`             // 英数字、-、_のみ。"global"は予約値
    Title     *string        `json:"title"`               // nullable
    Text      string         `json:"text"`                // 必須
    Tags      []string       `json:"tags"`                // 空配列可
    Source    *string        `json:"source"`              // nullable
    CreatedAt *string        `json:"createdAt"`           // ISO8601 UTC形式、nullable（nullならサーバー側で現在時刻設定）
    Metadata  map[string]any `json:"metadata,omitempty"`  // nullable（JSON null許容）、省略可
}
```

**バリデーションルール**:
- `ID`: UUID形式（空文字不可）
- `ProjectID`: 空文字不可
- `GroupID`: 正規表現 `^[a-zA-Z0-9_-]+$` にマッチすること
- `Text`: 空文字不可
- `CreatedAt`: nullableだが、値がある場合はISO8601 UTC形式（例: "2024-01-15T10:30:00Z"）
- `Metadata`: null許容。JSONの`null`は`nil`として受理する

**バリデーション関数**:
```go
func (n *Note) Validate() error
func ValidateGroupID(groupID string) error
```

### 3.2 GlobalConfig構造体 (global_config.go)

```go
// GlobalConfig はプロジェクト単位のグローバル設定を表す
type GlobalConfig struct {
    ID        string  `json:"id"`        // UUID形式
    ProjectID string  `json:"projectId"` // 正規化済みパス
    Key       string  `json:"key"`       // "global."プレフィックス必須
    Value     any     `json:"value"`     // 任意のJSON値
    UpdatedAt *string `json:"updatedAt"` // ISO8601 UTC形式、nullable（nullならサーバー側で現在時刻設定）
}
```

**バリデーションルール**:
- `Key`: "global."プレフィックス必須
- `UpdatedAt`: nullableだが、値がある場合はISO8601 UTC形式

**標準キー定数**:
```go
const (
    GlobalKeyEmbedderProvider = "global.memory.embedder.provider"
    GlobalKeyEmbedderModel    = "global.memory.embedder.model"
    GlobalKeyGroupDefaults    = "global.memory.groupDefaults"
    GlobalKeyProjectConventions = "global.project.conventions"
)
```

**バリデーション関数**:
```go
func (g *GlobalConfig) Validate() error
func ValidateGlobalKey(key string) error
```

### 3.3 Config構造体 (config.go)

```go
// Config はサーバー全体の設定を表す
type Config struct {
    TransportDefaults TransportDefaults `json:"transportDefaults"`
    Embedder          EmbedderConfig    `json:"embedder"`
    Store             StoreConfig       `json:"store"`
    Paths             PathsConfig       `json:"paths"`
}

// TransportDefaults はtransportのデフォルト設定
type TransportDefaults struct {
    DefaultTransport string `json:"defaultTransport"` // "stdio" | "http"
}

// EmbedderConfig はembedder設定
type EmbedderConfig struct {
    Provider string  `json:"provider"`          // "openai" | "ollama" | "local"
    Model    string  `json:"model"`             // モデル名
    Dim      int     `json:"dim"`               // ベクトル次元（0は未設定）
    BaseURL  *string `json:"baseUrl,omitempty"` // nullable、省略可
    APIKey   *string `json:"apiKey,omitempty"`  // nullable、省略可（セキュリティ注意）
}

// StoreConfig はvector store設定
type StoreConfig struct {
    Type string  `json:"type"`           // "chroma" | "sqlite" | "qdrant" | "faiss"
    Path *string `json:"path,omitempty"` // nullable（SQLite用）
    URL  *string `json:"url,omitempty"`  // nullable（Chroma/Qdrant用）
}

// PathsConfig はファイルパス設定
type PathsConfig struct {
    ConfigPath string `json:"configPath"` // 設定ファイルパス
    DataDir    string `json:"dataDir"`    // データディレクトリ
}
```

**定数定義**:
```go
const (
    TransportStdio = "stdio"
    TransportHTTP  = "http"

    ProviderOpenAI = "openai"
    ProviderOllama = "ollama"
    ProviderLocal  = "local"

    StoreTypeChroma = "chroma"
    StoreTypeSQLite = "sqlite"
    StoreTypeQdrant = "qdrant"
    StoreTypeFAISS  = "faiss"
)
```

### 3.4 JSON-RPC 2.0構造体 (jsonrpc.go)

```go
// Request はJSON-RPC 2.0リクエスト
type Request struct {
    JSONRPC string `json:"jsonrpc"`        // 常に "2.0"
    ID      any    `json:"id"`             // string | number | null
    Method  string `json:"method"`         // メソッド名
    Params  any    `json:"params,omitempty"` // 任意のオブジェクト、省略可
}

// Response はJSON-RPC 2.0レスポンス（成功時）
type Response struct {
    JSONRPC string `json:"jsonrpc"`       // 常に "2.0"
    ID      any    `json:"id"`            // リクエストのIDと同一
    Result  any    `json:"result"`        // 結果オブジェクト
}

// ErrorResponse はJSON-RPC 2.0エラーレスポンス
type ErrorResponse struct {
    JSONRPC string     `json:"jsonrpc"`   // 常に "2.0"
    ID      any        `json:"id"`        // リクエストのIDと同一（パース失敗時はnull）
    Error   RPCError   `json:"error"`     // エラーオブジェクト
}

// RPCError はJSON-RPC 2.0エラーオブジェクト
type RPCError struct {
    Code    int    `json:"code"`           // エラーコード
    Message string `json:"message"`        // エラーメッセージ
    Data    any    `json:"data,omitempty"` // 追加情報、省略可
}
```

**標準エラーコード定数**:
```go
const (
    // JSON-RPC 2.0 標準エラーコード
    ErrCodeParseError     = -32700 // Invalid JSON
    ErrCodeInvalidRequest = -32600 // Invalid Request
    ErrCodeMethodNotFound = -32601 // Method not found
    ErrCodeInvalidParams  = -32602 // Invalid params
    ErrCodeInternalError  = -32603 // Internal error

    // カスタムエラーコード（-32000 〜 -32099 はサーバー予約）
    ErrCodeAPIKeyMissing     = -32001 // API key not configured
    ErrCodeInvalidKeyPrefix  = -32002 // Invalid key prefix (global.* required)
    ErrCodeNotFound          = -32003 // Resource not found
    ErrCodeProviderError     = -32004 // Embedding provider error
)
```

**ヘルパー関数**:
```go
// NewResponse は成功レスポンスを生成
func NewResponse(id any, result any) *Response

// NewErrorResponse はエラーレスポンスを生成
func NewErrorResponse(id any, code int, message string, data any) *ErrorResponse

// 標準エラー生成ヘルパー
func NewParseError(data any) *ErrorResponse
func NewInvalidRequest(id any, data any) *ErrorResponse
func NewMethodNotFound(id any, method string) *ErrorResponse
func NewInvalidParams(id any, message string) *ErrorResponse
func NewInternalError(id any, message string) *ErrorResponse
```

## 4. テストケース一覧

### 4.1 Note テスト

| テストケース | 説明 |
|------------|------|
| TestNote_Validate_Valid | 有効なNoteがバリデーションを通過 |
| TestNote_Validate_EmptyID | 空のIDでエラー |
| TestNote_Validate_EmptyProjectID | 空のProjectIDでエラー |
| TestNote_Validate_EmptyText | 空のTextでエラー |
| TestNote_Validate_InvalidGroupID | 不正なGroupID（特殊文字含む）でエラー |
| TestValidateGroupID_Valid | 有効なGroupID（英数字、-、_）が通過 |
| TestValidateGroupID_GlobalReserved | "global"が予約値として受理される |
| TestValidateGroupID_Invalid | スペース、日本語、その他特殊文字でエラー |

### 4.2 GlobalConfig テスト

| テストケース | 説明 |
|------------|------|
| TestGlobalConfig_Validate_Valid | 有効なGlobalConfigがバリデーションを通過 |
| TestGlobalConfig_Validate_InvalidKey | "global."プレフィックスなしでエラー |
| TestValidateGlobalKey_Valid | 有効なキー（標準キー含む）が通過 |
| TestValidateGlobalKey_Invalid | プレフィックスなし/不正プレフィックスでエラー |

### 4.3 Config テスト

| テストケース | 説明 |
|------------|------|
| TestConfig_JSONMarshal | Configが正しくJSONシリアライズされる |
| TestConfig_JSONUnmarshal | JSONからConfigが正しくデシリアライズされる |
| TestEmbedderConfig_OmitEmpty | 空のBaseURL/APIKeyがJSON出力で省略される |

### 4.4 JSON-RPC テスト

| テストケース | 説明 |
|------------|------|
| TestRequest_JSONMarshal | Requestが正しくJSONシリアライズされる |
| TestRequest_JSONUnmarshal | JSONからRequestが正しくデシリアライズされる |
| TestResponse_JSONMarshal | Responseが正しくJSONシリアライズされる |
| TestErrorResponse_JSONMarshal | ErrorResponseが正しくJSONシリアライズされる |
| TestErrorResponse_ParseError_IDNull | パース失敗時のErrorResponseでIDがnullになる |
| TestNewResponse | NewResponseが正しいレスポンスを生成 |
| TestNewErrorResponse | NewErrorResponseが正しいエラーレスポンスを生成 |
| TestNewParseError | ParseErrorが-32700コードを持ち、IDがnullになる |
| TestNewMethodNotFound | MethodNotFoundが-32601コードを持つ |
| TestNewInvalidParams | InvalidParamsが-32602コードを持つ |
| TestNewInternalError | InternalErrorが-32603コードを持つ |

## 5. 実装の注意点

### 5.1 nullable フィールドの扱い

- Goでは`*string`でnullableを表現
- JSONの`null`は`nil`ポインタに対応
- `omitempty`タグはnilの場合にフィールドを省略

### 5.2 時刻形式

- 全ての時刻はUTCで統一
- ISO8601形式: `2024-01-15T10:30:00Z`
- Goでのフォーマット: `time.RFC3339`

```go
const TimeFormat = time.RFC3339  // "2006-01-02T15:04:05Z07:00"
```

### 5.3 GroupID バリデーション

正規表現: `^[a-zA-Z0-9_-]+$`
- 英数字（大小文字区別）
- ハイフン `-`
- アンダースコア `_`
- "global" は予約値だが、バリデーションでは特別扱いしない（正規表現を通過すればOK）

### 5.4 JSON-RPC IDの型

JSON-RPC 2.0 仕様では ID は `string | number | null` のいずれか。
Goでは `any` 型で受け取り、nilチェックで判定する。

### 5.5 セキュリティ考慮

- `APIKey` フィールドはログ出力時にマスクすること
- 将来の実装で `String()` メソッドを定義する際は要注意

### 5.6 仕様との整合性

本設計は `requirements/embedded_spec.md` の以下のセクションに基づく:
- #6 データモデル
- #7 global設定
- #8 JSON-RPC methods
- #10 JSON-RPC 2.0の厳密さ

## 6. 設計方針: モデルとDTOの分離

`internal/model` パッケージは**内部データモデル**を定義する。
API入出力用のDTO（Data Transfer Object）は別のパッケージで定義する:

- **内部モデル** (`internal/model`): 永続化・ビジネスロジック用
  - `Note`, `GlobalConfig`, `Config`
  - `namespace` は持たない（サービス層で動的に算出）

- **API DTO** (`internal/jsonrpc` または `internal/dto`):
  - `NoteResponse`: `Note` + `Namespace` + `Score`（search用）
  - `SearchResponse`: `Namespace` + `Results`
  - `ListRecentResponse`: `Namespace` + `Items`

この分離により:
1. 内部モデルはシンプルに保てる
2. API仕様変更時の影響範囲を限定できる
3. 将来の拡張に柔軟に対応できる

## 7. 完了条件

```bash
go test ./internal/model/... -v
```

上記コマンドが全てPASSすること。
