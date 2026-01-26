"""MCP Memory Client data models."""
from typing import Any

from pydantic import BaseModel, Field


class Note(BaseModel):
    """Note model."""

    id: str
    project_id: str = Field(alias="projectId")
    group_id: str = Field(alias="groupId")
    title: str | None = None
    text: str
    tags: list[str] = Field(default_factory=list)
    source: str | None = None
    created_at: str = Field(alias="createdAt")
    namespace: str | None = None
    score: float | None = None  # search result only
    metadata: dict[str, Any] | None = None

    model_config = {"populate_by_name": True}


class SearchResult(BaseModel):
    """Search result (for memory.search)."""

    namespace: str
    results: list[Note] = Field(default_factory=list)


class ListRecentResult(BaseModel):
    """List recent result (for memory.list_recent)."""

    namespace: str
    items: list[Note] = Field(default_factory=list)


class EmbedderConfig(BaseModel):
    """Embedder config."""

    provider: str
    model: str
    dim: int
    base_url: str | None = Field(default=None, alias="baseUrl")

    model_config = {"populate_by_name": True}


class StoreConfig(BaseModel):
    """Store config."""

    type: str
    path: str | None = None
    url: str | None = None


class PathsConfig(BaseModel):
    """Paths config."""

    config_path: str = Field(alias="configPath")
    data_dir: str = Field(alias="dataDir")

    model_config = {"populate_by_name": True}


class TransportDefaults(BaseModel):
    """Transport defaults config."""

    default_transport: str = Field(alias="defaultTransport")

    model_config = {"populate_by_name": True}


class ConfigResult(BaseModel):
    """get_config result."""

    transport_defaults: TransportDefaults = Field(alias="transportDefaults")
    embedder: EmbedderConfig
    store: StoreConfig
    paths: PathsConfig

    model_config = {"populate_by_name": True}


class GlobalValue(BaseModel):
    """get_global result."""

    namespace: str
    found: bool
    id: str | None = None
    value: Any | None = None
    updated_at: str | None = Field(default=None, alias="updatedAt")

    model_config = {"populate_by_name": True}
