# Phase 9 Task 13: Claude Code用Skill定義 実装計画

## 概要

Claude Code用の `/memory` スキルを実装する。MCP Memory Serverを利用して、プロジェクトの決定事項・仕様・ノウハウ等を永続化し、セッション間で知識を活用できるようにする。

---

## 1. 実装するファイル一覧

| ファイル | 説明 |
|---------|------|
| `.claude/skills/memory/SKILL.md` | メインのスキル定義ファイル |

---

## 2. SKILL.mdの構成案

### 2.1 ヘッダー（メタデータ）

```yaml
---
description: プロジェクトメモリを管理するスキル（保存・検索・取得）
allowed-tools: Bash, Read, Write, Edit
---
```

### 2.2 スキル概要

- 目的: プロジェクトの決定事項、仕様、ノウハウ、用語集をMCPメモリサーバーに保存・検索
- 対象ユーザー: Claude Codeで開発作業を行うユーザー

### 2.3 projectIdの扱い

```markdown
## projectIdの扱い

1. 初回呼び出し時: プロジェクトルートパス（例: `/Users/user/git/my-project`）を渡す
2. レスポンスで canonical projectId を取得
3. 以降の呼び出しでは canonical projectId を使用

例:
- 入力: `~/git/my-project` または `/Users/user/git/my-project`
- 出力（canonical）: `/Users/user/git/my-project`（絶対パス、symlink解決済み）
```

### 2.4 groupIdの決め方

```markdown
## groupIdの決め方

| 状況 | groupId | 例 |
|-----|---------|-----|
| デフォルト（プロジェクト全体設定） | `global` | persona, conventions |
| 機能実装中 | `feature-xxx` | `feature-auth`, `feature-api` |
| タスク単位 | `task-xxx` | `task-123`, `task-refactor` |

- groupIdの制約: 英数字、ハイフン `-`、アンダースコア `_` のみ
- 大小文字は区別する
- `global` は予約値（共通設定用）
```

### 2.5 セッション開始時の動作

```markdown
## セッション開始時の動作

1. プロジェクトルートを検出（git rootまたはカレントディレクトリ）
2. 以下のglobal keysを取得（未設定でもエラーにしない）:
   - `global.memory.embedder.provider` - 推奨embedder provider
   - `global.memory.embedder.model` - 推奨embedder model
   - `global.memory.groupDefaults` - groupId命名ルール
   - `global.project.conventions` - プロジェクト規約
3. `memory.get_config` で実稼働設定を取得
4. global設定とconfigに矛盾があれば明示:
   - 「方針（global）と実設定（config）がズレています」
   - どちらを正にするか確認し、同期を提案
```

### 2.6 検索タイミング定義

```markdown
## 検索タイミング

### パターンA: 仕様/方針/規約が関係する話題

1. まず `search(groupId="global", topK=5)` で共通設定を検索
2. 不足なら `search(groupId=null, topK=8)` で全group横断検索

### パターンB: 機能/タスクを進めるとき

1. まず `search(groupId="feature-x", topK=5)` で該当機能の情報を検索
2. 矛盾回避が必要なら `search(groupId="global", topK=3)` で共通設定も検索

### パターンC: 直近の状況が必要なとき

- `list_recent(groupId指定 or null)` で最新のノートを取得
```

### 2.7 保存タイミング定義

```markdown
## 保存タイミング

### ノートとして保存（add_note）

以下を検出したら保存を提案:
- **decision**: 技術的な決定事項
- **spec**: 仕様の詳細
- **gotcha**: ハマりポイント、注意事項
- **glossary**: 用語定義

metadata に含める情報（将来の全文ingest拡張用）:
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
```

### 2.8 矛盾検出と解決フロー

```markdown
## 矛盾検出と解決

### 検索結果が矛盾している場合

1. 矛盾を明示:
   「以下の情報が矛盾しています:
    - Note A (2024-01-01): XXXはYYYとする
    - Note B (2024-01-15): XXXはZZZとする」
2. ユーザーに確認:
   「どちらが正しいですか？」
3. 解決:
   - 正しい方を更新（memory.update）
   - 古い方に `superseded` タグを付与

### global設定とconfigが矛盾している場合

1. 矛盾を明示:
   「方針（global）と実設定（config）がズレています:
    - global.memory.embedder.provider: ollama
    - config.embedder.provider: openai」
2. 同期を提案:
   - 「globalに合わせる（set_config）」
   - 「configに合わせる（upsert_global）」
   - 「現状維持」
```

---

## 3. 典型フロー例（tool呼び出し込み）

### フローA: 新しい仕様を相談して保存

```
ユーザー: 「認証機能の仕様を決めたい」

1. 既存情報を検索
   → memory.search(projectId, groupId="feature-auth", query="認証 仕様", topK=5)
   → memory.search(projectId, groupId="global", query="認証 セキュリティ", topK=3)

2. 検索結果を踏まえてユーザーと議論

3. 決定事項を保存
   → memory.add_note(
       projectId: "/Users/user/git/my-project",
       groupId: "feature-auth",
       title: "認証方式の決定",
       text: "JWT + Refresh Tokenを採用。有効期限は...",
       tags: ["decision", "auth", "security"],
       metadata: { "conversationId": "abc123" }
     )
   → レスポンス: { "id": "note-xxx", "namespace": "openai:text-embedding-3-small:1536" }
```

### フローB: 過去の決定が怪しい場合

```
ユーザー: 「前に決めたAPI設計、矛盾してない？」

1. 全groupを横断検索
   → memory.search(projectId, groupId=null, query="API設計 エンドポイント", topK=10)

2. 矛盾を検出
   「以下の情報が矛盾しています:
    - Note A (feature-api): RESTful設計を採用
    - Note B (task-v2): GraphQL APIに移行」

3. ユーザーに確認
   「どちらが最新の方針ですか？」

4. 解決（GraphQLが正とする場合）
   → memory.update(
       id: "note-a-id",
       patch: { tags: ["decision", "api", "superseded"] }
     )
   → memory.add_note(
       projectId: "/Users/user/git/my-project",
       groupId: "global",
       title: "API設計方針の統一",
       text: "GraphQL APIを正式採用。REST APIは廃止予定。",
       tags: ["decision", "api", "official"]
     )
```

### フローC: セッション開始時の初期化

```
1. プロジェクトルートを検出
   → /Users/user/git/my-project

2. global keysを取得
   → memory.get_global(projectId, key="global.memory.embedder.provider")
   → memory.get_global(projectId, key="global.memory.embedder.model")
   → memory.get_global(projectId, key="global.memory.groupDefaults")
   → memory.get_global(projectId, key="global.project.conventions")

3. 実稼働設定を取得
   → memory.get_config()

4. 矛盾チェック（global.memory.embedder.* vs config.embedder.*）

5. 矛盾があれば報告、なければサイレントに続行
```

### フローD: グローバル設定の保存

```
ユーザー: 「このプロジェクトでは日本語でコメントを書くルールにしたい」

1. 規約として保存
   → memory.upsert_global(
       projectId: "/Users/user/git/my-project",
       key: "global.project.conventions",
       value: {
         "language": "ja",
         "comments": "日本語でコメントを記述する",
         "naming": "変数名・関数名は英語"
       }
     )
   → レスポンス: { "ok": true, "id": "global-xxx", "namespace": "..." }
```

---

## 4. テストケース一覧（動作確認手順）

スキル自体のユニットテストは困難なため、以下の手動動作確認を行う。

### 4.1 前提条件

1. MCP Memory Serverが起動していること
2. OpenAI API Keyが設定されていること
3. Claude Codeで `/memory` スキルが認識されていること

### 4.2 動作確認チェックリスト

| # | 確認項目 | 操作 | 期待結果 |
|---|---------|------|---------|
| 1 | スキル認識 | `/memory` を入力 | スキルが起動する |
| 2 | セッション初期化 | 新規セッションで `/memory` 実行 | global keysの取得が試行される（未設定でもエラーにならない） |
| 3 | ノート保存 | 「決定事項を保存して」と依頼 | `memory.add_note` が呼ばれ、id/namespaceが返る |
| 4 | ノート検索 | 「前に決めた〇〇を教えて」と依頼 | `memory.search` が呼ばれ、関連ノートが返る |
| 5 | groupId指定検索 | 「feature-authの情報を検索して」と依頼 | groupId="feature-auth"で検索される |
| 6 | 全group検索 | 「全体から検索して」と依頼 | groupId=nullで検索される |
| 7 | global設定保存 | 「このルールをプロジェクト規約として保存」と依頼 | `memory.upsert_global` が呼ばれる |
| 8 | global設定取得 | 「プロジェクト規約を確認して」と依頼 | `memory.get_global` で規約が取得される |
| 9 | 矛盾検出 | 矛盾するノートを2件保存後、検索 | 矛盾が明示される |
| 10 | 矛盾解決 | 矛盾を解決するよう依頼 | supersededタグ付与 + 新規ノート作成 |
| 11 | config取得 | 「memory設定を確認して」と依頼 | `memory.get_config` が呼ばれる |
| 12 | config/global矛盾 | global設定とconfigに矛盾がある状態で確認 | 矛盾が明示され、同期が提案される |

### 4.3 エラーケース確認

| # | 確認項目 | 操作 | 期待結果 |
|---|---------|------|---------|
| E1 | サーバー未起動 | サーバー停止状態で操作 | 適切なエラーメッセージ |
| E2 | 不正なgroupId | 「groupId=@invalid」で保存 | バリデーションエラー |
| E3 | 不正なglobal key | 「key=invalid」でupsert_global | エラー（"global."プレフィックス必須） |

---

## 5. 実装時の注意事項

### 5.1 MCP呼び出し方法

```bash
# stdio transport経由でJSON-RPCを送信
echo '{"jsonrpc":"2.0","id":1,"method":"memory.search","params":{"projectId":"/path/to/project","query":"検索クエリ"}}' | mcp-memory serve
```

または HTTP transport:

```bash
curl -X POST http://localhost:8765/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"memory.search","params":{"projectId":"/path/to/project","query":"検索クエリ"}}'
```

### 5.2 projectIdの検出

```bash
# git rootを検出
git rev-parse --show-toplevel 2>/dev/null || pwd
```

### 5.3 allowed-tools

スキルで使用するツール:
- **Bash**: MCP呼び出し（curl/echo）
- **Read**: 結果の確認
- **Write**: 必要に応じてノート内容のファイル出力
- **Edit**: 既存ファイルの編集

---

## 6. 将来の拡張ポイント

1. **conversation ingest**: 会話全文の自動保存（現在はmetadataにconversationIdのみ）
2. **自動タグ付け**: LLMによるノートの自動分類
3. **定期的なクリーンアップ**: 古いノートのアーカイブ
4. **複数プロジェクト対応**: プロジェクト切り替え時の自動初期化

---

## 7. 完了条件

- [x] `.claude/skills/memory/SKILL.md` が作成されている
- [ ] `/memory` スキルでClaude Codeからメモリ操作ができる
- [ ] セッション開始時にglobal keysが取得される（未設定でもエラーにならない）
- [ ] ノートの保存・検索・更新が動作する
- [ ] global設定の保存・取得が動作する
- [ ] 矛盾検出時に適切な対応が提案される

**注**: Claude Codeの再起動が必要な場合は「再起動が必要」と報告してタスク完了とする。
