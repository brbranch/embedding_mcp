# MCP Memory Server

ローカルで動作する MCP メモリサーバー（Go実装）。会話メモ/仕様/ノートの埋め込み検索基盤。

## これがあると何が嬉しいか

### AIの「忘却」問題を解決

Claude Code などのAIアシスタントは、セッションが終わると会話内容を忘れてしまいます。そのため：

- 毎回同じ説明をする必要がある（「このプロジェクトはReactで...」）
- 以前決めたルールや方針を覚えていない
- 過去の議論で出た重要な決定事項が失われる

このMCPメモリサーバーを導入すると、**AIがプロジェクトの文脈を覚え続けられる**ようになります。

### 具体的なメリット

| 課題 | 導入後 |
|------|--------|
| プロジェクトのコーディング規約を毎回説明 | 一度保存すれば自動で参照 |
| 「前に話した機能Xの仕様って何だっけ？」 | セマンティック検索で即座に取得 |
| チーム内で決めた用語の定義がブレる | 用語集をメモリに保存して統一 |
| 過去の設計判断の理由が分からない | 決定事項と理由をセットで記録 |

## AIが何をできるようになるか

### 1. コンテキストを持った回答

```
ユーザー: 「認証機能を追加して」

AIの内部動作:
1. memory.search で過去の認証関連の議論を検索
2. 「JWTを使う」「refresh tokenは7日」という過去の決定を発見
3. その方針に沿った実装を提案
```

### 2. プロジェクト知識の蓄積

AIは以下のような情報を自動的に保存・参照できます：

- **仕様・設計**: 機能の詳細仕様、API設計、データモデル
- **決定事項**: 技術選定の理由、アーキテクチャ判断
- **規約・ルール**: コーディング規約、命名規則、レビュー基準
- **用語集**: プロジェクト固有の用語定義
- **注意点・落とし穴**: 過去にハマった問題とその解決策

### 3. 複数プロジェクトの知識を分離管理

`projectId` でプロジェクトごとに知識を分離。さらに `groupId` で：

- `global`: 全体方針、コーディング規約、用語集
- `feature-xxx`: 機能単位の仕様・議論
- `task-xxx`: タスク単位の作業メモ

### 4. セマンティック検索

単純なキーワード検索ではなく、意味的に近いメモを検索できます：

```
検索クエリ: 「ユーザー認証の方法」
ヒットするメモ: 「ログイン機能ではJWTを採用する」
```

## クイックスタート（導入方法）

### 重要: MCP登録だけでは不十分

MCPサーバーを登録すると、Claude Codeは `memory.add_note` や `memory.search` などのツールを**使える**ようになります。しかし、**いつ使うかは自発的に判断しません**。

| 設定レベル | 効果 |
|------------|------|
| MCP登録のみ | ツールは使えるが、自発的には使わない |
| MCP登録 + CLAUDE.md指示 | 指示に従って使うようになる |

つまり、導入には以下の両方が必要です：
1. MCPサーバーの登録（ツールを使えるようにする）
2. CLAUDE.md への運用ルール追記（いつ使うかを指示する）

---

### Step 1: ビルド

```bash
git clone https://github.com/miistin/embedding_mcp.git
cd embedding_mcp
go build ./cmd/mcp-memory
```

### Step 2: OpenAI APIキーの設定

```bash
export OPENAI_API_KEY="sk-..."
```

または設定ファイル（`~/.local-mcp-memory/config.json`）に記載：

```json
{
  "embedder": {
    "provider": "openai",
    "model": "text-embedding-3-small",
    "apiKey": "sk-..."
  }
}
```

### Step 3: 動作確認（HTTP モード）

```bash
# サーバー起動
./mcp-memory serve --transport http --port 8765
```

別ターミナルで動作確認：

```bash
curl -X POST http://localhost:8765/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}'
```

レスポンスが返ってくれば成功です。

### Step 4: Claude Codeへの MCP 登録

Claude Code の設定ファイル `~/.claude/settings.json` に追加：

```json
{
  "mcpServers": {
    "memory": {
      "command": "/path/to/mcp-memory",
      "args": ["serve"]
    }
  }
}
```

`/path/to/mcp-memory` は実際のパスに置き換えてください（例: `/Users/yourname/embedding_mcp/mcp-memory`）。

### Step 5: CLAUDE.md に運用ルールを追記（重要）

プロジェクトの `CLAUDE.md` に以下を追記します。これにより、Claude Codeが適切なタイミングでメモリを使うようになります：

```markdown
## Memory MCP 運用ルール

MCPサーバー「memory」が利用可能。以下のルールに従って使用すること。

### 検索タイミング
- セッション開始時、プロジェクトの方針・規約が関係する話題では:
  → memory.search(projectId=カレントディレクトリ, groupId="global", topK=5)
- 特定機能の実装中:
  → memory.search(projectId=カレントディレクトリ, groupId="feature-xxx", topK=5)

### 保存タイミング
- 重要な決定・仕様が確定したら:
  → memory.add_note で保存
- 保存すべき情報の例:
  - 技術選定の決定と理由
  - API設計・データモデル
  - コーディング規約
  - 用語の定義
  - ハマった問題と解決策

### groupId の使い分け
- "global": 全体方針、規約、用語集
- "feature-xxx": 機能単位の仕様（例: feature-auth, feature-payment）
- "task-xxx": タスク単位の作業メモ

### tags の付け方
- ["仕様"], ["決定"], ["規約"], ["用語"], ["注意点"] など
```

### Step 6: 使い始める

Claude Code を再起動し、以下のように使えます：

```
ユーザー: 「このプロジェクトのコーディング規約を覚えておいて」
AI: memory.add_note で規約を保存

ユーザー: 「認証機能について過去に話したことある？」
AI: memory.search で関連メモを検索して回答
```

## ユースケース例

### ケース1: プロジェクト規約の記録

```json
// memory.add_note
{
  "projectId": "~/myproject",
  "groupId": "global",
  "title": "コーディング規約",
  "text": "1. 変数名はcamelCase\n2. 関数は単一責任\n3. エラーは早期return",
  "tags": ["規約", "コーディング"]
}
```

### ケース2: 機能仕様の保存

```json
// memory.add_note
{
  "projectId": "~/myproject",
  "groupId": "feature-auth",
  "title": "認証機能の仕様",
  "text": "JWTを使用。access tokenは1時間、refresh tokenは7日。",
  "tags": ["仕様", "認証"]
}
```

### ケース3: 過去の議論を検索

```json
// memory.search
{
  "projectId": "~/myproject",
  "query": "データベースの選定理由",
  "topK": 5
}
```

---

## 概要

Claude Code から JSON-RPC 2.0 で呼び出せるローカル RAG メモリ基盤を提供します。

## 機能

- 会話メモ/仕様/ノートの保存と検索
- プロジェクト単位・グループ単位でのメモ管理
- OpenAI による埋め込み生成（Ollama は将来実装予定）
- MemoryStore（インメモリ）によるベクトル検索（Chroma/SQLite は将来実装予定）

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

### 起動

```bash
# stdio transport（デフォルト）
mcp-memory serve
# または
./mcp-memory serve

# 直接実行（ビルドなし）
go run ./cmd/mcp-memory serve

# HTTP transport
mcp-memory serve --transport http --host 127.0.0.1 --port 8765

# カスタム設定ファイル
./mcp-memory serve --config /path/to/config.json
```

### ワンショット検索コマンド（v9.13.0～）

MCPサーバーを起動せずに、コマンドラインから直接検索を実行できます。SessionStart Hookとの連携に最適です。

```bash
# 基本的な使い方
mcp-memory search -p /path/to/project "検索クエリ"

# グループとタグでフィルタ
mcp-memory search -p ~/myproject -g global -k 10 "コーディング規約"

# JSON形式で出力（スクリプト連携用）
mcp-memory search -p /path/to/project -f json "API設計"

# stdinからクエリを読み取り（セキュリティ向上）
echo "機密クエリ" | mcp-memory search -p /path/to/project --stdin
```

**search オプション:**

| オプション | 短縮形 | デフォルト | 説明 |
|------------|--------|------------|------|
| `--project` | `-p` | (必須) | プロジェクトID/パス |
| `--group` | `-g` | (全グループ) | グループID |
| `--top-k` | `-k` | 5 | 取得件数 |
| `--tags` | - | - | タグフィルタ（カンマ区切り） |
| `--format` | `-f` | text | 出力形式: text, json |
| `--config` | `-c` | ~/.local-mcp-memory/config.json | 設定ファイルパス |
| `--stdin` | - | false | stdinからクエリを読み取る |

**出力例（text形式）:**
```
[1] コーディング規約 (score: 0.92)
    変数名はcamelCase、関数は単一責任...
    tags: 規約, コーディング

[2] API設計方針 (score: 0.85)
    RESTful APIを採用、エンドポイントは...
    tags: 仕様, API
```

**SessionStart Hookでの使用例:**

`~/.claude/settings.json`:
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/memory-init.sh"
          }
        ]
      }
    ]
  }
}
```

`~/.claude/hooks/memory-init.sh`:
```bash
#!/bin/bash
RESULTS=$(/path/to/mcp-memory search \
  --project "$CLAUDE_PROJECT_DIR" \
  --group global \
  --top-k 5 \
  "プロジェクト方針 規約 コーディング")

if [ -n "$RESULTS" ]; then
  echo "## プロジェクトメモリから取得した情報"
  echo ""
  echo "$RESULTS"
fi
exit 0
```

### serve CLI オプション

| オプション | 短縮形 | デフォルト | 説明 |
|------------|--------|------------|------|
| `--transport` | `-t` | stdio | Transport type: stdio, http |
| `--host` | - | 127.0.0.1 | HTTP bind host |
| `--port` | `-p` | 8765 | HTTP bind port |
| `--config` | `-c` | ~/.local-mcp-memory/config.json | Config file path |

### ビルド時デフォルト変更

```bash
# HTTP をデフォルトにしてビルド
go build -ldflags "-X main.defaultTransport=http" -o mcp-memory ./cmd/mcp-memory

# このバイナリは --transport 指定なしでHTTPで起動
./mcp-memory serve
```

### シグナルハンドリング

- `SIGINT` (Ctrl+C): Graceful shutdown
- `SIGTERM`: Graceful shutdown

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
    "dim": 0,
    "apiKey": "sk-..."
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

**注**: `embedder.apiKey` に OpenAI API キーを設定できます。

**セキュリティ注意**: 設定ファイルにAPIキーを保存する場合は、ファイルのパーミッションを適切に設定してください（例: `chmod 600 ~/.local-mcp-memory/config.json`）。可能であれば環境変数での設定を推奨します。

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

**重要**: providerやmodelを変更すると、namespaceも変わります。これは異なる埋め込みモデル間での次元数（dim）の不一致を防ぐためです。古いnamespaceのデータはそのまま残りますが、新しいnamespaceからは検索されません。同じデータを新しいモデルで検索したい場合は、再度 `add_note` で追加してください。

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

### Chromaのセットアップ（将来実装予定）

Chroma連携機能は現在開発中です。完成後は以下の方法で使用できます:

**サーバーモード:**

```bash
# Docker で起動
docker run -d -p 8000:8000 chromadb/chroma

# または pip でインストールして起動
pip install chromadb
chroma run --host localhost --port 8000
```

サーバー起動後、MCP Memory Serverは自動的に `localhost:8000` に接続します。

**現在の状態:**

現在はMemoryStore（インメモリ実装）を使用しています。これはテスト・開発用途向けで、サーバー再起動時にデータは失われます。本番環境ではChroma実装完成後に切り替えてください。

**注**: Embedded mode（インプロセスでのChroma実行）は現在未対応です。

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

## Service Layer

### NoteService

ノートのCRUD操作と検索を提供:

| メソッド | 説明 |
|----------|------|
| `AddNote` | ノート追加（埋め込み生成→Store保存） |
| `Search` | ベクトル検索（クエリ埋め込み→cosine検索） |
| `Get` | ID指定でノート取得 |
| `Update` | ノート更新（text変更時のみ再埋め込み） |
| `ListRecent` | 最新ノート取得（createdAt降順） |

### ConfigService

設定の取得・変更を提供:

| メソッド | 説明 |
|----------|------|
| `GetConfig` | 現在の設定を取得 |
| `SetConfig` | Embedder設定を変更（provider/model変更時はdimリセット） |

### GlobalService

プロジェクト単位のグローバル設定を提供:

| メソッド | 説明 |
|----------|------|
| `UpsertGlobal` | グローバル設定のupsert（`global.`プレフィックス必須） |
| `GetGlobal` | グローバル設定の取得 |

### テスト実行

```bash
go test ./internal/service/...
```

## JSON-RPC Handler

### Handler

JSON-RPC 2.0リクエストをパースし、適切なサービスメソッドにディスパッチ:

```go
handler := jsonrpc.New(noteService, configService, globalService)
response := handler.Handle(ctx, requestBytes)
```

### 対応メソッド

| メソッド | 説明 |
|----------|------|
| `memory.add_note` | ノート追加 |
| `memory.search` | ベクトル検索（topKデフォルト: 5） |
| `memory.get` | ノート取得 |
| `memory.update` | ノート更新 |
| `memory.list_recent` | 最新ノート取得 |
| `memory.get_config` | 設定取得 |
| `memory.set_config` | 設定変更 |
| `memory.upsert_global` | グローバル設定upsert |
| `memory.get_global` | グローバル設定取得 |

### エラーコード

| コード | 名前 | 説明 |
|--------|------|------|
| -32700 | Parse Error | 不正なJSON |
| -32600 | Invalid Request | 不正なリクエスト（jsonrpc != "2.0"等） |
| -32601 | Method Not Found | 未知のメソッド |
| -32602 | Invalid Params | 不正なパラメータ |
| -32603 | Internal Error | 内部エラー |
| -32001 | API Key Missing | APIキー未設定 |
| -32002 | Invalid Key Prefix | global.プレフィックスなし |
| -32003 | Not Found | リソース未検出 |

### テスト実行

```bash
go test ./internal/jsonrpc/...
```

## Transport

### stdio Transport

標準入出力（stdin/stdout）でJSON-RPC 2.0を処理。NDJSON形式（1行1リクエスト/レスポンス）。

**特徴:**
- 1リクエスト = 1行（改行で区切る）
- JSON内の改行は `\n` でエスケープ（複数行JSONは不可）
- 最大バッファサイズ: 1MB
- contextキャンセルで graceful shutdown

**使用例（テスト用）:**

```go
handler := jsonrpc.New(noteService, configService, globalService)
server := stdio.New(handler)
err := server.Run(ctx)
```

**コマンドライン例:**

```bash
# パイプでリクエスト送信
echo '{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}' | ./mcp-memory serve

# 複数リクエスト（1行1リクエスト）
cat <<'EOF' | ./mcp-memory serve
{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}
{"jsonrpc":"2.0","id":2,"method":"memory.add_note","params":{"projectId":"~/test","text":"Hello"}}
EOF
```

**改行を含むテキストの例:**

```bash
# text内の改行は \n でエスケープ（JSONの仕様通り）
echo '{"jsonrpc":"2.0","id":1,"method":"memory.add_note","params":{"projectId":"~/test","text":"Line1\nLine2\nLine3"}}' | ./mcp-memory serve
```

**テスト実行:**

```bash
go test ./internal/transport/stdio/...
```

### HTTP Transport

HTTP経由でJSON-RPC 2.0を処理。`POST /rpc` エンドポイントを提供。

**特徴:**
- エンドポイント: `POST /rpc`
- Content-Type: `application/json`
- CORS設定可能（デフォルトは無効）
- contextキャンセルで graceful shutdown

**設定:**

```go
type Config struct {
    Addr        string   // listen address (例: "127.0.0.1:8765")
    CORSOrigins []string // 許可するオリジンリスト、空ならCORS無効
}
```

**使用例:**

```go
handler := jsonrpc.New(noteService, configService, globalService)
server := http.New(handler, http.Config{
    Addr: "127.0.0.1:8765",
})
err := server.Run(ctx)
```

**テスト実行:**

```bash
go test ./internal/transport/http/...
```

**動作確認:**

```bash
# HTTPサーバー起動
./mcp-memory serve --transport http --port 8765

# 別ターミナルでJSON-RPC呼び出し
curl -X POST http://localhost:8765/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}'

# レスポンス例
# {"jsonrpc":"2.0","id":1,"result":{"transportDefaults":{"defaultTransport":"stdio"},...}}
```

**CORS設定例:**

```go
server := http.New(handler, http.Config{
    Addr:        "127.0.0.1:8765",
    CORSOrigins: []string{"http://localhost:3000", "http://example.com"},
})
```

CORS有効時のレスポンスヘッダー:
- `Access-Control-Allow-Origin`: リクエストのOrigin（許可リストに含まれる場合）
- `Access-Control-Allow-Methods`: POST, OPTIONS
- `Access-Control-Allow-Headers`: Content-Type
- `Vary: Origin`: キャッシュ安全のため

## テスト

### ユニットテスト

```bash
go test ./...
```

### E2Eテスト（統合テスト）

E2Eテストは外部依存なしで実行可能です（MemoryStore + MockEmbedder使用）。

```bash
# E2Eテストのみ実行
go test ./e2e/... -tags=e2e -v

# 全テスト（E2E含む）を実行
go test ./... -tags=e2e
```

E2Eテストで検証される項目:
- projectIdの正規化（`~/tmp/demo` → `/Users/xxx/tmp/demo`）
- add_note（グループ別ノート追加）
- search（groupIdフィルタ、全検索）
- upsert_global/get_global（標準キー、エラーケース）

## 開発状況

コア機能は実装済みです。以下は将来実装予定:

- **ChromaStore完全実装**: 現在はスタブのみ、本番環境向けChroma連携
- **Ollama embedder**: ローカルLLMによる埋め込み生成
- **SQLite VectorStore**: 軽量用途向けの組み込みベクトルストア
