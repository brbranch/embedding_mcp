# Phase 8 Task 12: README最終確認・整理 - 実装計画

## 概要

READMEに必要な項目が全て記載されているか確認し、不足があれば追記する。

## 現状分析

### チェック項目と現状

| # | 項目 | 現状 | 対応 |
|---|------|------|------|
| 1 | stdio起動例: `mcp-memory serve` | ✅ L36 に記載あり | 不要 |
| 2 | HTTP起動例: `mcp-memory serve --transport http --host 127.0.0.1 --port 8765` | ⚠️ L42 に `--host` なし | 修正 |
| 3 | curl でHTTP JSON-RPCを叩く例 | ✅ L394-396 に記載あり | 不要 |
| 4 | stdio の NDJSON例（1行1JSON、改行エスケープ） | ⚠️ 説明あるが具体例なし | 追記 |
| 5 | OpenAI apiKey設定方法（環境変数 or 設定ファイル） | ✅ L139-144 に環境変数、L126 に設定ファイル | 不要 |
| 6 | provider切替でnamespaceが変わる説明 | ⚠️ L158 に形式説明あるが「なぜ変わるか」の理由なし | 追記 |
| 7 | Ollama embedderは将来実装予定の旨 | ✅ L221 に記載あり | 不要 |
| 8 | Chromaのセットアップ方法 | ❌ 記載なし | 追記 |
| 9 | OpenAI apiKeyの注意喚起（セキュリティ） | ❌ 記載なし | 追記 |

### 全体構成確認

現在のセクション構成:
1. 概要
2. 機能
3. ビルド方法（起動、CLIオプション、ビルド時設定、シグナル）
4. データモデル（Note, GlobalConfig）
5. 設定管理（ファイル、環境変数、projectId正規化、namespace）
6. VectorStore
7. Embedder
8. Service Layer
9. JSON-RPC Handler
10. Transport（stdio, HTTP）
11. テスト
12. 開発状況

→ 構成は適切。不足項目を既存セクションに追記する。

## 実装ステップ

### Step 1: HTTP起動例に `--host` を追加

**場所**: L42
**変更前**:
```bash
./mcp-memory serve --transport http --port 8765
```
**変更後**:
```bash
./mcp-memory serve --transport http --host 127.0.0.1 --port 8765
```

### Step 2: stdio NDJSON具体例を追加

**場所**: L337 の後（stdio Transport セクション内）

**追加内容**:
```markdown
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
```

### Step 3: namespace説明を拡充

**場所**: L158-160 の namespace セクション

**変更前**:
```markdown
### namespace

埋め込みのnamespaceは `{provider}:{model}:{dim}` 形式で自動生成されます。

例: `openai:text-embedding-3-small:1536`
```

**変更後**:
```markdown
### namespace

埋め込みのnamespaceは `{provider}:{model}:{dim}` 形式で自動生成されます。

例: `openai:text-embedding-3-small:1536`

**重要**: providerやmodelを変更すると、namespaceも変わります。これは異なる埋め込みモデル間での次元数（dim）の不一致を防ぐためです。古いnamespaceのデータはそのまま残りますが、新しいnamespaceからは検索されません。同じデータを新しいモデルで検索したい場合は、再度 `add_note` で追加してください。
```

### Step 4: Chromaセットアップ方法を追加

**場所**: VectorStoreセクション（L162）の「実装」の後（L192-193付近）

**実装状況の確認結果**:
- `internal/store/chroma.go`: ChromaStoreはスタブ実装のみ（全メソッドが "not yet implemented" を返す）
- `internal/store/memory.go`: MemoryStoreは完全実装済み（テスト用インメモリ）
- 現在のデフォルトはMemoryStoreを使用

**追加内容**（実装状況を反映）:
```markdown
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
```

### Step 5: OpenAI APIキーのセキュリティ注意喚起を追加

**場所**: 設定ファイルセクション（L136-137の後、環境変数セクションの前）

**追加内容**:
```markdown
**セキュリティ注意**: 設定ファイルにAPIキーを保存する場合は、ファイルのパーミッションを適切に設定してください（例: `chmod 600 ~/.local-mcp-memory/config.json`）。可能であれば環境変数での設定を推奨します。
```

### Step 6: 全体の整理

- セクション間の一貫性確認
- 「開発状況」セクションの更新
  - 変更前: 「現在開発中です。」
  - 変更後: 「コア機能は実装済みです。以下は将来実装予定:」+ リスト（ChromaStore完全実装、Ollama embedder、SQLite VectorStore）

## テストケース

このタスクはドキュメント更新のみのため、テストコード作成は不要。

確認項目:
- [ ] 全9項目がREADMEに記載されている
- [ ] 各セクションの説明が整合している
- [ ] コード例が実際に動作する形式になっている

## 完了条件

- README.mdが上記全項目を含み、整理されていること
