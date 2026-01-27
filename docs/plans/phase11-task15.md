# Phase 11 Task 15: SQLite VectorStore実装 設計計画

## 概要

軽量用途向けのSQLiteベースのVectorStore実装。cgo不要のmodernc.org/sqliteを使用し、全件スキャン方式でcosine類似度検索を行う。

## 要件トレーサビリティ表

### TODO.md 要件

| 要件ID | 要件内容 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-01 | SQLite実装（modernc.org/sqlite使用、cgo不要） | TestSQLiteStore_NewStore | internal/store/sqlite.go:NewSQLiteStore |
| REQ-02 | cosine類似度による全件スキャン検索 | TestSQLiteStore_Search_* | internal/store/sqlite.go:Search |
| REQ-03 | embeddingsをSQLiteに保存 | TestSQLiteStore_AddNote_* | internal/store/sqlite.go:AddNote |
| REQ-04 | 5,000件超過時の警告ログ出力 | TestSQLiteStore_AddNote_WarningAt5000 | internal/store/sqlite.go:AddNote |

### Store interface準拠要件

| 要件ID | 要件内容 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-05 | AddNote（ノート追加） | TestSQLiteStore_AddNote_* | internal/store/sqlite.go:AddNote |
| REQ-06 | Get（ID指定取得） | TestSQLiteStore_Get_* | internal/store/sqlite.go:Get |
| REQ-07 | Update（ノート更新） | TestSQLiteStore_Update_* | internal/store/sqlite.go:Update |
| REQ-08 | Delete（ノート削除） | TestSQLiteStore_Delete_* | internal/store/sqlite.go:Delete |
| REQ-09 | Search（ベクトル検索） | TestSQLiteStore_Search_* | internal/store/sqlite.go:Search |
| REQ-10 | ListRecent（最新一覧、createdAt降順） | TestSQLiteStore_ListRecent_* | internal/store/sqlite.go:ListRecent |
| REQ-11 | UpsertGlobal（GlobalConfig追加/更新） | TestSQLiteStore_UpsertGlobal_* | internal/store/sqlite.go:UpsertGlobal |
| REQ-12 | GetGlobal（GlobalConfig取得） | TestSQLiteStore_GetGlobal_* | internal/store/sqlite.go:GetGlobal |
| REQ-13 | Initialize（namespace設定、テーブル作成） | TestSQLiteStore_Initialize | internal/store/sqlite.go:Initialize |
| REQ-14 | Close（DB接続クローズ） | TestSQLiteStore_Close | internal/store/sqlite.go:Close |

### 仕様要件（embedded_spec.md / types.go準拠）

| 要件ID | 要件内容 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-15 | projectIDフィルタ（Search/ListRecent必須） | TestSQLiteStore_Search_ProjectIDFilter, TestSQLiteStore_ListRecent_ProjectIDFilter | Search, ListRecent |
| REQ-16 | groupIDフィルタ（nil時はフィルタなし） | TestSQLiteStore_Search_WithGroupIDFilter, TestSQLiteStore_ListRecent_WithGroupIDFilter | Search, ListRecent |
| REQ-17 | tagsフィルタ（AND検索、大小文字区別） | TestSQLiteStore_Search_WithTagsFilter, TestSQLiteStore_ListRecent_WithTagsFilter | Search, ListRecent |
| REQ-18 | since/untilフィルタ（since <= createdAt < until） | TestSQLiteStore_Search_WithTimeFilter_Boundary | Search |
| REQ-19 | スコア0-1正規化（1が最も類似） | TestSQLiteStore_Search_ScoreNormalization | Search |
| REQ-20 | TopK制限（default=5） | TestSQLiteStore_Search_TopK | Search |
| REQ-21 | Limit制限（default=10） | TestSQLiteStore_ListRecent_Limit | ListRecent |
| REQ-22 | createdAt降順ソート | TestSQLiteStore_ListRecent_SortOrder | ListRecent |
| REQ-23 | createdAt UTC ISO8601形式 | TestSQLiteStore_AddNote_CreatedAtFormat | AddNote, Search |

### エラーハンドリング要件

| 要件ID | 要件内容 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-24 | ErrNotFound（存在しないID） | TestSQLiteStore_Get_NotFound, TestSQLiteStore_Update_NotFound, TestSQLiteStore_Delete_NotFound | Get, Update, Delete |
| REQ-25 | ErrNotInitialized（Initialize前の操作） | TestSQLiteStore_NotInitialized | 全メソッド |

## 作成するファイル

| ファイル | 用途 |
|----------|------|
| `internal/store/sqlite.go` | SQLite Store実装 |
| `internal/store/sqlite_test.go` | テスト |
| `internal/store/helpers.go` | 共通関数（cosineSimilarity, containsAllTags） |

## データベーススキーマ

### namespace分離方針

- テーブルにnamespaceカラムを追加し、全クエリでnamespaceフィルタを適用
- 同一DBファイル内で複数namespaceのデータを分離管理
- Initialize時に設定されたnamespaceを全操作で使用

### notes テーブル

```sql
CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    namespace TEXT NOT NULL,  -- namespace分離用
    project_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    title TEXT,
    text TEXT NOT NULL,
    tags TEXT,           -- JSON配列として格納
    source TEXT,
    created_at TEXT,     -- ISO8601 UTC形式（NULL許容、NULL時は末尾ソート）
    metadata TEXT,       -- JSONとして格納
    embedding BLOB       -- []float32をLittle Endian形式で保存
);

CREATE INDEX IF NOT EXISTS idx_notes_namespace ON notes(namespace);
CREATE INDEX IF NOT EXISTS idx_notes_project_id ON notes(namespace, project_id);
CREATE INDEX IF NOT EXISTS idx_notes_group_id ON notes(namespace, group_id);
CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(namespace, created_at);
```

### global_configs テーブル

```sql
CREATE TABLE IF NOT EXISTS global_configs (
    id TEXT PRIMARY KEY,
    namespace TEXT NOT NULL,  -- namespace分離用
    project_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT,         -- JSONとして格納
    updated_at TEXT,    -- ISO8601 UTC形式
    UNIQUE(namespace, project_id, key)
);

CREATE INDEX IF NOT EXISTS idx_global_configs_namespace ON global_configs(namespace);
CREATE INDEX IF NOT EXISTS idx_global_configs_project_key ON global_configs(namespace, project_id, key);
```

## 主要設計

### 1. Embedding保存形式

`[]float32` を `[]byte` に変換してBLOBとして保存:
- Little Endian形式
- 各float32は4バイト
- 例: 1536次元のembeddingは6144バイト

```go
// 保存時
func encodeEmbedding(embedding []float32) []byte {
    buf := make([]byte, len(embedding)*4)
    for i, v := range embedding {
        binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
    }
    return buf
}

// 読み込み時
func decodeEmbedding(data []byte) []float32 {
    embedding := make([]float32, len(data)/4)
    for i := range embedding {
        embedding[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
    }
    return embedding
}
```

### 2. 検索方式

全件スキャンによるcosine類似度検索:
1. SQLでnamespace + projectIDフィルタを適用
2. Go側でgroupID/tags/since/untilフィルタを適用
3. Go側でcosine類似度を計算
4. スコア降順でソートしてTopK件を返却

**フィルタ仕様**:
- namespace: 全クエリで必須フィルタ（Initialize時に設定）
- projectID: 必須フィルタ
- groupID: nil時はフィルタなし
- tags: AND検索（全tagを含む）、大小文字区別、**空配列/nilはフィルタなし**
- since/until: 境界条件は `since <= createdAt < until`
- スコア: 0-1正規化（`score = 1.0 - (cosine_distance / 2.0)`、1が最も類似）

**createdAt nil時の扱い**:
- ListRecent: SQLiteのデフォルト動作に従い、NULL値は末尾に配置
- Search: createdAtがnilのノートはsince/untilフィルタをスキップ（フィルタ対象外）

**注意**: 5,000件程度までの軽量用途向け。大規模データにはChromaStore推奨。

### 3. 共通関数の再利用

`memory.go`で定義済みの関数を`helpers.go`に抽出して共有:
- `CosineSimilarity(a, b []float32) float64` - cosine distance計算
- `ContainsAllTags(tags, targets []string) bool` - AND検索

### 4. createdAt補完処理

embedded_spec.mdに従い、AddNote時にcreatedAtがnullの場合はサーバー側で現在時刻（UTC ISO8601）を設定する。

```go
func (s *SQLiteStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
    // createdAtがnilの場合は現在時刻を設定
    if note.CreatedAt == nil {
        now := time.Now().UTC().Format(time.RFC3339)
        note.CreatedAt = &now
    }
    // ... 保存処理 ...
}
```

### 5. 警告ログ出力

```go
const noteCountWarningThreshold = 5000

func (s *SQLiteStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
    // ... 保存処理 ...

    // 件数チェック
    count, _ := s.countNotes(ctx)
    if count >= noteCountWarningThreshold {
        slog.Warn("note count exceeded threshold",
            "count", count,
            "threshold", noteCountWarningThreshold,
            "recommendation", "consider using ChromaStore for better performance")
    }
    return nil
}
```

### 6. 初期化状態チェック

Initialize()呼び出し前の操作は`ErrNotInitialized`を返す（MemoryStoreと同じ動作）。

## SQLiteStore構造体

```go
type SQLiteStore struct {
    db          *sql.DB
    dbPath      string
    namespace   string
    initialized bool
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error)
func (s *SQLiteStore) Initialize(ctx context.Context, namespace string) error
func (s *SQLiteStore) Close() error
func (s *SQLiteStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error
func (s *SQLiteStore) Get(ctx context.Context, id string) (*model.Note, error)
func (s *SQLiteStore) Update(ctx context.Context, note *model.Note, embedding []float32) error
func (s *SQLiteStore) Delete(ctx context.Context, id string) error
func (s *SQLiteStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error)
func (s *SQLiteStore) ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error)
func (s *SQLiteStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error
func (s *SQLiteStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error)
```

## テストケース

### 基本操作

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_NewStore | インスタンス作成、DBファイル作成 |
| TestSQLiteStore_Initialize | テーブル作成、namespace設定 |
| TestSQLiteStore_Close | DB接続クローズ |
| TestSQLiteStore_NotInitialized | Initialize前の各操作がErrNotInitializedを返す |

### AddNote

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_AddNote_Basic | 基本的なノート追加 |
| TestSQLiteStore_AddNote_WithAllFields | 全フィールド指定でのノート追加 |
| TestSQLiteStore_AddNote_WarningAt5000 | 5,000件超過時の警告ログ |
| TestSQLiteStore_AddNote_CreatedAtFormat | createdAtがUTC ISO8601形式で保存される |

### Get

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_Get_Found | 存在するノート取得 |
| TestSQLiteStore_Get_NotFound | 存在しないノート取得→ErrNotFound |

### Update

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_Update_Basic | ノート更新 |
| TestSQLiteStore_Update_WithReembedding | embedding変更を伴う更新 |
| TestSQLiteStore_Update_NotFound | 存在しないノート更新→ErrNotFound |

### Delete

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_Delete_Basic | ノート削除 |
| TestSQLiteStore_Delete_NotFound | 存在しないノート削除→ErrNotFound |

### Search

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_Search_Basic | 基本的なベクトル検索 |
| TestSQLiteStore_Search_ProjectIDFilter | projectIDフィルタ |
| TestSQLiteStore_Search_WithGroupIDFilter | groupIDフィルタ（nil時はフィルタなし） |
| TestSQLiteStore_Search_WithTagsFilter | tagsフィルタ（AND検索、大小文字区別） |
| TestSQLiteStore_Search_WithTimeFilter_Boundary | since/untilフィルタ（境界条件: since <= createdAt < until） |
| TestSQLiteStore_Search_TopK | TopK制限（default=5） |
| TestSQLiteStore_Search_ScoreNormalization | スコア0-1正規化（1が最も類似） |
| TestSQLiteStore_Search_SortOrder | スコア降順ソート |

### ListRecent

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_ListRecent_Basic | 基本的な最新ノート一覧 |
| TestSQLiteStore_ListRecent_ProjectIDFilter | projectIDフィルタ |
| TestSQLiteStore_ListRecent_WithGroupIDFilter | groupIDフィルタ |
| TestSQLiteStore_ListRecent_WithTagsFilter | tagsフィルタ（AND検索） |
| TestSQLiteStore_ListRecent_Limit | Limit制限（default=10） |
| TestSQLiteStore_ListRecent_SortOrder | createdAt降順ソート |

### GlobalConfig

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_UpsertGlobal_Insert | 新規global config作成 |
| TestSQLiteStore_UpsertGlobal_Update | global config更新 |
| TestSQLiteStore_GetGlobal_Found | 存在するconfig取得 |
| TestSQLiteStore_GetGlobal_NotFound | 存在しないconfig取得→(nil, false, nil) |

## 依存パッケージ

```go
import (
    "modernc.org/sqlite" // cgo不要のSQLiteドライバ
)
```

go.mod追加:
```
require modernc.org/sqlite v1.x.x
```

## 実装順序

1. `go get modernc.org/sqlite` で依存追加
2. `helpers.go` に共通関数を抽出（CosineSimilarity, ContainsAllTags）
3. `sqlite.go` の基本構造体とコンストラクタ
4. Initialize（スキーマ作成）
5. AddNote / Get / Update / Delete
6. Search（cosine類似度検索）
7. ListRecent
8. UpsertGlobal / GetGlobal
9. 5,000件警告ログ
10. テスト作成

## リスク・考慮事項

1. **パフォーマンス**: 全件スキャンのため、5,000件超で性能劣化の可能性
2. **メモリ使用量**: 検索時に全embeddingをメモリに読み込むため注意
3. **並行アクセス**: SQLiteのWALモードで対応
4. **ファイルロック**: 複数プロセスからのアクセス時の挙動確認が必要
