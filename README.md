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

### Step 3: 動作確認（stdio モード）

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"memory.get_config","params":{}}' | ./mcp-memory serve
```

JSON形式のレスポンスが返ってくれば成功です。

### Step 4: Claude Codeへの MCP 登録

Claude Code の設定ファイル `~/.claude/settings.json` に追加：

```json
{
  "mcpServers": {
    "mcp-memory": {
      "command": "/path/to/mcp-memory",
      "args": ["serve"],
      "env": {
        "OPENAI_API_KEY": "sk-..."
      }
    }
  }
}
```

**設定項目の説明:**

| 項目 | 説明 | 例 |
|------|------|-----|
| キー名 | MCPサーバーの識別名 | `"mcp-memory"` |
| `command` | 実行ファイルの絶対パス | `/Users/yourname/embedding_mcp/mcp-memory` |
| `args` | コマンドライン引数 | `["serve"]` |
| `env` | 環境変数（オプション） | `{"OPENAI_API_KEY": "sk-..."}` |

### Step 5: CLAUDE.md に運用ルールを追記（重要）

`~/.claude/CLAUDE.md`（グローバル）または プロジェクトの `CLAUDE.md` に以下を追記します。これにより、Claude Codeが適切なタイミングでメモリを使うようになります：

```markdown
# mcp-memory 使用ガイド

ユーザーの指示・学び・バグ対応などを保存し、セマンティック検索で参照する。
※ 一時的な雑談・試行錯誤・詳細な要件定義（別ファイル管理）は保存不要

## groupId

| groupId | 用途 |
|---------|------|
| `reviews` | レビュー指摘 |
| `learning` | デグレ・エラーでハマった内容 |
| `research` | 調査依頼の内容 |
| `global` | その他 |

## tags

状態タグ（複数可）:
- `rule`: 「覚えておいて」「徹底して」と言われた内容
- `important`: 「必ず」「絶対に」など強調された内容
- `decision`: 決定事項
- `want`: 将来作りたい内容
- `wip`: 検討中
- `done`: 完了
- `putoff`: 先送り
- `issue`: ユーザー/レビュワーからの指摘
- `duplicate-<N>`: 重複ノート統合時、残す側に付与

関連タグ:
- CLAUDE.mdに記載のディレクトリ名（例: `cmd/mcp-memory`）
- CLAUDE.mdに記載の機能名（あれば）
- 特定コマンド名（例: `copilot`）

## 保存ルール

関連するやりとりを短く要約して保存する。指摘・エラーは原因と解決策をセットで。

**保存する（メインエージェントのみ）:**
- セッション開始時の最初のユーザー発言と回答（MUST）
- プロジェクト方針・決定事項
- 「覚えておいて」「徹底して」と言われた内容
- コマンドエラー発生時（解決方法とセット）
- 計画・プランへの指摘（実装詳細の指摘は除く）
- Web調査の結果
- プロジェクト固有の重要情報（有名サービス名は除く）
- 迷ったらとりあえず保存

**保存しない:**
- Task tool でサブエージェントを呼び出した際の内容
- サブエージェント内部の会話
- 明らかに一時的な雑談

## 更新ルール

- タグ変更が適切と判断した場合
- 参照時に同じ指摘が重複 → `important` タグ追加
- 参照時に類似ノートが3件以上 → 自明なら統合、迷ったら提案

## 削除ルール

- 以前の内容と矛盾する場合 → ユーザーに確認してから削除

## 参照ルール

**セッション開始時:**
- `global` を直近10件取得
- `rule` かつ `important` を取得（topK=50）

**適宜検索:**
- 新しい作業開始時
- ユーザーからの質問時
- コマンドエラー発生時

## 注意事項

- `important` タグが30件超えたらユーザーにエスカレーション
```

### Step 6: 使い始める

Claude Code を再起動し、以下のように使えます：

```
ユーザー: 「このプロジェクトのコーディング規約を覚えておいて」
AI: memory.add_note で規約を保存

ユーザー: 「認証機能について過去に話したことある？」
AI: memory.search で関連メモを検索して回答
```

## CLIオプション

### serve コマンド

| オプション | 短縮形 | デフォルト | 説明 |
|------------|--------|------------|------|
| `--transport` | `-t` | stdio | トランスポート種別: stdio, http |
| `--host` | - | 127.0.0.1 | HTTPバインドホスト |
| `--port` | `-p` | 8765 | HTTPバインドポート |
| `--config` | `-c` | ~/.local-mcp-memory/config.json | 設定ファイルパス |

### search コマンド（ワンショット検索）

MCPサーバーを起動せずに、コマンドラインから直接検索を実行できます。

```bash
# 基本的な使い方
mcp-memory search -p /path/to/project "検索クエリ"

# グループとタグでフィルタ
mcp-memory search -p ~/myproject -g global -k 10 "コーディング規約"

# JSON形式で出力（スクリプト連携用）
mcp-memory search -p /path/to/project -f json "API設計"
```

| オプション | 短縮形 | デフォルト | 説明 |
|------------|--------|------------|------|
| `--project` | `-p` | (必須) | プロジェクトID/パス |
| `--group` | `-g` | (全グループ) | グループID |
| `--top-k` | `-k` | 5 | 取得件数 |
| `--tags` | - | - | タグフィルタ（カンマ区切り） |
| `--format` | `-f` | text | 出力形式: text, json |
| `--config` | `-c` | ~/.local-mcp-memory/config.json | 設定ファイルパス |
| `--stdin` | - | false | stdinからクエリを読み取る |

## SessionStart Hook連携

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

## 設定ファイル

デフォルトパス: `~/.local-mcp-memory/config.json`

### 設定項目一覧

| セクション | 項目 | デフォルト | 説明 |
|------------|------|------------|------|
| embedder | provider | openai | 埋め込みプロバイダ (openai) |
| embedder | model | text-embedding-3-small | 埋め込みモデル名 |
| embedder | apiKey | null | APIキー（環境変数優先） |
| embedder | dim | 0 | 埋め込み次元数（0=自動） |
| store | type | sqlite | ストア種別 (sqlite) |
| store | path | \<dataDir>/memory.db | SQLiteデータベースパス |
| transportDefaults | defaultTransport | stdio | デフォルトトランスポート |
| paths | configPath | ~/.local-mcp-memory/config.json | 設定ファイルパス |
| paths | dataDir | ~/.local-mcp-memory/data | データディレクトリ |

### 設定例

```json
{
  "embedder": {
    "provider": "openai",
    "model": "text-embedding-3-small",
    "apiKey": "sk-..."
  },
  "store": {
    "type": "sqlite",
    "path": "~/.local-mcp-memory/data/memory.db"
  }
}
```

**セキュリティ注意**: 設定ファイルにAPIキーを保存する場合は、ファイルのパーミッションを適切に設定してください（例: `chmod 600 ~/.local-mcp-memory/config.json`）。可能であれば環境変数での設定を推奨します。

### 環境変数

| 環境変数 | 説明 |
|----------|------|
| `OPENAI_API_KEY` | OpenAI APIキー（設定ファイルより優先） |

### namespace

埋め込みのnamespaceは `{provider}:{model}:{dim}` 形式で自動生成されます。

例: `openai:text-embedding-3-small:1536`

**重要**: providerやmodelを変更すると、namespaceも変わります。異なるnamespaceのデータは検索されません。同じデータを新しいモデルで検索したい場合は、再度 `add_note` で追加してください。

## GlobalConfig

プロジェクト単位でグローバル設定を保存できます。AIが参照すべきプロジェクト固有の設定に使用します。

### 用途

- プロジェクト固有の埋め込み設定
- グループのデフォルト設定
- コーディング規約や方針の構造化データ

### 標準キー一覧

| キー | 説明 |
|------|------|
| `global.memory.embedder.provider` | プロジェクト固有の埋め込みプロバイダ |
| `global.memory.embedder.model` | プロジェクト固有の埋め込みモデル |
| `global.memory.groupDefaults` | グループデフォルト設定 |
| `global.project.conventions` | コーディング規約（構造化データ） |

**注意**: キーは必ず `global.` プレフィックスで始める必要があります。

## 対応メソッド一覧

| メソッド | 説明 |
|----------|------|
| `memory.add_note` | ノート追加 |
| `memory.search` | ベクトル検索（topKデフォルト: 5） |
| `memory.get` | ノート取得 |
| `memory.update` | ノート更新 |
| `memory.delete` | ノート/グローバル設定削除（物理削除） |
| `memory.list_recent` | 最新ノート取得 |
| `memory.get_config` | 設定取得 |
| `memory.set_config` | 設定変更 |
| `memory.upsert_global` | グローバル設定upsert |
| `memory.get_global` | グローバル設定取得 |

## エラーコードとトラブルシューティング

| コード | 名前 | 原因 | 対処法 |
|--------|------|------|--------|
| -32700 | Parse Error | 不正なJSON | JSONの構文を確認 |
| -32600 | Invalid Request | 不正なリクエスト | `jsonrpc: "2.0"` を確認 |
| -32601 | Method Not Found | 未知のメソッド | メソッド名を確認（例: `memory.add_note`） |
| -32602 | Invalid Params | 不正なパラメータ | 必須パラメータを確認 |
| -32603 | Internal Error | 内部エラー | ログを確認 |
| -32001 | API Key Missing | APIキー未設定 | `OPENAI_API_KEY` 環境変数または設定ファイルを確認 |
| -32002 | Invalid Key Prefix | `global.`プレフィックスなし | GlobalConfigのキーは `global.` で始める |
| -32003 | Not Found | リソース未検出 | IDが正しいか確認 |
| -32004 | Provider Error | APIリクエスト失敗 | APIキーの有効性、ネットワーク接続を確認 |

### よくあるトラブル

**Q: `memory.search` で結果が返ってこない**

- `projectId` が正しいパスか確認（`~` は展開されます）
- ノートが追加されているか `memory.list_recent` で確認
- 埋め込みモデルを変更した場合、古いノートは検索されません

**Q: サーバーが起動しない**

- 設定ファイルのJSONが正しいか確認
- APIキーが設定されているか確認

**Q: Claude Code がメモリツールを使わない**

- CLAUDE.md に運用ルールを追記してください（Step 5参照）

## Pythonクライアント

`clients/python` に Python クライアントライブラリが含まれています。

### インストール

```bash
cd clients/python
pip install -e .

# LangGraph統合が必要な場合
pip install -e ".[langchain]"
```

### 使用例

```python
from mcp_memory_client import MCPMemoryClient

# クライアント作成（デフォルト: http://localhost:8765）
with MCPMemoryClient() as client:
    # ノート追加
    result = client.add_note(
        project_id="/path/to/project",
        group_id="global",
        text="プロジェクトのコーディング規約: TypeScript + ESLint使用",
        tags=["conventions"],
    )
    print(f"Note ID: {result['id']}")

    # 検索
    results = client.search(
        project_id="/path/to/project",
        query="コーディング規約",
    )
    for note in results.results:
        print(f"- {note.text[:50]}... (score: {note.score})")
```

詳細は `clients/python/README.md` を参照してください。

## テスト

```bash
# ユニットテスト
go test ./...

# E2Eテスト（統合テスト）
go test ./e2e/... -tags=e2e -v
```

## 開発状況

### 実装済み

- **SQLiteStore**: 軽量用途向けSQLite実装（5,000件程度まで推奨、cgo不要）
- **MemoryStore**: インメモリ実装（テスト・開発用）
- **OpenAI Embedder**: OpenAI Embeddings API連携

### 未実装（将来実装予定）

- **ChromaStore**: 大規模用途向けChroma連携
- **Ollama Embedder**: ローカルLLMによる埋め込み生成
- **Local Embedder**: ローカルモデル対応

**現在の推奨構成**:
- 小規模〜中規模（〜5,000件）: **SQLiteStore** + OpenAI Embedder
- 開発・テスト: **MemoryStore** + OpenAI Embedder
