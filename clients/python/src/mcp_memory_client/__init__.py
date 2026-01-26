"""MCP Memory Client - Python client for MCP Memory Server."""
from mcp_memory_client.client import MCPMemoryClient
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

__all__ = [
    "MCPMemoryClient",
    "MCPMemoryError",
    "ConnectionError",
    "TimeoutError",
    "RPCError",
    "Note",
    "SearchResult",
    "ListRecentResult",
    "ConfigResult",
    "GlobalValue",
]

__version__ = "0.1.0"
