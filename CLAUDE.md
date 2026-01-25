# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ローカルMCPメモリサーバー（Go実装）。会話メモ/仕様/ノートの埋め込み検索基盤。

## Tech Stack

- Go 1.22+
- SQLite (modernc.org/sqlite推奨、cgo不要)
- JSON-RPC 2.0

## Project Structure

```
cmd/mcp-memory/       # エントリポイント
internal/
  config/             # 設定管理
  model/              # データモデル
  service/            # ビジネスロジック
  embedder/           # Embedding provider
  store/              # VectorStore
  jsonrpc/            # JSON-RPC handler
  transport/stdio/    # stdio transport
  transport/http/     # HTTP transport
```

## Code Style

- K&R / 1TBS: 開き括弧 `{` は同一行に配置

## Documentation
### 作成するプロダクト

- 仕様: `requirements/embedded_spec.md`
- Skill仕様: `requirements/embedded_skill_spec.md`
