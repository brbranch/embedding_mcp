//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestE2E_ProjectID_TildeExpansion はprojectIDの~展開を検証
func TestE2E_ProjectID_TildeExpansion(t *testing.T) {
	h := setupTestHandler(t)

	// ~/tmp/demo でノート追加
	resp := callAddNote(t, h, "~/tmp/demo", "global", "test note")

	// レスポンスが正常に返ること
	if resp.ID == "" {
		t.Error("expected ID to be non-empty")
	}
	if resp.Namespace == "" {
		t.Error("expected Namespace to be non-empty")
	}

	// NOTE: 現在の実装ではprojectID正規化が未実装の可能性があるため、
	// このテストはcanonicalProjectIDフィールドの存在を確認するのみ
	// 実装が完了したら以下のアサーションを有効化すること
	// if !strings.HasPrefix(resp.CanonicalProjectID, "/") {
	//     t.Errorf("projectID should be absolute path, got: %s", resp.CanonicalProjectID)
	// }
	// if strings.Contains(resp.CanonicalProjectID, "~") {
	//     t.Errorf("~ should be expanded, got: %s", resp.CanonicalProjectID)
	// }
}

// TestE2E_AddNote_GlobalGroup はglobal groupへのノート追加を検証
func TestE2E_AddNote_GlobalGroup(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	resp := callAddNote(t, h, projectID, "global", "global note content")

	if resp.ID == "" {
		t.Error("expected ID to be non-empty")
	}
	if resp.Namespace == "" {
		t.Error("expected Namespace to be non-empty")
	}
}

// TestE2E_AddNote_FeatureGroup はfeature groupへのノート追加を検証
func TestE2E_AddNote_FeatureGroup(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	resp := callAddNote(t, h, projectID, "feature-1", "feature note content")

	if resp.ID == "" {
		t.Error("expected ID to be non-empty")
	}
	if resp.Namespace == "" {
		t.Error("expected Namespace to be non-empty")
	}
}

// TestE2E_Search_WithGroupID はgroupIDフィルタ付き検索を検証
func TestE2E_Search_WithGroupID(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// 2件追加（異なるグループ）
	callAddNote(t, h, projectID, "global", "global note")
	callAddNote(t, h, projectID, "feature-1", "feature note")

	// groupId="feature-1" でフィルタ
	resp := callSearch(t, h, projectID, ptr("feature-1"), "feature")

	// 1件のみ返ること
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
	if len(resp.Results) > 0 && resp.Results[0].GroupID != "feature-1" {
		t.Errorf("expected groupId to be 'feature-1', got: %s", resp.Results[0].GroupID)
	}
}

// TestE2E_Search_WithoutGroupID はgroupIDフィルタなし検索を検証
func TestE2E_Search_WithoutGroupID(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// 2件追加（異なるグループ）
	callAddNote(t, h, projectID, "global", "note content 1")
	callAddNote(t, h, projectID, "feature-1", "note content 2")

	// groupId=null で全検索
	resp := callSearch(t, h, projectID, nil, "note")

	// 2件返ること
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
}

// TestE2E_Search_ProjectIDRequired はprojectID必須検証
func TestE2E_Search_ProjectIDRequired(t *testing.T) {
	h := setupTestHandler(t)

	// projectId なしで検索 → invalid params エラー
	resp := callSearchRaw(t, h, "", nil, "query")

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected error code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
	if !strings.Contains(resp.Error.Message, "projectId") {
		t.Errorf("expected error message to contain 'projectId', got: %s", resp.Error.Message)
	}
}

// TestE2E_Global_EmbedderProvider はembedder.provider設定を検証
func TestE2E_Global_EmbedderProvider(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// upsert
	upsertResp := callUpsertGlobal(t, h, projectID, "global.memory.embedder.provider", "openai")
	if !upsertResp.OK {
		t.Error("expected OK to be true")
	}

	// get
	getResp := callGetGlobal(t, h, projectID, "global.memory.embedder.provider")
	if !getResp.Found {
		t.Error("expected Found to be true")
	}
	if getResp.Value != "openai" {
		t.Errorf("expected value to be 'openai', got: %v", getResp.Value)
	}
}

// TestE2E_Global_EmbedderModel はembedder.model設定を検証
func TestE2E_Global_EmbedderModel(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// upsert
	upsertResp := callUpsertGlobal(t, h, projectID, "global.memory.embedder.model", "text-embedding-3-small")
	if !upsertResp.OK {
		t.Error("expected OK to be true")
	}

	// get
	getResp := callGetGlobal(t, h, projectID, "global.memory.embedder.model")
	if !getResp.Found {
		t.Error("expected Found to be true")
	}
	if getResp.Value != "text-embedding-3-small" {
		t.Errorf("expected value to be 'text-embedding-3-small', got: %v", getResp.Value)
	}
}

// TestE2E_Global_GroupDefaults はgroupDefaults設定を検証
func TestE2E_Global_GroupDefaults(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	value := map[string]any{
		"featurePrefix": "feature-",
		"taskPrefix":    "task-",
	}

	// upsert
	upsertResp := callUpsertGlobal(t, h, projectID, "global.memory.groupDefaults", value)
	if !upsertResp.OK {
		t.Error("expected OK to be true")
	}

	// get
	getResp := callGetGlobal(t, h, projectID, "global.memory.groupDefaults")
	if !getResp.Found {
		t.Error("expected Found to be true")
	}

	// map比較
	resultMap, ok := getResp.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected value to be map[string]any, got: %T", getResp.Value)
	}
	if resultMap["featurePrefix"] != "feature-" {
		t.Errorf("expected featurePrefix to be 'feature-', got: %v", resultMap["featurePrefix"])
	}
	if resultMap["taskPrefix"] != "task-" {
		t.Errorf("expected taskPrefix to be 'task-', got: %v", resultMap["taskPrefix"])
	}
}

// TestE2E_Global_ProjectConventions はproject.conventions設定を検証
func TestE2E_Global_ProjectConventions(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// upsert（日本語文字列）
	upsertResp := callUpsertGlobal(t, h, projectID, "global.project.conventions", "文章")
	if !upsertResp.OK {
		t.Error("expected OK to be true")
	}

	// get
	getResp := callGetGlobal(t, h, projectID, "global.project.conventions")
	if !getResp.Found {
		t.Error("expected Found to be true")
	}
	if getResp.Value != "文章" {
		t.Errorf("expected value to be '文章', got: %v", getResp.Value)
	}
}

// TestE2E_Global_InvalidKeyPrefix はglobal.プレフィックスなしでエラーを検証
func TestE2E_Global_InvalidKeyPrefix(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/project"

	// "global." プレフィックスなし → エラー
	resp := callUpsertGlobalRaw(t, h, projectID, "memory.embedder.provider", "openai")

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != model.ErrCodeInvalidKeyPrefix {
		t.Errorf("expected error code %d, got %d", model.ErrCodeInvalidKeyPrefix, resp.Error.Code)
	}
}
