# MCP Memory Server

ローカルで動作する MCP メモリサーバー（Go実装）。会話メモ/仕様/ノートの埋め込み検索基盤。

## 概要

Claude Code から JSON-RPC 2.0 で呼び出せるローカル RAG メモリ基盤を提供します。

## 機能

- 会話メモ/仕様/ノートの保存と検索
- プロジェクト単位・グループ単位でのメモ管理
- OpenAI/Ollama による埋め込み生成
- Chroma/SQLite によるベクトル検索

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

### 起動（予定）

```bash
# stdio（デフォルト）
./mcp-memory serve

# HTTP
./mcp-memory serve --transport http --port 8765
```

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

## 開発状況

現在開発中です。

---

**NOTE**: 各タスク完了時に、該当機能の動作確認方法をこのREADMEに追記します。
