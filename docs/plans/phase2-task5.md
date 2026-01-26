# Phase 2 Task 5: Embedder抽象化とOpenAI実装 実装計画

## 1. 概要

`internal/embedder` パッケージは以下の機能を提供する:

1. **Embedder interface定義**: テキストから埋め込みベクトルを生成する抽象インターフェース
2. **OpenAI embedder実装**: OpenAI embeddings API（text-embedding-3-small等）を`net/http`で呼び出し
3. **Ollama/local embedder stub**: NotImplementedエラーを返すスタブ実装（将来拡張用）
4. **初回埋め込み時のdim取得**: API応答からベクトル次元を取得し、設定ファイルへ記録

## 2. 作成するファイル一覧

```
internal/embedder/
├── embedder.go          # Embedder interface定義、エラー定義
├── openai.go            # OpenAI embedder実装
├── openai_test.go       # OpenAIテスト（モック使用）
├── ollama.go            # Ollama embedder stub
├── local.go             # local embedder stub
├── factory.go           # Provider名からEmbedder生成するファクトリ
└── factory_test.go      # ファクトリテスト
```

## 3. Embedder interface詳細設計

### 3.1 Embedder interface (embedder.go)

```go
package embedder

import (
    "context"
    "errors"
)

// Embedder はテキストから埋め込みベクトルを生成するインターフェース
type Embedder interface {
    // Embed はテキストを埋め込みベクトルに変換する
    Embed(ctx context.Context, text string) ([]float32, error)

    // GetDimension はこのEmbedderが生成するベクトルの次元数を返す
    // 初回埋め込み前（dim未確定時）は 0 を返す
    GetDimension() int
}

// DimUpdater は次元数が確定した際に呼び出されるコールバック
type DimUpdater interface {
    UpdateDim(dim int) error
}

// エラー定義
var (
    ErrAPIKeyRequired   = errors.New("api key is required")
    ErrNotImplemented   = errors.New("embedder not implemented")
    ErrAPIRequestFailed = errors.New("API request failed")
    ErrInvalidResponse  = errors.New("invalid API response")
    ErrEmptyEmbedding   = errors.New("empty embedding returned")
    ErrUnknownProvider  = errors.New("unknown embedder provider")
)

// APIError は詳細なAPIエラー情報を保持
type APIError struct {
    StatusCode int
    Message    string
}

func (e *APIError) Error() string
func (e *APIError) Is(target error) bool  // ErrAPIRequestFailed との比較用
```

## 4. OpenAI Embedder実装

### 4.1 OpenAI API仕様

**エンドポイント**: `https://api.openai.com/v1/embeddings`

**リクエスト**:
```json
{
    "model": "text-embedding-3-small",
    "input": "テキスト",
    "encoding_format": "float"
}
```

**レスポンス**:
```json
{
    "data": [
        {
            "embedding": [0.0023064255, -0.009327292, ...],
            "index": 0
        }
    ],
    "model": "text-embedding-3-small"
}
```

### 4.2 OpenAIEmbedder構造体

```go
const (
    DefaultOpenAIBaseURL = "https://api.openai.com/v1"
    DefaultOpenAIModel   = "text-embedding-3-small"
)

type OpenAIEmbedder struct {
    httpClient *http.Client
    baseURL    string
    apiKey     string
    model      string
    dim        int
    dimOnce    sync.Once
    dimUpdater DimUpdater
}

func NewOpenAIEmbedder(apiKey string, opts ...OpenAIOption) (*OpenAIEmbedder, error)
```

### 4.3 オプション

| オプション | 説明 |
|-----------|------|
| `WithBaseURL(url)` | ベースURL設定 |
| `WithModel(model)` | モデル設定 |
| `WithDim(dim)` | 既知のdim設定（設定ファイルから） |
| `WithDimUpdater(updater)` | dim確定時のコールバック |
| `WithHTTPClient(client)` | HTTPクライアント（テスト用） |

### 4.4 エラーハンドリング詳細

| エラー条件 | 処理 |
|-----------|------|
| HTTP 4xx (400, 401, 403, 404) | `&APIError{StatusCode, body}` を返す |
| HTTP 429 (Rate Limit) | `&APIError{429, "rate limit exceeded"}` を返す |
| HTTP 5xx | `&APIError{StatusCode, body}` を返す |
| JSONデコード失敗 | `ErrInvalidResponse` をラップして返す |
| `data` 配列が空 | `ErrEmptyEmbedding` を返す |
| `embedding` 配列が空 | `ErrEmptyEmbedding` を返す |
| context.Canceled | そのままcontext.Canceledを返す |
| context.DeadlineExceeded | そのままcontext.DeadlineExceededを返す |
| DimUpdater.UpdateDim失敗 | ログ出力のみ（embedは正常に返す） |

## 5. Stub実装

### 5.1 Ollama Embedder

```go
type OllamaEmbedder struct {
    baseURL string
    model   string
}

func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder
func (e *OllamaEmbedder) Embed(ctx, text) -> ErrNotImplemented
func (e *OllamaEmbedder) GetDimension() -> 0
```

### 5.2 Local Embedder

```go
type LocalEmbedder struct{}

func NewLocalEmbedder() *LocalEmbedder
func (e *LocalEmbedder) Embed(ctx, text) -> ErrNotImplemented
func (e *LocalEmbedder) GetDimension() -> 0
```

## 6. Factory

### 6.1 シグネチャ

```go
func NewEmbedder(cfg *model.EmbedderConfig, envAPIKey string, dimUpdater DimUpdater) (Embedder, error)
```

### 6.2 APIKey解決優先順位

1. `cfg.APIKey` (設定ファイル由来) が非nil・非空なら使用
2. それ以外は `envAPIKey` (環境変数由来) を使用
3. 両方空ならOpenAI provider時に `ErrAPIKeyRequired`

### 6.3 cfg.BaseURL の適用

- `cfg.BaseURL` が非nil・非空なら `WithBaseURL` オプションで適用
- nilまたは空ならデフォルト (`https://api.openai.com/v1`)

### 6.4 Provider別処理

| Provider | 返り値 |
|----------|--------|
| `openai` | OpenAIEmbedder (cfg.BaseURL, cfg.Model, cfg.Dim を適用) |
| `ollama` | OllamaEmbedder (cfg.BaseURL or default, cfg.Model) |
| `local` | LocalEmbedder |
| その他 | ErrUnknownProvider |

## 7. テストケース一覧

### OpenAI Embedderテスト

| テストケース | 説明 |
|------------|------|
| `TestOpenAIEmbedder_NewEmbedder_APIKeyRequired` | apiKey空でErrAPIKeyRequired |
| `TestOpenAIEmbedder_Embed_Success` | 正常なAPI応答で埋め込みが返る |
| `TestOpenAIEmbedder_Embed_DimUpdatedOnFirstCall` | 初回埋め込み時にdimが更新される |
| `TestOpenAIEmbedder_Embed_DimUpdatedOnlyOnce` | dimは1回だけ更新される |
| `TestOpenAIEmbedder_Embed_DimPreset` | WithDimで事前設定時はUpdaterが呼ばれない |
| `TestOpenAIEmbedder_Embed_DimUpdaterError` | UpdateDim失敗でもembedは成功する |
| `TestOpenAIEmbedder_Embed_APIError_4xx` | 401/403等でAPIErrorが返る |
| `TestOpenAIEmbedder_Embed_APIError_RateLimit` | 429でAPIError(rate limit)が返る |
| `TestOpenAIEmbedder_Embed_APIError_5xx` | 500等でAPIErrorが返る |
| `TestOpenAIEmbedder_Embed_InvalidJSON` | JSONパース失敗でErrInvalidResponse |
| `TestOpenAIEmbedder_Embed_EmptyData` | data配列空でErrEmptyEmbedding |
| `TestOpenAIEmbedder_Embed_EmptyEmbedding` | embedding配列空でErrEmptyEmbedding |
| `TestOpenAIEmbedder_Embed_ContextCanceled` | キャンセル時context.Canceled |
| `TestOpenAIEmbedder_GetDimension_BeforeEmbed` | 埋め込み前は0を返す |
| `TestOpenAIEmbedder_GetDimension_AfterEmbed` | 埋め込み後は正しいdimを返す |
| `TestOpenAIEmbedder_WithBaseURL` | BaseURLオプション適用確認 |
| `TestOpenAIEmbedder_WithModel` | Modelオプション適用確認 |

### Stubテスト

| テストケース | 説明 |
|------------|------|
| `TestOllamaEmbedder_Embed_NotImplemented` | ErrNotImplementedを返す |
| `TestOllamaEmbedder_GetDimension_Zero` | 0を返す |
| `TestLocalEmbedder_Embed_NotImplemented` | ErrNotImplementedを返す |
| `TestLocalEmbedder_GetDimension_Zero` | 0を返す |

### Factoryテスト

| テストケース | 説明 |
|------------|------|
| `TestNewEmbedder_OpenAI` | OpenAI providerでOpenAIEmbedderを返す |
| `TestNewEmbedder_OpenAI_CfgAPIKey` | cfg.APIKey優先で使用される |
| `TestNewEmbedder_OpenAI_EnvAPIKey` | cfg.APIKeyがnilならenvAPIKeyを使用 |
| `TestNewEmbedder_OpenAI_NoAPIKey` | 両方空でErrAPIKeyRequired |
| `TestNewEmbedder_OpenAI_BaseURL` | cfg.BaseURLがOpenAIEmbedderに適用される |
| `TestNewEmbedder_OpenAI_Model` | cfg.ModelがOpenAIEmbedderに適用される |
| `TestNewEmbedder_OpenAI_Dim` | cfg.DimがOpenAIEmbedderに適用される |
| `TestNewEmbedder_Ollama` | Ollama providerでOllamaEmbedderを返す |
| `TestNewEmbedder_Local` | Local providerでLocalEmbedderを返す |
| `TestNewEmbedder_Unknown` | 未知providerでErrUnknownProvider |

## 8. テスト方法

`httptest.Server` を使用してOpenAI APIをモック:

```go
func newMockOpenAIServer(handler http.HandlerFunc) *httptest.Server
func successHandler(embedding []float32) http.HandlerFunc
```

## 9. 依存関係

```
internal/embedder
  ├── internal/model  (EmbedderConfig, Provider定数)
  └── net/http        (OpenAI API呼び出し)
```

## 10. 完了条件

```bash
go test ./internal/embedder/... -v
```

上記コマンドが全てPASSし、以下の動作が確認できること:

1. OpenAI embedderでAPIキー必須チェックが動作
2. 正常なAPI応答で埋め込みベクトルが返る（モックサーバー使用）
3. 初回埋め込み時にdimが取得され、DimUpdaterが呼ばれる
4. Ollama/local stubがErrNotImplementedを返す
5. Factory関数が正しいEmbedder実装を返す
