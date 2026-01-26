"""Tests for MCP Memory Client."""
import pytest
from pytest_httpx import HTTPXMock

from mcp_memory_client import MCPMemoryClient
from mcp_memory_client.exceptions import ConnectionError, RPCError, TimeoutError


class TestClientInit:
    """Tests for client initialization."""

    def test_client_init_default(self):
        """Test default initialization."""
        client = MCPMemoryClient()
        assert client._base_url == "http://localhost:8765"
        assert client._timeout == 30.0
        client.close()

    def test_client_init_custom(self):
        """Test custom URL and timeout."""
        client = MCPMemoryClient(base_url="http://localhost:9000", timeout=60.0)
        assert client._base_url == "http://localhost:9000"
        assert client._timeout == 60.0
        client.close()

    def test_client_init_trailing_slash(self):
        """Test URL trailing slash is removed."""
        client = MCPMemoryClient(base_url="http://localhost:8765/")
        assert client._base_url == "http://localhost:8765"
        client.close()

    def test_client_context_manager(self):
        """Test context manager usage."""
        with MCPMemoryClient() as client:
            assert client._base_url == "http://localhost:8765"


class TestAddNote:
    """Tests for add_note method."""

    def test_add_note_minimal(self, httpx_mock: HTTPXMock, rpc_response):
        """Test add_note with minimal parameters."""
        httpx_mock.add_response(
            json=rpc_response({"id": "note-123", "namespace": "openai:model:1536"})
        )

        with MCPMemoryClient() as client:
            result = client.add_note(
                project_id="/test/project",
                group_id="global",
                text="Test note content",
            )

        assert result["id"] == "note-123"
        assert result["namespace"] == "openai:model:1536"

        # Verify request
        request = httpx_mock.get_request()
        assert request is not None
        body = request.read()
        import json

        data = json.loads(body)
        assert data["method"] == "memory.add_note"
        assert data["params"]["projectId"] == "/test/project"
        assert data["params"]["groupId"] == "global"
        assert data["params"]["text"] == "Test note content"

    def test_add_note_full(self, httpx_mock: HTTPXMock, rpc_response):
        """Test add_note with all parameters."""
        httpx_mock.add_response(
            json=rpc_response({"id": "note-456", "namespace": "openai:model:1536"})
        )

        with MCPMemoryClient() as client:
            result = client.add_note(
                project_id="/test/project",
                group_id="feature-1",
                text="Full note content",
                title="My Note",
                tags=["tag1", "tag2"],
                source="test",
                created_at="2024-01-15T10:30:00Z",
                metadata={"key": "value"},
            )

        assert result["id"] == "note-456"

        # Verify request
        request = httpx_mock.get_request()
        import json

        data = json.loads(request.read())
        assert data["params"]["title"] == "My Note"
        assert data["params"]["tags"] == ["tag1", "tag2"]
        assert data["params"]["source"] == "test"
        assert data["params"]["createdAt"] == "2024-01-15T10:30:00Z"
        assert data["params"]["metadata"] == {"key": "value"}

    def test_add_note_invalid_params(self, httpx_mock: HTTPXMock, rpc_response):
        """Test add_note with missing required parameter."""
        httpx_mock.add_response(
            json=rpc_response(
                error={"code": -32602, "message": "projectId is required"}
            )
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.add_note(
                    project_id="",
                    group_id="global",
                    text="Test",
                )

        assert exc_info.value.is_invalid_params


class TestSearch:
    """Tests for search method."""

    def test_search_basic(self, httpx_mock: HTTPXMock, rpc_response, sample_note_data):
        """Test basic search."""
        httpx_mock.add_response(
            json=rpc_response(
                {
                    "namespace": "openai:model:1536",
                    "results": [sample_note_data],
                }
            )
        )

        with MCPMemoryClient() as client:
            result = client.search(
                project_id="/test/project",
                query="test query",
            )

        assert result.namespace == "openai:model:1536"
        assert len(result.results) == 1
        assert result.results[0].id == "note-123"

    def test_search_with_filters(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with filters."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "results": []})
        )

        with MCPMemoryClient() as client:
            result = client.search(
                project_id="/test/project",
                query="test",
                group_id="feature-1",
                top_k=10,
                tags=["important"],
                since="2024-01-01T00:00:00Z",
                until="2024-12-31T23:59:59Z",
            )

        # Verify request
        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["groupId"] == "feature-1"
        assert data["params"]["topK"] == 10
        assert data["params"]["tags"] == ["important"]
        assert data["params"]["since"] == "2024-01-01T00:00:00Z"
        assert data["params"]["until"] == "2024-12-31T23:59:59Z"

    def test_search_empty_result(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with no results."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "results": []})
        )

        with MCPMemoryClient() as client:
            result = client.search(
                project_id="/test/project",
                query="nonexistent",
            )

        assert result.results == []

    def test_search_topk_default(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search uses default topK=5."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "results": []})
        )

        with MCPMemoryClient() as client:
            client.search(project_id="/test", query="test")

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["topK"] == 5

    def test_search_topk_boundary_zero(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with topK=0 (error expected from server)."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32602, "message": "topK must be > 0"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.search(project_id="/test", query="test", top_k=0)

        assert exc_info.value.is_invalid_params

    def test_search_topk_boundary_negative(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with topK=-1 (error expected from server)."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32602, "message": "topK must be > 0"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.search(project_id="/test", query="test", top_k=-1)

        assert exc_info.value.is_invalid_params

    def test_search_topk_boundary_large(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with large topK."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "results": []})
        )

        with MCPMemoryClient() as client:
            client.search(project_id="/test", query="test", top_k=1000)

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["topK"] == 1000

    def test_search_since_until(self, httpx_mock: HTTPXMock, rpc_response):
        """Test search with since/until boundary conditions."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "results": []})
        )

        with MCPMemoryClient() as client:
            from datetime import datetime

            client.search(
                project_id="/test",
                query="test",
                since=datetime(2024, 1, 1, 0, 0, 0),
                until=datetime(2024, 1, 2, 0, 0, 0),
            )

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["since"] == "2024-01-01T00:00:00Z"
        assert data["params"]["until"] == "2024-01-02T00:00:00Z"


class TestGet:
    """Tests for get method."""

    def test_get_existing(self, httpx_mock: HTTPXMock, rpc_response, sample_note_data):
        """Test get existing note."""
        httpx_mock.add_response(json=rpc_response(sample_note_data))

        with MCPMemoryClient() as client:
            note = client.get("note-123")

        assert note.id == "note-123"
        assert note.text == "This is a test note"

    def test_get_not_found(self, httpx_mock: HTTPXMock, rpc_response):
        """Test get non-existing note."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32001, "message": "Note not found"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.get("nonexistent")

        assert exc_info.value.is_not_found


class TestUpdate:
    """Tests for update method."""

    def test_update_title(self, httpx_mock: HTTPXMock, rpc_response):
        """Test update title only."""
        httpx_mock.add_response(json=rpc_response({"ok": True}))

        with MCPMemoryClient() as client:
            result = client.update("note-123", title="New Title")

        assert result["ok"] is True

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["patch"]["title"] == "New Title"
        assert "text" not in data["params"]["patch"]

    def test_update_text(self, httpx_mock: HTTPXMock, rpc_response):
        """Test update text (triggers re-embedding)."""
        httpx_mock.add_response(json=rpc_response({"ok": True}))

        with MCPMemoryClient() as client:
            result = client.update("note-123", text="New content")

        assert result["ok"] is True

    def test_update_not_found(self, httpx_mock: HTTPXMock, rpc_response):
        """Test update non-existing note."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32001, "message": "Note not found"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.update("nonexistent", title="New Title")

        assert exc_info.value.is_not_found


class TestListRecent:
    """Tests for list_recent method."""

    def test_list_recent_default(
        self, httpx_mock: HTTPXMock, rpc_response, sample_note_data
    ):
        """Test list_recent with default parameters."""
        httpx_mock.add_response(
            json=rpc_response(
                {"namespace": "openai:model:1536", "items": [sample_note_data]}
            )
        )

        with MCPMemoryClient() as client:
            result = client.list_recent(project_id="/test/project")

        assert result.namespace == "openai:model:1536"
        assert len(result.items) == 1

    def test_list_recent_with_limit(self, httpx_mock: HTTPXMock, rpc_response):
        """Test list_recent with limit."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "items": []})
        )

        with MCPMemoryClient() as client:
            client.list_recent(project_id="/test", limit=20)

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["limit"] == 20

    def test_list_recent_with_group(self, httpx_mock: HTTPXMock, rpc_response):
        """Test list_recent with groupId."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "items": []})
        )

        with MCPMemoryClient() as client:
            client.list_recent(project_id="/test", group_id="feature-1")

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["groupId"] == "feature-1"

    def test_list_recent_with_tags(self, httpx_mock: HTTPXMock, rpc_response):
        """Test list_recent with tags filter."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "items": []})
        )

        with MCPMemoryClient() as client:
            client.list_recent(project_id="/test", tags=["important", "review"])

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["tags"] == ["important", "review"]

    def test_list_recent_limit_zero(self, httpx_mock: HTTPXMock, rpc_response):
        """Test list_recent with limit=0."""
        httpx_mock.add_response(
            json=rpc_response({"namespace": "openai:model:1536", "items": []})
        )

        with MCPMemoryClient() as client:
            result = client.list_recent(project_id="/test", limit=0)

        assert result.items == []

    def test_list_recent_limit_negative(self, httpx_mock: HTTPXMock, rpc_response):
        """Test list_recent with limit=-1 (error expected from server)."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32602, "message": "limit must be >= 0"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.list_recent(project_id="/test", limit=-1)

        assert exc_info.value.is_invalid_params


class TestGetConfig:
    """Tests for get_config method."""

    def test_get_config(self, httpx_mock: HTTPXMock, rpc_response, sample_config_data):
        """Test get_config."""
        httpx_mock.add_response(json=rpc_response(sample_config_data))

        with MCPMemoryClient() as client:
            result = client.get_config()

        assert result.transport_defaults.default_transport == "stdio"
        assert result.embedder.provider == "openai"
        assert result.embedder.model == "text-embedding-3-small"

    def test_get_config_response_format(
        self, httpx_mock: HTTPXMock, rpc_response, sample_config_data
    ):
        """Test get_config response format validation."""
        httpx_mock.add_response(json=rpc_response(sample_config_data))

        with MCPMemoryClient() as client:
            result = client.get_config()

        # Verify all fields are present
        assert result.embedder.dim == 1536
        assert result.store.type == "chroma"
        assert result.paths.config_path.endswith("config.json")
        assert result.paths.data_dir.endswith("data")


class TestSetConfig:
    """Tests for set_config method."""

    def test_set_config_provider(self, httpx_mock: HTTPXMock, rpc_response):
        """Test set_config provider change."""
        httpx_mock.add_response(
            json=rpc_response({"ok": True, "effectiveNamespace": "ollama:llama:4096"})
        )

        with MCPMemoryClient() as client:
            result = client.set_config(provider="ollama", model="llama")

        assert result["ok"] is True
        assert result["effectiveNamespace"] == "ollama:llama:4096"

        # Verify request format
        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["embedder"]["provider"] == "ollama"
        assert data["params"]["embedder"]["model"] == "llama"

    def test_set_config_partial(self, httpx_mock: HTTPXMock, rpc_response):
        """Test set_config with partial update."""
        httpx_mock.add_response(
            json=rpc_response({"ok": True, "effectiveNamespace": "openai:new-model:1536"})
        )

        with MCPMemoryClient() as client:
            result = client.set_config(model="new-model")

        assert result["ok"] is True

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["embedder"]["model"] == "new-model"
        assert "provider" not in data["params"]["embedder"]

    def test_set_config_invalid_provider(self, httpx_mock: HTTPXMock, rpc_response):
        """Test set_config with invalid provider."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32602, "message": "invalid provider"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.set_config(provider="invalid")

        assert exc_info.value.is_invalid_params

    def test_set_config_empty(self, httpx_mock: HTTPXMock, rpc_response):
        """Test set_config with empty parameters."""
        httpx_mock.add_response(
            json=rpc_response({"ok": True, "effectiveNamespace": "openai:model:1536"})
        )

        with MCPMemoryClient() as client:
            result = client.set_config()

        assert result["ok"] is True


class TestUpsertGlobal:
    """Tests for upsert_global method."""

    def test_upsert_global_string(self, httpx_mock: HTTPXMock, rpc_response):
        """Test upsert_global with string value."""
        httpx_mock.add_response(
            json=rpc_response(
                {"ok": True, "id": "global-123", "namespace": "openai:model:1536"}
            )
        )

        with MCPMemoryClient() as client:
            result = client.upsert_global(
                project_id="/test",
                key="global.project.conventions",
                value="Use TypeScript",
            )

        assert result["ok"] is True
        assert result["id"] == "global-123"

    def test_upsert_global_object(self, httpx_mock: HTTPXMock, rpc_response):
        """Test upsert_global with object value."""
        httpx_mock.add_response(
            json=rpc_response(
                {"ok": True, "id": "global-456", "namespace": "openai:model:1536"}
            )
        )

        with MCPMemoryClient() as client:
            result = client.upsert_global(
                project_id="/test",
                key="global.memory.groupDefaults",
                value={"featurePrefix": "feature-", "taskPrefix": "task-"},
            )

        assert result["ok"] is True

    def test_upsert_global_with_updated_at(self, httpx_mock: HTTPXMock, rpc_response):
        """Test upsert_global with updated_at specified."""
        httpx_mock.add_response(
            json=rpc_response(
                {"ok": True, "id": "global-789", "namespace": "openai:model:1536"}
            )
        )

        with MCPMemoryClient() as client:
            result = client.upsert_global(
                project_id="/test",
                key="global.test",
                value="test",
                updated_at="2024-01-15T10:30:00Z",
            )

        import json

        request = httpx_mock.get_request()
        data = json.loads(request.read())
        assert data["params"]["updatedAt"] == "2024-01-15T10:30:00Z"

    def test_upsert_global_invalid_prefix(self, httpx_mock: HTTPXMock, rpc_response):
        """Test upsert_global with invalid key prefix."""
        httpx_mock.add_response(
            json=rpc_response(
                error={"code": -32003, "message": "key must start with 'global.'"}
            )
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.upsert_global(
                    project_id="/test",
                    key="invalid.key",
                    value="test",
                )

        assert exc_info.value.is_invalid_key_prefix


class TestGetGlobal:
    """Tests for get_global method."""

    def test_get_global_existing(self, httpx_mock: HTTPXMock, rpc_response):
        """Test get_global with existing key."""
        httpx_mock.add_response(
            json=rpc_response(
                {
                    "namespace": "openai:model:1536",
                    "found": True,
                    "id": "global-123",
                    "value": {"setting": "value"},
                    "updatedAt": "2024-01-15T10:30:00Z",
                }
            )
        )

        with MCPMemoryClient() as client:
            result = client.get_global(
                project_id="/test",
                key="global.project.conventions",
            )

        assert result.found is True
        assert result.id == "global-123"
        assert result.value == {"setting": "value"}

    def test_get_global_not_found(self, httpx_mock: HTTPXMock, rpc_response):
        """Test get_global with non-existing key."""
        httpx_mock.add_response(
            json=rpc_response(
                {
                    "namespace": "openai:model:1536",
                    "found": False,
                }
            )
        )

        with MCPMemoryClient() as client:
            result = client.get_global(
                project_id="/test",
                key="global.nonexistent",
            )

        assert result.found is False

    def test_get_global_not_found_fields(self, httpx_mock: HTTPXMock, rpc_response):
        """Test get_global found=false has null fields."""
        httpx_mock.add_response(
            json=rpc_response(
                {
                    "namespace": "openai:model:1536",
                    "found": False,
                }
            )
        )

        with MCPMemoryClient() as client:
            result = client.get_global(project_id="/test", key="global.missing")

        assert result.found is False
        assert result.id is None
        assert result.value is None
        assert result.updated_at is None


class TestErrors:
    """Tests for error handling."""

    def test_rpc_error_handling(self, httpx_mock: HTTPXMock, rpc_response):
        """Test general RPC error handling."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32603, "message": "Internal error"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.get_config()

        assert exc_info.value.code == -32603
        assert "Internal error" in str(exc_info.value)

    def test_rpc_error_invalid_params(self, httpx_mock: HTTPXMock, rpc_response):
        """Test -32602 error."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32602, "message": "Invalid params"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.get("test")

        assert exc_info.value.is_invalid_params

    def test_rpc_error_method_not_found(self, httpx_mock: HTTPXMock, rpc_response):
        """Test -32601 error."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32601, "message": "Method not found"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client._call("memory.unknown")

        assert exc_info.value.is_method_not_found

    def test_rpc_error_api_key_missing(self, httpx_mock: HTTPXMock, rpc_response):
        """Test -32002 error."""
        httpx_mock.add_response(
            json=rpc_response(error={"code": -32002, "message": "API key required"})
        )

        with MCPMemoryClient() as client:
            with pytest.raises(RPCError) as exc_info:
                client.add_note("/test", "global", "text")

        assert exc_info.value.is_api_key_missing

    def test_connection_error(self, httpx_mock: HTTPXMock):
        """Test connection error."""
        import httpx as httpx_lib

        httpx_mock.add_exception(httpx_lib.ConnectError("Connection refused"))

        with MCPMemoryClient() as client:
            with pytest.raises(ConnectionError):
                client.get_config()

    def test_timeout_error(self, httpx_mock: HTTPXMock):
        """Test timeout error."""
        import httpx as httpx_lib

        httpx_mock.add_exception(httpx_lib.TimeoutException("Request timeout"))

        with MCPMemoryClient() as client:
            with pytest.raises(TimeoutError):
                client.get_config()
