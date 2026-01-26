"""LangGraph/LangChain tools for MCP Memory Client.

Usage:
    from mcp_memory_client.langchain_tools import configure_memory_client, MEMORY_TOOLS

    # Configure the client
    configure_memory_client(base_url="http://localhost:8765")

    # Use tools in LangGraph
    from langgraph.prebuilt import create_react_agent
    agent = create_react_agent(llm, tools=MEMORY_TOOLS)
"""
from typing import Any

try:
    from langchain_core.tools import tool
except ImportError:
    raise ImportError(
        "langchain-core is required for LangGraph tools. "
        "Install with: pip install mcp-memory-client[langchain]"
    )

from mcp_memory_client.client import MCPMemoryClient

# Global client instance
_client: MCPMemoryClient | None = None


def configure_memory_client(
    base_url: str = "http://localhost:8765",
    timeout: float = 30.0,
) -> None:
    """Configure the global MCP Memory client for LangGraph tools.

    Args:
        base_url: Server URL (default: http://localhost:8765)
        timeout: Request timeout in seconds (default: 30.0)
    """
    global _client
    if _client is not None:
        _client.close()
    _client = MCPMemoryClient(base_url=base_url, timeout=timeout)


def get_client() -> MCPMemoryClient:
    """Get the configured client.

    Raises:
        RuntimeError: If configure_memory_client() hasn't been called
    """
    if _client is None:
        raise RuntimeError("Call configure_memory_client() first")
    return _client


@tool
def memory_search(
    project_id: str,
    query: str,
    group_id: str | None = None,
    top_k: int = 5,
) -> str:
    """Search project memory by semantic similarity.

    Args:
        project_id: Project ID (path)
        query: Search query
        group_id: Group ID filter (optional, null searches all groups)
        top_k: Number of results (default: 5)

    Returns:
        JSON string of search results
    """
    import json

    client = get_client()
    result = client.search(project_id, query, group_id=group_id, top_k=top_k)
    return json.dumps(
        [n.model_dump(by_alias=True) for n in result.results], ensure_ascii=False
    )


@tool
def memory_add_note(
    project_id: str,
    group_id: str,
    text: str,
    title: str | None = None,
    tags: list[str] | None = None,
) -> str:
    """Add a note to project memory.

    Args:
        project_id: Project ID (path)
        group_id: Group ID ("global", "feature-xxx", "task-xxx", etc.)
        text: Note content
        title: Note title (optional)
        tags: Tag list (optional)

    Returns:
        JSON string with id and namespace
    """
    import json

    client = get_client()
    result = client.add_note(project_id, group_id, text, title=title, tags=tags)
    return json.dumps(result, ensure_ascii=False)


@tool
def memory_get_note(note_id: str) -> str:
    """Get a note by ID.

    Args:
        note_id: Note ID

    Returns:
        JSON string of the note
    """
    import json

    client = get_client()
    note = client.get(note_id)
    return json.dumps(note.model_dump(by_alias=True), ensure_ascii=False)


@tool
def memory_list_recent(
    project_id: str,
    group_id: str | None = None,
    limit: int = 10,
) -> str:
    """List recent notes from project memory.

    Args:
        project_id: Project ID (path)
        group_id: Group ID filter (optional)
        limit: Number of results (default: 10)

    Returns:
        JSON string of recent notes
    """
    import json

    client = get_client()
    result = client.list_recent(project_id, group_id=group_id, limit=limit)
    return json.dumps(
        [n.model_dump(by_alias=True) for n in result.items], ensure_ascii=False
    )


@tool
def memory_upsert_global(
    project_id: str,
    key: str,
    value: Any,
) -> str:
    """Save a global setting.

    Args:
        project_id: Project ID (path)
        key: Key (must start with "global.")
        value: Value (any JSON value)

    Returns:
        JSON string with result
    """
    import json

    client = get_client()
    result = client.upsert_global(project_id, key, value)
    return json.dumps(result, ensure_ascii=False)


@tool
def memory_get_global(
    project_id: str,
    key: str,
) -> str:
    """Get a global setting.

    Args:
        project_id: Project ID (path)
        key: Key

    Returns:
        JSON string with found status and value
    """
    import json

    client = get_client()
    result = client.get_global(project_id, key)
    return json.dumps(result.model_dump(by_alias=True), ensure_ascii=False)


# Export all tools as a list
MEMORY_TOOLS = [
    memory_search,
    memory_add_note,
    memory_get_note,
    memory_list_recent,
    memory_upsert_global,
    memory_get_global,
]
