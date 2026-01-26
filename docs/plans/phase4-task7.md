# Phase 4 Task 7: JSON-RPCハンドラー (internal/jsonrpc)

## 概要
JSON-RPC 2.0のリクエストをパースし、適切なサービスメソッドにディスパッチするハンドラー層を実装する。

## ディレクトリ構成

```
internal/jsonrpc/
├── handler.go      # メインのHandler構造体とDispatch
├── methods.go      # 各メソッドハンドラーの実装
├── params.go       # JSON-RPCパラメータ→サービスリクエスト変換
└── handler_test.go # テスト
```

## 設計

### Handler構造体

```go
// Handler はJSON-RPCリクエストを処理する
type Handler struct {
    noteService   service.NoteService
    configService service.ConfigService
    globalService service.GlobalService
}

// New は新しいHandlerを生成
func New(
    noteService service.NoteService,
    configService service.ConfigService,
    globalService service.GlobalService,
) *Handler

// Handle はJSON-RPCリクエストをパースしてディスパッチ
// 戻り値は *model.Response または *model.ErrorResponse のJSON bytes
func (h *Handler) Handle(ctx context.Context, requestBytes []byte) []byte
```

### メソッドディスパッチ

| JSON-RPCメソッド | サービスメソッド |
|-----------------|-----------------|
| memory.add_note | NoteService.AddNote |
| memory.search | NoteService.Search |
| memory.get | NoteService.Get |
| memory.update | NoteService.Update |
| memory.list_recent | NoteService.ListRecent |
| memory.get_config | ConfigService.GetConfig |
| memory.set_config | ConfigService.SetConfig |
| memory.upsert_global | GlobalService.UpsertGlobal |
| memory.get_global | GlobalService.GetGlobal |

### パラメータ型定義（params.go）

JSON-RPCのparams（map[string]any）をサービスのリクエスト型に変換する。

```go
// AddNoteParams は memory.add_note のパラメータ
type AddNoteParams struct {
    ProjectID string          `json:"projectId"`
    GroupID   string          `json:"groupId"`
    Title     *string         `json:"title"`
    Text      string          `json:"text"`
    Tags      []string        `json:"tags"`
    Source    *string         `json:"source"`
    CreatedAt *string         `json:"createdAt"`
    Metadata  map[string]any  `json:"metadata"`
}

// SearchParams は memory.search のパラメータ
type SearchParams struct {
    ProjectID string    `json:"projectId"`
    GroupID   *string   `json:"groupId"`
    Query     string    `json:"query"`
    TopK      *int      `json:"topK"`
    Tags      []string  `json:"tags"`
    Since     *string   `json:"since"`
    Until     *string   `json:"until"`
}

// GetParams は memory.get のパラメータ
type GetParams struct {
    ID string `json:"id"`
}

// UpdateParams は memory.update のパラメータ
type UpdateParams struct {
    ID    string      `json:"id"`
    Patch PatchParams `json:"patch"`
}

// PatchParams は memory.update のパッチパラメータ
// 注意: JSON-RPCでは「キーが存在しない」と「null」を区別できる
// - キーが存在しない → フィールドを変更しない
// - null → フィールドをnullにクリアする（title, source, metadataのみ）
// この区別を実現するため、json.RawMessageを使用してパース時に判定する
//
// groupIdについて:
// - groupIdは必須フィールドのため、nullでのクリアは不可
// - groupIdがnullの場合はInvalidParams（groupId is required）エラー
// - 空文字の場合も同様にInvalidParams
type PatchParams struct {
    Title    json.RawMessage `json:"title,omitempty"`    // 存在しない/null/string
    Text     *string         `json:"text,omitempty"`     // 存在しない/string（nullはtext必須違反）
    Tags     *[]string       `json:"tags,omitempty"`
    Source   json.RawMessage `json:"source,omitempty"`   // 存在しない/null/string
    GroupID  *string         `json:"groupId,omitempty"`  // 存在しない/string（nullは不可）
    Metadata json.RawMessage `json:"metadata,omitempty"` // 存在しない/null/object
}

// ParsePatch はPatchParamsをservice.NotePatchに変換
// json.RawMessageを解釈して「未指定」「null」「値あり」を判定する
func ParsePatch(p *PatchParams) (*service.NotePatch, error)

// ListRecentParams は memory.list_recent のパラメータ
type ListRecentParams struct {
    ProjectID string   `json:"projectId"`
    GroupID   *string  `json:"groupId"`
    Limit     *int     `json:"limit"`
    Tags      []string `json:"tags"`
}

// SetConfigParams は memory.set_config のパラメータ
type SetConfigParams struct {
    Embedder *EmbedderParams `json:"embedder"`
}

type EmbedderParams struct {
    Provider *string `json:"provider"`
    Model    *string `json:"model"`
    BaseURL  *string `json:"baseUrl"`
    APIKey   *string `json:"apiKey"`
}

// UpsertGlobalParams は memory.upsert_global のパラメータ
type UpsertGlobalParams struct {
    ProjectID string  `json:"projectId"`
    Key       string  `json:"key"`
    Value     any     `json:"value"`
    UpdatedAt *string `json:"updatedAt"`
}

// GetGlobalParams は memory.get_global のパラメータ
type GetGlobalParams struct {
    ProjectID string `json:"projectId"`
    Key       string `json:"key"`
}
```

### エラーハンドリング

serviceから返されるエラーをJSON-RPCエラーコードに変換:

| サービスエラー | JSON-RPCエラーコード |
|---------------|---------------------|
| service.ErrNoteNotFound | -32003 (NotFound) |
| service.ErrInvalidGlobalKey | -32002 (InvalidKeyPrefix) |
| service.ErrProjectIDRequired | -32602 (InvalidParams) |
| service.ErrGroupIDRequired | -32602 (InvalidParams) |
| service.ErrInvalidGroupID | -32602 (InvalidParams) |
| service.ErrTextRequired | -32602 (InvalidParams) |
| service.ErrQueryRequired | -32602 (InvalidParams) |
| service.ErrIDRequired | -32602 (InvalidParams) |
| service.ErrInvalidTimeFormat | -32602 (InvalidParams) |
| service.ErrKeyRequired（get_global key空） | -32602 (InvalidParams) |
| embedder.ErrAPIKeyRequired | -32001 (APIKeyMissing) |
| config/store由来のエラー | -32603 (InternalError) |
| その他 | -32603 (InternalError) |

### key必須チェック

`memory.get_global` で key が空文字の場合、Handler側で InvalidParams を返す。
（service層では空keyをnot found扱いにしているため、Handler側で事前検証が必要）

### Handle処理フロー

```
1. requestBytesをmodel.Requestにアンマーシャル
   - 失敗 → ParseError (-32700)
2. JSONRPCバージョン確認（"2.0"必須）
   - 不一致 → InvalidRequest (-32600)
3. methodに基づいてディスパッチ
   - 未知のmethod → MethodNotFound (-32601)
4. paramsを適切な型にマッピング
   - 失敗 → InvalidParams (-32602)
5. サービスメソッド呼び出し
   - エラー → 上記エラーマッピング
6. 結果をJSON-RPCレスポンスとして返却
```

## テストケース

### handler_test.go

1. **パース系**
   - 不正なJSON → ParseError
   - jsonrpc != "2.0" → InvalidRequest
   - method未指定 → InvalidRequest

2. **ディスパッチ系**
   - 未知のmethod → MethodNotFound
   - 正しいmethod → 対応するサービスメソッドが呼ばれる

3. **memory.add_note**
   - 正常系: id, namespaceが返る
   - projectId未指定 → InvalidParams
   - groupId未指定 → InvalidParams
   - text未指定 → InvalidParams

4. **memory.search**
   - 正常系: namespace, resultsが返る
   - topK未指定時にdefault 5が使われる
   - projectId未指定 → InvalidParams
   - query未指定 → InvalidParams

5. **memory.get**
   - 正常系: noteが返る
   - id未指定 → InvalidParams
   - 存在しないid → NotFound

6. **memory.update**
   - 正常系: {ok: true}が返る
   - id未指定 → InvalidParams
   - 存在しないid → NotFound

7. **memory.list_recent**
   - 正常系: namespace, itemsが返る
   - projectId未指定 → InvalidParams

8. **memory.get_config**
   - 正常系: transportDefaults, embedder, store, pathsが返る

9. **memory.set_config**
   - 正常系: {ok: true, effectiveNamespace}が返る
   - apiKey未設定でOpenAI使用 → APIKeyMissing

10. **memory.upsert_global**
    - 正常系: {ok: true, id, namespace}が返る
    - keyが"global."で始まらない → InvalidKeyPrefix
    - projectId未指定 → InvalidParams

11. **memory.get_global**
    - 正常系（存在する）: found: true, value返る
    - 正常系（存在しない）: found: false
    - projectId未指定 → InvalidParams
    - key未指定 → InvalidParams

12. **エラーハンドリング追加テスト**
    - groupId不正文字（memory.add_note） → InvalidParams
    - createdAt不正形式（memory.add_note） → InvalidParams
    - since/until不正形式（memory.search） → InvalidParams
    - updatedAt不正形式（memory.upsert_global） → InvalidParams
    - memory.updateでtitle/source/metadataをnullでクリア → 正常に更新される
    - memory.updateでgroupIdをnull → InvalidParams
    - set_config で embedder未指定 → {ok: true}が返る（変更なし）

## 実装順序

1. params.go: パラメータ型定義
2. handler.go: Handler構造体、New関数、Handle関数（ディスパッチロジック）
3. methods.go: 各メソッドハンドラーの実装
4. handler_test.go: テスト（モックサービス使用）

## 依存関係

- internal/model: JSON-RPC型、エラーコード
- internal/service: NoteService, ConfigService, GlobalService インターフェース

## 注意事項

- Handler はtransport非依存。stdio/HTTPどちらからも同じHandlerを使う
- バッチリクエスト（配列形式）は今回は非対応（将来拡張可能）
- IDがnullのリクエスト（notification）も今回は非対応

## 既知の制限事項

### nullクリアの実装について

memory.updateのpatchにおいて、title/source/metadataのnullクリアは以下のように実装:
- JSON-RPCの`null`は空文字列（title/source）または空map（metadata）として渡される
- service層の`*string`/`*map[string]any`では「nil=変更なし」のため、厳密なnullクリアを表現できない
- 将来的にはservice層でnullクリアフラグを明示的にサポートする設計変更が望ましい

### 型不正時の動作

- patch内の型が不正な場合（例: titleに数値）は無視される（未指定扱い）
- エラーにせず寛容に処理する方針としている
