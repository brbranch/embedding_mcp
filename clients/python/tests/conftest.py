"""Pytest configuration and fixtures."""
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
