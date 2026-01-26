# Phase 2 Task 3: 設定管理 (internal/config) 実装計画

## 1. 概要

`internal/config` パッケージは以下の機能を提供する:

1. **設定ファイルの読み書き**: `~/.local-mcp-memory/config.json`
2. **projectId正規化**: `~`展開、絶対パス化、シンボリックリンク解決
3. **namespace生成**: `{provider}:{model}:{dim}` 形式
4. **環境変数によるapiKey上書き**: `OPENAI_API_KEY` 等

## 2. 作成するファイル一覧

```
internal/config/
├── config.go      # 既存ファイル（スケルトン → 実装）
├── manager.go     # 設定ロード/セーブ、Manager構造体
├── path.go        # projectId正規化、パス関連ユーティリティ
├── namespace.go   # namespace生成ロジック
├── env.go         # 環境変数オーバーライド処理
└── config_test.go # 全テスト
```

## 3. 各機能の詳細設計

### 3.1 Manager構造体 (manager.go)

設定の読み書きを管理するマネージャー。

```go
package config

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sync"

    "github.com/brbranch/embedding_mcp/internal/model"
)

// DefaultConfigDir はデフォルトの設定ディレクトリ名
const DefaultConfigDir = ".local-mcp-memory"

// DefaultConfigFile はデフォルトの設定ファイル名
const DefaultConfigFile = "config.json"

// Manager は設定の読み書きを管理する
type Manager struct {
    mu         sync.RWMutex
    config     *model.Config
    configPath string
}

// NewManager は新しいManagerを作成する
// configPathが空文字の場合、デフォルトパス（~/.local-mcp-memory/config.json）を使用
func NewManager(configPath string) (*Manager, error)

// Load は設定ファイルを読み込む
// ファイルが存在しない場合はデフォルト設定を返す
func (m *Manager) Load() (*model.Config, error)

// Save は設定ファイルを保存する
func (m *Manager) Save(config *model.Config) error

// GetConfig は現在の設定を返す（ロード済みの場合）
func (m *Manager) GetConfig() *model.Config

// GetConfigPath は設定ファイルパスを返す
func (m *Manager) GetConfigPath() string

// UpdateEmbedder はembedder設定のみを更新する
// 他のフィールド（store/paths）は変更不可
func (m *Manager) UpdateEmbedder(embedder *model.EmbedderConfig) error

// UpdateDim は埋め込み次元を更新する（初回埋め込み時に使用）
func (m *Manager) UpdateDim(dim int) error
```

**デフォルト設定**:
```go
func DefaultConfig(configPath, dataDir string) *model.Config {
    return &model.Config{
        TransportDefaults: model.TransportDefaults{
            DefaultTransport: model.TransportStdio,
        },
        Embedder: model.EmbedderConfig{
            Provider: model.ProviderOpenAI,
            Model:    "text-embedding-3-small",
            Dim:      0, // 初回埋め込み時に設定
            BaseURL:  nil,
            APIKey:   nil,
        },
        Store: model.StoreConfig{
            Type: model.StoreTypeChroma,
            Path: nil,
            URL:  nil,
        },
        Paths: model.PathsConfig{
            ConfigPath: configPath,
            DataDir:    dataDir,
        },
    }
}
```

### 3.2 projectId正規化 (path.go)

```go
package config

import (
    "os"
    "path/filepath"
    "strings"
)

// CanonicalizeProjectID はprojectIdを正規化する
// 1. "~" をホームディレクトリに展開
// 2. 絶対パス化（filepath.Abs）
// 3. シンボリックリンク解決（filepath.EvalSymlinks）※失敗時はAbsまで
func CanonicalizeProjectID(projectID string) (string, error)

// ExpandTilde は"~"をホームディレクトリに展開する
// "~/" で始まる場合のみ展開し、それ以外はそのまま返す
func ExpandTilde(path string) (string, error)

// GetDefaultConfigPath はデフォルトの設定ファイルパスを返す
// ~/.local-mcp-memory/config.json
func GetDefaultConfigPath() (string, error)

// GetDefaultDataDir はデフォルトのデータディレクトリを返す
// ~/.local-mcp-memory/data
func GetDefaultDataDir() (string, error)

// EnsureDir はディレクトリが存在することを確認し、なければ作成する
func EnsureDir(dir string) error
```

**正規化の詳細ロジック**:

```go
func CanonicalizeProjectID(projectID string) (string, error) {
    // 1. "~" をホームに展開
    expanded, err := ExpandTilde(projectID)
    if err != nil {
        return "", fmt.Errorf("failed to expand tilde: %w", err)
    }

    // 2. 絶対パス化
    abs, err := filepath.Abs(expanded)
    if err != nil {
        return "", fmt.Errorf("failed to get absolute path: %w", err)
    }

    // 3. シンボリックリンク解決（失敗時はAbsまで）
    canonical, err := filepath.EvalSymlinks(abs)
    if err != nil {
        // EvalSymlinks が失敗した場合は Abs の結果を使用
        return abs, nil
    }

    return canonical, nil
}
```

### 3.3 namespace生成 (namespace.go)

```go
package config

import "fmt"

// GenerateNamespace はembedder設定からnamespaceを生成する
// 形式: "{provider}:{model}:{dim}"
// dimが0（未設定）の場合も0をそのまま使用（仕様準拠）
func GenerateNamespace(provider, model string, dim int) string

// ParseNamespace はnamespaceをprovider, model, dimに分解する
// 不正な形式の場合はエラーを返す
// dimは0以上の整数であること（負数の場合はエラー）
func ParseNamespace(namespace string) (provider, model string, dim int, err error)
```

**実装**:

```go
func GenerateNamespace(provider, modelName string, dim int) string {
    return fmt.Sprintf("%s:%s:%d", provider, modelName, dim)
}
```

### 3.4 環境変数オーバーライド (env.go)

```go
package config

import "os"

// 環境変数名の定数
const (
    EnvOpenAIAPIKey = "OPENAI_API_KEY"
)

// ApplyEnvOverrides は環境変数による設定上書きを適用する
// config を直接変更する
func ApplyEnvOverrides(config *model.Config)

// GetOpenAIAPIKey は環境変数からOpenAI APIキーを取得する
// 設定ファイルの値より環境変数を優先
func GetOpenAIAPIKey(config *model.Config) string
```

**実装**:

```go
func ApplyEnvOverrides(config *model.Config) {
    // OpenAI APIキーの環境変数上書き
    if apiKey := os.Getenv(EnvOpenAIAPIKey); apiKey != "" {
        config.Embedder.APIKey = &apiKey
    }
}

func GetOpenAIAPIKey(config *model.Config) string {
    // 環境変数を優先
    if apiKey := os.Getenv(EnvOpenAIAPIKey); apiKey != "" {
        return apiKey
    }
    // 設定ファイルの値
    if config.Embedder.APIKey != nil {
        return *config.Embedder.APIKey
    }
    return ""
}
```

## 4. 関数シグネチャ一覧

### manager.go

| 関数名 | シグネチャ | 説明 |
|--------|----------|------|
| `NewManager` | `(configPath string) (*Manager, error)` | 新しいManagerを作成 |
| `Load` | `(m *Manager) Load() (*model.Config, error)` | 設定ファイルを読み込み |
| `Save` | `(m *Manager) Save(config *model.Config) error` | 設定ファイルを保存 |
| `GetConfig` | `(m *Manager) GetConfig() *model.Config` | 現在の設定を返す |
| `GetConfigPath` | `(m *Manager) GetConfigPath() string` | 設定ファイルパスを返す |
| `UpdateEmbedder` | `(m *Manager) UpdateEmbedder(embedder *model.EmbedderConfig) error` | embedder設定を更新 |
| `UpdateDim` | `(m *Manager) UpdateDim(dim int) error` | 埋め込み次元を更新 |
| `DefaultConfig` | `(configPath, dataDir string) *model.Config` | デフォルト設定を返す |

### path.go

| 関数名 | シグネチャ | 説明 |
|--------|----------|------|
| `CanonicalizeProjectID` | `(projectID string) (string, error)` | projectIdを正規化 |
| `ExpandTilde` | `(path string) (string, error)` | "~"をホームに展開 |
| `GetDefaultConfigPath` | `() (string, error)` | デフォルト設定ファイルパス |
| `GetDefaultDataDir` | `() (string, error)` | デフォルトデータディレクトリ |
| `EnsureDir` | `(dir string) error` | ディレクトリ作成を確認 |

### namespace.go

| 関数名 | シグネチャ | 説明 |
|--------|----------|------|
| `GenerateNamespace` | `(provider, model string, dim int) string` | namespace生成 |
| `ParseNamespace` | `(namespace string) (provider, model string, dim int, err error)` | namespace解析 |

### env.go

| 関数名 | シグネチャ | 説明 |
|--------|----------|------|
| `ApplyEnvOverrides` | `(config *model.Config)` | 環境変数上書き適用 |
| `GetOpenAIAPIKey` | `(config *model.Config) string` | APIキー取得（環境変数優先） |

## 5. テストケース一覧

### 5.1 projectId正規化テスト (path_test.go)

| テストケース | 説明 |
|------------|------|
| `TestCanonicalizeProjectID_TildeExpand` | `~/project` が `/Users/xxx/project` に展開される |
| `TestCanonicalizeProjectID_AbsolutePath` | `./relative` が絶対パスになる |
| `TestCanonicalizeProjectID_Symlink` | シンボリックリンクが解決される |
| `TestCanonicalizeProjectID_SymlinkFail` | 存在しないパスでもAbsまでは返る |
| `TestCanonicalizeProjectID_AlreadyAbsolute` | `/absolute/path` はそのまま |
| `TestExpandTilde_Valid` | `~/path` が正しく展開される |
| `TestExpandTilde_NoTilde` | `~` なしはそのまま |
| `TestExpandTilde_TildeOnly` | `~` のみはホームに展開 |
| `TestExpandTilde_TildeUser` | `~user` は展開しない（非対応） |
| `TestGetDefaultConfigPath` | 正しいパスが返る |
| `TestGetDefaultDataDir` | 正しいパスが返る |

### 5.2 namespace生成テスト (namespace_test.go)

| テストケース | 説明 |
|------------|------|
| `TestGenerateNamespace_WithDim` | `openai:text-embedding-3-small:1536` |
| `TestGenerateNamespace_DimZero` | `openai:text-embedding-3-small:0` |
| `TestGenerateNamespace_Ollama` | `ollama:nomic-embed-text:768` |
| `TestParseNamespace_Valid` | 正しく分解される |
| `TestParseNamespace_InvalidFormat` | 不正形式でエラー |
| `TestParseNamespace_DimZero` | dim=0でも正常にパース |
| `TestParseNamespace_NegativeDim` | 負のdimでエラー |

### 5.3 設定管理テスト (manager_test.go)

| テストケース | 説明 |
|------------|------|
| `TestManager_NewManager_DefaultPath` | 空パスでデフォルトパス使用 |
| `TestManager_NewManager_CustomPath` | カスタムパス指定 |
| `TestManager_Load_NotExist` | ファイルなしでデフォルト設定返却 |
| `TestManager_Load_Exist` | ファイルありで正しく読み込み |
| `TestManager_Load_Invalid` | 不正JSONでエラー |
| `TestManager_Save` | 正しく保存される |
| `TestManager_SaveAndLoad` | 保存後読み込みで同一 |
| `TestManager_UpdateEmbedder` | embedder設定のみ更新 |
| `TestManager_UpdateEmbedder_StorePathsUnchanged` | store/pathsが変更されないことを検証 |
| `TestManager_UpdateDim` | dim更新が反映される |
| `TestDefaultConfig` | デフォルト値が正しい |

### 5.4 環境変数オーバーライドテスト (env_test.go)

| テストケース | 説明 |
|------------|------|
| `TestApplyEnvOverrides_OpenAIAPIKey` | OPENAI_API_KEYで上書き |
| `TestApplyEnvOverrides_NoEnv` | 環境変数なしで設定ファイル値維持 |
| `TestGetOpenAIAPIKey_EnvPriority` | 環境変数が優先 |
| `TestGetOpenAIAPIKey_ConfigFallback` | 環境変数なしで設定ファイル値 |
| `TestGetOpenAIAPIKey_Empty` | どちらもなしで空文字 |

## 6. 実装の注意点

### 6.1 ファイル操作

- 設定ファイル保存時はディレクトリを自動作成（`os.MkdirAll`）
- ファイル書き込みは一時ファイル経由でatomicに（データ破損防止）
- パーミッションは `0644`（ファイル）、`0755`（ディレクトリ）

### 6.2 スレッドセーフティ

- `Manager` は `sync.RWMutex` でスレッドセーフに
- 読み込み操作は `RLock`、書き込み操作は `Lock`

### 6.3 エラーハンドリング

- パス操作のエラーは詳細なコンテキストを付与
- シンボリックリンク解決失敗は非致命的エラー（Absまでで続行）

### 6.4 セキュリティ考慮

- APIキーはログ出力時にマスクすること（将来の実装で）
- 環境変数による上書きは明示的にドキュメント化

### 6.5 テスト環境

- 一時ディレクトリを使用（`t.TempDir()`）
- 環境変数テストは `t.Setenv()` を使用（Go 1.17+）
- シンボリックリンクテストはプラットフォーム依存に注意

## 7. 依存関係

```
internal/config
  └── internal/model  (Config, EmbedderConfig 等の構造体)
```

## 8. 完了条件

```bash
go test ./internal/config/... -v
```

上記コマンドが全てPASSし、以下の動作が確認できること:

1. `~/tmp/demo` を渡して canonical化（絶対パス）される
2. シンボリックリンクが解決される（失敗時はAbsまで）
3. namespace が `{provider}:{model}:{dim}` 形式で生成される
4. 環境変数 `OPENAI_API_KEY` で apiKey が上書きされる
5. 設定ファイルの読み書きが正常に動作する
