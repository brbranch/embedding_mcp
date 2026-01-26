# 実装計画: ワンショットCLIコマンド

## 概要

Claude Code の SessionStart Hook から効率的にメモリ検索を実行するため、ワンショット実行可能なCLIサブコマンドを追加する。

## 背景

### 現状の問題

1. **`serve` コマンドのみ**: 現在は `mcp-memory serve` でサーバーモード起動のみサポート
2. **Hookとの連携が煩雑**:
   - HTTP: サーバーを別途常駐させる必要あり
   - stdio: 毎回JSON-RPCをパイプで渡す必要あり（初期化コスト1-2秒）

### 目標

SessionStart Hook から以下のようにシンプルに呼び出せるようにする：

```bash
mcp-memory search --project "$CLAUDE_PROJECT_DIR" --group global "プロジェクト方針"
```

---

## 追加するサブコマンド

### 1. `search` コマンド

```bash
mcp-memory search [options] <query>

Options:
  --project, -p  string   プロジェクトID/パス（必須）
  --group, -g    string   グループID（省略時: 全グループ検索）
  --top-k, -k    int      取得件数（デフォルト: 5）
  --tags         string   タグフィルタ（カンマ区切り）
  --format, -f   string   出力形式: text|json（デフォルト: text）
  --config, -c   string   設定ファイルパス
```

**出力例（text形式）**:
```
[1] コーディング規約 (score: 0.92)
    変数名はcamelCase、関数は単一責任...
    tags: 規約, コーディング

[2] API設計方針 (score: 0.85)
    RESTful APIを採用、エンドポイントは...
    tags: 仕様, API
```

**出力例（json形式）**:
```json
{
  "results": [
    {"id": "...", "title": "コーディング規約", "text": "...", "score": 0.92, "tags": ["規約"]}
  ]
}
```

### 2. `add` コマンド（将来検討）

```bash
mcp-memory add --project <path> --group <group> --title <title> [--tags <tags>] <text>
```

※ 今回のスコープ外。Hookでの主要ユースケースは検索のため、まず `search` のみ実装。

---

## 実装設計

### ディレクトリ構成（変更箇所）

```
cmd/mcp-memory/
  main.go           # 変更: サブコマンド分岐追加
  serve.go          # 新規: serveコマンドのロジック分離
  search.go         # 新規: searchコマンド実装
```

### 変更ファイル一覧

| ファイル | 変更種別 | 内容 |
|----------|----------|------|
| `cmd/mcp-memory/main.go` | 修正 | サブコマンド分岐ロジック追加 |
| `cmd/mcp-memory/serve.go` | 新規 | 既存serveロジックを分離 |
| `cmd/mcp-memory/search.go` | 新規 | searchコマンド実装 |

### main.go の変更

```go
func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    switch os.Args[1] {
    case "serve":
        runServe(os.Args[2:])
    case "search":
        runSearch(os.Args[2:])
    case "version":
        printVersion()
    case "help", "-h", "--help":
        printUsage()
    default:
        fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
        os.Exit(1)
    }
}
```

### search.go の実装

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"

    "embedding_mcp/internal/config"
    "embedding_mcp/internal/embedder"
    "embedding_mcp/internal/service"
    "embedding_mcp/internal/store"
)

func runSearch(args []string) {
    fs := flag.NewFlagSet("search", flag.ExitOnError)

    projectID := fs.String("project", "", "Project ID/path (required)")
    groupID := fs.String("group", "", "Group ID (optional)")
    topK := fs.Int("top-k", 5, "Number of results")
    tags := fs.String("tags", "", "Tag filter (comma-separated)")
    format := fs.String("format", "text", "Output format: text|json")
    configPath := fs.String("config", "", "Config file path")

    // 短縮形
    fs.StringVar(projectID, "p", "", "Project ID/path (required)")
    fs.StringVar(groupID, "g", "", "Group ID (optional)")
    fs.IntVar(topK, "k", 5, "Number of results")
    fs.StringVar(format, "f", "text", "Output format: text|json")
    fs.StringVar(configPath, "c", "", "Config file path")

    fs.Parse(args)

    // クエリは残りの引数
    query := strings.Join(fs.Args(), " ")
    if query == "" || *projectID == "" {
        fmt.Fprintln(os.Stderr, "usage: mcp-memory search -p <project> <query>")
        os.Exit(1)
    }

    // 初期化と検索実行
    ctx := context.Background()
    results, err := executeSearch(ctx, *configPath, *projectID, *groupID, query, *topK, *tags)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }

    // 出力
    outputResults(results, *format)
}
```

### 共通初期化ロジックの再利用

既存の初期化ロジック（config読み込み、embedder/store生成、service構築）を共通化：

```go
// internal/bootstrap/bootstrap.go（新規）
package bootstrap

type Services struct {
    NoteService   *service.NoteService
    ConfigService *service.ConfigService
    GlobalService *service.GlobalService
    Config        *config.Config
}

func Initialize(ctx context.Context, configPath string) (*Services, error) {
    // 既存の初期化ロジックをここに集約
}
```

---

## 既存コードへの影響

### 影響範囲

| コンポーネント | 影響 | 詳細 |
|----------------|------|------|
| `cmd/mcp-memory/main.go` | 中 | サブコマンド分岐追加。既存ロジックはserve.goに移動 |
| `internal/config/` | なし | 変更不要 |
| `internal/service/` | なし | 変更不要（そのまま利用） |
| `internal/embedder/` | なし | 変更不要 |
| `internal/store/` | なし | 変更不要 |
| `internal/jsonrpc/` | なし | searchコマンドでは使用しない |
| `internal/transport/` | なし | searchコマンドでは使用しない |

### 後方互換性

- `mcp-memory serve` は従来通り動作
- 既存のMCP連携（stdio/HTTP）に影響なし

---

## セキュリティ考慮事項

### 1. コマンドライン引数の露出

```bash
# ps コマンドで他ユーザーに見える可能性
mcp-memory search --project /secret/path "機密クエリ"
```

**対策**:
- APIキーはコマンドライン引数で渡さない（環境変数/設定ファイルのみ）
- クエリ内容の露出は許容範囲と判断（ローカル実行前提）

### 2. 出力先の考慮

**対策**:
- 結果は stdout に出力（リダイレクト可能）
- エラーは stderr に出力
- ログファイルへの機密情報出力を避ける

### 3. 設定ファイルのパーミッション

**既存の対策を継続**:
- `~/.local-mcp-memory/config.json` は 600 推奨
- APIキーは環境変数 `OPENAI_API_KEY` 優先

### 4. パス traversal

```bash
mcp-memory search --project "../../etc/passwd" "query"
```

**対策**:
- projectID は正規化処理（既存ロジック）で絶対パス化
- ファイルシステムへの直接アクセスはなし（VectorStore経由のみ）

---

## テスト計画

### ユニットテスト

```go
// cmd/mcp-memory/search_test.go
func TestSearchCommand_ParseFlags(t *testing.T) { ... }
func TestSearchCommand_RequiredFlags(t *testing.T) { ... }
func TestSearchCommand_OutputFormat(t *testing.T) { ... }
```

### 統合テスト

```bash
# E2Eテストに追加
# 1. add_note でデータ追加
# 2. search コマンドで検索
# 3. 結果の検証
```

---

## 実装手順

1. [ ] `internal/bootstrap/bootstrap.go` 作成（共通初期化ロジック）
2. [ ] `cmd/mcp-memory/serve.go` 作成（既存ロジック分離）
3. [ ] `cmd/mcp-memory/main.go` 修正（サブコマンド分岐）
4. [ ] `cmd/mcp-memory/search.go` 作成
5. [ ] ユニットテスト作成
6. [ ] E2Eテスト追加
7. [ ] README更新（Hookでの使用例追加）

---

## Hookでの使用例（完成後）

### settings.json

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

### ~/.claude/hooks/memory-init.sh

```bash
#!/bin/bash

# プロジェクトのメモリを検索してコンテキストに追加
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

---

## 要改善チェック（レビュー用）

### 既存コードへの影響

- [x] **要改善**: main.go の構造変更
  - **問題**: 計画では `main()` を直接 switch 分岐にしているが、現行の `run() -> parseFlags() -> runServe()` フローを迂回してしまう
  - **影響**: `flag.ContinueOnError` の挙動、シグナルハンドラー設定が serve 以外で適用されない
  - **改善案**: serve サブコマンドは現行の `run()` フローを維持し、search は別関数で処理する
    ```go
    func main() {
        if len(os.Args) < 2 {
            printUsage()
            os.Exit(1)
        }
        switch os.Args[1] {
        case "serve":
            // 既存の run() フローをそのまま使用
            if err := run(os.Args[1:]); err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
            }
        case "search":
            if err := runSearchCmd(os.Args[2:]); err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
            }
        // ...
        }
    }
    ```

- [x] **要改善**: Search 時の ProjectID 正規化
  - **問題**: `internal/service/note.go` の `Search()` は ProjectID を正規化しない
  - **影響**: AddNote で保存した canonical path と一致せず検索漏れが発生する
  - **改善案**: CLI の search コマンド内で `config.CanonicalizeProjectID()` を呼び出してから Search を実行
    ```go
    // search.go 内で
    canonicalProjectID, err := config.CanonicalizeProjectID(*projectID)
    if err != nil {
        return fmt.Errorf("failed to canonicalize projectId: %w", err)
    }
    // 正規化後の projectID で検索
    results, err := noteService.Search(ctx, &service.SearchRequest{
        ProjectID: canonicalProjectID,
        // ...
    })
    ```

### セキュリティ上の懸念

- [x] **要改善**: コマンドライン引数の露出
  - **問題**: クエリやパスがプロセス引数に残り、`ps` コマンドで他ユーザーに見える
  - **改善案**: stdin からクエリを入力できるオプションを追加
    ```bash
    # 引数で渡す（従来）
    mcp-memory search -p /path "クエリ"

    # stdin で渡す（追加オプション）
    echo "機密クエリ" | mcp-memory search -p /path --stdin
    ```
  - **実装**: `--stdin` フラグを追加し、クエリを stdin から読み取る
    ```go
    if *useStdin {
        scanner := bufio.NewScanner(os.Stdin)
        if scanner.Scan() {
            query = scanner.Text()
        }
    }
    ```

- [ ] 出力内容の機密性: OK
  - 検索結果には保存されたノート内容が含まれるが、これは意図した動作
  - ログファイルへの出力は行わない設計

- [ ] パス操作の安全性: OK
  - `config.CanonicalizeProjectID()` で正規化済み
  - ファイルシステムへの直接アクセスはなし

### パフォーマンス

- [ ] 初期化コストの許容範囲確認: 要検討
  - 毎回 Embedder/Store 初期化（1-2秒）
  - Hook 用途では許容範囲内と判断（セッション開始時のみ実行）

### エラーハンドリング

- [ ] 網羅性確認: OK
  - 必須引数チェック（projectID, query）
  - 設定ファイル読み込みエラー
  - Embedder/Store 初期化エラー
  - 検索実行エラー
