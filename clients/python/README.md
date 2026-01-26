# MCP Memory Client

Python client for MCP Memory Server - a local memory server with vector search capabilities.

## Installation

```bash
# Basic installation
pip install mcp-memory-client

# With LangGraph support
pip install mcp-memory-client[langchain]

# Development
pip install mcp-memory-client[dev]
```

## Quick Start

```python
from mcp_memory_client import MCPMemoryClient

# Create client (default: http://localhost:8765)
client = MCPMemoryClient()

# Or with custom settings
client = MCPMemoryClient(
    base_url="http://localhost:8765",
    timeout=30.0,
)

# Add a note
result = client.add_note(
    project_id="/path/to/project",
    group_id="global",
    text="Important project decision: Use TypeScript for the frontend.",
    title="Tech Stack Decision",
    tags=["decision", "frontend"],
)
print(f"Note ID: {result['id']}")

# Search notes
results = client.search(
    project_id="/path/to/project",
    query="frontend technology",
    top_k=5,
)
for note in results.results:
    print(f"- {note.title}: {note.text[:50]}... (score: {note.score})")

# Context manager support
with MCPMemoryClient() as client:
    note = client.get("note-123")
    print(note.text)
```

## API Reference

### Note Operations

#### `add_note(project_id, group_id, text, **kwargs)`

Add a note to the memory.

```python
result = client.add_note(
    project_id="/path/to/project",
    group_id="feature-auth",  # or "global", "task-123", etc.
    text="User authentication should use JWT tokens.",
    title="Auth Implementation",  # optional
    tags=["auth", "security"],   # optional
    source="meeting-notes",       # optional
    created_at="2024-01-15T10:30:00Z",  # optional
    metadata={"meeting_id": "123"},     # optional
)
# Returns: {"id": "note-xxx", "namespace": "openai:model:dim"}
```

#### `search(project_id, query, **kwargs)`

Search notes by vector similarity.

```python
results = client.search(
    project_id="/path/to/project",
    query="authentication implementation",
    group_id="feature-auth",  # optional, null = all groups
    top_k=10,                 # default: 5
    tags=["security"],        # optional, AND search
    since="2024-01-01T00:00:00Z",  # optional
    until="2024-12-31T23:59:59Z",  # optional
)
# Returns: SearchResult(namespace=str, results=[Note, ...])
```

#### `get(note_id)`

Get a note by ID.

```python
note = client.get("note-123")
# Returns: Note
```

#### `update(note_id, **kwargs)`

Update a note (patch).

```python
result = client.update(
    "note-123",
    title="Updated Title",
    text="Updated content",  # triggers re-embedding
    tags=["new-tag"],
)
# Returns: {"ok": True}
```

#### `list_recent(project_id, **kwargs)`

List recent notes.

```python
results = client.list_recent(
    project_id="/path/to/project",
    group_id="feature-auth",  # optional
    limit=20,                 # optional
    tags=["important"],       # optional
)
# Returns: ListRecentResult(namespace=str, items=[Note, ...])
```

### Config Operations

#### `get_config()`

Get server configuration.

```python
config = client.get_config()
print(f"Provider: {config.embedder.provider}")
print(f"Model: {config.embedder.model}")
```

#### `set_config(**kwargs)`

Update embedder configuration.

```python
result = client.set_config(
    provider="openai",
    model="text-embedding-3-small",
    api_key="sk-xxx",  # optional
)
# Returns: {"ok": True, "effectiveNamespace": "openai:model:dim"}
```

### Global Settings

#### `upsert_global(project_id, key, value, **kwargs)`

Save a global setting.

```python
result = client.upsert_global(
    project_id="/path/to/project",
    key="global.project.conventions",  # must start with "global."
    value={"coding_style": "pep8"},
)
# Returns: {"ok": True, "id": "global-xxx", "namespace": "..."}
```

#### `get_global(project_id, key)`

Get a global setting.

```python
result = client.get_global(
    project_id="/path/to/project",
    key="global.project.conventions",
)
if result.found:
    print(f"Value: {result.value}")
```

## LangGraph Integration

```python
from mcp_memory_client.langchain_tools import configure_memory_client, MEMORY_TOOLS
from langgraph.prebuilt import create_react_agent
from langchain_openai import ChatOpenAI

# Configure the memory client
configure_memory_client(base_url="http://localhost:8765")

# Create agent with memory tools
llm = ChatOpenAI(model="gpt-4")
agent = create_react_agent(llm, tools=MEMORY_TOOLS)

# The agent can now use:
# - memory_search: Search project memory
# - memory_add_note: Add notes
# - memory_get_note: Get note by ID
# - memory_list_recent: List recent notes
# - memory_upsert_global: Save global settings
# - memory_get_global: Get global settings
```

## Error Handling

```python
from mcp_memory_client import MCPMemoryClient
from mcp_memory_client.exceptions import (
    MCPMemoryError,
    ConnectionError,
    TimeoutError,
    RPCError,
)

try:
    with MCPMemoryClient() as client:
        note = client.get("nonexistent-id")
except RPCError as e:
    if e.is_not_found:
        print("Note not found")
    elif e.is_api_key_missing:
        print("API key required")
    elif e.is_invalid_params:
        print(f"Invalid parameters: {e.message}")
    else:
        print(f"RPC error [{e.code}]: {e.message}")
except ConnectionError:
    print("Failed to connect to server")
except TimeoutError:
    print("Request timeout")
except MCPMemoryError as e:
    print(f"Error: {e}")
```

## Security Considerations

- **API Key handling**: The `api_key` parameter in `set_config()` is sent to the server in plain text. Do not log or expose it.
- **base_url validation**: This client defaults to `http://localhost:8765`. Connecting to external URLs is the user's responsibility. Ensure the server is trusted.
- **HTTPS**: For production use with external servers, always use HTTPS to encrypt communication.

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Type checking
mypy src/mcp_memory_client

# Linting
ruff check src tests
```

## License

MIT
