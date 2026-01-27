# Phase 10 Task 14 残り作業 実装計画

## 現状分析

### 実装済み
- `client.py` - MCPMemoryClient（全9メソッド）
- `models.py` - Pydanticデータモデル
- `exceptions.py` - カスタム例外クラス
- `langchain_tools.py` - LangGraph Tools（6ツール）
- `pyproject.toml` - パッケージ設定（基本構成完了）
- `README.md` - ドキュメント（基本構成完了）

### 未実装・拡充必要
1. `memory_update_note` ツール（client.updateに対応）
2. `test_langchain_tools.py` - LangGraph Tools用テスト（新規作成）
3. README.md - memory_update_note説明追加、ReActエージェント例拡充

## 要件トレーサビリティ表

| 要件ID | TODO項目 | テストケース | 実装箇所 | 検証方法 |
|--------|----------|--------------|----------|----------|
| REQ-1 | @tool デコレータでの定義例 | test_tool_decorator_applied | langchain_tools.py | 各ツールに@tool適用確認 |
| REQ-2 | memory_search Tool | test_memory_search_normal, test_memory_search_with_group | langchain_tools.py:57 | JSON返却確認 |
| REQ-3 | memory_add_note Tool | test_memory_add_note_normal, test_memory_add_note_with_options | langchain_tools.py:83 | id/namespace返却確認 |
| REQ-4 | memory_get_note Tool | test_memory_get_note_normal | langchain_tools.py:110 | Note JSON返却確認 |
| REQ-5 | memory_list_recent Tool | test_memory_list_recent_normal | langchain_tools.py:127 | 配列JSON返却確認 |
| REQ-6 | memory_upsert_global Tool | test_memory_upsert_global_normal | langchain_tools.py:152 | ok/id/namespace返却確認 |
| REQ-7 | memory_get_global Tool | test_memory_get_global_normal | langchain_tools.py:175 | found/value返却確認 |
| REQ-8 | memory_update_note Tool | test_memory_update_note_title, test_memory_update_note_text | langchain_tools.py（新規追加） | ok返却確認 |
| REQ-9 | pyproject.toml | - | pyproject.toml | `pip install -e .` 成功 |
| REQ-10 | 使用例ドキュメント | - | README.md | memory_update_note説明あり |
| REQ-11 | LangGraphから呼び出し可能 | test_memory_tools_list, test_configure_memory_client | MEMORY_TOOLS | 7ツール登録確認 |

## 実装内容

### 1. memory_update_note ツール追加 (langchain_tools.py)

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

### 3. テストコード新規作成 (tests/test_langchain_tools.py)

```python
"""Tests for LangGraph tools."""
import json
import pytest
from unittest.mock import MagicMock, patch

class TestConfigureMemoryClient:
    """configure_memory_client のテスト"""

    def test_configure_creates_client(self):
        """クライアント作成確認"""

    def test_get_client_raises_without_configure(self):
        """configure前のget_clientでRuntimeError"""

class TestMemoryToolsList:
    """MEMORY_TOOLS のテスト"""

    def test_memory_tools_contains_7_tools(self):
        """7ツール登録確認"""

    def test_all_tools_have_tool_decorator(self):
        """@tool適用確認"""

class TestMemorySearch:
    """memory_search のテスト"""

    def test_returns_json_string(self):
        """JSON文字列返却"""

    def test_with_group_id_filter(self):
        """group_idフィルタ動作"""

class TestMemoryAddNote:
    """memory_add_note のテスト"""

    def test_returns_id_and_namespace(self):
        """id/namespace返却"""

class TestMemoryUpdateNote:
    """memory_update_note のテスト"""

    def test_update_title_only(self):
        """titleのみ更新"""

    def test_update_text_triggers_reembedding(self):
        """text更新時の再埋め込み"""

    def test_update_tags_only(self):
        """tagsのみ更新"""
```

### 4. README.md 更新

LangGraph Integration セクションに memory_update_note を追加:
```markdown
# The agent can now use:
# - memory_search: Search project memory
# - memory_add_note: Add notes
# - memory_get_note: Get note by ID
# - memory_update_note: Update existing note  # 追加
# - memory_list_recent: List recent notes
# - memory_upsert_global: Save global settings
# - memory_get_global: Get global settings
```

## 完了条件の検証手順

1. **ユニットテスト**: `pytest tests/test_langchain_tools.py -v`
2. **パッケージインストール**: `pip install -e ".[langchain]"` 成功
3. **LangGraph統合確認**:
   ```python
   from mcp_memory_client.langchain_tools import configure_memory_client, MEMORY_TOOLS
   configure_memory_client()
   assert len(MEMORY_TOOLS) == 7
   print([t.name for t in MEMORY_TOOLS])
   # ['memory_search', 'memory_add_note', 'memory_get_note', 'memory_update_note',
   #  'memory_list_recent', 'memory_upsert_global', 'memory_get_global']
   ```

## ファイル変更一覧

| ファイル | 変更内容 |
|----------|----------|
| langchain_tools.py | memory_update_note追加、MEMORY_TOOLS更新 |
| tests/test_langchain_tools.py | 新規作成（全ツールのテスト） |
| README.md | memory_update_note説明追加 |
