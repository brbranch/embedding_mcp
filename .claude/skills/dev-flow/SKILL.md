---
description: 開発フローを管理するスキル（TODO管理、エージェント呼び出し）
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Task, AskUserQuestion
---

# 開発フロー管理スキル (Dev Flow Skill)

TODO.mdに基づいて開発フローを管理し、各エージェントを呼び出して開発を進める。

## 使用方法

ユーザーは以下のコマンドだけを使う:

```
/dev-flow
```

これだけでOK。スキルが自動で以下を判断する:
- 🚧 着手中のタスクがあれば → 続きから再開
- 着手中がなければ → 最初の未着手タスク `[ ]` を開始

## スキル実行時の動作

### 1. TODO.mdを読み込み、状態を判断

```
🚧 着手中あり → そのタスクの続きを実行
着手中なし → 最初の [ ] タスクを開始
```

### 2. ユーザーに確認

```
Phase {N} Task {M}: {タスク名}
ステータス: {着手中 or 未着手}
次のステップ: {Step X: 〇〇}

続行しますか？
```

### 3. 承認後、該当ステップを実行

各エージェントを呼び出し、PRの作成・レビュー依頼等を自動で行う。

## 内部ステップ管理

スキルは以下のステップを順番に実行する（ユーザーが個別に呼び出す必要はない）:

| Step | 内容 | 担当エージェント |
|------|------|------------------|
| 1 | Worktree作成 | - |
| 2 | 実装計画作成 | 設計者 (Opus) |
| 3 | 設計レビュー | Copilot |
| 4 | テストコード作成 | テスト実装者 (Sonnet) |
| 5 | テストレビュー | Copilot |
| 6 | 本実装 | 実装者 (Sonnet) |
| 7 | エスカレーション対応 | テックリーダー (Opus) ※必要時のみ |
| 8 | E2Eテスト | E2E担当者 (Sonnet) |
| 9 | README更新 | ドキュメント担当 (Opus) |
| 10 | 全体レビュー | Copilot |
| 11 | マージ＆リリース | - |

## ステップ進行の記録

各タスクの進行状況は `docs/progress/phase{N}-task{M}.json` に記録:

```json
{
  "phase": 1,
  "task": 1,
  "taskName": "プロジェクト初期化とディレクトリ構造",
  "currentStep": 4,
  "status": "in_progress",
  "branch": "feature/phase1-task1",
  "worktree": "../embedding_mcp-phase1-task1",
  "designPR": {
    "number": 1,
    "url": "https://github.com/.../pull/1",
    "status": "merged"
  },
  "implPR": {
    "number": 2,
    "url": "https://github.com/.../pull/2",
    "status": "draft"
  },
  "startedAt": "2025-01-26T10:00:00Z",
  "history": [
    {"step": 1, "action": "worktree作成", "completedAt": "..."},
    {"step": 2, "action": "実装計画作成", "completedAt": "..."},
    {"step": 3, "action": "設計レビューLGTM", "completedAt": "..."}
  ]
}
```

### 再開時の動作

1. `docs/progress/` 内の `status: "in_progress"` のファイルを検索
2. **複数ある場合**: `startedAt` が最も新しいものを選択（ユーザーに確認）
3. 見つかったら以下を復元:
   - `worktree` のパスに移動（存在しなければ再作成）
   - `branch` をチェックアウト
   - `currentStep` から再開
   - PRがあれば `designPR.url` / `implPR.url` を参照

### 進捗ファイルの管理ルール

- **作成タイミング**: Step 1（Worktree作成）完了時
- **更新タイミング**: 各Step完了時
- **削除タイミング**: Step 11（マージ＆リリース）完了時に `status: "completed"` に変更
- **責任者**: スキルが自動で管理（手動編集不可）

## TODO.mdの自動更新

- タスク開始時: `[ ]` → `🚧`
- タスク完了時: `🚧` → `[x]`

## エージェント呼び出し（内部用）

スキルが内部で以下を実行する（ユーザーは意識不要）:

### Claude Code エージェント

```bash
# 設計者
claude --model opus --agent .claude/agents/designer.md -p "{指示}"

# テスト実装者
claude --model sonnet --agent .claude/agents/test-implementer.md -p "{指示}"

# 実装者
claude --model sonnet --agent .claude/agents/implementer.md -p "{指示}"

# テックリーダー
claude --model opus --agent .claude/agents/tech-leader.md -p "{指示}"

# E2E担当者
claude --model sonnet --agent .claude/agents/e2e-tester.md -p "{指示}"

# ドキュメント担当者
claude --model opus --agent .claude/agents/documenter.md -p "{指示}"
```

### GitHub Copilot（レビュー）

```bash
copilot --model gpt-5.2-codex --add-dir . --allow-all-tools -p "{レビュープロンプト}"
```

**レビュープロンプトに必ず含めること**:
- 「資料が不足している場合は `[資料不足]` プレフィックスを付けて、どのような資料が必要か明記してください」
- 「日本語で回答してください」

**`[資料不足]` が返ってきた場合**:
1. まず自分で解決を試みる:
   - 該当ファイルをGlob/Grepで検索
   - 既存ドキュメント（requirements/、docs/）から推測
   - コードベースから情報を収集
2. 自分で解決できた場合 → 資料を補完して再レビュー
3. 自分で解決できない場合 → ユーザーに質問（AskUserQuestion）:
   - 「以下の資料が不足しています: {不足内容}」
   - 「{試した内容}を確認しましたが見つかりませんでした」
   - 選択肢: 「資料を提供する」「資料なしで再レビュー」「現状でOK」

## ユーザーへの確認タイミング

以下のタイミングでユーザーに確認を求める:

1. **タスク開始時**: どのタスクを開始/再開するか
2. **レビュー指摘時**: 指摘への対応方針
3. **エスカレーション時**: 問題の対応方針
4. **マージ前**: 最終確認

それ以外は自動で進行する。

## エラー時の動作

- レビューでRequest Changes → 指摘内容を表示し、対応方針を確認
- テスト失敗 → エスカレーションするか確認
- PR作成失敗 → エラー内容を表示し、リトライするか確認

## 完了時

```bash
terminal-notifier -title "Dev Flow" -message "Phase{N} Task{M} 完了" -sound default
```

## 参照ドキュメント

- 開発フロー仕様: `docs/development-flow.md`
- TODO: `TODO.md`
- 要件仕様: `requirements/embedded_spec.md`
- Skill仕様: `requirements/embedded_skill_spec.md`

## エージェント一覧

| 役割 | ファイル | モデル |
|------|----------|--------|
| 設計者 | `.claude/agents/designer.md` | Opus 4.5 |
| レビュワー | `.claude/agents/reviewer.md` | Copilot GPT-5.2-Codex |
| テスト実装者 | `.claude/agents/test-implementer.md` | Sonnet 4.5 |
| 実装者 | `.claude/agents/implementer.md` | Sonnet 4.5 |
| テックリーダー | `.claude/agents/tech-leader.md` | Opus 4.5 |
| E2E担当者 | `.claude/agents/e2e-tester.md` | Sonnet 4.5 |
| ドキュメント担当 | `.claude/agents/documenter.md` | Opus 4.5 |
