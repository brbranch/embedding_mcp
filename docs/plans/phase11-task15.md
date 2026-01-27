# Phase 11 Task 15: SQLite VectorStore実装 設計計画

## 概要

軽量用途向けのSQLiteベースのVectorStore実装。cgo不要のmodernc.org/sqliteを使用し、全件スキャン方式でcosine類似度検索を行う。

## 要件トレーサビリティ表

| 要件ID | 要件内容 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-01 | SQLite実装（modernc.org/sqlite使用、cgo不要） | TestSQLiteStore_NewStore | internal/store/sqlite.go:NewSQLiteStore |
| REQ-02 | cosine類似度による全件スキャン検索 | TestSQLiteStore_Search_* | internal/store/sqlite.go:Search |
| REQ-03 | embeddingsをSQLiteに保存 | TestSQLiteStore_AddNote_* | internal/store/sqlite.go:AddNote |
| REQ-04 | 5,000件超過時の警告ログ出力 | TestSQLiteStore_AddNote_WarningAt5000 | internal/store/sqlite.go:AddNote |
| REQ-05 | Store interface準拠 | 全テストケース | internal/store/sqlite.go |

## 作成するファイル

| ファイル | 用途 |
|----------|------|
| `internal/store/sqlite.go` | SQLite Store実装 |
| `internal/store/sqlite_test.go` | テスト |

## データベーススキーマ

### notes テーブル

```sql
CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    title TEXT,
    text TEXT NOT NULL,
    tags TEXT,           -- JSON配列として格納
    source TEXT,
    created_at TEXT,     -- ISO8601形式
    metadata TEXT,       -- JSONとして格納
    embedding BLOB       -- []float32をLittle Endian形式で保存
);

CREATE INDEX IF NOT EXISTS idx_notes_project_id ON notes(project_id);
CREATE INDEX IF NOT EXISTS idx_notes_group_id ON notes(group_id);
CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at);
```

### global_configs テーブル

```sql
CREATE TABLE IF NOT EXISTS global_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT,         -- JSONとして格納
    updated_at TEXT,    -- ISO8601形式
    UNIQUE(project_id, key)
);

CREATE INDEX IF NOT EXISTS idx_global_configs_project_key ON global_configs(project_id, key);
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
1. SQLでprojectID/groupID/tags/since/untilフィルタを適用
2. フィルタ後の全件をメモリに読み込み
3. Go側でcosine類似度を計算
4. スコア降順でソートしてTopK件を返却

**注意**: 5,000件程度までの軽量用途向け。大規模データにはChromaStore推奨。

### 3. 共通関数の再利用

`memory.go`で定義済みの関数を再利用:
- `cosineSimilarity(a, b []float32) float64`
- `containsAllTags(tags, targets []string) bool`

これらの関数はexportして共有するか、`helpers.go`に抽出する。

### 4. 警告ログ出力

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

## SQLiteStore構造体

```go
type SQLiteStore struct {
    db        *sql.DB
    dbPath    string
    namespace string
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

| テストケース | 検証内容 |
|-------------|----------|
| TestSQLiteStore_NewStore | インスタンス作成、DBファイル作成 |
| TestSQLiteStore_Initialize | テーブル作成、namespace設定 |
| TestSQLiteStore_AddNote_Basic | 基本的なノート追加 |
| TestSQLiteStore_AddNote_WithAllFields | 全フィールド指定でのノート追加 |
| TestSQLiteStore_AddNote_WarningAt5000 | 5,000件超過時の警告ログ |
| TestSQLiteStore_Get_Found | 存在するノート取得 |
| TestSQLiteStore_Get_NotFound | 存在しないノート取得 |
| TestSQLiteStore_Update_Basic | ノート更新 |
| TestSQLiteStore_Update_WithReembedding | embedding変更を伴う更新 |
| TestSQLiteStore_Delete | ノート削除 |
| TestSQLiteStore_Search_Basic | 基本的なベクトル検索 |
| TestSQLiteStore_Search_WithGroupIDFilter | groupIDフィルタ |
| TestSQLiteStore_Search_WithTagsFilter | tagsフィルタ（AND検索） |
| TestSQLiteStore_Search_WithTimeFilter | since/untilフィルタ |
| TestSQLiteStore_Search_TopK | TopK制限 |
| TestSQLiteStore_ListRecent_Basic | 最新ノート一覧 |
| TestSQLiteStore_ListRecent_WithFilters | フィルタ付き一覧 |
| TestSQLiteStore_UpsertGlobal_Insert | 新規global config作成 |
| TestSQLiteStore_UpsertGlobal_Update | global config更新 |
| TestSQLiteStore_GetGlobal_Found | 存在するconfig取得 |
| TestSQLiteStore_GetGlobal_NotFound | 存在しないconfig取得 |
| TestSQLiteStore_Close | DB接続クローズ |

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
2. `helpers.go` に共通関数を抽出（cosineSimilarity, containsAllTags）
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
