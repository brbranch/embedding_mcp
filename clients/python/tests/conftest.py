"""Pytest configuration and fixtures."""
from unittest.mock import MagicMock, patch

import pytest
from pytest_httpx import HTTPXMock


@pytest.fixture
def httpx_mock_factory() -> type[HTTPXMock]:
    """Return HTTPXMock class for use in tests."""
    return HTTPXMock


@pytest.fixture
def rpc_response():
    """Create a JSON-RPC 2.0 response."""

    def _create(result: dict | list | None = None, error: dict | None = None, id: int = 1):
        response = {"jsonrpc": "2.0", "id": id}
        if error is not None:
            response["error"] = error
        else:
            response["result"] = result
        return response

    return _create


@pytest.fixture
def sample_note_data():
    """Sample note data for testing."""
    return {
        "id": "note-123",
        "projectId": "/Users/test/project",
        "groupId": "global",
        "title": "Test Note",
        "text": "This is a test note",
        "tags": ["test", "sample"],
        "source": "test",
        "createdAt": "2024-01-15T10:30:00Z",
        "namespace": "openai:text-embedding-3-small:1536",
        "metadata": {"key": "value"},
    }


@pytest.fixture
def sample_config_data():
    """Sample config data for testing."""
    return {
        "transportDefaults": {"defaultTransport": "stdio"},
        "embedder": {
            "provider": "openai",
            "model": "text-embedding-3-small",
            "dim": 1536,
            "baseUrl": None,
        },
        "store": {"type": "chroma", "path": None, "url": "http://localhost:8000"},
        "paths": {
            "configPath": "/Users/test/.local-mcp-memory/config.json",
            "dataDir": "/Users/test/.local-mcp-memory/data",
        },
    }


@pytest.fixture
def mock_client(sample_note_data):
    """Mock MCPMemoryClient for langchain_tools tests."""
    from mcp_memory_client.models import GlobalValue, ListRecentResult, Note, SearchResult

    mock = MagicMock()

    # search returns SearchResult
    mock.search.return_value = SearchResult(
        namespace="openai:text-embedding-3-small:1536",
        results=[Note.model_validate(sample_note_data)],
    )

    # add_note returns dict
    mock.add_note.return_value = {
        "id": "note-new",
        "namespace": "openai:text-embedding-3-small:1536",
    }

    # get returns Note
    mock.get.return_value = Note.model_validate(sample_note_data)

    # update returns dict
    mock.update.return_value = {"ok": True}

    # list_recent returns ListRecentResult
    mock.list_recent.return_value = ListRecentResult(
        namespace="openai:text-embedding-3-small:1536",
        items=[Note.model_validate(sample_note_data)],
    )

    # upsert_global returns dict
    mock.upsert_global.return_value = {
        "ok": True,
        "id": "global-123",
        "namespace": "openai:text-embedding-3-small:1536",
    }

    # get_global returns GlobalValue
    mock.get_global.return_value = GlobalValue(
        namespace="openai:text-embedding-3-small:1536",
        found=True,
        id="global-123",
        value={"test": "value"},
        updated_at="2024-01-15T10:30:00Z",
    )

    with patch("mcp_memory_client.langchain_tools._client", mock):
        yield mock
