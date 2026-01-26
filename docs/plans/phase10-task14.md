# Phase 10 Task 14: Pythonクライアント実装計画

## 概要

MCP Memory ServerのPythonクライアントライブラリを実装する。HTTP JSON-RPC 2.0を通じてサーバーと通信し、LangGraphとの統合サンプルも提供する。

---

## 要件トレーサビリティ

| 要件ID | TODO項目 | テストケース | 実装箇所 | 備考 |
|--------|----------|--------------|----------|------|
| REQ-1 | MCPMemoryClient クラス | TestClient_* | clients/python/src/mcp_memory_client/client.py | |
| REQ-2 | 全9メソッド対応 | TestClient_{method}_* | clients/python/src/mcp_memory_client/client.py | |
| REQ-3 | 型ヒント付き | (型チェッカーで検証) | 全.pyファイル | |
| REQ-4 | 接続設定（base_url, timeout） | TestClient_connection_* | clients/python/src/mcp_memory_client/client.py | |
| REQ-5 | エラーハンドリング | TestClient_error_* | clients/python/src/mcp_memory_client/exceptions.py | 本計画で方針を確定 |
| REQ-6 | LangGraph Tool定義サンプル | (手動検証) | clients/python/src/mcp_memory_client/langchain_tools.py | LangGraph >= 0.2.0 を前提 |
| REQ-7 | pyproject.toml | (pip install -e . で検証) | clients/python/pyproject.toml | |
| REQ-8 | 使用例ドキュメント | (手動検証) | clients/python/README.md | |

### 未確定事項（実装時に最終決定）

| 項目 | 現状の方針 | 実装時に検討すべき点 |
|------|------------|----------------------|
| LangGraphバージョン | >= 0.2.0 を前提 | 最新バージョンでのAPI変更に対応 |
| ツール登録方法 | `@tool` デコレータ使用 | LangGraph側のベストプラクティスに合わせる |

---

## ディレクトリ構成

```
clients/python/
├── pyproject.toml              # パッケージ定義
├── README.md                   # 使用例ドキュメント
├── src/
│   └── mcp_memory_client/
│       ├── __init__.py         # パッケージエクスポート
│       ├── client.py           # MCPMemoryClient クラス
│       ├── models.py           # データモデル（Pydantic）
│       ├── exceptions.py       # カスタム例外
│       └── langchain_tools.py  # LangGraph Tool定義
└── tests/
    ├── __init__.py
    ├── conftest.py             # pytest fixtures
    ├── test_client.py          # クライアントテスト
    └── test_models.py          # モデルテスト
```

---

## 依存関係

### 必須
- Python >= 3.10
- httpx >= 0.27.0 (HTTP クライアント、async対応)
- pydantic >= 2.0.0 (データバリデーション)

### オプション（LangGraph統合用）
- langchain-core >= 0.3.0
- langgraph >= 0.2.0

### 開発用
- pytest >= 8.0.0
- pytest-asyncio >= 0.24.0
- pytest-httpx >= 0.32.0 (httpx モック)
- mypy >= 1.11.0
- ruff >= 0.6.0

---

## クラス設計

### 1. MCPMemoryClient (client.py)

```python
from typing import Any
from datetime import datetime
from .models import (
    Note, NoteInput, SearchResult, SearchInput,
    UpdatePatch, ConfigResult, SetConfigInput,
    GlobalValue, ListRecentInput
)
from .exceptions import MCPMemoryError, RPCError

class MCPMemoryClient:
    """MCP Memory Server HTTP JSON-RPC クライアント"""

    def __init__(
        self,
        base_url: str = "http://localhost:8765",
        timeout: float = 30.0,
    ) -> None:
        """
        Args:
            base_url: サーバーURL（デフォルト: http://localhost:8765）
            timeout: リクエストタイムアウト秒数（デフォルト: 30.0）
        """
        ...

    # --- Note操作 ---

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
        """
        ノートを追加

        Returns:
            {"id": str, "namespace": str}

        Raises:
            RPCError: JSON-RPCエラー
            MCPMemoryError: 接続/タイムアウトエラー
        """
        ...

    def search(
        self,
        project_id: str,
        query: str,
        *,
        group_id: str | None = None,
        top_k: int = 5,  # サーバーのデフォルトと同じ
        tags: list[str] | None = None,
        since: datetime | str | None = None,
        until: datetime | str | None = None,
    ) -> SearchResult:
        """
        ベクトル検索

        Returns:
            SearchResult(namespace=str, results=[Note, ...])

        Raises:
            RPCError: JSON-RPCエラー
            MCPMemoryError: 接続/タイムアウトエラー
        """
        ...

    def get(self, note_id: str) -> Note:
        """
        ID指定で1件取得

        Raises:
            RPCError: JSON-RPCエラー（not found含む）
            MCPMemoryError: 接続/タイムアウトエラー
        """
        ...

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
        """
        ノートを更新（patch）

        Returns:
            {"ok": True}

        Raises:
            RPCError: JSON-RPCエラー
            MCPMemoryError: 接続/タイムアウトエラー
        """
        ...

    def list_recent(
        self,
        project_id: str,
        *,
        group_id: str | None = None,
        limit: int | None = None,
        tags: list[str] | None = None,
    ) -> ListRecentResult:
        """
        最近のノート一覧（createdAt降順）

        Returns:
            ListRecentResult(namespace=str, items=[Note, ...])

        Raises:
            RPCError: JSON-RPCエラー
            MCPMemoryError: 接続/タイムアウトエラー
        """
        ...

    # --- Config操作 ---

    def get_config(self) -> ConfigResult:
        """
        設定を取得

        Returns:
            ConfigResult(transport_defaults=..., embedder=..., store=..., paths=...)
        """
        ...

    def set_config(
        self,
        *,
        provider: str | None = None,
        model: str | None = None,
        base_url: str | None = None,
        api_key: str | None = None,
    ) -> dict[str, Any]:
        """
        埋め込み設定を変更

        Note:
            内部で embedder オブジェクトに変換してJSON-RPC送信:
            {"embedder": {"provider": ..., "model": ..., ...}}

        Returns:
            {"ok": True, "effectiveNamespace": str}
        """
        ...

    # --- Global KV操作 ---

    def upsert_global(
        self,
        project_id: str,
        key: str,
        value: Any,
        *,
        updated_at: datetime | str | None = None,
    ) -> dict[str, Any]:
        """
        グローバル設定を保存（upsert）

        Note:
            keyは "global." プレフィックスが必須

        Returns:
            {"ok": True, "id": str, "namespace": str}

        Raises:
            RPCError: キープレフィックスエラー等
        """
        ...

    def get_global(
        self,
        project_id: str,
        key: str,
    ) -> GlobalValue:
        """
        グローバル設定を取得

        Returns:
            GlobalValue(namespace=str, found=bool, id=str|None, value=Any|None, updated_at=str|None)
        """
        ...

    # --- 内部メソッド ---

    def _call(self, method: str, params: dict[str, Any] | None = None) -> Any:
        """JSON-RPC 2.0 呼び出し（内部）"""
        ...

    def close(self) -> None:
        """HTTPクライアントをクローズ"""
        ...

    def __enter__(self) -> "MCPMemoryClient":
        ...

    def __exit__(self, *args: Any) -> None:
        ...
```

### 2. データモデル (models.py)

```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Any

class Note(BaseModel):
    """ノートモデル"""
    id: str
    project_id: str = Field(alias="projectId")
    group_id: str = Field(alias="groupId")
    title: str | None = None
    text: str
    tags: list[str] = Field(default_factory=list)
    source: str | None = None
    created_at: str = Field(alias="createdAt")
    namespace: str | None = None
    score: float | None = None  # search結果のみ
    metadata: dict[str, Any] | None = None

    model_config = {"populate_by_name": True}

class SearchResult(BaseModel):
    """検索結果（search用）"""
    namespace: str
    results: list[Note] = Field(default_factory=list)

class ListRecentResult(BaseModel):
    """list_recent結果（仕様では items キー）"""
    namespace: str
    items: list[Note] = Field(default_factory=list)

class EmbedderConfig(BaseModel):
    """Embedder設定"""
    provider: str
    model: str
    dim: int
    base_url: str | None = Field(default=None, alias="baseUrl")

    model_config = {"populate_by_name": True}

class StoreConfig(BaseModel):
    """Store設定"""
    type: str
    path: str | None = None
    url: str | None = None

class PathsConfig(BaseModel):
    """パス設定"""
    config_path: str = Field(alias="configPath")
    data_dir: str = Field(alias="dataDir")

    model_config = {"populate_by_name": True}

class TransportDefaults(BaseModel):
    """Transport設定"""
    default_transport: str = Field(alias="defaultTransport")

    model_config = {"populate_by_name": True}

class ConfigResult(BaseModel):
    """get_config結果"""
    transport_defaults: TransportDefaults = Field(alias="transportDefaults")
    embedder: EmbedderConfig
    store: StoreConfig
    paths: PathsConfig

    model_config = {"populate_by_name": True}

class GlobalValue(BaseModel):
    """get_global結果"""
    namespace: str
    found: bool
    id: str | None = None
    value: Any | None = None
    updated_at: str | None = Field(default=None, alias="updatedAt")

    model_config = {"populate_by_name": True}
```

### 3. 例外クラス (exceptions.py)

```python
class MCPMemoryError(Exception):
    """MCP Memory クライアントの基底例外"""
    pass

class ConnectionError(MCPMemoryError):
    """接続エラー"""
    pass

class TimeoutError(MCPMemoryError):
    """タイムアウトエラー"""
    pass

class RPCError(MCPMemoryError):
    """JSON-RPC エラー"""

    def __init__(self, code: int, message: str, data: Any = None):
        self.code = code
        self.message = message
        self.data = data
        super().__init__(f"[{code}] {message}")

    @property
    def is_invalid_params(self) -> bool:
        return self.code == -32602

    @property
    def is_method_not_found(self) -> bool:
        return self.code == -32601

    @property
    def is_not_found(self) -> bool:
        return self.code == -32001  # カスタムコード

    @property
    def is_api_key_missing(self) -> bool:
        return self.code == -32002  # カスタムコード
```

### 4. LangGraph Tools (langchain_tools.py)

```python
from typing import Any
from langchain_core.tools import tool

# グローバルクライアントインスタンス（設定用）
_client: "MCPMemoryClient | None" = None

def configure_memory_client(
    base_url: str = "http://localhost:8765",
    timeout: float = 30.0,
) -> None:
    """LangGraph用クライアントを設定"""
    global _client
    from .client import MCPMemoryClient
    _client = MCPMemoryClient(base_url=base_url, timeout=timeout)

def get_client() -> "MCPMemoryClient":
    """設定済みクライアントを取得"""
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
    """
    プロジェクトのメモリを検索する

    Args:
        project_id: プロジェクトID（パス）
        query: 検索クエリ
        group_id: グループID（オプション、nullで全グループ検索）
        top_k: 取得件数（デフォルト5）

    Returns:
        検索結果のJSON文字列
    """
    import json
    client = get_client()
    result = client.search(project_id, query, group_id=group_id, top_k=top_k)
    return json.dumps([n.model_dump() for n in result.results], ensure_ascii=False)

@tool
def memory_add_note(
    project_id: str,
    group_id: str,
    text: str,
    title: str | None = None,
    tags: list[str] | None = None,
) -> str:
    """
    プロジェクトにメモを追加する

    Args:
        project_id: プロジェクトID（パス）
        group_id: グループID（"global", "feature-xxx", "task-xxx"等）
        text: メモ内容
        title: タイトル（オプション）
        tags: タグリスト（オプション）

    Returns:
        追加結果のJSON（id, namespace）
    """
    import json
    client = get_client()
    result = client.add_note(project_id, group_id, text, title=title, tags=tags)
    return json.dumps(result, ensure_ascii=False)

@tool
def memory_get_note(note_id: str) -> str:
    """
    IDでメモを取得する

    Args:
        note_id: ノートID

    Returns:
        ノートのJSON
    """
    import json
    client = get_client()
    note = client.get(note_id)
    return json.dumps(note.model_dump(), ensure_ascii=False)

@tool
def memory_list_recent(
    project_id: str,
    group_id: str | None = None,
    limit: int = 10,
) -> str:
    """
    最近のメモを取得する

    Args:
        project_id: プロジェクトID
        group_id: グループID（オプション）
        limit: 取得件数（デフォルト10）

    Returns:
        メモリストのJSON
    """
    import json
    client = get_client()
    result = client.list_recent(project_id, group_id=group_id, limit=limit)
    return json.dumps([n.model_dump() for n in result.items], ensure_ascii=False)

@tool
def memory_upsert_global(
    project_id: str,
    key: str,
    value: Any,
) -> str:
    """
    グローバル設定を保存する

    Args:
        project_id: プロジェクトID
        key: キー（"global."プレフィックス必須）
        value: 値（任意のJSON値）

    Returns:
        結果のJSON
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
    """
    グローバル設定を取得する

    Args:
        project_id: プロジェクトID
        key: キー

    Returns:
        設定値のJSON
    """
    import json
    client = get_client()
    result = client.get_global(project_id, key)
    return json.dumps(result.model_dump(), ensure_ascii=False)

# ツール一覧をエクスポート
MEMORY_TOOLS = [
    memory_search,
    memory_add_note,
    memory_get_note,
    memory_list_recent,
    memory_upsert_global,
    memory_get_global,
]
```

---

## エラーハンドリング方針

### HTTPエラー
| 状況 | 例外 |
|------|------|
| 接続失敗 | `ConnectionError` |
| タイムアウト | `TimeoutError` |
| HTTP 4xx/5xx | `MCPMemoryError` |

### JSON-RPCエラー
| code | 意味 | 例外 |
|------|------|------|
| -32700 | Parse error | `RPCError` |
| -32600 | Invalid Request | `RPCError` |
| -32601 | Method not found | `RPCError` |
| -32602 | Invalid params | `RPCError` |
| -32603 | Internal error | `RPCError` |
| -32001 | Not found | `RPCError` |
| -32002 | API key missing | `RPCError` |
| -32003 | Invalid key prefix | `RPCError` |

---

## テストケース一覧

### test_client.py

#### 接続テスト
- `test_client_init_default`: デフォルト設定での初期化
- `test_client_init_custom`: カスタムURL/timeoutでの初期化
- `test_client_context_manager`: with文での使用

#### add_note テスト
- `test_add_note_minimal`: 最小パラメータ
- `test_add_note_full`: 全パラメータ指定
- `test_add_note_invalid_params`: 必須パラメータ欠落

#### search テスト
- `test_search_basic`: 基本検索
- `test_search_with_filters`: フィルタ付き検索
- `test_search_empty_result`: 結果なし
- `test_search_topk_default`: topKデフォルト値（5）
- `test_search_topk_boundary_zero`: topK=0（エラー期待）
- `test_search_topk_boundary_large`: topK=1000（大きな値）
- `test_search_since_until`: since/until境界条件

#### get テスト
- `test_get_existing`: 存在するID
- `test_get_not_found`: 存在しないID

#### update テスト
- `test_update_title`: タイトルのみ更新
- `test_update_text`: テキスト更新（再埋め込み発生）
- `test_update_not_found`: 存在しないID

#### list_recent テスト
- `test_list_recent_default`: デフォルトパラメータ
- `test_list_recent_with_limit`: limit指定
- `test_list_recent_with_group`: groupId指定
- `test_list_recent_with_tags`: tagsフィルタ
- `test_list_recent_limit_zero`: limit=0（空リスト期待）
- `test_list_recent_limit_negative`: limit=-1（エラー期待）

#### get_config テスト
- `test_get_config`: 設定取得
- `test_get_config_response_format`: レスポンス形式の検証

#### set_config テスト
- `test_set_config_provider`: provider変更
- `test_set_config_partial`: 一部のみ変更
- `test_set_config_invalid_provider`: 不正なprovider値
- `test_set_config_empty`: 空のパラメータ

#### upsert_global テスト
- `test_upsert_global_string`: 文字列値
- `test_upsert_global_object`: オブジェクト値
- `test_upsert_global_with_updated_at`: updated_at指定
- `test_upsert_global_invalid_prefix`: 不正なキープレフィックス

#### get_global テスト
- `test_get_global_existing`: 存在するキー
- `test_get_global_not_found`: 存在しないキー（found=false）
- `test_get_global_not_found_fields`: found=false時のid/value/updated_at確認

#### エラーテスト
- `test_rpc_error_handling`: JSON-RPCエラーの処理
- `test_rpc_error_invalid_params`: -32602エラー
- `test_rpc_error_method_not_found`: -32601エラー
- `test_rpc_error_api_key_missing`: -32002エラー
- `test_connection_error`: 接続エラー
- `test_timeout_error`: タイムアウト

### test_models.py

- `test_note_from_dict`: dict→Noteの変換
- `test_note_alias`: camelCase alias の動作
- `test_search_result`: SearchResult の構築
- `test_list_recent_result`: ListRecentResult の構築
- `test_config_result`: ConfigResult の構築
- `test_global_value_found_true`: found=true時の構築
- `test_global_value_found_false`: found=false時の構築

---

## pyproject.toml

```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "mcp-memory-client"
version = "0.1.0"
description = "Python client for MCP Memory Server"
readme = "README.md"
requires-python = ">=3.10"
license = "MIT"
authors = [
    { name = "Your Name", email = "your@email.com" }
]
classifiers = [
    "Development Status :: 4 - Beta",
    "Intended Audience :: Developers",
    "License :: OSI Approved :: MIT License",
    "Programming Language :: Python :: 3",
    "Programming Language :: Python :: 3.10",
    "Programming Language :: Python :: 3.11",
    "Programming Language :: Python :: 3.12",
]
dependencies = [
    "httpx>=0.27.0",
    "pydantic>=2.0.0",
]

[project.optional-dependencies]
langchain = [
    "langchain-core>=0.3.0",
]
dev = [
    "pytest>=8.0.0",
    "pytest-asyncio>=0.24.0",
    "pytest-httpx>=0.32.0",
    "mypy>=1.11.0",
    "ruff>=0.6.0",
]

[tool.hatch.build.targets.wheel]
packages = ["src/mcp_memory_client"]

[tool.ruff]
target-version = "py310"
line-length = 100

[tool.ruff.lint]
select = ["E", "F", "I", "W"]

[tool.mypy]
python_version = "3.10"
strict = true
```

---

## 実装順序

1. **models.py**: データモデル定義
2. **exceptions.py**: 例外クラス定義
3. **client.py**: MCPMemoryClient実装
4. **langchain_tools.py**: LangGraph Tools実装
5. **pyproject.toml**: パッケージ設定
6. **tests/**: テスト実装
7. **README.md**: 使用例ドキュメント

---

## 設計方針

### パラメータ名変換（snake_case ↔ camelCase）

クライアント側はPython慣習のsnake_case、サーバー側はcamelCaseを使用。

| クライアント (Python) | サーバー (JSON-RPC) |
|----------------------|---------------------|
| `project_id` | `projectId` |
| `group_id` | `groupId` |
| `top_k` | `topK` |
| `created_at` | `createdAt` |
| `updated_at` | `updatedAt` |
| `base_url` | `baseUrl` |
| `api_key` | `apiKey` |

**実装方針**: `_call()` メソッド内で params を camelCase に変換して送信。レスポンスは Pydantic の `alias` で自動変換。

### 同期/非同期 設計

- **v0.1.0**: 同期クライアントのみ（`httpx` の同期API使用）
- **将来**: 非同期クライアント `AsyncMCPMemoryClient` を追加可能（pytest-asyncioは将来の拡張用に依存関係に含む）

---

## セキュリティ考慮事項

1. **API Key の取り扱い**
   - `set_config` で渡す `api_key` はログに出力しない
   - メモリ上に平文で保持されるため、使用後は参照を切る

2. **入力検証**
   - Pydanticによる型検証を徹底
   - サーバー側でも検証されるが、クライアント側でも早期エラーを返す

3. **接続先の検証**
   - `base_url` は localhost のみを想定
   - 外部URLへの接続はユーザーの責任

---

## README.md 構成

1. インストール方法
2. クイックスタート
3. API リファレンス
4. LangGraph 統合
5. エラーハンドリング
6. 開発（テスト実行方法）
