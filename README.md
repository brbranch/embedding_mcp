# MCP Memory Server

ローカルで動作する MCP メモリサーバー（Go実装）。会話メモ/仕様/ノートの埋め込み検索基盤。

## 概要

Claude Code から JSON-RPC 2.0 で呼び出せるローカル RAG メモリ基盤を提供します。

## 機能

- 会話メモ/仕様/ノートの保存と検索
- プロジェクト単位・グループ単位でのメモ管理
- OpenAI/Ollama による埋め込み生成
- Chroma/SQLite によるベクトル検索

## ビルド方法

### 前提条件

- Go 1.22 以上

### ビルド

```bash
# 全パッケージのビルド確認
go build ./...

# 実行可能バイナリのビルド
go build ./cmd/mcp-memory
```

### 起動（予定）

```bash
# stdio（デフォルト）
./mcp-memory serve

# HTTP
./mcp-memory serve --transport http --port 8765
```

## データモデル

### Note

メモリノートの基本構造:

```go
type Note struct {
    ID        string         // UUID
    ProjectID string         // 正規化済みパス
    GroupID   string         // 英数字、-、_のみ（"global"は予約値）
    Title     *string        // nullable
    Text      string         // 必須
    Tags      []string       // 空配列可
    Source    *string        // nullable
    CreatedAt *string        // ISO8601 UTC（nullならサーバーが設定）
    Metadata  map[string]any // nullable
}
```

### GlobalConfig

プロジェクト単位のグローバル設定:

```go
type GlobalConfig struct {
    ID        string  // UUID
    ProjectID string  // 正規化済みパス
    Key       string  // "global."プレフィックス必須
    Value     any     // 任意のJSON値
    UpdatedAt *string // ISO8601 UTC
}
```

標準キー:
- `global.memory.embedder.provider`
- `global.memory.embedder.model`
- `global.memory.groupDefaults`
- `global.project.conventions`

## 設定管理

### 設定ファイル

デフォルトの設定ファイルパス: `~/.local-mcp-memory/config.json`

```json
{
  "transportDefaults": {
    "defaultTransport": "stdio"
  },
  "embedder": {
    "provider": "openai",
    "model": "text-embedding-3-small",
    "dim": 0
  },
  "store": {
    "type": "chroma"
  },
  "paths": {
    "configPath": "~/.local-mcp-memory/config.json",
    "dataDir": "~/.local-mcp-memory/data"
  }
}
```

### 環境変数

OpenAI APIキーは環境変数で設定可能（設定ファイルより優先）:

```bash
export OPENAI_API_KEY="sk-..."
```

### projectId正規化

projectIdは以下の順序で正規化されます:

1. `~` をホームディレクトリに展開
2. 絶対パス化
3. シンボリックリンク解決（失敗時は絶対パスまで）

例: `~/project` → `/Users/xxx/project`

### namespace

埋め込みのnamespaceは `{provider}:{model}:{dim}` 形式で自動生成されます。

例: `openai:text-embedding-3-small:1536`

## VectorStore

### Store Interface

ベクトルストアの抽象インターフェース。以下の操作をサポート:

- **Note操作**: AddNote, Get, Update, Delete
- **検索**: Search（ベクトル類似度検索）, ListRecent（最新順取得）
- **GlobalConfig**: UpsertGlobal, GetGlobal

### 検索フィルタ

```go
SearchOptions{
    ProjectID: "...",      // 必須
    GroupID:   nil,        // nilの場合は全group対象
    TopK:      5,          // 取得件数
    Tags:      []string{}, // AND検索（大小文字区別）
    Since:     &time,      // since <= createdAt
    Until:     &time,      // createdAt < until
}
```

### スコア

検索結果のスコアは0-1に正規化（1が最も類似）。

### 実装

- **MemoryStore**: テスト用インメモリ実装
- **ChromaStore**: Chroma連携（実装予定）

## Embedder

### Embedder Interface

テキストから埋め込みベクトルを生成する抽象インターフェース:

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    GetDimension() int  // 未確定時は0
}
```

### DimUpdater

初回埋め込み時に次元数を通知するコールバック:

```go
type DimUpdater interface {
    UpdateDim(dim int) error
}
```

### 実装

- **OpenAIEmbedder**: OpenAI Embeddings API (`text-embedding-3-small`等)
- **OllamaEmbedder**: Ollama連携（実装予定）
- **LocalEmbedder**: ローカルモデル（実装予定）

### Factory

```go
emb, err := embedder.NewEmbedder(cfg, apiKey, dimUpdater)
```

APIキー解決優先順位:
1. `cfg.APIKey` (設定ファイル)
2. `apiKey` パラメータ (環境変数)

### エラー

| エラー | 説明 |
|--------|------|
| `ErrAPIKeyRequired` | APIキー未設定 |
| `ErrNotImplemented` | 未実装プロバイダ |
| `ErrAPIRequestFailed` | APIリクエスト失敗 |
| `ErrInvalidResponse` | 不正なAPIレスポンス |
| `ErrEmptyEmbedding` | 空の埋め込み |
| `ErrUnknownProvider` | 未知のプロバイダ |

## 開発状況

現在開発中です。

---

**NOTE**: 各タスク完了時に、該当機能の動作確認方法をこのREADMEに追記します。
