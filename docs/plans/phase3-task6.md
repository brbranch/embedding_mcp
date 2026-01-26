# Phase 3 Task 6: サービス層 (internal/service) 実装計画

## 1. 概要

`internal/service` パッケージは以下の機能を提供する:

1. **NoteService**: ノートのCRUD + 検索（埋め込み生成→Store操作）
2. **ConfigService**: 設定の取得・変更
3. **GlobalService**: グローバル設定のUpsert/Get

## 2. 作成するファイル一覧

```
internal/service/
├── service.go         # Service interface定義、共通エラー
├── note.go            # NoteService実装
├── note_test.go       # NoteServiceテスト
├── config.go          # ConfigService実装
├── config_test.go     # ConfigServiceテスト
├── global.go          # GlobalService実装
├── global_test.go     # GlobalServiceテスト
└── options.go         # オプション定義
```

## 3. 依存関係

```
internal/service
  ├── internal/model    (Note, GlobalConfig, Config)
  ├── internal/embedder (Embedder interface)
  ├── internal/store    (Store interface)
  └── internal/config   (Manager, namespace生成)
```

## 4. NoteService詳細設計

### 4.1 インターフェース

```go
type NoteService interface {
    AddNote(ctx context.Context, req *AddNoteRequest) (*AddNoteResponse, error)
    Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
    Get(ctx context.Context, id string) (*GetResponse, error)
    Update(ctx context.Context, req *UpdateRequest) error
    ListRecent(ctx context.Context, req *ListRecentRequest) (*ListRecentResponse, error)
}
```

### 4.2 AddNote

**入力:**
```go
type AddNoteRequest struct {
    ProjectID string
    GroupID   string
    Title     *string
    Text      string
    Tags      []string
    Source    *string
    CreatedAt *string   // nullならサーバー側で設定
    Metadata  map[string]any
}
```

**処理:**
1. ProjectIDを正規化（config.CanonicalizeProjectID）
2. UUIDを生成
3. CreatedAtがnullなら現在時刻を設定
4. Embedder.Embed(text)で埋め込みを生成
5. Store.AddNote(note, embedding)で保存
6. namespace（config.GenerateNamespace）と共にIDを返却

**出力:**
```go
type AddNoteResponse struct {
    ID        string
    Namespace string
}
```

### 4.3 Search

**入力:**
```go
type SearchRequest struct {
    ProjectID string
    GroupID   *string  // nilなら全group
    Query     string
    TopK      *int     // default 5
    Tags      []string // AND検索
    Since     *string  // UTC ISO8601
    Until     *string  // UTC ISO8601
}
```

**処理:**
1. ProjectIDを正規化
2. Embedder.Embed(query)でクエリを埋め込み化
3. Store.Search(embedding, opts)で検索
4. スコア降順でソート（Store側でソート済み想定）
5. namespace + resultsを返却

**出力:**
```go
type SearchResponse struct {
    Namespace string
    Results   []SearchResult
}

type SearchResult struct {
    ID        string
    ProjectID string
    GroupID   string
    Title     *string
    Text      string
    Tags      []string
    Source    *string
    CreatedAt string
    Score     float64  // 0-1正規化
    Metadata  map[string]any
}
```

### 4.4 Get

**処理:**
1. Store.Get(id)でノートを取得
2. 見つからなければエラー

**出力:**
```go
type GetResponse struct {
    ID        string
    ProjectID string
    GroupID   string
    Title     *string
    Text      string
    Tags      []string
    Source    *string
    CreatedAt string
    Namespace string
    Metadata  map[string]any
}
```

### 4.5 Update

**入力:**
```go
type UpdateRequest struct {
    ID    string
    Patch NotePatch
}

type NotePatch struct {
    Title    *string         // nilは変更なし、""は空文字に設定
    Text     *string         // 変更時のみ再埋め込み
    Tags     *[]string
    Source   *string
    GroupID  *string         // 再埋め込み不要
    Metadata *map[string]any
}
```

**処理:**
1. Store.Get(id)で既存ノートを取得
2. patchを適用
3. Text変更時のみEmbedder.Embed(newText)
4. Store.Update(note, embedding)で保存

### 4.6 ListRecent

**入力:**
```go
type ListRecentRequest struct {
    ProjectID string
    GroupID   *string
    Limit     *int     // default 10
    Tags      []string
}
```

**処理:**
1. ProjectIDを正規化
2. Store.ListRecent(opts)で取得
3. createdAt降順でソート（Store側でソート済み想定）

## 5. ConfigService詳細設計

### 5.1 インターフェース

```go
type ConfigService interface {
    GetConfig(ctx context.Context) (*GetConfigResponse, error)
    SetConfig(ctx context.Context, req *SetConfigRequest) (*SetConfigResponse, error)
}
```

### 5.2 GetConfig

**出力:**
```go
type GetConfigResponse struct {
    TransportDefaults TransportDefaults
    Embedder          EmbedderConfig
    Store             StoreConfig
    Paths             PathsConfig
}
```

**処理:**
1. config.Manager.Get()で現在設定を取得
2. build-time defaultsを含めて返却

### 5.3 SetConfig

**入力:**
```go
type SetConfigRequest struct {
    Embedder *EmbedderPatch // embedderのみ変更可能
}

type EmbedderPatch struct {
    Provider *string
    Model    *string
    BaseURL  *string
    APIKey   *string
}
```

**処理:**
1. config.Manager.Update()で設定を更新
2. provider/model変更時はnamespaceが変わる（dimリセット→初回埋め込みで再取得）
3. effectiveNamespace（新namespace）を返却

**出力:**
```go
type SetConfigResponse struct {
    OK                 bool
    EffectiveNamespace string
}
```

## 6. GlobalService詳細設計

### 6.1 インターフェース

```go
type GlobalService interface {
    UpsertGlobal(ctx context.Context, req *UpsertGlobalRequest) (*UpsertGlobalResponse, error)
    GetGlobal(ctx context.Context, projectID, key string) (*GetGlobalResponse, error)
}
```

### 6.2 UpsertGlobal

**入力:**
```go
type UpsertGlobalRequest struct {
    ProjectID string
    Key       string // "global." プレフィックス必須
    Value     any
    UpdatedAt *string
}
```

**処理:**
1. Key が "global." で始まることを検証（でなければエラー）
2. ProjectIDを正規化
3. UpdatedAtがnullなら現在時刻を設定
4. Store.UpsertGlobal(config)で保存

**出力:**
```go
type UpsertGlobalResponse struct {
    OK        bool
    ID        string
    Namespace string
}
```

### 6.3 GetGlobal

**処理:**
1. ProjectIDを正規化
2. Store.GetGlobal(projectID, key)で取得

**出力:**
```go
type GetGlobalResponse struct {
    Namespace string
    Found     bool
    ID        *string
    Value     any
    UpdatedAt *string
}
```

## 7. エラー定義

```go
var (
    ErrNoteNotFound       = errors.New("note not found")
    ErrInvalidGlobalKey   = errors.New("key must start with 'global.'")
    ErrProjectIDRequired  = errors.New("projectId is required")
    ErrTextRequired       = errors.New("text is required")
    ErrQueryRequired      = errors.New("query is required")
    ErrIDRequired         = errors.New("id is required")
)
```

## 8. テストケース一覧

### NoteServiceテスト

| テストケース | 説明 |
|------------|------|
| `TestNoteService_AddNote_Success` | 正常追加 |
| `TestNoteService_AddNote_ProjectIDRequired` | projectID必須 |
| `TestNoteService_AddNote_TextRequired` | text必須 |
| `TestNoteService_AddNote_CreatedAtDefault` | createdAt自動設定 |
| `TestNoteService_Search_Success` | 正常検索 |
| `TestNoteService_Search_WithGroupID` | groupIDフィルタ |
| `TestNoteService_Search_WithTags` | tagsフィルタ |
| `TestNoteService_Search_WithTimeRange` | since/untilフィルタ |
| `TestNoteService_Get_Success` | 正常取得 |
| `TestNoteService_Get_NotFound` | 存在しないID |
| `TestNoteService_Update_Success` | 正常更新 |
| `TestNoteService_Update_TextReembed` | text変更時再埋め込み |
| `TestNoteService_Update_NotFound` | 存在しないID |
| `TestNoteService_ListRecent_Success` | 正常取得 |
| `TestNoteService_ListRecent_WithGroupID` | groupIDフィルタ |

### ConfigServiceテスト

| テストケース | 説明 |
|------------|------|
| `TestConfigService_GetConfig_Success` | 正常取得 |
| `TestConfigService_SetConfig_Provider` | provider変更 |
| `TestConfigService_SetConfig_Model` | model変更 |
| `TestConfigService_SetConfig_NamespaceChange` | 変更後namespace確認 |

### GlobalServiceテスト

| テストケース | 説明 |
|------------|------|
| `TestGlobalService_UpsertGlobal_Success` | 正常追加 |
| `TestGlobalService_UpsertGlobal_InvalidKey` | "global."なしでエラー |
| `TestGlobalService_UpsertGlobal_UpdatedAtDefault` | updatedAt自動設定 |
| `TestGlobalService_GetGlobal_Found` | 正常取得 |
| `TestGlobalService_GetGlobal_NotFound` | 見つからない場合 |

## 9. テスト方法

モックを使用してEmbedder/Storeを注入:

```go
type mockEmbedder struct {
    embedFunc func(ctx context.Context, text string) ([]float32, error)
    dim       int
}

type mockStore struct {
    notes  map[string]*model.Note
    global map[string]*model.GlobalConfig
}
```

## 10. 完了条件

```bash
go test ./internal/service/... -v
```

上記コマンドが全てPASSし、以下の動作が確認できること:

1. NoteServiceでノートのCRUD + 検索が動作
2. ConfigServiceで設定の取得・変更が動作
3. GlobalServiceでグローバル設定のUpsert/Getが動作
4. projectID正規化が全メソッドで適用
5. namespace生成が正しく行われる
