"""MCP Memory Client exceptions."""
from typing import Any


class MCPMemoryError(Exception):
    """MCP Memory client base exception."""

    pass


class ConnectionError(MCPMemoryError):
    """Connection error."""

    pass


class TimeoutError(MCPMemoryError):
    """Timeout error."""

    pass


class RPCError(MCPMemoryError):
    """JSON-RPC error."""

    def __init__(self, code: int, message: str, data: Any = None) -> None:
        self.code = code
        self.message = message
        self.data = data
        super().__init__(f"[{code}] {message}")

    @property
    def is_invalid_params(self) -> bool:
        """Check if error is invalid params (-32602)."""
        return self.code == -32602

    @property
    def is_method_not_found(self) -> bool:
        """Check if error is method not found (-32601)."""
        return self.code == -32601

    @property
    def is_not_found(self) -> bool:
        """Check if error is not found (-32001)."""
        return self.code == -32001

    @property
    def is_api_key_missing(self) -> bool:
        """Check if error is API key missing (-32002)."""
        return self.code == -32002

    @property
    def is_invalid_key_prefix(self) -> bool:
        """Check if error is invalid key prefix (-32003)."""
        return self.code == -32003
