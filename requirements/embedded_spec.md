あなたは Go エンジニアです。
ローカルで動作する MCP メモリサーバーを Go で実装してください。
目的は「会話メモ/仕様/ノート」を埋め込み検索できるローカルRAGメモリ基盤を、Claude Code から JSON-RPC 2.0 で呼べるようにすることです。

# 参考（Auto-Claude）
この設計は Auto-Claude の Memory/Graphiti の思想（会話や作業中の “insights/episodes” を蓄積して検索し、次回以降に再利用する）を参考にする。
- Memory/Graphiti の用途説明（episodes/insights 等）:
  https://github.com/AndyMik90/Auto-Claude/blob/53111dbb95e7737565859fe7c711f0a5a384469c/CLAUDE.md#L314-L356
- provider構成と実装場所の説明:
  https://github.com/AndyMik90/Auto-Claude/blob/53111dbb95e7737565859fe7c711f0a5a384469c/.planning/codebase/INTEGRATIONS.md#L61-L105
- Ollama embedder を OpenAI互換的に扱う発想の参考:
  https://github.com/AndyMik90/Auto-Claude/blob/ae4e48e8bf0950f77a204ee1c716bcd57b2ce553/apps/backend/integrations/graphiti/providers_pkg/embedder_providers/ollama_embedder.py#L84-L127
- provider切替時の embedding 次元 mismatch の問題意識の参考:
  https://github.com/AndyMik90/Auto-Claude/blob/ae4e48e8bf0950f77a204ee1c716bcd57b2ce553/apps/backend/integrations/graphiti/migrate_embeddings.py#L1-L36

# 0. 大方針（重要）
- JSON-RPCの method 実装は transport 非依存にし、stdio/HTTP の2transportを薄いアダプタとして載せる。
- 将来的に「会話全文 ingest（conversation ingest）」を追加できる拡張余地を設計に入れる。
  - ただし現時点では会話全文を保存しない（メモリー用途として要点のみ）。
  - 代わりに、ノートの metadata に会話参照（conversationId等）を保存できる設計にする。

# 1. Transport（stdio / HTTP の切替）
- transport は起動オプションで切り替え可能:
  - stdio: 標準入出力で JSON-RPC 2.0（NDJSON: 1行=1request/response）
  - http: localhost HTTPで JSON-RPC 2.0（POST /rpc）
- デフォルト transport は stdio。
- ただし「ビルド時にデフォルトtransportを差し替え可能」にする：
  - Go の -ldflags でデフォルト値を変更できること。
  - 例: `go build -ldflags "-X main.defaultTransport=http"`
- CLI 例:
  - `mcp-memory serve`（デフォルトtransport）
  - `mcp-memory serve --transport stdio`
  - `mcp-memory serve --transport http --host 127.0.0.1 --port 8765`

## NDJSON仕様（stdio）
- 1リクエスト = 1行を厳守（改行で区切る）
- JSON内のtext等に含まれる改行は `\n` でエスケープすること
- 複数行にまたがるJSONは不可

## HTTP transport
- CORS設定を可能にする（設定ファイルで許可オリジンを指定）
- デフォルトはCORS無効（localhost直接アクセスのみ）

# 2. マルチプロジェクト対応（projectId/groupId）
- 複数プロジェクトで共用するため、すべてのメモリは projectId と groupId を必ず持つ。
- search では projectId が必須、groupId は任意（nullならフィルタしない）。
- list_recent では projectId が必須、groupId は任意。

## groupId の意味
- "global": 共通設定 / 全体方針 / persona / 規約
- "feature-1": 機能単位の話題
- "task-1": タスク単位の話題

## groupId の制約
- 許容文字: 英数字、ハイフン `-`、アンダースコア `_`
- 大小文字は区別する（正規化なし）
- "global" は予約値（共通設定用）

# 3. projectId（パス）正規化（macOS/Linux対応）
- projectId はプロジェクトIDでもパスでもよいが、パスの場合はサーバー側で正規化する。
- 正規化ルール（macOS/Linux）:
  - "~" をホームに展開
  - 絶対パス化（filepath.Abs）
  - シンボリックリンク解決（filepath.EvalSymlinks）※失敗時はAbsまで
  - パス区切りはOS標準に統一
  - 最終的に canonicalProjectId を内部キーとして使う
- すべてのレスポンスでは projectId は canonicalProjectId を返す（ブレ防止）。
- 注意: macOSとLinuxでパスが違うため、同一projectの共有は前提にしない（別project扱いでOK）。

# 4. Embedding Provider 抽象化（設定で切替）
- Embedding Provider は設定で切り替え可能に抽象化:
  - provider: "openai" | "ollama" | "local"
- 最初のバージョンは "openai" を実装する。
  - OpenAI: embeddings endpoint を net/http で叩く（apiKey必須。未設定ならJSON-RPC error）
  - Ollama: 将来実装（stub: NotImplemented）
  - local: 将来実装（stub: NotImplemented）
- Embedding の次元 mismatch を避けるため、namespace を provider+model+dim から自動生成する。
  - 例: "{provider}:{model}:{dim}"
  - dim は初回埋め込み時にprovider応答から取得し、設定に記録する
- provider変更時は namespace が変わる（古いデータは残るが別namespace）。

## Ollama実装（将来）
- Ollama: http://localhost:11434 の /api/embeddings を叩く実装を用意
  - request: { "model": "<model>", "prompt": "<text>" }
  - response から embedding を取得（ベクトル長からdimを判定）

# 5. VectorStore 抽象化（設定で切替）
- Vector store も設定で切り替え可能に抽象化:
  - store: "chroma" | "sqlite" | "qdrant" | "faiss"
- 最初のバージョンは "chroma" を実装する（ベクトル検索に最適化・ローカル永続）。
  - Chroma: https://www.trychroma.com/
  - Go クライアント: github.com/amikos-tech/chroma-go を使用
  - embeddings を Chroma に保存し、ベクトル検索を実行
  - 後で SQLite/Qdrant/FAISS に差し替え可能なインターフェースにする
- SQLite実装は将来オプションとして追加可能（軽量用途向け）

## Chroma運用上の注意
- Chromaサーバーをローカルで起動する必要がある（デフォルト: localhost:8000）
- または embedded mode（インプロセス）での利用も可能
- 将来的に cleanup / archive 機能を追加可能な設計にしておく

# 6. データモデル（拡張しやすく）
- Note の最低限フィールド:
  - id (uuid)
  - projectId (canonical)
  - groupId
  - title (nullable)
  - text
  - tags ([]string)
  - source (nullable)
  - createdAt (ISO8601 UTC) - nullの場合はサーバー側で現在時刻を設定
- 将来の conversation ingest に備え、metadata を持てる設計にする:
  - metadata map[string]any を保存可能にする（SQLiteでは JSON カラム等）
  - 例: metadata = { "conversationId": "...", "messageRange": "12-20" }
  - 今は会話全文は保存しないが、参照情報だけ入れられるようにする

## 時刻の扱い
- 全ての時刻はUTCで統一
- ISO8601形式（例: "2024-01-15T10:30:00Z"）
- since/until の境界条件: since <= createdAt < until

# 7. global 設定（AIだけでなくシステムからも更新）— 具体的なキー
- groupId="global" は予約値として扱う。
- システム（AI以外）からも “設定” を機械的に更新・取得できるよう、global KV API を提供する。
- 以下のキーは「最低限サポートすべき標準キー」としてREADMEとSkillにも明記する:

## 標準キー（required-to-support）
- "global.memory.embedder.provider"   // 値: "ollama" | "openai"
- "global.memory.embedder.model"      // 値: string（例: "nomic-embed-text"）
- "global.memory.groupDefaults"       // 値: object（例: { "featurePrefix": "feature-", "taskPrefix": "task-" } 等）
- "global.project.conventions"        // 値: string または object（文章でもJSONでもOK）

- これらの値は「プロジェクトの推奨/方針」として保存する（システム側が set_config と同期してもよい）。
- 注意: MCPサーバー自身の「実稼働 embedder 設定」と混同しないこと。
  - 実稼働 embedder 設定（実際に埋め込み生成に効く）は memory.set_config で変更される
  - global.memory.embedder.* は「推奨/方針」として保存され、Skillが参照して判断に使う

# 8. JSON-RPC methods（必須）
1) memory.add_note
Input:
{
  "projectId": string,
  "groupId": string,
  "title": string|null,
  "text": string,
  "tags": []string|null,
  "source": string|null,
  "createdAt": string|null,
  "metadata": object|null
}
Output:
{ "id": string, "namespace": string }

2) memory.search
Input:
{
  "projectId": string,
  "groupId": string|null,
  "query": string,
  "topK": number|null,     // default 5
  "tags": []string|null,   // AND検索、空配列/nullはフィルタなし、大小文字区別
  "since": string|null,    // UTC ISO8601
  "until": string|null     // UTC ISO8601
}
Output:
{
  "namespace": string,
  "results": [             // score降順でソート
    {
      "id": string,
      "projectId": string,
      "groupId": string,
      "title": string|null,
      "text": string,
      "tags": []string,
      "source": string|null,
      "createdAt": string,
      "score": number,     // 0-1に正規化（1が最も類似）
      "metadata": object|null
    }
  ]
}

3) memory.get
Input: { "id": string }
Output:
{
  "id": string,
  "projectId": string,
  "groupId": string,
  "title": string|null,
  "text": string,
  "tags": []string,
  "source": string|null,
  "createdAt": string,
  "namespace": string,
  "metadata": object|null
}

4) memory.update
Input:
{
  "id": string,
  "patch": {
    "title"?: string|null,
    "text"?: string,        // text変更時のみ再埋め込み
    "tags"?: []string,
    "source"?: string|null,
    "groupId"?: string,     // groupId変更は再埋め込み不要（メタデータのみ）
    "metadata"?: object|null
  }
}
Output: { "ok": true }

5) memory.list_recent
Input:
{
  "projectId": string,
  "groupId": string|null,
  "limit": number|null,
  "tags": []string|null
}
Output:
{
  "namespace": string,
  "items": [
    {
      "id": string,
      "projectId": string,
      "groupId": string,
      "title": string|null,
      "text": string,
      "tags": []string,
      "source": string|null,
      "createdAt": string,
      "namespace": string,
      "metadata": object|null
    }
  ]
}

6) memory.get_config
Output:
{
  "transportDefaults": { "defaultTransport": string }, // build-time defaultも含める
  "embedder": { "provider": string, "model": string, "dim": number, "baseUrl"?: string },
  "store": { "type": string, "path"?: string, "url"?: string },
  "paths": { "configPath": string, "dataDir": string }
}

7) memory.set_config
Input: embedder設定のみ変更可能
{
  "embedder"?: {
    "provider"?: string,   // "ollama" | "openai" | "local"
    "model"?: string,
    "baseUrl"?: string,
    "apiKey"?: string
  }
}
// store/paths の変更は再起動が必要（set_configでは不可）
Output: { "ok": true, "effectiveNamespace": string }

8) memory.upsert_global
Input:
{
  "projectId": string,
  "key": string,            // "global." プレフィックス必須、それ以外はエラー
  "value": any,             // JSON値
  "updatedAt": string|null  // nullならサーバー側で現在時刻を設定
}
// key例: "global.persona", "global.project.conventions", "global.memory.embedder.provider"
Output:
{ "ok": true, "id": string, "namespace": string }

9) memory.get_global
Input:
{ "projectId": string, "key": string }
Output:
{
  "namespace": string,
  "found": boolean,
  "id": string|null,
  "value": any|null,
  "updatedAt": string|null
}

# 9. 設定ファイル & データ保存
- 設定ファイルはローカルに保存する（例: ~/.local-mcp-memory/config.json）。
- データディレクトリもローカル（例: ~/.local-mcp-memory/data/）。
- OpenAI apiKey は設定ファイル保存でもよいがREADMEで注意喚起する。
  - 可能なら環境変数で上書きも用意（任意）。

# 10. JSON-RPC 2.0 の厳密さ
- request: {"jsonrpc":"2.0","id":...,"method":...,"params":...}
- response: result または error を返す
- error は code/message/data を適切に（invalid params, method not found, internal error 等）

# 11. テスト / README
- README:
  - 各タスク完了時に該当機能の動作確認方法を追記すること
  - 最終的に以下を含むこと:
    - stdio と HTTP の起動例
    - curl で HTTP JSON-RPC を叩く例
    - stdio の NDJSON 例
    - OpenAI apiKey 設定方法（環境変数 or 設定ファイル）
    - provider切替で namespace が変わる説明（embedding dim mismatch回避のため）
    - Ollama embedder は将来実装予定の旨
- go test のスモークテスト:
  - projectId="~/tmp/demo" を渡して canonical化されること
  - add_note 2件（groupId="global" と "feature-1"）
  - search(projectId必須, groupId="feature-1") が返る
  - search(projectId必須, groupId=null) でも返る
  - upsert_global("global.memory.embedder.provider","openai") → get_global で取れる
  - upsert_global("global.memory.embedder.model","text-embedding-3-small") → get_global で取れる
  - upsert_global("global.memory.groupDefaults", {...}) → get_global で取れる
  - upsert_global("global.project.conventions", "文章") → get_global で取れる

# 12. 期待するディレクトリ構成（例）
- cmd/mcp-memory/main.go
- internal/config/...
- internal/model/...
- internal/service/...
- internal/embedder/...
- internal/store/...
- internal/jsonrpc/...
- internal/transport/stdio/...
- internal/transport/http/...

最初に「設計（ディレクトリ構成・主要IF・データスキーマ・namespace戦略）」を提示してから実装に入ってください。