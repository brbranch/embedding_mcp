"""Tests for LangGraph tools."""
import json
from unittest.mock import patch

import pytest

from mcp_memory_client import langchain_tools
from mcp_memory_client.langchain_tools import (
    MEMORY_TOOLS,
    configure_memory_client,
    get_client,
    memory_add_note,
    memory_get_global,
    memory_get_note,
    memory_list_recent,
    memory_search,
    memory_update_note,
    memory_upsert_global,
)


class TestConfigureMemoryClient:
    """configure_memory_client tests."""

    def test_configure_creates_client(self):
        """Verify client creation."""
        with patch.object(langchain_tools, "_client", None):
            configure_memory_client(base_url="http://test:8765")
            client = get_client()
            assert client is not None
            client.close()

    def test_get_client_raises_without_configure(self):
        """RuntimeError when get_client called before configure."""
        with patch.object(langchain_tools, "_client", None):
            with pytest.raises(RuntimeError, match="configure_memory_client"):
                get_client()


class TestMemoryToolsList:
    """MEMORY_TOOLS tests."""

    def test_memory_tools_contains_7_tools(self):
        """7 tools registered."""
        assert len(MEMORY_TOOLS) == 7

    def test_all_tools_have_tool_decorator(self):
        """@tool decorator applied (StructuredTool instance)."""
        from langchain_core.tools import StructuredTool

        for tool in MEMORY_TOOLS:
            assert isinstance(tool, StructuredTool)

    def test_tool_names(self):
        """Tool names are correct."""
        names = [t.name for t in MEMORY_TOOLS]
        assert names == [
            "memory_search",
            "memory_add_note",
            "memory_get_note",
            "memory_update_note",
            "memory_list_recent",
            "memory_upsert_global",
            "memory_get_global",
        ]


class TestMemorySearch:
    """memory_search tests."""

    def test_returns_json_string(self, mock_client):
        """Returns JSON string."""
        result = memory_search.invoke({"project_id": "/test", "query": "test query"})
        parsed = json.loads(result)
        assert isinstance(parsed, list)
        assert len(parsed) == 1
        assert "id" in parsed[0]

    def test_with_group_id_filter(self, mock_client):
        """group_id filter works."""
        result = memory_search.invoke({
            "project_id": "/test",
            "query": "test",
            "group_id": "feature-1",
        })
        assert isinstance(json.loads(result), list)
        mock_client.search.assert_called_once()
        call_kwargs = mock_client.search.call_args
        assert call_kwargs[1]["group_id"] == "feature-1"


class TestMemoryAddNote:
    """memory_add_note tests."""

    def test_returns_id_and_namespace(self, mock_client):
        """Returns id and namespace."""
        result = memory_add_note.invoke({
            "project_id": "/test",
            "group_id": "global",
            "text": "test note",
        })
        parsed = json.loads(result)
        assert "id" in parsed
        assert "namespace" in parsed

    def test_with_optional_params(self, mock_client):
        """Optional params passed correctly."""
        memory_add_note.invoke({
            "project_id": "/test",
            "group_id": "global",
            "text": "test note",
            "title": "Test Title",
            "tags": ["tag1", "tag2"],
        })
        mock_client.add_note.assert_called_once()
        call_args = mock_client.add_note.call_args
        assert call_args[1]["title"] == "Test Title"
        assert call_args[1]["tags"] == ["tag1", "tag2"]


class TestMemoryGetNote:
    """memory_get_note tests."""

    def test_returns_note_json(self, mock_client):
        """Returns Note JSON."""
        result = memory_get_note.invoke({"note_id": "test-id"})
        parsed = json.loads(result)
        assert "id" in parsed
        assert "text" in parsed


class TestMemoryUpdateNote:
    """memory_update_note tests."""

    def test_update_title_only(self, mock_client):
        """Update title only."""
        result = memory_update_note.invoke({"note_id": "test-id", "title": "new title"})
        parsed = json.loads(result)
        assert parsed.get("ok") is True
        mock_client.update.assert_called_once_with(
            "test-id",
            title="new title",
            text=None,
            tags=None,
            source=None,
            group_id=None,
            metadata=None,
        )

    def test_update_text_triggers_reembedding(self, mock_client):
        """Update text (triggers re-embedding)."""
        result = memory_update_note.invoke({"note_id": "test-id", "text": "new text"})
        assert json.loads(result).get("ok") is True
        mock_client.update.assert_called_once()
        call_args = mock_client.update.call_args
        assert call_args[1]["text"] == "new text"

    def test_update_all_fields(self, mock_client):
        """Update all fields."""
        result = memory_update_note.invoke({
            "note_id": "test-id",
            "title": "new title",
            "text": "new text",
            "tags": ["tag1"],
            "source": "new-source",
            "group_id": "feature-2",
            "metadata": {"key": "value"},
        })
        assert json.loads(result).get("ok") is True
        mock_client.update.assert_called_once_with(
            "test-id",
            title="new title",
            text="new text",
            tags=["tag1"],
            source="new-source",
            group_id="feature-2",
            metadata={"key": "value"},
        )


class TestMemoryListRecent:
    """memory_list_recent tests."""

    def test_returns_array(self, mock_client):
        """Returns array JSON."""
        result = memory_list_recent.invoke({"project_id": "/test"})
        parsed = json.loads(result)
        assert isinstance(parsed, list)

    def test_with_group_id_and_limit(self, mock_client):
        """group_id and limit passed correctly."""
        memory_list_recent.invoke({
            "project_id": "/test",
            "group_id": "feature-1",
            "limit": 5,
        })
        mock_client.list_recent.assert_called_once()
        call_kwargs = mock_client.list_recent.call_args
        assert call_kwargs[1]["group_id"] == "feature-1"
        assert call_kwargs[1]["limit"] == 5


class TestMemoryUpsertGlobal:
    """memory_upsert_global tests."""

    def test_returns_ok_id_namespace(self, mock_client):
        """Returns ok, id, namespace."""
        result = memory_upsert_global.invoke({
            "project_id": "/test",
            "key": "global.test",
            "value": {"foo": "bar"},
        })
        parsed = json.loads(result)
        assert parsed.get("ok") is True
        assert "id" in parsed
        assert "namespace" in parsed


class TestMemoryGetGlobal:
    """memory_get_global tests."""

    def test_returns_found_and_value(self, mock_client):
        """Returns found and value."""
        result = memory_get_global.invoke({"project_id": "/test", "key": "global.test"})
        parsed = json.loads(result)
        assert "found" in parsed
        assert parsed["found"] is True
        assert "value" in parsed
