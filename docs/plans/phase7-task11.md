# Phase 7 Task 11: E2Eテスト/スモークテスト 実装計画

## 1. 概要

本タスクでは、MCPメモリサーバーの主要機能を検証するE2Eテスト（スモークテスト）を実装する。

**完了条件:** `go test ./... -tags=e2e` が成功すること

## 2. ファイル構成

```
e2e/
├── e2e_test.go        # E2Eテスト本体（//go:build e2e）
├── helper_test.go     # テストヘルパー
└── README.md          # E2Eテスト実行方法
```

## 3. 要件に基づくテストケース一覧

### 3.1 projectId正規化テスト

| テストケース | 説明 |
|------------|------|
| `TestE2E_ProjectID_TildeExpansion` | `~/tmp/demo` が正規化（canonical化）されること |
| `TestE2E_ProjectID_Consistency` | 同じパスは常に同じcanonicalパスになること |

### 3.2 add_note テスト

| テストケース | 説明 |
|------------|------|
| `TestE2E_AddNote_GlobalGroup` | groupId="global" でノート追加 |
| `TestE2E_AddNote_FeatureGroup` | groupId="feature-1" でノート追加 |

### 3.3 search テスト

| テストケース | 説明 |
|------------|------|
| `TestE2E_Search_WithGroupID` | groupId="feature-1" フィルタで検索、該当ノートが返る |
| `TestE2E_Search_WithoutGroupID` | groupId=null で検索、全グループのノートが返る |
| `TestE2E_Search_ProjectIDRequired` | projectId必須の検証 |

### 3.4 upsert_global/get_global テスト

| テストケース | 説明 |
|------------|------|
| `TestE2E_Global_EmbedderProvider` | `global.memory.embedder.provider` = "openai" |
| `TestE2E_Global_EmbedderModel` | `global.memory.embedder.model` = "text-embedding-3-small" |
| `TestE2E_Global_GroupDefaults` | `global.memory.groupDefaults` = { "featurePrefix": "feature-", "taskPrefix": "task-" } |
| `TestE2E_Global_ProjectConventions` | `global.project.conventions` = "文章" |
| `TestE2E_Global_InvalidKeyPrefix` | `global.` プレフィックスなしでエラー |

## 4. 実装アプローチ

### 4.1 テスト実行方式

2つのアプローチを検討し、**アプローチB（インプロセス統合テスト）**を採用する:

#### アプローチA: 外部プロセステスト（不採用）
- サーバーを起動し、HTTP/stdioで接続
- 利点: 本番に近い環境でテスト
- 欠点: Chromaサーバー起動が必須、CI/CDで複雑

#### アプローチB: インプロセス統合テスト（採用）
- MemoryStoreを使用し、サーバーを起動せずにテスト
- 利点: 外部依存なし、CI/CDで容易に実行
- 欠点: transport層はテストされない（別途テスト済み）

### 4.2 コンポーネント構成

```
E2Eテスト
    ↓
jsonrpc.Handler
    ↓
NoteService / ConfigService / GlobalService
    ↓
MemoryStore + MockEmbedder
```

### 4.3 モック戦略

#### Embedder モック
- 実際のOpenAI APIを呼ばない
- 決定論的な埋め込みベクトルを返す（テスト再現性のため）
- 例: テキストのハッシュから擬似ベクトルを生成

```go
type mockEmbedder struct {
    dim int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // テキストから決定論的なベクトルを生成
    // 同じテキストには同じベクトル、異なるテキストには異なるベクトル
    return generateDeterministicVector(text, m.dim), nil
}

func (m *mockEmbedder) GetDimension() int {
    return m.dim
}
```

#### Store: MemoryStore（既存実装を使用）
- 外部依存なし
- テストごとにクリーンな状態で開始可能

## 5. テストの詳細設計

### 5.1 TestE2E_ProjectID_TildeExpansion

```go
func TestE2E_ProjectID_TildeExpansion(t *testing.T) {
    h := setupTestHandler(t)

    // ~/tmp/demo でノート追加
    resp := callAddNote(t, h, "~/tmp/demo", "global", "test note")

    // レスポンスにはcanonical化されたprojectIdが返るべき
    // 検証条件（環境依存を考慮し緩い判定）:
    // - "~" が展開されていること（先頭が"~"でない）
    // - 絶対パスであること（先頭が"/"）
    assert.NotEmpty(t, resp.ID)
    assert.True(t, strings.HasPrefix(resp.CanonicalProjectID, "/"),
        "projectIDは絶対パスであるべき")
    assert.False(t, strings.Contains(resp.CanonicalProjectID, "~"),
        "~は展開されているべき")
}
```

**注意:** 現在の実装を確認したところ、projectIDの正規化はサービス層で行われる設計だが、実際の呼び出しコードがない可能性がある。実装時に確認し、必要であればサービス層に正規化処理を追加する。

### 5.1.1 TestE2E_Search_ProjectIDRequired

```go
func TestE2E_Search_ProjectIDRequired(t *testing.T) {
    h := setupTestHandler(t)

    // projectId なしで検索 → invalid params エラー (-32602)
    resp := callSearchRaw(t, h, "", nil, "query")

    assert.NotNil(t, resp.Error)
    assert.Equal(t, jsonrpc.ErrCodeInvalidParams, resp.Error.Code)
    // エラーメッセージに"projectId"が含まれることを確認
    assert.Contains(t, resp.Error.Message, "projectId")
}
```

### 5.2 TestE2E_AddNote_GlobalGroup / FeatureGroup

```go
func TestE2E_AddNote_GlobalGroup(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    resp := callAddNote(t, h, projectID, "global", "global note content")

    assert.NotEmpty(t, resp.ID)
    assert.NotEmpty(t, resp.Namespace)
}

func TestE2E_AddNote_FeatureGroup(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    resp := callAddNote(t, h, projectID, "feature-1", "feature note content")

    assert.NotEmpty(t, resp.ID)
    assert.NotEmpty(t, resp.Namespace)
}
```

### 5.3 TestE2E_Search_WithGroupID / WithoutGroupID

```go
func TestE2E_Search_WithGroupID(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // 2件追加
    callAddNote(t, h, projectID, "global", "global note")
    callAddNote(t, h, projectID, "feature-1", "feature note")

    // groupId="feature-1" でフィルタ
    resp := callSearch(t, h, projectID, ptr("feature-1"), "feature")

    // 1件のみ返る
    assert.Len(t, resp.Results, 1)
    assert.Equal(t, "feature-1", resp.Results[0].GroupID)
}

func TestE2E_Search_WithoutGroupID(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // 2件追加
    callAddNote(t, h, projectID, "global", "note content 1")
    callAddNote(t, h, projectID, "feature-1", "note content 2")

    // groupId=null で全検索
    resp := callSearch(t, h, projectID, nil, "note")

    // 2件返る
    assert.Len(t, resp.Results, 2)
}
```

### 5.4 TestE2E_Global_* シリーズ

```go
func TestE2E_Global_EmbedderProvider(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // upsert
    upsertResp := callUpsertGlobal(t, h, projectID,
        "global.memory.embedder.provider", "openai")
    assert.True(t, upsertResp.OK)

    // get
    getResp := callGetGlobal(t, h, projectID,
        "global.memory.embedder.provider")
    assert.True(t, getResp.Found)
    assert.Equal(t, "openai", getResp.Value)
}

func TestE2E_Global_EmbedderModel(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // upsert
    upsertResp := callUpsertGlobal(t, h, projectID,
        "global.memory.embedder.model", "text-embedding-3-small")
    assert.True(t, upsertResp.OK)

    // get
    getResp := callGetGlobal(t, h, projectID,
        "global.memory.embedder.model")
    assert.True(t, getResp.Found)
    assert.Equal(t, "text-embedding-3-small", getResp.Value)
}

func TestE2E_Global_GroupDefaults(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    value := map[string]any{
        "featurePrefix": "feature-",
        "taskPrefix":    "task-",
    }

    upsertResp := callUpsertGlobal(t, h, projectID,
        "global.memory.groupDefaults", value)
    assert.True(t, upsertResp.OK)

    getResp := callGetGlobal(t, h, projectID,
        "global.memory.groupDefaults")
    assert.True(t, getResp.Found)
    // map比較
    resultMap, ok := getResp.Value.(map[string]any)
    assert.True(t, ok)
    assert.Equal(t, "feature-", resultMap["featurePrefix"])
    assert.Equal(t, "task-", resultMap["taskPrefix"])
}

func TestE2E_Global_ProjectConventions(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // upsert（日本語文字列）
    upsertResp := callUpsertGlobal(t, h, projectID,
        "global.project.conventions", "文章")
    assert.True(t, upsertResp.OK)

    // get
    getResp := callGetGlobal(t, h, projectID,
        "global.project.conventions")
    assert.True(t, getResp.Found)
    assert.Equal(t, "文章", getResp.Value)
}

func TestE2E_Global_InvalidKeyPrefix(t *testing.T) {
    h := setupTestHandler(t)
    projectID := "/test/project"

    // "global." プレフィックスなし → エラー
    resp := callUpsertGlobalRaw(t, h, projectID,
        "memory.embedder.provider", "openai")

    assert.Equal(t, jsonrpc.ErrCodeInvalidParams, resp.Error.Code)
}
```

## 6. ヘルパー関数

```go
// setupTestHandler はテスト用のHandlerを構築
func setupTestHandler(t *testing.T) *jsonrpc.Handler {
    t.Helper()

    // 1. MockEmbedder作成
    emb := &mockEmbedder{dim: 128}

    // 2. MemoryStore作成・初期化
    store := store.NewMemoryStore()
    namespace := "test:mock:128"
    if err := store.Initialize(context.Background(), namespace); err != nil {
        t.Fatal(err)
    }

    // 3. Services作成
    noteService := service.NewNoteService(emb, store, namespace)
    globalService := service.NewGlobalService(store, namespace)

    // 4. ConfigService（モック）
    configService := &mockConfigService{}

    // 5. Handler作成
    return jsonrpc.New(noteService, configService, globalService)
}

// callAddNote はmemory.add_noteを呼び出す
func callAddNote(t *testing.T, h *jsonrpc.Handler, projectID, groupID, text string) *AddNoteResponse {
    // JSON-RPCリクエストを構築して呼び出し
}

// callSearch はmemory.searchを呼び出す
func callSearch(t *testing.T, h *jsonrpc.Handler, projectID string, groupID *string, query string) *SearchResponse {
    // JSON-RPCリクエストを構築して呼び出し
}
```

## 7. ビルドタグ

E2EテストはCIで時間がかかる可能性があるため、ビルドタグで分離:

```go
//go:build e2e

package e2e

import (
    "testing"
)
```

通常のテスト: `go test ./...`
E2Eテスト含む: `go test ./... -tags=e2e`

## 8. projectID正規化の実装確認

### 現状の分析

コードを確認したところ、`config.CanonicalizeProjectID` 関数は存在するが、サービス層（note.go, global.go）での呼び出しがない。設計文書（phase3-task6.md）には「ProjectIDを正規化」と記載されているが、実装されていない可能性がある。

### 対応方針

E2Eテスト実装時に以下を確認・修正:

1. **サービス層での正規化呼び出し追加**
   - `noteService.AddNote` 内でprojectIDを正規化
   - `noteService.Search` 内でprojectIDを正規化
   - `globalService.UpsertGlobal` 内でprojectIDを正規化
   - `globalService.GetGlobal` 内でprojectIDを正規化

2. **レスポンスにcanonical化されたprojectIDを返す**
   - search/get/list_recentの結果にはcanonicalなprojectIDが含まれるべき

## 9. テスト実行前提条件

### 必須ではないもの
- Chromaサーバー: MemoryStoreを使用するため不要
- OpenAI APIキー: MockEmbedderを使用するため不要

### 必須なもの
- Go 1.22+

## 10. 実装手順

1. `e2e/` ディレクトリ作成
2. `e2e/helper_test.go` 実装（setupTestHandler, callXxx関数群）
3. `e2e/e2e_test.go` 実装（各テストケース）
4. projectID正規化が未実装の場合、サービス層を修正
5. `go test ./... -tags=e2e` で全テスト実行
6. `e2e/README.md` 作成（実行方法の説明）

## 11. 完了条件チェックリスト

- [ ] `TestE2E_ProjectID_TildeExpansion` がPASS
- [ ] `TestE2E_AddNote_GlobalGroup` がPASS
- [ ] `TestE2E_AddNote_FeatureGroup` がPASS
- [ ] `TestE2E_Search_WithGroupID` がPASS
- [ ] `TestE2E_Search_WithoutGroupID` がPASS
- [ ] `TestE2E_Global_EmbedderProvider` がPASS
- [ ] `TestE2E_Global_EmbedderModel` がPASS
- [ ] `TestE2E_Global_GroupDefaults` がPASS
- [ ] `TestE2E_Global_ProjectConventions` がPASS
- [ ] `TestE2E_Global_InvalidKeyPrefix` がPASS
- [ ] `go test ./... -tags=e2e` が成功

## 12. 補足: 将来のE2E拡張

本タスクはスモークテストであるため、主要な動作確認に絞る。将来的には以下のE2Eテストを追加可能:

- HTTP transport経由のE2Eテスト（Chromaサーバー必須）
- stdio transport経由のE2Eテスト
- 大量データ投入時のパフォーマンステスト
- provider切替時のnamespace変更確認
