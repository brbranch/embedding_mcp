# Phase 2 Task 4: VectorStore抽象化とChroma実装 (internal/store) 実装計画

**注意**: この計画書はChromaStore完全実装を想定していますが、実際にはスタブのみが実装されました。MemoryStoreとSQLiteStoreが実装済みです。

## 1. 概要

`internal/store` パッケージは以下の機能を提供する:

1. **Store interface定義**: Note/GlobalConfigのCRUD操作とベクトル検索の抽象化
2. **Chroma実装**: `github.com/amikos-tech/chroma-go` を使用したベクトルストア（**未実装、スタブのみ**）
3. **namespace分離**: Chromaのcollection単位でnamespace（provider:model:dim）を管理
4. **検索フィルタ**: tags（AND検索）、since/until（時刻範囲）のサポート

## 実装状況

- ✅ **Store interface**: 実装済み (internal/store/store.go, types.go)
- ✅ **MemoryStore**: 完全実装（テスト・開発用）
- ✅ **SQLiteStore**: 完全実装（Phase 11 Task 15で追加、軽量用途向け）
- ❌ **ChromaStore**: スタブのみ（internal/store/chroma.go、全メソッドが未実装エラーを返す）

## 2. 作成するファイル一覧

```
internal/store/
├── store.go           # Store interface定義
├── options.go         # SearchOptions、ListOptions等のオプション型
├── result.go          # SearchResult等の結果型
├── chroma.go          # Chroma Store実装
├── chroma_test.go     # Chromaテスト
├── mock.go            # テスト用モックStore
└── store_test.go      # interface準拠テスト
```

## 3. Store interface詳細設計

### 3.1 Store interface (store.go)

```go
package store

import (
    "context"

    "github.com/brbranch/embedding_mcp/internal/model"
)

// Store はベクトルストアの抽象インターフェース
type Store interface {
    // Note操作
    AddNote(ctx context.Context, note *model.Note, embedding []float32) error
    Get(ctx context.Context, id string) (*model.Note, error)
    Update(ctx context.Context, note *model.Note, embedding []float32) error
    Delete(ctx context.Context, id string) error

    // ベクトル検索
    Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error)

    // 最新一覧取得（createdAt降順）
    ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error)

    // GlobalConfig操作
    UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error
    GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error)

    // 初期化・終了
    Initialize(ctx context.Context, namespace string) error
    Close() error
}

// エラー定義
var (
    ErrNotFound         = errors.New("resource not found")
    ErrNotInitialized   = errors.New("store not initialized")
    ErrConnectionFailed = errors.New("failed to connect to store")
)
```

### 3.2 オプション型 (options.go)

```go
package store

import "time"

// SearchOptions はSearch操作のオプション
type SearchOptions struct {
    ProjectID string     // 必須
    GroupID   *string    // nullable（nilの場合はフィルタなし）
    TopK      int        // default: 5
    Tags      []string   // AND検索、空/nilはフィルタなし、大小文字区別
    Since     *time.Time // UTC、境界条件: since <= createdAt
    Until     *time.Time // UTC、境界条件: createdAt < until
}

// ListOptions はListRecent操作のオプション
type ListOptions struct {
    ProjectID string   // 必須
    GroupID   *string  // nullable（nilの場合は全group）
    Limit     int      // default: 10
    Tags      []string // AND検索、空/nilはフィルタなし
}

// DefaultSearchOptions はSearchOptionsのデフォルト値を返す
func DefaultSearchOptions() SearchOptions {
    return SearchOptions{TopK: 5}
}

// DefaultListOptions はListOptionsのデフォルト値を返す
func DefaultListOptions() ListOptions {
    return ListOptions{Limit: 10}
}
```

### 3.3 結果型 (result.go)

```go
package store

import "github.com/brbranch/embedding_mcp/internal/model"

// SearchResult はベクトル検索結果の1件を表す
type SearchResult struct {
    Note  *model.Note
    Score float64 // 0-1に正規化（1が最も類似）
}
```

## 4. Chroma実装詳細

### 4.1 ChromaStore構造体 (chroma.go)

```go
const (
    DefaultChromaURL = "http://localhost:8000"
    notesCollectionPrefix  = "notes_"
    globalCollectionPrefix = "global_"
)

type ChromaStore struct {
    client           *chroma.Client
    baseURL          string
    namespace        string
    notesCollection  *chroma.Collection
    globalCollection *chroma.Collection
}

func NewChromaStore(url string) (*ChromaStore, error)
func (s *ChromaStore) Initialize(ctx context.Context, namespace string) error
func (s *ChromaStore) Close() error
```

### 4.2 メタデータキー

| キー | 用途 | 型 |
|-----|------|-----|
| `projectId` | プロジェクトID | string |
| `groupId` | グループID | string |
| `title` | タイトル | string (nullable) |
| `tags` | タグ配列 | JSON string |
| `source` | ソース | string (nullable) |
| `createdAt` | 作成日時 | ISO8601 string |
| `metadata` | 任意メタデータ | JSON string |
| `key` | GlobalConfig用キー | string |
| `value` | GlobalConfig用値 | JSON string |
| `updatedAt` | GlobalConfig更新日時 | ISO8601 string |

### 4.3 フィルタ処理の分担

| フィルタ | 処理場所 | 理由 |
|---------|---------|------|
| projectID | Chroma側 | 基本フィルタ |
| groupID | Chroma側 | 基本フィルタ |
| tags AND検索 | Go側（後処理） | Chromaは配列AND未サポート |
| since/until | Go側（後処理） | Chromaは時刻比較が限定的 |

### 4.4 スコア正規化

```go
// Chromaはcosine distance（0=同一、2=正反対）を返す
score := 1.0 - (distance / 2.0) // 0-1に正規化（1が最も類似）
```

## 5. テストケース一覧

### 5.1 Note CRUDテスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_AddNote` | ノート追加 |
| `TestChromaStore_AddNote_AllFields` | 全フィールド含むノート追加 |
| `TestChromaStore_Get` | ID指定でノート取得 |
| `TestChromaStore_Get_NotFound` | 存在しないIDでErrNotFound |
| `TestChromaStore_Update` | ノート更新（embedding含む） |
| `TestChromaStore_Update_MetadataOnly` | metadataのみ更新 |
| `TestChromaStore_Delete` | ノート削除 |

### 5.2 ベクトル検索テスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_Search_Basic` | 基本的なベクトル検索 |
| `TestChromaStore_Search_ProjectIDFilter` | projectIDでフィルタ |
| `TestChromaStore_Search_GroupIDFilter` | groupIDでフィルタ |
| `TestChromaStore_Search_TopK` | TopK指定で件数制限 |
| `TestChromaStore_Search_ScoreOrder` | スコア降順でソート |

### 5.3 タグフィルタテスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_Search_TagsFilter_AND` | tags AND検索 |
| `TestChromaStore_Search_TagsFilter_Empty` | 空配列はフィルタなし |
| `TestChromaStore_Search_TagsFilter_CaseSensitive` | 大小文字区別 |

### 5.4 時刻フィルタテスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_Search_SinceFilter` | since <= createdAt |
| `TestChromaStore_Search_UntilFilter` | createdAt < until |
| `TestChromaStore_Search_BoundaryCondition` | 境界条件テスト |

### 5.5 ListRecentテスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_ListRecent_Basic` | createdAt降順で取得 |
| `TestChromaStore_ListRecent_Limit` | Limit指定で件数制限 |
| `TestChromaStore_ListRecent_Filters` | フィルタ適用 |

### 5.6 GlobalConfigテスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_UpsertGlobal_Insert` | 新規追加 |
| `TestChromaStore_UpsertGlobal_Update` | 更新 |
| `TestChromaStore_GetGlobal_Found` | 取得成功 |
| `TestChromaStore_GetGlobal_NotFound` | found=false |

### 5.7 namespace分離テスト

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_Namespace_Isolation` | 異なるnamespaceは別コレクション |

## 6. 依存関係

```
internal/store
  ├── internal/model  (Note, GlobalConfig)
  └── github.com/amikos-tech/chroma-go
```

## 7. 責務分離（設計レビュー対応）

### 7.1 Store層 vs Service層

| 責務 | 担当 | 説明 |
|------|------|------|
| ProjectID必須チェック | Service層 | JSON-RPC errorとして返す |
| GlobalConfig key検証 | Service層 | "global."プレフィックス必須 |
| 再埋め込み判断 | Service層 | text変更時のみ再埋め込み |
| embedding生成 | Embedder | Store層はembeddingを受け取るのみ |
| TopK/Limitデフォルト | Service層 | Optionsのデフォルト値を設定 |

### 7.2 groupID=nil時の動作

- **Search**: groupIDフィルタを適用しない（全group対象）
- **ListRecent**: groupIDフィルタを適用しない（全group対象）
- Chroma側ではWhereフィルタにgroupIDを含めない

### 7.3 追加テストケース

| テストケース | 説明 |
|------------|------|
| `TestChromaStore_Search_GroupIDNil` | groupID=nilで全group検索 |
| `TestChromaStore_ListRecent_GroupIDNil` | groupID=nilで全group取得 |
| `TestChromaStore_Search_TagsNil` | tags=nilでフィルタなし |
| `TestChromaStore_Search_TagsEmpty` | tags=空配列でフィルタなし |

### 7.4 Chroma API仕様

- **Query結果**: `Distances`フィールドでcosine distance（0-2）を返す
- **スコア変換**: `score = 1.0 - (distance / 2.0)` で0-1に正規化
- **metadataフィルタ**: string型のみサポート、nullableフィールドは空文字列で保存

## 8. 完了条件

```bash
go test ./internal/store/... -v
```

上記コマンドが全てPASSし、以下の動作が確認できること:

1. Note CRUD（AddNote, Get, Update, Delete）が正常動作
2. ベクトル検索（Search）がスコア降順で結果を返す
3. tags AND検索が正しく動作（大小文字区別）
4. since/until時刻フィルタが境界条件 `since <= createdAt < until` で動作
5. ListRecentがcreatedAt降順で結果を返す
6. GlobalConfig CRUD（UpsertGlobal, GetGlobal）が正常動作
7. namespace分離が正しく機能
8. groupID=nil時に全group対象で検索/取得
