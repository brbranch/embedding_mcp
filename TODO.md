# TODO - MCP Memory Server Implementation

## ステータス凡例
- [ ] 未着手
- [x] 完了
- 🚧 着手中

## タスク一覧

### Phase 1: プロジェクト基盤

#### 1. プロジェクト初期化とディレクトリ構造
- [ ] go.mod作成（Go 1.22+）
- [ ] ディレクトリ構造作成
  ```
  cmd/mcp-memory/
  internal/config/
  internal/model/
  internal/service/
  internal/embedder/
  internal/store/
  internal/jsonrpc/
  internal/transport/stdio/
  internal/transport/http/
  ```
- [ ] README.md スケルトン作成（プロジェクト概要、ビルド方法）

**完了条件**: `go build ./...` がエラーなく成功すること、README.mdにビルド方法が記載されていること

**README更新方針**: 以降の各タスク完了時に、該当機能の動作確認方法をREADMEに追記すること

---

### Phase 2: コアコンポーネント

#### 2. データモデル定義 (internal/model)
- [ ] Note構造体
  - id (uuid)
  - projectId (canonical)
  - groupId (許容文字: 英数字、`-`、`_`。大小文字区別。"global"は予約値)
  - title (nullable)
  - text
  - tags ([]string)
  - source (nullable)
  - createdAt (ISO8601 UTC、nullならサーバー側で現在時刻設定)
  - metadata (map[string]any、SQLiteではJSONカラム)
- [ ] GlobalConfig構造体（id, projectId, key, value, updatedAt）
- [ ] Config構造体
  - transportDefaults: { defaultTransport: string }
  - embedder: { provider, model, dim, baseUrl?, apiKey? }
  - store: { type, path?, url? }
  - paths: { configPath, dataDir }
- [ ] JSON-RPC 2.0 Request/Response/Error構造体
  - request: { jsonrpc: "2.0", id, method, params }
  - response: result または error
  - error: { code, message, data }

**完了条件**: `go test ./internal/model/...` が成功すること

---

#### 3. 設定管理 (internal/config)
- [ ] 設定ファイルの読み書き（~/.local-mcp-memory/config.json）
- [ ] projectId正規化
  - "~" をホームに展開
  - 絶対パス化（filepath.Abs）
  - シンボリックリンク解決（filepath.EvalSymlinks）※失敗時はAbsまで
  - レスポンスでは常にcanonicalProjectIdを返す
- [ ] namespace生成（`{provider}:{model}:{dim}` 形式）
  - dimは初回埋め込み時にprovider応答から取得し記録
  - provider変更時はnamespace変更（古いデータは残るが別namespace）
- [ ] 環境変数によるapiKey上書き（OpenAI用）

**完了条件**: `go test ./internal/config/...` が成功し、projectId正規化が動作すること

---

#### 4. VectorStore抽象化とChroma実装 (internal/store)
- [ ] Store interface定義（AddNote, Search, Get, Update, Delete, ListRecent, UpsertGlobal, GetGlobal）
- [ ] Chroma実装（github.com/amikos-tech/chroma-go使用）
  - Chromaサーバー接続（デフォルト: localhost:8000）
  - または embedded mode（インプロセス）対応
- [ ] ベクトル検索実装
- [ ] namespace分離対応（Chromaのcollection単位）
- [ ] 検索フィルタ実装
  - tags: AND検索、空配列/nullはフィルタなし、大小文字区別
  - since/until: UTC ISO8601、境界条件は `since <= createdAt < until`

**完了条件**: `go test ./internal/store/...` が成功し、CRUD+ベクトル検索+フィルタが動作すること

---

#### 5. Embedder抽象化とOpenAI実装 (internal/embedder)
- [ ] Embedder interface定義（Embed(text) -> []float64, GetDimension() -> int）
- [ ] OpenAI embedder実装
  - embeddings endpoint を net/http で呼び出し
  - apiKey必須（未設定ならJSON-RPC error）
- [ ] Ollama embedder stub（NotImplemented error返却、将来実装）
- [ ] local embedder stub（NotImplemented error返却）
- [ ] 初回埋め込み時のdim取得・設定ファイルへの記録

**完了条件**: `go test ./internal/embedder/...` が成功すること（OpenAI embedderでembedding取得確認）

---

### Phase 3: ビジネスロジック

#### 6. サービス層 (internal/service)
- [ ] NoteService
  - add_note: 埋め込み生成→Store保存、{ id, namespace }返却
  - search: 埋め込み生成→cosine検索、score降順ソート(0-1正規化)
  - get: ID指定で1件取得
  - update: patch適用、text変更時のみ再埋め込み
  - list_recent: createdAt降順でlimit件取得
- [ ] ConfigService
  - get_config: transportDefaults/embedder/store/paths返却（build-time default含む）
  - set_config: embedder設定のみ変更可（store/pathsは再起動必要）、effectiveNamespace返却
- [ ] GlobalService
  - upsert_global: key制約("global."プレフィックス必須、それ以外はerror)、{ ok, id, namespace }返却
  - get_global: { namespace, found, id?, value?, updatedAt? }返却
  - 標準キー: global.memory.embedder.provider, global.memory.embedder.model, global.memory.groupDefaults, global.project.conventions
- [ ] 時刻処理: 全てUTC、ISO8601形式

**完了条件**: `go test ./internal/service/...` が成功すること

---

### Phase 4: JSON-RPC層

#### 7. JSON-RPCハンドラー (internal/jsonrpc)
- [ ] JSON-RPC 2.0パーサー実装
- [ ] method dispatcher実装
- [ ] 各メソッドハンドラー登録（入出力仕様は embedded_spec.md #8 参照）
  - memory.add_note
  - memory.search (topK default=5)
  - memory.get
  - memory.update
  - memory.list_recent
  - memory.get_config
  - memory.set_config
  - memory.upsert_global
  - memory.get_global
- [ ] エラーハンドリング
  - invalid params (-32602)
  - method not found (-32601)
  - internal error (-32603)
  - カスタムエラー（apiKey未設定、invalid key prefix等）

**完了条件**: `go test ./internal/jsonrpc/...` が成功し、全9メソッドのJSON-RPC呼び出しが動作すること

---

### Phase 5: Transport層

#### 8. stdio transport (internal/transport/stdio)
- [ ] NDJSON形式の入出力
  - 1リクエスト = 1行を厳守（改行で区切る）
  - JSON内のtext等に含まれる改行は `\n` でエスケープ
  - 複数行にまたがるJSONは不可
- [ ] graceful shutdown（SIGINT/SIGTERM対応）

**完了条件**: stdio経由でJSON-RPCリクエストを送り、正しいレスポンスが返ること

---

#### 9. HTTP transport (internal/transport/http)
- [ ] POST /rpc エンドポイント
- [ ] CORS設定
  - 設定ファイルで許可オリジン指定可能
  - デフォルトはCORS無効（localhost直接アクセスのみ）
- [ ] graceful shutdown

**完了条件**: `curl -X POST http://localhost:8765/rpc` でJSON-RPCが動作すること

---

### Phase 6: CLI

#### 10. CLIエントリポイント (cmd/mcp-memory)
- [ ] serveコマンド実装
- [ ] --transport オプション（stdio/http）
- [ ] --host, --port オプション（HTTP用）
- [ ] -ldflags でデフォルトtransport切替対応
  - 例: `go build -ldflags "-X main.defaultTransport=http"`
- [ ] シグナルハンドリング（SIGINT/SIGTERM）

**完了条件**: 以下が動作すること
- `go run ./cmd/mcp-memory serve` でstdio起動
- `go run ./cmd/mcp-memory serve --transport http --port 8765` でHTTP起動

---

### Phase 7: 統合テスト

#### 11. E2Eテスト/スモークテスト
- [ ] projectId="~/tmp/demo" の正規化確認（canonical化されること）
- [ ] add_note 2件（groupId="global" と "feature-1"）
- [ ] search(projectId必須, groupId="feature-1") が返る
- [ ] search(projectId必須, groupId=null) でも返る
- [ ] upsert_global/get_global テスト
  - "global.memory.embedder.provider" = "openai"
  - "global.memory.embedder.model" = "text-embedding-3-small"
  - "global.memory.groupDefaults" = { "featurePrefix": "feature-", "taskPrefix": "task-" }
  - "global.project.conventions" = "文章"
- [ ] upsert_global で "global." プレフィックスなしはエラーになること

**完了条件**: `go test ./... -tags=e2e` が成功すること

---

### Phase 8: ドキュメント最終確認

#### 12. README最終確認・整理
- [ ] 以下の項目がREADMEに記載されていることを確認:
  - stdio起動例: `mcp-memory serve`
  - HTTP起動例: `mcp-memory serve --transport http --host 127.0.0.1 --port 8765`
  - curl でHTTP JSON-RPCを叩く例
  - stdio の NDJSON例（1行1JSON、改行エスケープ）
  - OpenAI apiKey設定方法（環境変数 or 設定ファイル）
  - provider切替でnamespaceが変わる説明（embedding dim mismatch回避のため）
  - Ollama embedderは将来実装予定の旨
  - Chromaのセットアップ方法（サーバー起動 or embedded mode）
  - OpenAI apiKeyの注意喚起（設定ファイル保存時のセキュリティ）
- [ ] 不足項目があれば追記
- [ ] 全体の構成・読みやすさを確認

**完了条件**: README.mdが上記項目を全て含み、整理されていること

---

### Phase 9: Skill定義

#### 13. Claude Code用Skill定義 (.claude/skills/memory)
- [ ] SKILL.md作成（embedded_skill_spec.md に基づく）
- [ ] projectIdの扱い
  - プロジェクトルートパスをprojectIdとして渡す
  - レスポンスのcanonical projectIdを以降の呼び出しで使用
- [ ] フィルタ要件
  - search: projectId必須、groupId任意（nullはフィルタなし）
  - list_recent: projectId必須、groupId任意（nullは全groupから取得）
- [ ] groupIdの決め方
  - デフォルト: "global"
  - 機能実装中: "feature-xxx"
  - タスク単位: "task-xxx"
- [ ] 検索タイミング定義
  - 仕様/方針/規約: search(groupId="global") → search(groupId=null)
  - 機能/タスク進行: search(groupId="feature-x") → search(groupId="global")
  - 直近状況が必要: list_recent(groupId指定 or null)
- [ ] 保存タイミング定義
  - decision/spec/gotcha/glossary → add_note
  - metadataにconversationId等を入れる（将来の全文ingest拡張用）
  - persona/共通規約 → upsert_global (key例: global.persona, global.project.conventions)
- [ ] セッション開始時のglobal keys取得
  - global.memory.embedder.provider
  - global.memory.embedder.model
  - global.memory.groupDefaults
  - global.project.conventions
  - 未設定時: サイレントにデフォルト動作を続行（エラーにしない）
- [ ] globalとconfigの矛盾時
  - 「方針（global）と実設定（config）がズレている」と明示
  - どちらを正にするか確認、同期（set_config/upsert_global）を提案
- [ ] 矛盾検出時のフロー定義
  - 検索結果が矛盾 → 明示してユーザーに確認
  - 更新時は memory.update または upsert_global で上書き
  - 必要に応じて superseded を tags に付与

**完了条件**: `/memory` スキルでClaude Codeからメモリ操作ができること

---

### Phase 10: クライアントライブラリ（optional）

#### 14. Pythonクライアント (clients/python)
- [ ] mcp_memory_client.py 作成
  - MCPMemoryClient クラス（HTTP JSON-RPC呼び出し）
  - 全9メソッド対応（add_note, search, get, update, list_recent, get_config, set_config, upsert_global, get_global）
  - 型ヒント付き
  - 接続設定（base_url, timeout）、エラーハンドリング方針は実装時に決定
- [ ] LangGraph Tool定義サンプル
  - @tool デコレータでの定義例
  - memory_search, memory_add_note 等
  - 対象LangGraphバージョン、ツール登録方法の詳細は実装時に決定
- [ ] pyproject.toml / setup.py
- [ ] 使用例ドキュメント

**完了条件**: LangGraphからmcp-memoryのメソッドを呼び出せること

**注記**: 本フェーズはoptional。LangGraph統合の詳細（バージョン、認証、接続設定、エラー処理方針）は実装着手時に要件を確定する。

---

### Phase 11: 追加Store/Embedder実装（optional）

#### 15. SQLite VectorStore実装 (internal/store)
- [ ] SQLite実装（modernc.org/sqlite使用、cgo不要）
- [ ] cosine類似度による全件スキャン検索
- [ ] embeddings を SQLite に保存（軽量用途向け）
- [ ] 5,000件超過時の警告ログ出力

**完了条件**: `go test ./internal/store/...` でSQLite実装のテストが成功すること

---

#### 16. Ollama Embedder実装 (internal/embedder)
- [ ] Ollama embedder実装
  - endpoint: http://localhost:11434/api/embeddings
  - request: { "model": "<model>", "prompt": "<text>" }
  - responseからembeddingを取得（ベクトル長からdim判定）
- [ ] Ollamaが無い/起動してない場合のエラーハンドリング

**完了条件**: `go test ./internal/embedder/...` でOllama実装のテストが成功すること（Ollama起動環境で確認）

**注記**: 本フェーズはoptional。Store/Embedderは抽象化されているため、必要に応じて追加実装可能。

---

## 依存関係

```
1 → 2 → 3 → 4, 5 → 6 → 7 → 8, 9 → 10 → 11 → 12 → 13
         ↘     ↗                              ↓
          並行可能                      14, 15, 16 (optional)
```

- Phase 1完了後にPhase 2開始
- Phase 2内の4（Store）と5（Embedder）は並行実装可能
- Phase 3以降は順次依存
- Phase 9はサーバー完成後に実施
- Phase 10（タスク14）はHTTP transport完成後いつでも実施可能（optional）
- Phase 11（タスク15, 16）はStore/Embedder interface完成後いつでも実施可能（optional）

## 仕様参照
- サーバー仕様: `requirements/embedded_spec.md`
- Skill仕様: `requirements/embedded_skill_spec.md`
