"""Tests for MCP Memory Client models."""
import pytest

from mcp_memory_client.models import (
    ConfigResult,
    GlobalValue,
    ListRecentResult,
    Note,
    SearchResult,
)


class TestNote:
    """Tests for Note model."""

    def test_note_from_dict(self, sample_note_data):
        """Test creating Note from dict with camelCase keys."""
        note = Note.model_validate(sample_note_data)

        assert note.id == "note-123"
        assert note.project_id == "/Users/test/project"
        assert note.group_id == "global"
        assert note.title == "Test Note"
        assert note.text == "This is a test note"
        assert note.tags == ["test", "sample"]
        assert note.source == "test"
        assert note.created_at == "2024-01-15T10:30:00Z"
        assert note.namespace == "openai:text-embedding-3-small:1536"
        assert note.metadata == {"key": "value"}

    def test_note_alias(self):
        """Test that camelCase alias works correctly."""
        data = {
            "id": "note-1",
            "projectId": "/test",
            "groupId": "test-group",
            "text": "test",
            "createdAt": "2024-01-01T00:00:00Z",
        }
        note = Note.model_validate(data)

        # Check snake_case attributes
        assert note.project_id == "/test"
        assert note.group_id == "test-group"
        assert note.created_at == "2024-01-01T00:00:00Z"

        # Check serialization with aliases
        dumped = note.model_dump(by_alias=True)
        assert "projectId" in dumped
        assert "groupId" in dumped
        assert "createdAt" in dumped

    def test_note_with_score(self, sample_note_data):
        """Test Note with score field (search result)."""
        sample_note_data["score"] = 0.95
        note = Note.model_validate(sample_note_data)

        assert note.score == 0.95

    def test_note_optional_fields(self):
        """Test Note with minimal required fields."""
        data = {
            "id": "note-1",
            "projectId": "/test",
            "groupId": "global",
            "text": "test content",
            "createdAt": "2024-01-01T00:00:00Z",
        }
        note = Note.model_validate(data)

        assert note.title is None
        assert note.tags == []
        assert note.source is None
        assert note.namespace is None
        assert note.score is None
        assert note.metadata is None


class TestSearchResult:
    """Tests for SearchResult model."""

    def test_search_result(self, sample_note_data):
        """Test SearchResult construction."""
        result = SearchResult(
            namespace="openai:text-embedding-3-small:1536",
            results=[Note.model_validate(sample_note_data)],
        )

        assert result.namespace == "openai:text-embedding-3-small:1536"
        assert len(result.results) == 1
        assert result.results[0].id == "note-123"

    def test_search_result_empty(self):
        """Test SearchResult with empty results."""
        result = SearchResult(namespace="test:ns:123", results=[])

        assert result.namespace == "test:ns:123"
        assert result.results == []


class TestListRecentResult:
    """Tests for ListRecentResult model."""

    def test_list_recent_result(self, sample_note_data):
        """Test ListRecentResult construction."""
        result = ListRecentResult(
            namespace="openai:text-embedding-3-small:1536",
            items=[Note.model_validate(sample_note_data)],
        )

        assert result.namespace == "openai:text-embedding-3-small:1536"
        assert len(result.items) == 1
        assert result.items[0].id == "note-123"

    def test_list_recent_result_empty(self):
        """Test ListRecentResult with empty items."""
        result = ListRecentResult(namespace="test:ns:123", items=[])

        assert result.namespace == "test:ns:123"
        assert result.items == []


class TestConfigResult:
    """Tests for ConfigResult model."""

    def test_config_result(self, sample_config_data):
        """Test ConfigResult construction."""
        result = ConfigResult.model_validate(sample_config_data)

        assert result.transport_defaults.default_transport == "stdio"
        assert result.embedder.provider == "openai"
        assert result.embedder.model == "text-embedding-3-small"
        assert result.embedder.dim == 1536
        assert result.store.type == "chroma"
        assert result.paths.config_path == "/Users/test/.local-mcp-memory/config.json"
        assert result.paths.data_dir == "/Users/test/.local-mcp-memory/data"


class TestGlobalValue:
    """Tests for GlobalValue model."""

    def test_global_value_found_true(self):
        """Test GlobalValue when found=true."""
        data = {
            "namespace": "openai:text-embedding-3-small:1536",
            "found": True,
            "id": "global-123",
            "value": {"setting": "value"},
            "updatedAt": "2024-01-15T10:30:00Z",
        }
        result = GlobalValue.model_validate(data)

        assert result.namespace == "openai:text-embedding-3-small:1536"
        assert result.found is True
        assert result.id == "global-123"
        assert result.value == {"setting": "value"}
        assert result.updated_at == "2024-01-15T10:30:00Z"

    def test_global_value_found_false(self):
        """Test GlobalValue when found=false."""
        data = {
            "namespace": "openai:text-embedding-3-small:1536",
            "found": False,
        }
        result = GlobalValue.model_validate(data)

        assert result.namespace == "openai:text-embedding-3-small:1536"
        assert result.found is False
        assert result.id is None
        assert result.value is None
        assert result.updated_at is None

    def test_global_value_with_string_value(self):
        """Test GlobalValue with string value."""
        data = {
            "namespace": "test:ns:123",
            "found": True,
            "id": "global-1",
            "value": "simple string value",
            "updatedAt": "2024-01-01T00:00:00Z",
        }
        result = GlobalValue.model_validate(data)

        assert result.value == "simple string value"
