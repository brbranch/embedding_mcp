# E2Eテスト

このディレクトリには、MCPメモリサーバーの主要機能を検証するE2Eテスト（スモークテスト）が含まれています。

## 概要

E2Eテストは以下の機能を検証します:

- **projectID正規化**: `~/tmp/demo` などのチルダ展開
- **ノート追加**: global/feature グループへのノート追加
- **検索**: groupIDフィルタ付き/なし検索、projectID必須検証
- **グローバル設定**: embedder設定、groupDefaults、projectConventions の upsert/get

## テスト実行方法

### 前提条件

- Go 1.22+
- 外部依存なし（MemoryStoreを使用）

### 実行コマンド

```bash
# E2Eテストのみ実行
go test ./e2e/... -tags=e2e -v

# すべてのテスト（E2E含む）を実行
go test ./... -tags=e2e -v
```

### 通常のテストとの違い

E2Eテストはビルドタグ `//go:build e2e` で分離されています。

- 通常のテスト: `go test ./...` （E2Eテストは実行されない）
- E2Eテスト含む: `go test ./... -tags=e2e` （すべてのテストが実行される）

## テストの内部構造

### テスト戦略

- **アプローチ**: インプロセス統合テスト
- **Store**: MemoryStore（外部依存なし）
- **Embedder**: MockEmbedder（決定論的なベクトル生成）

### モック実装

#### mockEmbedder

テキストのSHA256ハッシュから決定論的なベクトルを生成します。同じテキストには常に同じベクトルが返されるため、テストの再現性が保証されます。

#### mockConfigService

テスト用の固定設定を返すConfigServiceモックです。

### ヘルパー関数

- `setupTestHandler`: テスト用のHandlerを構築
- `callAddNote`: memory.add_note を呼び出す
- `callSearch`: memory.search を呼び出す
- `callUpsertGlobal`: memory.upsert_global を呼び出す
- `callGetGlobal`: memory.get_global を呼び出す

## テストケース一覧

| テストケース | 検証内容 |
|-------------|---------|
| `TestE2E_ProjectID_TildeExpansion` | projectIDの~展開と正規化 |
| `TestE2E_ProjectID_Consistency` | 同一パスの正規化一貫性 |
| `TestE2E_AddNote_GlobalGroup` | global groupへのノート追加 |
| `TestE2E_AddNote_FeatureGroup` | feature groupへのノート追加 |
| `TestE2E_Search_WithGroupID` | groupIDフィルタ付き検索 |
| `TestE2E_Search_WithoutGroupID` | groupIDフィルタなし検索 |
| `TestE2E_Search_ProjectIDRequired` | projectID必須検証 |
| `TestE2E_Global_EmbedderProvider` | embedder.provider 設定 |
| `TestE2E_Global_EmbedderModel` | embedder.model 設定 |
| `TestE2E_Global_GroupDefaults` | groupDefaults 設定 |
| `TestE2E_Global_ProjectConventions` | project.conventions 設定 |
| `TestE2E_Global_InvalidKeyPrefix` | global.プレフィックスなしでエラー |

## 制限事項

### 現在未検証の項目

- **transport層**: stdio/HTTP transportは別途テスト済み（ここではテストしない）
- **実際のEmbedder API**: MockEmbedderを使用するため、OpenAI API等の実APIは呼ばれない
- **Search/ListRecentのprojectID正規化**: 現在はadd_noteのみで正規化が実行される

### 将来の拡張

以下のE2Eテストを追加可能:

- HTTP transport経由のE2Eテスト（Chromaサーバー必須）
- stdio transport経由のE2Eテスト
- 大量データ投入時のパフォーマンステスト
- provider切替時のnamespace変更確認
