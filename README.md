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

## 開発状況

現在開発中です。

---

**NOTE**: 各タスク完了時に、該当機能の動作確認方法をこのREADMEに追記します。
