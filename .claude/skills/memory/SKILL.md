---
description: プロジェクトメモリを管理するスキル（保存・検索・取得）
allowed-tools: Bash, Read, Write, Edit
---

# memory スキル

## 概要

プロジェクトの決定事項、仕様、ノウハウ、用語集をMCPメモリサーバーに保存・検索するスキル。

**目的**
- 会話セッション間で知識を共有
- プロジェクト全体の決定事項を永続化
- 矛盾を検出して品質を向上

**対象ユーザー**
- Claude Codeで開発作業を行うユーザー

---

## projectIdの扱い

### ルール

1. **初回呼び出し時**: プロジェクトルートパス（例: `/Users/user/git/my-project`）を渡す
2. **canonical化**: サーバー側で以下を実行
   - `~` をホームディレクトリに展開
   - 絶対パス化
   - シンボリックリンク解決
3. **以降の呼び出し**: レスポンスで取得した canonical projectId を使用

### 例

- 入力: `~/git/my-project` または `/Users/user/git/my-project`
- 出力（canonical）: `/Users/user/git/my-project`

### 検出方法

```bash
# git rootを検出（gitリポジトリの場合）
git rev-parse --show-toplevel 2>/dev/null || pwd
```

---

## groupIdの決め方

### 命名ルール

| 状況 | groupId | 例 |
|-----|---------|-----|
| プロジェクト全体設定 | `global` | persona, conventions |
| 機能実装中 | `feature-xxx` | `feature-auth`, `feature-api` |
| タスク単位 | `task-xxx` | `task-123`, `task-refactor` |

### 制約

- 許容文字: 英数字、ハイフン `-`、アンダースコア `_`
- 大小文字区別: あり
- 予約値: `global`（共通設定用）

### デフォルト値

- ユーザーが指定しない場合は `"global"` をデフォルトとする
- 不明な場合はユーザーに確認
  - 「globalでよいですか？それとも特定のfeature/task用ですか？」

---

## セッション開始時の動作

### 初期化手順

1. **プロジェクトルートを検出**
   - git rootまたはカレントディレクトリ

2. **global keysを取得**（未設定でもエラーにしない）
   - `global.memory.embedder.provider` - 推奨embedder provider
   - `global.memory.embedder.model` - 推奨embedder model
   - `global.memory.groupDefaults` - groupId命名ルール
   - `global.project.conventions` - プロジェクト規約

3. **実稼働設定を取得**
   - `memory.get_config` で実際の設定を取得

4. **矛盾チェック**
   - global設定（推奨/方針）とconfig（実稼働）を比較
   - 矛盾があれば明示:
     - 「方針（global）と実設定（config）がズレています」
     - どちらを正にするか確認
     - 同期を提案（`set_config` / `upsert_global`）

---

## 検索タイミング

### 重要: projectIdは必須

全ての検索・取得操作で `projectId` は必須。

### パターンA: 仕様/方針/規約が関係する話題

1. まず `search(projectId, groupId="global", topK=5)` で共通設定を検索
2. 不足なら `search(projectId, groupId=null, topK=8)` で全group横断検索

### パターンB: 機能/タスクを進めるとき

1. まず `search(projectId, groupId="feature-x", topK=5)` で該当機能の情報を検索
2. 矛盾回避が必要なら `search(projectId, groupId="global", topK=3)` で共通設定も検索

### パターンC: 直近の状況が必要なとき

- `list_recent(projectId, groupId指定 or null)` で最新のノートを取得

### パターンD: 特定ノートを直接取得

- `memory.get(projectId, id)` でIDを指定して取得

### 検索フィルタオプション

- `tags`: タグでフィルタ（AND検索、空配列/nullはフィルタなし）
- `since`/`until`: 期間でフィルタ（UTC ISO8601、`since <= createdAt < until`）

---

## 保存タイミング

### ノートとして保存（add_note）

以下を検出したら保存を提案:
- **decision**: 技術的な決定事項
- **spec**: 仕様の詳細
- **gotcha**: ハマりポイント、注意事項
- **glossary**: 用語定義

**metadata に含める情報**（将来の全文ingest拡張用）:
- `conversationId`: 会話ID（取得可能な場合）
- `timestamp`: 保存時刻

### グローバル設定として保存（upsert_global）

以下はグローバル設定として保存:
- **persona**: AIの振る舞い設定
  - key: `global.persona`
- **conventions**: プロジェクト規約
  - key: `global.project.conventions`
- **embedder推奨設定**:
  - key: `global.memory.embedder.provider`
  - key: `global.memory.embedder.model`
- **groupDefaults**: groupId命名ルール
  - key: `global.memory.groupDefaults`

**注意**: keyは `"global."` プレフィックス必須（それ以外はエラー）

### 設定の変更（set_config）

embedder設定を変更する場合:
- `memory.set_config(embedder: { provider, model, ... })`
- **注意**: store/pathsは再起動が必要なため変更不可
- `effectiveNamespace` が返却される

---

## 矛盾検出と解決

### 検索結果が矛盾している場合

1. **矛盾を明示**:
   ```
   以下の情報が矛盾しています:
   - Note A (2024-01-01): XXXはYYYとする
   - Note B (2024-01-15): XXXはZZZとする
   ```

2. **ユーザーに確認**:
   「どちらが正しいですか？」

3. **解決**:
   - 正しい方を更新（`memory.update`）
   - 古い方に `superseded` タグを付与

### global設定とconfigが矛盾している場合

1. **矛盾を明示**:
   ```
   方針（global）と実設定（config）がズレています:
   - global.memory.embedder.provider: ollama
   - config.embedder.provider: openai
   ```

2. **同期を提案**:
   - 「globalに合わせる（`set_config`）」
   - 「configに合わせる（`upsert_global`）」
   - 「現状維持」

---

## MCP呼び出し方法

### HTTP transport（推奨）

MCPサーバーがHTTP transportで起動している場合:

```bash
curl -X POST http://localhost:8765/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "memory.search",
    "params": {
      "projectId": "/Users/user/git/my-project",
      "query": "認証 仕様",
      "topK": 5
    }
  }'
```

### stdio transport

MCPサーバーがstdio transportで起動している場合:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"memory.search","params":{"projectId":"/path/to/project","query":"検索クエリ"}}' | mcp-memory serve
```

**注意**: NDJSON形式（1リクエスト = 1行）を厳守

---

## 典型フロー例

### フローA: 新しい仕様を相談して保存

```
ユーザー: 「認証機能の仕様を決めたい」

1. 既存情報を検索
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "memory.search",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "groupId": "feature-auth",
         "query": "認証 仕様",
         "topK": 5
       }
     }'

   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 2,
       "method": "memory.search",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "groupId": "global",
         "query": "認証 セキュリティ",
         "topK": 3
       }
     }'

2. 検索結果を踏まえてユーザーと議論

3. 決定事項を保存
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 3,
       "method": "memory.add_note",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "groupId": "feature-auth",
         "title": "認証方式の決定",
         "text": "JWT + Refresh Tokenを採用。有効期限は...",
         "tags": ["decision", "auth", "security"],
         "metadata": {"conversationId": "abc123"}
       }
     }'

   レスポンス: {"id": "note-xxx", "namespace": "openai:text-embedding-3-small:1536"}
```

### フローB: 過去の決定が怪しい場合

```
ユーザー: 「前に決めたAPI設計、矛盾してない？」

1. 全groupを横断検索
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "memory.search",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "groupId": null,
         "query": "API設計 エンドポイント",
         "topK": 10
       }
     }'

2. 矛盾を検出して明示
   「以下の情報が矛盾しています:
    - Note A (feature-api): RESTful設計を採用
    - Note B (task-v2): GraphQL APIに移行」

3. ユーザーに確認
   「どちらが最新の方針ですか？」

4. 解決（GraphQLが正とする場合）
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 2,
       "method": "memory.update",
       "params": {
         "id": "note-a-id",
         "patch": {
           "tags": ["decision", "api", "superseded"]
         }
       }
     }'

   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 3,
       "method": "memory.add_note",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "groupId": "global",
         "title": "API設計方針の統一",
         "text": "GraphQL APIを正式採用。REST APIは廃止予定。",
         "tags": ["decision", "api", "official"]
       }
     }'
```

### フローC: セッション開始時の初期化

```
1. プロジェクトルートを検出
   PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

2. global keysを取得
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d "{
       \"jsonrpc\": \"2.0\",
       \"id\": 1,
       \"method\": \"memory.get_global\",
       \"params\": {
         \"projectId\": \"$PROJECT_ROOT\",
         \"key\": \"global.memory.embedder.provider\"
       }
     }"

   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d "{
       \"jsonrpc\": \"2.0\",
       \"id\": 2,
       \"method\": \"memory.get_global\",
       \"params\": {
         \"projectId\": \"$PROJECT_ROOT\",
         \"key\": \"global.memory.embedder.model\"
       }
     }"

   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d "{
       \"jsonrpc\": \"2.0\",
       \"id\": 3,
       \"method\": \"memory.get_global\",
       \"params\": {
         \"projectId\": \"$PROJECT_ROOT\",
         \"key\": \"global.memory.groupDefaults\"
       }
     }"

   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d "{
       \"jsonrpc\": \"2.0\",
       \"id\": 4,
       \"method\": \"memory.get_global\",
       \"params\": {
         \"projectId\": \"$PROJECT_ROOT\",
         \"key\": \"global.project.conventions\"
       }
     }"

3. 実稼働設定を取得
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc": "2.0", "id": 5, "method": "memory.get_config", "params": {}}'

4. 矛盾チェック（global.memory.embedder.* vs config.embedder.*）

5. 矛盾があれば報告、なければサイレントに続行
```

### フローD: グローバル設定の保存

```
ユーザー: 「このプロジェクトでは日本語でコメントを書くルールにしたい」

1. 規約として保存
   curl -X POST http://localhost:8765/rpc \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "memory.upsert_global",
       "params": {
         "projectId": "/Users/user/git/my-project",
         "key": "global.project.conventions",
         "value": {
           "language": "ja",
           "comments": "日本語でコメントを記述する",
           "naming": "変数名・関数名は英語"
         }
       }
     }'

   レスポンス: {"ok": true, "id": "global-xxx", "namespace": "..."}
```

---

## 利用可能なJSON-RPCメソッド

### memory.add_note

ノートを追加

**Input**:
```json
{
  "projectId": "string",
  "groupId": "string",
  "title": "string|null",
  "text": "string",
  "tags": ["string"]|null,
  "source": "string|null",
  "createdAt": "string|null",
  "metadata": "object|null"
}
```

**Output**:
```json
{
  "id": "string",
  "namespace": "string"
}
```

### memory.search

ノートを検索

**Input**:
```json
{
  "projectId": "string",
  "groupId": "string|null",
  "query": "string",
  "topK": "number|null",
  "tags": ["string"]|null,
  "since": "string|null",
  "until": "string|null"
}
```

**Output**:
```json
{
  "namespace": "string",
  "results": [
    {
      "id": "string",
      "projectId": "string",
      "groupId": "string",
      "title": "string|null",
      "text": "string",
      "tags": ["string"],
      "source": "string|null",
      "createdAt": "string",
      "score": "number",
      "metadata": "object|null"
    }
  ]
}
```

### memory.get

ノートをIDで取得

**Input**:
```json
{
  "id": "string"
}
```

**Output**:
```json
{
  "id": "string",
  "projectId": "string",
  "groupId": "string",
  "title": "string|null",
  "text": "string",
  "tags": ["string"],
  "source": "string|null",
  "createdAt": "string",
  "namespace": "string",
  "metadata": "object|null"
}
```

### memory.update

ノートを更新

**Input**:
```json
{
  "id": "string",
  "patch": {
    "title": "string|null",
    "text": "string",
    "tags": ["string"],
    "source": "string|null",
    "groupId": "string",
    "metadata": "object|null"
  }
}
```

**Output**:
```json
{
  "ok": true
}
```

### memory.list_recent

最近のノートを取得

**Input**:
```json
{
  "projectId": "string",
  "groupId": "string|null",
  "limit": "number|null",
  "tags": ["string"]|null
}
```

**Output**:
```json
{
  "namespace": "string",
  "items": [
    {
      "id": "string",
      "projectId": "string",
      "groupId": "string",
      "title": "string|null",
      "text": "string",
      "tags": ["string"],
      "source": "string|null",
      "createdAt": "string",
      "namespace": "string",
      "metadata": "object|null"
    }
  ]
}
```

### memory.get_config

実稼働設定を取得

**Input**: なし

**Output**:
```json
{
  "transportDefaults": {
    "defaultTransport": "string"
  },
  "embedder": {
    "provider": "string",
    "model": "string",
    "dim": "number",
    "baseUrl": "string"
  },
  "store": {
    "type": "string",
    "path": "string",
    "url": "string"
  },
  "paths": {
    "configPath": "string",
    "dataDir": "string"
  }
}
```

### memory.set_config

embedder設定を変更

**Input**:
```json
{
  "embedder": {
    "provider": "string",
    "model": "string",
    "baseUrl": "string",
    "apiKey": "string"
  }
}
```

**Output**:
```json
{
  "ok": true,
  "effectiveNamespace": "string"
}
```

### memory.upsert_global

グローバル設定を保存/更新

**Input**:
```json
{
  "projectId": "string",
  "key": "string",
  "value": "any",
  "updatedAt": "string|null"
}
```

**Output**:
```json
{
  "ok": true,
  "id": "string",
  "namespace": "string"
}
```

### memory.get_global

グローバル設定を取得

**Input**:
```json
{
  "projectId": "string",
  "key": "string"
}
```

**Output**:
```json
{
  "namespace": "string",
  "found": "boolean",
  "id": "string|null",
  "value": "any|null",
  "updatedAt": "string|null"
}
```

---

## 注意事項

### MCPサーバー起動

MCPサーバーが起動していない場合はエラーになります。

**HTTP transport**:
```bash
mcp-memory serve --transport http --host 127.0.0.1 --port 8765
```

**stdio transport**:
```bash
mcp-memory serve --transport stdio
```

### OpenAI API Key

OpenAI providerを使用する場合、API Keyの設定が必要です。

**環境変数**:
```bash
export OPENAI_API_KEY="sk-..."
```

**設定ファイル**:
```json
{
  "embedder": {
    "provider": "openai",
    "model": "text-embedding-3-small",
    "apiKey": "sk-..."
  }
}
```

### namespace変更

provider/modelを変更するとnamespaceが変わります。古いノートは別namespaceに残ります（検索できなくなる）。

**例**:
- OpenAI text-embedding-3-small: `openai:text-embedding-3-small:1536`
- Ollama nomic-embed-text: `ollama:nomic-embed-text:768`
