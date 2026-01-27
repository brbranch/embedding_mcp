# Phase 10 Task 14 残り作業 実装計画

## 現状分析

### 実装済み（確認済み）
- `client.py` - MCPMemoryClient（全9メソッド、updateはgroup_id/source/metadata対応済み）
- `models.py` - Pydanticデータモデル
- `exceptions.py` - カスタム例外クラス
- `langchain_tools.py` - LangGraph Tools（6ツール）
- `pyproject.toml` - パッケージ設定（langchain extras含む）
- `README.md` - 使用例ドキュメント（基本構成完了、LangGraph例あり）

### 今回実装するもの
1. `memory_update_note` ツール（client.updateに対応、全patch項目サポート）
2. `test_langchain_tools.py` - LangGraph Tools用テスト（新規作成、全7ツール網羅）
3. README.md - memory_update_note説明追加

## TODO.md要件との対応

| TODO項目 | 現状 | 対応 |
|----------|------|------|
| LangGraph Tool定義サンプル | 6ツール実装済み | memory_update_note追加で7ツール完成 |
| @toolデコレータでの定義例 | 実装済み（langchain_tools.py） | テストで検証 |
| memory_search, memory_add_note等 | 実装済み | テストで検証 |
| pyproject.toml / setup.py | pyproject.toml完成 | pip install検証 |
| 使用例ドキュメント | README.mdにLangGraph例あり | memory_update_note追加 |

## 要件トレーサビリティ表

| 要件ID | TODO項目 | テストケース | 実装箇所 | 検証方法 |
|--------|----------|--------------|----------|----------|
| REQ-1 | @tool デコレータでの定義例 | test_all_tools_have_tool_decorator | langchain_tools.py | hasattr(tool, 'tool') 確認 |
| REQ-2 | memory_search Tool | test_memory_search_returns_json, test_memory_search_with_group_id | langchain_tools.py:57 | JSONパース成功 |
| REQ-3 | memory_add_note Tool | test_memory_add_note_returns_id_namespace | langchain_tools.py:83 | id/namespace存在確認 |
| REQ-4 | memory_get_note Tool | test_memory_get_note_returns_note_json | langchain_tools.py:110 | Note JSON構造確認 |
| REQ-5 | memory_list_recent Tool | test_memory_list_recent_returns_array | langchain_tools.py:127 | 配列JSON確認 |
| REQ-6 | memory_upsert_global Tool | test_memory_upsert_global_returns_ok | langchain_tools.py:152 | ok/id/namespace確認 |
| REQ-7 | memory_get_global Tool | test_memory_get_global_returns_found | langchain_tools.py:175 | found/value確認 |
| REQ-8 | memory_update_note Tool | test_memory_update_note_title, test_memory_update_note_text, test_memory_update_note_all_fields | langchain_tools.py（新規） | ok確認、全patch項目対応 |
| REQ-9 | pyproject.toml | - | pyproject.toml | `pip install -e ".[langchain]"` 成功 |
| REQ-10 | 使用例ドキュメント | - | README.md | memory_update_note説明あり |
| REQ-11 | LangGraphから呼び出し可能 | test_memory_tools_contains_7_tools, test_configure_memory_client_creates_client | MEMORY_TOOLS | len(MEMORY_TOOLS)==7、get_client()成功 |

## 実装内容

### 1. memory_update_note ツール追加 (langchain_tools.py)

client.updateの全引数（group_id, source, metadata含む）をサポート:

```python
@tool
def memory_update_note(
    note_id: str,
    title: str | None = None,
    text: str | None = None,
    tags: list[str] | None = None,
    source: str | None = None,
    group_id: str | None = None,
    metadata: dict[str, Any] | None = None,
) -> str:
    """Update an existing note (patch).

    Args:
        note_id: Note ID
        title: New title (optional)
        text: New text (optional, triggers re-embedding)
        tags: New tags (optional)
        source: New source (optional)
        group_id: New group ID (optional)
        metadata: New metadata (optional)

    Returns:
        JSON string with result {"ok": true}
    """
    import json
    client = get_client()
    result = client.update(
        note_id,
        title=title,
        text=text,
        tags=tags,
        source=source,
        group_id=group_id,
        metadata=metadata,
    )
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

全7ツールを網羅するテスト:

```python
"""Tests for LangGraph tools."""
import json
import pytest
from unittest.mock import MagicMock, patch

from mcp_memory_client import langchain_tools
from mcp_memory_client.langchain_tools import (
    configure_memory_client,
    get_client,
    MEMORY_TOOLS,
    memory_search,
    memory_add_note,
    memory_get_note,
    memory_update_note,
    memory_list_recent,
    memory_upsert_global,
    memory_get_global,
)


class TestConfigureMemoryClient:
    """configure_memory_client のテスト"""

    def test_configure_creates_client(self):
        """クライアント作成確認"""
        with patch.object(langchain_tools, '_client', None):
            configure_memory_client(base_url="http://test:8765")
            client = get_client()
            assert client is not None
            client.close()

    def test_get_client_raises_without_configure(self):
        """configure前のget_clientでRuntimeError"""
        with patch.object(langchain_tools, '_client', None):
            with pytest.raises(RuntimeError, match="configure_memory_client"):
                get_client()


class TestMemoryToolsList:
    """MEMORY_TOOLS のテスト"""

    def test_memory_tools_contains_7_tools(self):
        """7ツール登録確認"""
        assert len(MEMORY_TOOLS) == 7

    def test_all_tools_have_tool_decorator(self):
        """@tool適用確認（StructuredToolインスタンス）"""
        from langchain_core.tools import StructuredTool
        for tool in MEMORY_TOOLS:
            assert isinstance(tool, StructuredTool)


class TestMemorySearch:
    """memory_search のテスト"""

    def test_returns_json_string(self, mock_client):
        """JSON文字列返却"""
        result = memory_search.invoke({
            "project_id": "/test",
            "query": "test query"
        })
        parsed = json.loads(result)
        assert isinstance(parsed, list)

    def test_with_group_id_filter(self, mock_client):
        """group_idフィルタ動作"""
        result = memory_search.invoke({
            "project_id": "/test",
            "query": "test",
            "group_id": "feature-1"
        })
        assert isinstance(json.loads(result), list)


class TestMemoryAddNote:
    """memory_add_note のテスト"""

    def test_returns_id_and_namespace(self, mock_client):
        """id/namespace返却"""
        result = memory_add_note.invoke({
            "project_id": "/test",
            "group_id": "global",
            "text": "test note"
        })
        parsed = json.loads(result)
        assert "id" in parsed
        assert "namespace" in parsed


class TestMemoryGetNote:
    """memory_get_note のテスト"""

    def test_returns_note_json(self, mock_client):
        """Note JSON返却"""
        result = memory_get_note.invoke({"note_id": "test-id"})
        parsed = json.loads(result)
        assert "id" in parsed
        assert "text" in parsed


class TestMemoryUpdateNote:
    """memory_update_note のテスト"""

    def test_update_title_only(self, mock_client):
        """titleのみ更新"""
        result = memory_update_note.invoke({
            "note_id": "test-id",
            "title": "new title"
        })
        parsed = json.loads(result)
        assert parsed.get("ok") is True

    def test_update_text_triggers_reembedding(self, mock_client):
        """text更新（再埋め込みトリガー）"""
        result = memory_update_note.invoke({
            "note_id": "test-id",
            "text": "new text"
        })
        assert json.loads(result).get("ok") is True

    def test_update_all_fields(self, mock_client):
        """全フィールド更新"""
        result = memory_update_note.invoke({
            "note_id": "test-id",
            "title": "new title",
            "text": "new text",
            "tags": ["tag1"],
            "source": "new-source",
            "group_id": "feature-2",
            "metadata": {"key": "value"}
        })
        assert json.loads(result).get("ok") is True


class TestMemoryListRecent:
    """memory_list_recent のテスト"""

    def test_returns_array(self, mock_client):
        """配列JSON返却"""
        result = memory_list_recent.invoke({
            "project_id": "/test"
        })
        parsed = json.loads(result)
        assert isinstance(parsed, list)


class TestMemoryUpsertGlobal:
    """memory_upsert_global のテスト"""

    def test_returns_ok_id_namespace(self, mock_client):
        """ok/id/namespace返却"""
        result = memory_upsert_global.invoke({
            "project_id": "/test",
            "key": "global.test",
            "value": {"foo": "bar"}
        })
        parsed = json.loads(result)
        assert parsed.get("ok") is True
        assert "id" in parsed
        assert "namespace" in parsed


class TestMemoryGetGlobal:
    """memory_get_global のテスト"""

    def test_returns_found_and_value(self, mock_client):
        """found/value返却"""
        result = memory_get_global.invoke({
            "project_id": "/test",
            "key": "global.test"
        })
        parsed = json.loads(result)
        assert "found" in parsed


# conftest.pyにmock_client fixtureを追加
```

### 4. README.md 更新

LangGraph Integrationセクションにmemory_update_noteを追加:
```markdown
# The agent can now use:
# - memory_search: Search project memory
# - memory_add_note: Add notes
# - memory_get_note: Get note by ID
# - memory_update_note: Update existing note
# - memory_list_recent: List recent notes
# - memory_upsert_global: Save global settings
# - memory_get_global: Get global settings
```

## 完了条件の検証手順

1. **パッケージインストール**:
   ```bash
   cd clients/python
   pip install -e ".[langchain,dev]"
   ```
   → 成功すること

2. **ユニットテスト**:
   ```bash
   pytest tests/test_langchain_tools.py -v
   ```
   → 全テストパス

3. **LangGraph統合確認**（Python REPL）:
   ```python
   from mcp_memory_client.langchain_tools import MEMORY_TOOLS
   assert len(MEMORY_TOOLS) == 7
   print([t.name for t in MEMORY_TOOLS])
   # ['memory_search', 'memory_add_note', 'memory_get_note', 'memory_update_note',
   #  'memory_list_recent', 'memory_upsert_global', 'memory_get_global']
   ```

## ファイル変更一覧

| ファイル | 変更内容 |
|----------|----------|
| src/mcp_memory_client/langchain_tools.py | memory_update_note追加、MEMORY_TOOLS更新 |
| tests/test_langchain_tools.py | 新規作成（全7ツールのテスト） |
| tests/conftest.py | mock_client fixture追加 |
| README.md | memory_update_note説明追加 |
