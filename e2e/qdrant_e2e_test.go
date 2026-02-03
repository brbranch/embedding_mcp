//go:build qdrant_e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/bootstrap"
	"github.com/brbranch/embedding_mcp/internal/jsonrpc"
	"github.com/brbranch/embedding_mcp/internal/model"
)

// setupQdrantTestHandler はQdrantを使用したテスト用Handlerを構築
func setupQdrantTestHandler(t *testing.T) (*jsonrpc.Handler, func()) {
	t.Helper()

	// 環境変数チェック
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}

	// テスト用設定ファイルパス（一時ファイル）
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	dataDir := filepath.Join(tmpDir, "data")

	// 設定ファイル作成（store: qdrant）
	qdrantURL := "http://localhost:6333"
	cfg := model.Config{
		TransportDefaults: model.TransportDefaults{
			DefaultTransport: "stdio",
		},
		Embedder: model.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
			Dim:      1536,
		},
		Store: model.StoreConfig{
			Type: "qdrant",
			URL:  &qdrantURL,
		},
		Paths: model.PathsConfig{
			ConfigPath: configPath,
			DataDir:    dataDir,
		},
	}

	// configファイルを書き込み
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// bootstrap.Initialize で初期化
	ctx := context.Background()
	services, cleanup, err := bootstrap.Initialize(ctx, configPath)
	if err != nil {
		// Qdrant接続失敗時はスキップ
		if strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "connection") {
			t.Skipf("Qdrant is not running: %v", err)
		}
		t.Fatalf("failed to initialize: %v", err)
	}

	// Handler作成
	handler := jsonrpc.New(
		services.NoteService,
		services.ConfigService,
		services.GlobalService,
		services.GroupService,
	)

	return handler, cleanup
}

// TestQdrantE2E_AddNote_Search はMCP経由でノートを追加し、検索できることを確認
func TestQdrantE2E_AddNote_Search(t *testing.T) {
	handler, cleanup := setupQdrantTestHandler(t)
	defer cleanup()

	// ユニークなproject ID（テスト間の干渉を避ける）
	projectID := fmt.Sprintf("/test/qdrant/e2e/%d", time.Now().UnixNano())
	groupID := "test-group"

	// 1. memory.add_noteでノートを追加（OpenAI Embeddingで実際にベクトル生成）
	noteText := "This is a test note for Qdrant E2E testing"
	addResult := callAddNote(t, handler, projectID, groupID, noteText)

	if addResult.ID == "" {
		t.Fatal("add_note returned empty ID")
	}

	// 2. memory.searchで検索し、追加したノートが返ることを確認
	searchResult := callSearch(t, handler, projectID, &groupID, "test note Qdrant")

	if len(searchResult.Results) == 0 {
		t.Fatal("search returned no results")
	}

	// 追加したノートが検索結果に含まれることを確認
	found := false
	for _, result := range searchResult.Results {
		if result.ID == addResult.ID {
			found = true
			if result.Text != noteText {
				t.Errorf("Expected text %q, got %q", noteText, result.Text)
			}
			if result.ProjectID != projectID {
				t.Errorf("Expected projectID %q, got %q", projectID, result.ProjectID)
			}
			if result.GroupID != groupID {
				t.Errorf("Expected groupID %q, got %q", groupID, result.GroupID)
			}
			break
		}
	}

	if !found {
		t.Errorf("Added note (ID: %s) not found in search results", addResult.ID)
	}
}

// TestQdrantE2E_MultipleNotes_Search は複数ノートを追加して検索できることを確認
func TestQdrantE2E_MultipleNotes_Search(t *testing.T) {
	handler, cleanup := setupQdrantTestHandler(t)
	defer cleanup()

	// ユニークなproject ID
	projectID := fmt.Sprintf("/test/qdrant/multi/%d", time.Now().UnixNano())
	groupID := "test-group"

	// 1. 異なるテキストで3つのノートを追加
	notes := []struct {
		text string
		id   string
	}{
		{text: "Golang is a programming language developed by Google"},
		{text: "Python is widely used for data science and machine learning"},
		{text: "JavaScript runs in web browsers and Node.js runtime"},
	}

	for i := range notes {
		result := callAddNote(t, handler, projectID, groupID, notes[i].text)
		notes[i].id = result.ID
		// APIレート制限を避けるため少し待機
		time.Sleep(100 * time.Millisecond)
	}

	// 2. 特定のクエリで検索し、関連するノートが上位に返ることを確認
	searchResult := callSearch(t, handler, projectID, &groupID, "programming languages")

	if len(searchResult.Results) == 0 {
		t.Fatal("search returned no results")
	}

	// 3つすべてのノートが返ることを確認
	if len(searchResult.Results) < 3 {
		t.Errorf("Expected at least 3 results, got %d", len(searchResult.Results))
	}

	// 最初の結果がGolangまたはPythonまたはJavaScriptに関連することを確認
	topResult := searchResult.Results[0]
	if !strings.Contains(topResult.Text, "Golang") &&
		!strings.Contains(topResult.Text, "Python") &&
		!strings.Contains(topResult.Text, "JavaScript") {
		t.Errorf("Top result seems unrelated: %s", topResult.Text)
	}

	// スコアが降順にソートされていることを確認
	for i := 1; i < len(searchResult.Results); i++ {
		if searchResult.Results[i].Score > searchResult.Results[i-1].Score {
			t.Errorf("Results not sorted by score: results[%d].Score (%f) > results[%d].Score (%f)",
				i, searchResult.Results[i].Score, i-1, searchResult.Results[i-1].Score)
		}
	}
}

// TestQdrantE2E_GlobalConfig はGlobalConfig操作を確認
func TestQdrantE2E_GlobalConfig(t *testing.T) {
	handler, cleanup := setupQdrantTestHandler(t)
	defer cleanup()

	// ユニークなproject ID
	projectID := fmt.Sprintf("/test/qdrant/global/%d", time.Now().UnixNano())
	key := "global.test_setting"
	value := map[string]interface{}{
		"feature_enabled": true,
		"max_results":     10,
		"name":            "Qdrant E2E Test",
	}

	// 1. memory.upsert_globalで設定保存
	upsertResult := callUpsertGlobal(t, handler, projectID, key, value)

	if !upsertResult.OK {
		t.Fatal("upsert_global returned OK=false")
	}
	if upsertResult.ID == "" {
		t.Fatal("upsert_global returned empty ID")
	}

	// 2. memory.get_globalで取得確認
	getResult := callGetGlobal(t, handler, projectID, key)

	if !getResult.Found {
		t.Fatal("get_global returned Found=false")
	}

	// 値の検証
	resultValue, ok := getResult.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected value to be map[string]interface{}, got %T", getResult.Value)
	}

	if resultValue["feature_enabled"] != true {
		t.Errorf("Expected feature_enabled=true, got %v", resultValue["feature_enabled"])
	}
	if resultValue["max_results"] != float64(10) { // JSON unmarshals numbers as float64
		t.Errorf("Expected max_results=10, got %v", resultValue["max_results"])
	}
	if resultValue["name"] != "Qdrant E2E Test" {
		t.Errorf("Expected name='Qdrant E2E Test', got %v", resultValue["name"])
	}
}

// TestQdrantE2E_Group はGroup操作を確認
func TestQdrantE2E_Group(t *testing.T) {
	handler, cleanup := setupQdrantTestHandler(t)
	defer cleanup()

	// ユニークなproject ID
	projectID := fmt.Sprintf("/test/qdrant/group/%d", time.Now().UnixNano())
	groupKey := "e2e-test-group"
	title := "E2E Test Group"
	description := "Group created for Qdrant E2E testing"

	// 1. memory.group_createでグループ作成
	createResult := callGroupCreate(t, handler, projectID, groupKey, title, description)

	if createResult.ID == "" {
		t.Fatal("group_create returned empty ID")
	}

	groupID := createResult.ID

	// 2. memory.group_getで取得確認
	getResult := callGroupGet(t, handler, groupID)

	if getResult.ID != groupID {
		t.Errorf("Expected ID %q, got %q", groupID, getResult.ID)
	}
	if getResult.ProjectID != projectID {
		t.Errorf("Expected ProjectID %q, got %q", projectID, getResult.ProjectID)
	}
	if getResult.GroupKey != groupKey {
		t.Errorf("Expected GroupKey %q, got %q", groupKey, getResult.GroupKey)
	}
	if getResult.Title != title {
		t.Errorf("Expected Title %q, got %q", title, getResult.Title)
	}
	if getResult.Description != description {
		t.Errorf("Expected Description %q, got %q", description, getResult.Description)
	}

	// 3. memory.group_listで一覧取得確認
	listResult := callGroupList(t, handler, projectID)

	if len(listResult.Groups) == 0 {
		t.Fatal("group_list returned no groups")
	}

	// 作成したグループが一覧に含まれることを確認
	found := false
	for _, group := range listResult.Groups {
		if group.ID == groupID {
			found = true
			if group.GroupKey != groupKey {
				t.Errorf("Expected GroupKey %q, got %q", groupKey, group.GroupKey)
			}
			if group.Title != title {
				t.Errorf("Expected Title %q, got %q", title, group.Title)
			}
			break
		}
	}

	if !found {
		t.Errorf("Created group (ID: %s) not found in list", groupID)
	}
}
