# Phase 11 Task 16: memory.delete メソッド実装

## 概要

JSON-RPCメソッド `memory.delete` を追加し、ノートおよびグローバル設定をIDで削除可能にする。

## 仕様設計

### メソッド定義

```
10) memory.delete
Input:
{
  "id": string  // 削除対象のID（Note ID または GlobalConfig ID）
}
Output:
{ "ok": true }
```

### 設計判断

| 項目 | 決定 | 理由 |
|------|------|------|
| 削除方式 | 物理削除 | 用途が「古いノートの整理、テストデータのクリーンアップ」と明確 |
| 対象 | Note または GlobalConfig | ID検索時、Noteを優先、なければGlobalConfigを検索 |
| 復元対応 | 非対応 | 論理削除が必要な場合は `tags: ["archived"]` でマーキング運用を推奨 |

### エラーケース

| エラー | コード | 条件 |
|--------|--------|------|
| Invalid params | -32602 | id が空または未指定 |
| Not found | -32001 | 指定IDのNote/GlobalConfigが存在しない |
| Internal error | -32603 | DB操作エラー等 |

## 影響範囲

### 1. Store interface (`internal/store/store.go`)

```go
// 追加
DeleteGlobal(ctx context.Context, projectID, key string) error
```

- `Delete(ctx, id)` は Note 用（既存）
- `DeleteGlobal(ctx, projectID, key)` は GlobalConfig 用（新規）

### 2. Store 実装

| 実装 | ファイル | 対応内容 |
|------|----------|----------|
| SQLite | `internal/store/sqlite.go` | DELETE FROM notes / DELETE FROM global_config |
| Memory | `internal/store/memory.go` | map からの削除 |
| Chroma | `internal/store/chroma.go` | stub（NotImplemented） |

### 3. Service 層

| サービス | 追加メソッド |
|----------|------------|
| NoteService | `Delete(ctx, id) error` |
| GlobalService | `DeleteGlobal(ctx, projectID, key) error` |

### 4. JSON-RPC Handler (`internal/jsonrpc/handler.go`)

- `memory.delete` ハンドラー追加
- dispatch に case 追加
- mapError に ErrGlobalConfigNotFound 追加

### 5. Python Client (`clients/python/`)

- `MCPMemoryClient.delete(id: str)` メソッド追加

### 6. LangGraph Tools (`clients/python/`)

- `memory_delete_note` ツール追加

### 7. Skill 定義 (`.claude/skills/memory/SKILL.md`)

- `memory.delete` の使用方法記載

## 要件トレーサビリティ表

| 要件ID | TODO項目 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-01 | memory.delete メソッド追加 | TestDelete_Note_Success | internal/jsonrpc/handler.go:handleDelete |
| REQ-02 | Note物理削除 | TestDelete_Note_Success | internal/store/sqlite.go:Delete |
| REQ-03 | GlobalConfig物理削除 | TestDelete_GlobalConfig_Success | internal/store/sqlite.go:DeleteGlobal |
| REQ-04 | ID必須バリデーション | TestDelete_EmptyID | internal/jsonrpc/methods.go:handleDelete |
| REQ-05 | Note優先検索 | TestDelete_Note_Priority | internal/service/note.go:Delete |
| REQ-06 | NotFound エラー | TestDelete_NotFound | internal/jsonrpc/handler.go:mapError |
| REQ-07 | Memory Store対応 | TestMemoryStore_Delete | internal/store/memory.go:Delete |
| REQ-08 | SQLite Store対応 | TestSQLiteStore_Delete | internal/store/sqlite.go:Delete |
| REQ-09 | Python Client追加 | test_client_delete | clients/python/mcp_memory_client.py |
| REQ-10 | LangGraph Tool追加 | - | clients/python/langgraph_tools.py |
| REQ-11 | Skill定義更新 | - | .claude/skills/memory/SKILL.md |

## 仕様追記案 (`requirements/embedded_spec.md`)

以下を「# 8. JSON-RPC methods（必須）」セクションの 9) の後に追加:

```markdown
10) memory.delete
Input:
{
  "id": string  // 削除対象のID（Note ID または GlobalConfig ID）
}
Output:
{ "ok": true }

- 物理削除を実行
- 検索順序: Note → GlobalConfig（Noteを優先）
- 指定IDが存在しない場合は Not found エラー (-32001)
```

## 実装順序

1. **仕様追記**: `requirements/embedded_spec.md` に memory.delete 追加
2. **Store interface**: `DeleteGlobal` メソッド追加
3. **Store 実装**: SQLite, Memory, Chroma（stub）
4. **Service 層**: NoteService.Delete, GlobalService.DeleteGlobal
5. **JSON-RPC Handler**: memory.delete ハンドラー
6. **テスト**: 単体テスト + E2Eテスト
7. **Python Client**: delete メソッド追加
8. **LangGraph Tools**: memory_delete_note 追加
9. **Skill 定義**: SKILL.md 更新
10. **README**: 使用例追記

## テスト計画

### 単体テスト

```go
// internal/jsonrpc/handler_test.go
func TestDelete_Note_Success(t *testing.T)
func TestDelete_GlobalConfig_Success(t *testing.T)
func TestDelete_EmptyID(t *testing.T)
func TestDelete_NotFound(t *testing.T)
func TestDelete_Note_Priority(t *testing.T)

// internal/store/sqlite_test.go
func TestSQLiteStore_Delete(t *testing.T)
func TestSQLiteStore_DeleteGlobal(t *testing.T)

// internal/store/memory_test.go
func TestMemoryStore_Delete(t *testing.T)
func TestMemoryStore_DeleteGlobal(t *testing.T)
```

### E2Eテスト

```go
// e2e/delete_test.go
func TestE2E_DeleteNote(t *testing.T)
func TestE2E_DeleteGlobalConfig(t *testing.T)
func TestE2E_DeleteNotFound(t *testing.T)
```

## 完了条件

- `go test ./...` が全てパス
- Python Client の delete メソッドが動作
- Skill 定義に memory.delete が記載
- README に使用例が追記
