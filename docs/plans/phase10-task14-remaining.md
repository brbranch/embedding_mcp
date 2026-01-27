# Phase 10 Task 14 残り作業 実装計画

## 現状分析

### 実装済み
- `client.py` - MCPMemoryClient（全9メソッド）
- `models.py` - Pydanticデータモデル
- `exceptions.py` - カスタム例外クラス
- `langchain_tools.py` - LangGraph Tools（6ツール）
- `pyproject.toml` - パッケージ設定
- `README.md` - ドキュメント

### 未実装
- `memory_update_note` ツール（client.updateに対応）

## 要件トレーサビリティ表

| 要件ID | TODO項目 | テストケース | 実装箇所 |
|--------|----------|--------------|----------|
| REQ-1 | @tool デコレータでの定義例 | test_langchain_tools.py | langchain_tools.py |
| REQ-2 | memory_search Tool | TestMemorySearch | langchain_tools.py:57 |
| REQ-3 | memory_add_note Tool | TestMemoryAddNote | langchain_tools.py:83 |
| REQ-4 | memory_get_note Tool | TestMemoryGetNote | langchain_tools.py:110 |
| REQ-5 | memory_list_recent Tool | TestMemoryListRecent | langchain_tools.py:127 |
| REQ-6 | memory_upsert_global Tool | TestMemoryUpsertGlobal | langchain_tools.py:152 |
| REQ-7 | memory_get_global Tool | TestMemoryGetGlobal | langchain_tools.py:175 |
| REQ-8 | memory_update_note Tool | TestMemoryUpdateNote | langchain_tools.py（新規追加） |
| REQ-9 | pyproject.toml | - | pyproject.toml |
| REQ-10 | 使用例ドキュメント | - | README.md |
| REQ-11 | LangGraphから呼び出し可能 | E2Eテスト | 検証手順で確認 |

## 実装内容

### 1. memory_update_note ツール追加

```python
@tool
def memory_update_note(
    note_id: str,
    title: str | None = None,
    text: str | None = None,
    tags: list[str] | None = None,
) -> str:
    """Update an existing note.

    Args:
        note_id: Note ID
        title: New title (optional)
        text: New text (optional, triggers re-embedding)
        tags: New tags (optional)

    Returns:
        JSON string with result
    """
    import json
    client = get_client()
    result = client.update(note_id, title=title, text=text, tags=tags)
    return json.dumps(result, ensure_ascii=False)
```

### 2. MEMORY_TOOLS に追加

```python
MEMORY_TOOLS = [
    memory_search,
    memory_add_note,
    memory_get_note,
    memory_update_note,  # 追加
    memory_list_recent,
    memory_upsert_global,
    memory_get_global,
]
```

### 3. テストコード追加

`tests/test_langchain_tools.py` に memory_update_note のテストを追加。

## 完了条件の検証手順

1. ユニットテスト: `pytest tests/test_langchain_tools.py`
2. LangGraph統合確認（手動）:
   ```python
   from mcp_memory_client.langchain_tools import configure_memory_client, MEMORY_TOOLS
   configure_memory_client()
   print([t.name for t in MEMORY_TOOLS])  # 7ツール表示
   ```

## ファイル変更一覧

| ファイル | 変更内容 |
|----------|----------|
| langchain_tools.py | memory_update_note追加、MEMORY_TOOLS更新 |
| tests/test_langchain_tools.py | memory_update_noteテスト追加 |
| README.md | memory_update_note説明追加 |
