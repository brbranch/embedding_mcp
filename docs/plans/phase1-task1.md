# Phase 1 Task 1: プロジェクト初期化とディレクトリ構造

## 概要

MCP Memory Server（Go実装）のプロジェクト基盤を構築する。

## タスク内容

### 1. go.mod 作成

```go
module github.com/brbranch/embedding_mcp

go 1.22
```

モジュール名は GitHub リポジトリに合わせる。

### 2. ディレクトリ構造

```
embedding_mcp/
├── cmd/
│   └── mcp-memory/
│       └── main.go           # エントリポイント（スケルトン）
├── internal/
│   ├── config/
│   │   └── config.go         # 設定管理（スケルトン）
│   ├── model/
│   │   └── model.go          # データモデル（スケルトン）
│   ├── service/
│   │   └── service.go        # ビジネスロジック（スケルトン）
│   ├── embedder/
│   │   └── embedder.go       # Embedding provider（スケルトン）
│   ├── store/
│   │   └── store.go          # VectorStore（スケルトン）
│   ├── jsonrpc/
│   │   └── handler.go        # JSON-RPC handler（スケルトン）
│   └── transport/
│       ├── stdio/
│       │   └── stdio.go      # stdio transport（スケルトン）
│       └── http/
│           └── http.go       # HTTP transport（スケルトン）
├── go.mod
└── README.md
```

### 3. 各スケルトンファイルの内容

#### cmd/mcp-memory/main.go

```go
// Package main is the entry point for mcp-memory server.
package main

func main() {
	// TODO: implement serve command
}
```

#### internal/config/config.go

```go
// Package config provides configuration management for mcp-memory.
package config

// TODO: implement config loading and management
```

#### internal/model/model.go

```go
// Package model defines data structures for mcp-memory.
package model

// TODO: implement Note, GlobalConfig, Config, JSON-RPC types
```

#### internal/service/service.go

```go
// Package service implements business logic for mcp-memory.
package service

// TODO: implement NoteService, ConfigService, GlobalService
```

#### internal/embedder/embedder.go

```go
// Package embedder provides embedding generation interfaces and implementations.
package embedder

// TODO: implement Embedder interface and OpenAI/Ollama/local embedders
```

#### internal/store/store.go

```go
// Package store provides vector storage interfaces and implementations.
package store

// TODO: implement Store interface and Chroma/SQLite implementations
```

#### internal/jsonrpc/handler.go

```go
// Package jsonrpc implements JSON-RPC 2.0 handlers for mcp-memory.
package jsonrpc

// TODO: implement JSON-RPC parser, dispatcher, and method handlers
```

#### internal/transport/stdio/stdio.go

```go
// Package stdio implements stdio transport for mcp-memory.
package stdio

// TODO: implement NDJSON stdio transport
```

#### internal/transport/http/http.go

```go
// Package http implements HTTP transport for mcp-memory.
package http

// TODO: implement HTTP JSON-RPC endpoint
```

### 4. README.md スケルトン

```markdown
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

\`\`\`bash
go build ./cmd/mcp-memory
\`\`\`

### 起動（予定）

\`\`\`bash
# stdio（デフォルト）
./mcp-memory serve

# HTTP
./mcp-memory serve --transport http --port 8765
\`\`\`

## 開発状況

現在開発中です。
```

## 完了条件

1. `go build ./...` がエラーなく成功すること
2. README.md にビルド方法が記載されていること

## テストケース

### 正常系

| # | テスト内容 | 期待結果 |
|---|-----------|----------|
| 1 | `go build ./...` 実行 | エラーなく成功 |
| 2 | `go mod tidy` 実行 | エラーなく成功 |
| 3 | 各パッケージのimport確認 | 正しくimportできる |

### 異常系

このフェーズでは該当なし（スケルトンのみ）。

## ファイル一覧

| ファイル | 内容 |
|---------|------|
| go.mod | モジュール定義 |
| cmd/mcp-memory/main.go | エントリポイント |
| internal/config/config.go | 設定管理 |
| internal/model/model.go | データモデル |
| internal/service/service.go | ビジネスロジック |
| internal/embedder/embedder.go | Embedding provider |
| internal/store/store.go | VectorStore |
| internal/jsonrpc/handler.go | JSON-RPC handler |
| internal/transport/stdio/stdio.go | stdio transport |
| internal/transport/http/http.go | HTTP transport |
| README.md | プロジェクト説明 |

## 依存関係

このフェーズでは外部依存なし（標準ライブラリのみ）。

## エラーハンドリング方針

このフェーズでは該当なし（スケルトンのみ）。

## セキュリティ考慮事項

このフェーズでは該当なし（コードなし）。

---

## 設計PRチェックリスト

- [x] TODO.mdの該当タスクの要件を全て含んでいる
- [x] requirements/ の仕様と整合している
- [x] 関数シグネチャが明記されている（スケルトンのみのため該当なし）
- [x] テストケース一覧がある
- [x] エラーハンドリング方針が定義されている（該当なし）
- [x] セキュリティ考慮事項が記載されている（該当なし）
