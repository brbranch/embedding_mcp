あなたは Claude Code の運用 Skill を設計する担当です。
ローカルの MCP メモリサーバー（JSON-RPC）に対してメモを保存・検索し、回答品質を上げます。
将来的に conversation ingest を追加できるよう、会話参照（conversationId等）を metadata に持てる前提で運用を定義してください。

# 前提: tools
- memory.add_note, memory.search, memory.get, memory.update, memory.list_recent
- memory.get_config, memory.set_config
- memory.upsert_global, memory.get_global

# フィルタ要件
- search は projectId 必須、groupId 任意（nullはフィルタしない）
- list_recent は projectId 必須、groupId 任意（nullは全groupから取得）

## groupId の制約（embedded_spec.md と同一）
- 許容文字: 英数字、ハイフン `-`、アンダースコア `_`
- 大小文字は区別する（正規化なし）
- 予約値: "global"（共通設定用）
- 例: global / feature-1 / task-auth

## projectId の扱い
- ローカル運用では「プロジェクトルートパス」を projectId として渡す
- サーバー側でcanonical化される（~展開、絶対パス化、symlink解決）
- レスポンスの projectId（canonical）を以降の呼び出しで使用すること

# まず読むべき global keys（プロジェクト推奨/方針）
- セッション開始時（またはプロジェクト切替時）、まず以下を get_global で取得する（あれば適用/提案に反映）:
  - global.memory.embedder.provider（ollama/openai）
  - global.memory.embedder.model
  - global.memory.groupDefaults（feature/task命名ルール等）
  - global.project.conventions（文章でもJSONでもOK）
- 取得できない場合（未設定）: サイレントにデフォルト動作を続行（エラーにしない、ユーザーに確認しない）
  - Skill側で必要に応じて判断し、適切なデフォルト値を使用
- これらの値が MCP の実稼働設定（memory.get_config）と矛盾している場合:
  - 「方針（global）と実設定（config）がズレている」と明示し、どちらを正にするか確認する
  - システム側で同期（set_config/upsert_global）する運用を提案してもよい

# Skillで定義すること
1) groupId の決め方
- ユーザーが指定しなければ暫定で "global"、必要に応じて確認
- 機能実装中は "feature-xxx"、タスク単位は "task-xxx" を推奨

2) 検索タイミング（推奨手順）
- 仕様/方針/規約/好み/ペルソナが関係する話題:
  1) memory.search(projectId, groupId="global", topK=5)
  2) 不足なら memory.search(projectId, groupId=null, topK=8)
- 機能/タスクを進めるとき:
  1) memory.search(projectId, groupId="<feature-x|task-x>", topK=5)
  2) 矛盾回避が必要なら memory.search(projectId, groupId="global", topK=3)
- 直近状況が必要なら memory.list_recent(projectId必須, groupId指定 or null)

3) 保存タイミング（要点のみ）
- decision/spec/gotcha/glossary を見つけたら memory.add_note
- ノートの metadata に conversationId 等が分かる場合は入れる（将来の全文ingest拡張のため）
- persona やプロジェクト共通規約は memory.upsert_global:
  - key 例: "global.persona", "global.project.conventions"
  - value は JSON で構造化してよい（language, tone, preferences, constraints 等）

4) 矛盾の扱い
- 検索結果が矛盾していたら、矛盾を明示してユーザーに確認。
- 決定が更新される場合は memory.update または upsert_global で上書きし、必要なら superseded を tags に付与。

# 成果物
- Claude Code に貼れる Skill 定義（箇条書き）
- 典型フロー2例（tool呼び出し込み）
  A) 新しい仕様相談（feature-1）:
     - search(projectId, "feature-1") → 必要なら search(projectId, "global") → add_note(projectId, "feature-1")
  B) 過去決定が怪しい:
     - search(projectId, null) → 矛盾指摘 → update / upsert_global