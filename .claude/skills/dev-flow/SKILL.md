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

---

## 内部ステップ管理

スキルは以下のステップを順番に実行する（ユーザーが個別に呼び出す必要はない）:

| Step | 内容 | 担当エージェント | レビュー必須 |
|------|------|------------------|--------------|
| 1 | Worktree作成 | - | - |
| 2 | 実装計画作成 | 設計者 (Opus) | - |
| 3 | 設計レビュー | Copilot | ✅ |
| 4 | テストコード作成 | テスト実装者 (Sonnet) | - |
| 5 | テストレビュー | Copilot | ✅ |
| 6 | 本実装 | 実装者 (Sonnet) | - |
| 7 | エスカレーション対応 | テックリーダー (Opus) | ※必要時のみ |
| 8 | E2Eテスト | E2E担当者 (Sonnet) | - |
| 9 | README更新 | ドキュメント担当 (Opus) | - |
| 10 | 全体レビュー | Copilot | ✅ |
| 11 | リリース | - | - |

---

## 各ステップのチェックリスト

**重要**: 各ステップ完了時に、次のステップへ進む前に必ずチェックリストを確認すること。

### Step 1: Worktree作成 チェックリスト

- [ ] worktree が正常に作成された
- [ ] feature ブランチが作成された
- [ ] 進捗ファイル (`docs/progress/phase{N}-task{M}.json`) が作成された
- [ ] TODO.md の該当タスクが `[ ]` → `🚧` に更新された

### Step 2: 実装計画作成 チェックリスト

- [ ] `docs/plans/phase{N}-task{M}.md` が作成された
- [ ] TODO.md の該当タスクの要件を全て含んでいる
- [ ] requirements/ の仕様と整合している
- [ ] 関数シグネチャが明記されている（該当する場合）
- [ ] テストケース一覧がある
- [ ] コミット & プッシュ済み

### Step 3: 設計レビュー チェックリスト

- [ ] **Copilot レビューを実行した**（レビュー必須）
- [ ] レビュー指摘があれば対応した
- [ ] LGTM を取得した（または指摘なし）
- [ ] 修正があればコミット & プッシュ済み

### Step 4: テストコード作成 チェックリスト

- [ ] テストファイルが作成された（`*_test.go`）
- [ ] 実装計画のテストケースを全てカバーしている
- [ ] テストが実行可能（コンパイルエラーなし）
- [ ] コミット & プッシュ済み
- [ ] （スキップした場合）スキップ理由を記録した

### Step 5: テストレビュー チェックリスト

- [ ] **Copilot レビューを実行した**（レビュー必須、テストコードがある場合）
- [ ] レビュー指摘があれば対応した
- [ ] LGTM を取得した（または指摘なし）
- [ ] 修正があればコミット & プッシュ済み
- [ ] （スキップした場合）スキップ理由を記録した

### Step 6: 本実装 チェックリスト

- [ ] 実装計画に基づいて実装した
- [ ] 全てのテストがパスした（`go test ./...`）
- [ ] コミット & プッシュ済み
- [ ] PR が Ready for Review 状態になった

### Step 7: エスカレーション対応 チェックリスト

- [ ] エスカレーションが必要か判断した
- [ ] （必要な場合）テックリーダーに報告し、対応した
- [ ] （不要な場合）「エスカレーション不要」と記録した

### Step 8: E2Eテスト チェックリスト

- [ ] E2E テストを実行した（該当する場合）
- [ ] テスト結果を確認した
- [ ] （スキップした場合）スキップ理由を記録した

### Step 9: README更新 チェックリスト

- [ ] README.md を確認した
- [ ] 該当機能の動作確認方法を追記した（必要な場合）
- [ ] コミット & プッシュ済み（変更があれば）
- [ ] （変更不要の場合）「README更新不要」と記録した

### Step 10: 全体レビュー チェックリスト

- [ ] **Copilot レビューを実行した**（レビュー必須）
- [ ] 完了条件を全て満たしている
- [ ] レビュー指摘があれば対応した
- [ ] LGTM を取得した
- [ ] 修正があればコミット & プッシュ済み

### Step 11: リリース チェックリスト

Step 11 は複数のサブステップで構成される。詳細は「Step 11: リリースフロー詳細」を参照。

---

## Step 11: リリースフロー詳細

**重要**: 全体レビューでLGTMが出たら、ユーザーへの確認なしに自動でリリースフローを最後まで実行する。

### 11-1. フロー全体チェックリスト確認

リリース前に、フロー全体のチェックリストを確認する:

```markdown
## フロー全体チェックリスト

### 必須項目（漏れがあれば切り戻し）
- [ ] Step 2: 実装計画が作成された
- [ ] Step 3: 設計レビューを実行した ⚠️レビュー必須
- [ ] Step 4: テストコードが作成された（または意図的スキップ）
- [ ] Step 5: テストレビューを実行した ⚠️レビュー必須（テストがある場合）
- [ ] Step 6: 本実装が完了した
- [ ] Step 8: E2Eテストを実行した（または意図的スキップ）
- [ ] Step 9: README更新を確認した
- [ ] Step 10: 全体レビューを実行した ⚠️レビュー必須

### 意図的スキップの記録
スキップしたステップがあれば記録:
- Step {N}: {スキップ理由}
```

### 11-2. 切り戻し判断

チェックリスト確認で漏れが見つかった場合:

**切り戻しが必要なケース**:
1. **レビュー漏れ**（最重要）
   - Step 3（設計レビュー）、Step 5（テストレビュー）、Step 10（全体レビュー）を飛ばした場合
   - 例外: レビュー対象がない場合（テストコードがない場合の Step 5 など）
2. **実装漏れ**
   - 実装計画、テストコード、本実装が未完了の場合

**切り戻し手順**:
1. 漏れたステップまで `currentStep` を戻す
2. 該当ステップから再実行
3. 以降のステップを順次実行

**意図的スキップの場合**:
- 切り戻し不要
- スキップ理由を記録し、リリース完了後にユーザーに報告

### 11-3. main へマージ

```bash
gh pr merge {PR番号} --squash --delete-branch
```

### 11-4. ローカルクリーンアップ

```bash
# worktree 削除
git worktree remove {worktreeパス}

# ローカルブランチ削除（残っていれば）
git branch -d {ブランチ名}

# main を最新化
git pull origin main
```

### 11-5. 進捗ファイル更新 & プッシュ

```bash
# 進捗ファイルを completed に更新
# docs/progress/phase{N}-task{M}.json の status を "completed" に変更

# TODO.md を更新（🚧 → [x]）

# コミット & プッシュ
git add docs/progress/ TODO.md
git commit -m "Complete Phase{N} Task{M}: {タスク名}"
git push origin main
```

### 11-6. GitHub Release 作成

```bash
# タグ名: v{phase}.{task}.0 （例: v1.1.0）
# または Phase0 の場合: v0.{task}.0

gh release create v{phase}.{task}.0 \
  --title "Phase {N} Task {M}: {タスク名}" \
  --notes "## 変更内容

- {主な変更点1}
- {主な変更点2}

## PR
- #{PR番号}
"
```

### 11-7. スキップ報告（該当する場合）

意図的にスキップしたステップがあれば、ユーザーに報告:

```
## 意図的にスキップしたステップ

- Step {N}: {スキップ理由}

これらは意図的なスキップであり、問題ありません。
```

### 11-8. 完了通知

```bash
terminal-notifier -title "Dev Flow" -message "Phase{N} Task{M} 完了 (v{phase}.{task}.0)" -sound default
```

---

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
  "skippedSteps": [
    {"step": 4, "reason": "スケルトンのみのためテスト不要"},
    {"step": 5, "reason": "テストコードがないためレビュー不要"}
  ],
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
- **完了タイミング**: Step 11 完了時に `status: "completed"` に変更し、mainにプッシュ
- **責任者**: スキルが自動で管理（手動編集不可）

---

## TODO.mdの自動更新

- タスク開始時: `[ ]` → `🚧`
- タスク完了時: `🚧` → `[x]`

---

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

---

## ユーザーへの確認タイミング

以下のタイミングでユーザーに確認を求める:

1. **タスク開始時**: どのタスクを開始/再開するか
2. **レビュー指摘時**: 指摘への対応方針
3. **エスカレーション時**: 問題の対応方針
4. **切り戻し発生時**: 漏れが見つかり切り戻しが必要な場合

**注意**: マージ＆リリースはユーザー確認なしに自動で実行する（全体レビューでLGTMが出ていれば問題なしと判断）

---

## エラー時の動作

- レビューでRequest Changes → 指摘内容を表示し、対応方針を確認
- テスト失敗 → エスカレーションするか確認
- PR作成失敗 → エラー内容を表示し、リトライするか確認
- マージ失敗 → エラー内容を表示し、対応方針を確認

---

## 完了時

```bash
terminal-notifier -title "Dev Flow" -message "Phase{N} Task{M} 完了 (v{phase}.{task}.0)" -sound default
```

---

## 参照ドキュメント

- 開発フロー仕様: `docs/development-flow.md`
- TODO: `TODO.md`
- 要件仕様: `requirements/embedded_spec.md`
- Skill仕様: `requirements/embedded_skill_spec.md`

---

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
