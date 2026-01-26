"""MCP Memory Client - HTTP JSON-RPC 2.0 client."""
from datetime import datetime
from typing import Any

import httpx

from mcp_memory_client.exceptions import (
    ConnectionError,
    MCPMemoryError,
    RPCError,
    TimeoutError,
)
from mcp_memory_client.models import (
    ConfigResult,
    GlobalValue,
    ListRecentResult,
    Note,
    SearchResult,
)


def _to_camel_case(snake_str: str) -> str:
    """Convert snake_case to camelCase."""
    components = snake_str.split("_")
    return components[0] + "".join(x.title() for x in components[1:])


def _convert_keys_to_camel(
    data: dict[str, Any], skip_keys: set[str] | None = None
) -> dict[str, Any]:
    """Convert dict keys from snake_case to camelCase recursively.

    Args:
        data: Dictionary to convert
        skip_keys: Keys whose values should not be converted (e.g., "metadata")

    Returns:
        Dictionary with camelCase keys
    """
    if skip_keys is None:
        skip_keys = {"metadata", "value"}  # Don't convert user-provided data

    result = {}
    for key, value in data.items():
        camel_key = _to_camel_case(key)
        if key in skip_keys:
            # Keep value as-is (user-provided data)
            result[camel_key] = value
        elif isinstance(value, dict):
            result[camel_key] = _convert_keys_to_camel(value, skip_keys)
        elif isinstance(value, list):
            result[camel_key] = [
                _convert_keys_to_camel(item, skip_keys) if isinstance(item, dict) else item
                for item in value
            ]
        else:
            result[camel_key] = value
    return result


def _format_datetime(dt: datetime | str | None) -> str | None:
    """Format datetime to ISO8601 string."""
    if dt is None:
        return None
    if isinstance(dt, str):
        return dt
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


class MCPMemoryClient:
    """MCP Memory Server HTTP JSON-RPC client."""

    def __init__(
        self,
        base_url: str = "http://localhost:8765",
        timeout: float = 30.0,
    ) -> None:
        """Initialize the client.

        Args:
            base_url: Server URL (default: http://localhost:8765)
            timeout: Request timeout in seconds (default: 30.0)
        """
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._client = httpx.Client(timeout=timeout)
        self._request_id = 0

    def _next_id(self) -> int:
        """Get next request ID."""
        self._request_id += 1
        return self._request_id

    def _call(self, method: str, params: dict[str, Any] | None = None) -> Any:
        """Make a JSON-RPC 2.0 call.

        Args:
            method: RPC method name
            params: Method parameters

        Returns:
            Result from server

        Raises:
            RPCError: JSON-RPC error
            ConnectionError: Connection failed
            TimeoutError: Request timeout
            MCPMemoryError: Other errors
        """
        request_body = {
            "jsonrpc": "2.0",
            "id": self._next_id(),
            "method": method,
        }
        if params is not None:
            # Convert snake_case to camelCase
            request_body["params"] = _convert_keys_to_camel(params)

        try:
            response = self._client.post(
                f"{self._base_url}/rpc",
                json=request_body,
            )
            response.raise_for_status()
        except httpx.ConnectError as e:
            raise ConnectionError(f"Failed to connect to {self._base_url}: {e}") from e
        except httpx.TimeoutException as e:
            raise TimeoutError(f"Request timeout: {e}") from e
        except httpx.HTTPStatusError as e:
            raise MCPMemoryError(f"HTTP error: {e}") from e

        try:
            data = response.json()
        except ValueError as e:
            raise MCPMemoryError(f"Invalid JSON response: {e}") from e

        if "error" in data:
            error = data["error"]
            raise RPCError(
                code=error.get("code", -32603),
                message=error.get("message", "Unknown error"),
                data=error.get("data"),
            )

        return data.get("result")

    # --- Note operations ---

    def add_note(
        self,
        project_id: str,
        group_id: str,
        text: str,
        *,
        title: str | None = None,
        tags: list[str] | None = None,
        source: str | None = None,
        created_at: datetime | str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, str]:
        """Add a note.

        Args:
            project_id: Project ID (path)
            group_id: Group ID
            text: Note content
            title: Note title (optional)
            tags: Tag list (optional)
            source: Source (optional)
            created_at: Created time (optional)
            metadata: Metadata (optional)

        Returns:
            {"id": str, "namespace": str}

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        params: dict[str, Any] = {
            "project_id": project_id,
            "group_id": group_id,
            "text": text,
        }
        if title is not None:
            params["title"] = title
        if tags is not None:
            params["tags"] = tags
        if source is not None:
            params["source"] = source
        if created_at is not None:
            params["created_at"] = _format_datetime(created_at)
        if metadata is not None:
            params["metadata"] = metadata

        return self._call("memory.add_note", params)

    def search(
        self,
        project_id: str,
        query: str,
        *,
        group_id: str | None = None,
        top_k: int = 5,
        tags: list[str] | None = None,
        since: datetime | str | None = None,
        until: datetime | str | None = None,
    ) -> SearchResult:
        """Search notes by vector similarity.

        Args:
            project_id: Project ID (path)
            query: Search query
            group_id: Group ID filter (optional)
            top_k: Number of results (default: 5)
            tags: Tag filter (AND search)
            since: Start time filter (inclusive)
            until: End time filter (exclusive)

        Returns:
            SearchResult with matching notes

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        params: dict[str, Any] = {
            "project_id": project_id,
            "query": query,
            "top_k": top_k,
        }
        if group_id is not None:
            params["group_id"] = group_id
        if tags is not None:
            params["tags"] = tags
        if since is not None:
            params["since"] = _format_datetime(since)
        if until is not None:
            params["until"] = _format_datetime(until)

        result = self._call("memory.search", params)
        return SearchResult(
            namespace=result["namespace"],
            results=[Note.model_validate(n) for n in result.get("results", [])],
        )

    def get(self, note_id: str) -> Note:
        """Get a note by ID.

        Args:
            note_id: Note ID

        Returns:
            Note

        Raises:
            RPCError: JSON-RPC error (including not found)
            MCPMemoryError: Connection/timeout error
        """
        result = self._call("memory.get", {"id": note_id})
        return Note.model_validate(result)

    def update(
        self,
        note_id: str,
        *,
        title: str | None = None,
        text: str | None = None,
        tags: list[str] | None = None,
        source: str | None = None,
        group_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, bool]:
        """Update a note (patch).

        Args:
            note_id: Note ID
            title: New title (optional)
            text: New text (triggers re-embedding)
            tags: New tags (optional)
            source: New source (optional)
            group_id: New group ID (optional)
            metadata: New metadata (optional)

        Returns:
            {"ok": True}

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        patch: dict[str, Any] = {}
        if title is not None:
            patch["title"] = title
        if text is not None:
            patch["text"] = text
        if tags is not None:
            patch["tags"] = tags
        if source is not None:
            patch["source"] = source
        if group_id is not None:
            patch["group_id"] = group_id
        if metadata is not None:
            patch["metadata"] = metadata

        return self._call("memory.update", {"id": note_id, "patch": patch})

    def list_recent(
        self,
        project_id: str,
        *,
        group_id: str | None = None,
        limit: int | None = None,
        tags: list[str] | None = None,
    ) -> ListRecentResult:
        """List recent notes.

        Args:
            project_id: Project ID (path)
            group_id: Group ID filter (optional)
            limit: Max number of results (optional)
            tags: Tag filter (AND search)

        Returns:
            ListRecentResult with notes (newest first)

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        params: dict[str, Any] = {"project_id": project_id}
        if group_id is not None:
            params["group_id"] = group_id
        if limit is not None:
            params["limit"] = limit
        if tags is not None:
            params["tags"] = tags

        result = self._call("memory.list_recent", params)
        return ListRecentResult(
            namespace=result["namespace"],
            items=[Note.model_validate(n) for n in result.get("items", [])],
        )

    # --- Config operations ---

    def get_config(self) -> ConfigResult:
        """Get server configuration.

        Returns:
            ConfigResult with server settings

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        result = self._call("memory.get_config")
        return ConfigResult.model_validate(result)

    def set_config(
        self,
        *,
        provider: str | None = None,
        model: str | None = None,
        base_url: str | None = None,
        api_key: str | None = None,
    ) -> dict[str, Any]:
        """Set embedder configuration.

        Args:
            provider: Embedder provider (openai, ollama, local)
            model: Model name
            base_url: API base URL
            api_key: API key

        Returns:
            {"ok": True, "effectiveNamespace": str}

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        embedder: dict[str, Any] = {}
        if provider is not None:
            embedder["provider"] = provider
        if model is not None:
            embedder["model"] = model
        if base_url is not None:
            embedder["base_url"] = base_url
        if api_key is not None:
            embedder["api_key"] = api_key

        return self._call("memory.set_config", {"embedder": embedder})

    # --- Global KV operations ---

    def upsert_global(
        self,
        project_id: str,
        key: str,
        value: Any,
        *,
        updated_at: datetime | str | None = None,
    ) -> dict[str, Any]:
        """Upsert a global setting.

        Args:
            project_id: Project ID (path)
            key: Key (must start with "global.")
            value: Value (any JSON value)
            updated_at: Updated time (optional)

        Returns:
            {"ok": True, "id": str, "namespace": str}

        Raises:
            RPCError: JSON-RPC error (including invalid key prefix)
            MCPMemoryError: Connection/timeout error
        """
        params: dict[str, Any] = {
            "project_id": project_id,
            "key": key,
            "value": value,
        }
        if updated_at is not None:
            params["updated_at"] = _format_datetime(updated_at)

        return self._call("memory.upsert_global", params)

    def get_global(
        self,
        project_id: str,
        key: str,
    ) -> GlobalValue:
        """Get a global setting.

        Args:
            project_id: Project ID (path)
            key: Key

        Returns:
            GlobalValue with found status and value

        Raises:
            RPCError: JSON-RPC error
            MCPMemoryError: Connection/timeout error
        """
        result = self._call("memory.get_global", {"project_id": project_id, "key": key})
        return GlobalValue.model_validate(result)

    # --- Lifecycle ---

    def close(self) -> None:
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self) -> "MCPMemoryClient":
        """Context manager enter."""
        return self

    def __exit__(self, *args: Any) -> None:
        """Context manager exit."""
        self.close()
