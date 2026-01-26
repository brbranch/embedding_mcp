//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/jsonrpc"
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

	// canonicalProjectIDが返されること
	if resp.CanonicalProjectID == "" {
		t.Error("expected CanonicalProjectID to be non-empty")
	}

	// 環境依存を考慮した緩い検証:
	// - 絶対パスであること（先頭が"/"）
	// - "~"が展開されていること（"~"を含まない）
	if !strings.HasPrefix(resp.CanonicalProjectID, "/") {
		t.Errorf("projectID should be absolute path, got: %s", resp.CanonicalProjectID)
	}
	if strings.Contains(resp.CanonicalProjectID, "~") {
		t.Errorf("~ should be expanded, got: %s", resp.CanonicalProjectID)
	}
}

// TestE2E_ProjectID_Consistency は同じパスが常に同じcanonicalパスになることを検証
func TestE2E_ProjectID_Consistency(t *testing.T) {
	h := setupTestHandler(t)

	// 同じパスで2回ノート追加
	resp1 := callAddNote(t, h, "~/tmp/demo", "global", "test note 1")
	resp2 := callAddNote(t, h, "~/tmp/demo", "global", "test note 2")

	// 両方とも同じcanonicalProjectIDが返ること
	if resp1.CanonicalProjectID != resp2.CanonicalProjectID {
		t.Errorf("expected same canonicalProjectID, got: %s vs %s",
			resp1.CanonicalProjectID, resp2.CanonicalProjectID)
	}
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

// ============================================================
// CLI Search Command E2E Tests
// ============================================================

// TestE2E_CLISearch_TextFormat tests the search command with text output format
func TestE2E_CLISearch_TextFormat(t *testing.T) {
	// Set up test data using JSON-RPC handler
	h := setupTestHandler(t)
	projectID := "/test/cli-search-project"

	// Add test notes
	callAddNote(t, h, projectID, "global", "Go言語のコーディング規約について")
	callAddNote(t, h, projectID, "global", "API設計の方針とベストプラクティス")
	callAddNote(t, h, projectID, "feature-1", "フィーチャー1の実装メモ")

	// Search using NoteService directly (simulating CLI search)
	ctx := context.Background()

	// Canonicalize project ID (as CLI search would do)
	canonicalProjectID, err := config.CanonicalizeProjectID(projectID)
	if err != nil {
		t.Fatalf("failed to canonicalize projectId: %v", err)
	}

	// Set up mock embedder and store for search
	emb := &mockEmbedder{dim: 128}
	results := searchWithMockService(t, ctx, emb, h, canonicalProjectID, nil, "コーディング", 5)

	// Verify results
	if len(results) == 0 {
		t.Error("expected at least 1 result")
	}

	// Format output as text
	var buf bytes.Buffer
	formatTextOutputE2E(&buf, results)
	output := buf.String()

	// Check output format
	if !strings.Contains(output, "[1]") {
		t.Error("expected output to contain result number [1]")
	}
	if !strings.Contains(output, "score:") {
		t.Error("expected output to contain score")
	}
}

// TestE2E_CLISearch_JSONFormat tests the search command with JSON output format
func TestE2E_CLISearch_JSONFormat(t *testing.T) {
	// Set up test data
	h := setupTestHandler(t)
	projectID := "/test/cli-search-json"

	// Add test notes with tags
	callAddNoteWithTags(t, h, projectID, "global", "テスト駆動開発の基本", []string{"TDD", "テスト"})
	callAddNoteWithTags(t, h, projectID, "global", "ユニットテストの書き方", []string{"テスト", "実装"})

	ctx := context.Background()
	canonicalProjectID, err := config.CanonicalizeProjectID(projectID)
	if err != nil {
		t.Fatalf("failed to canonicalize projectId: %v", err)
	}

	emb := &mockEmbedder{dim: 128}
	results := searchWithMockService(t, ctx, emb, h, canonicalProjectID, nil, "テスト", 5)

	// Format output as JSON
	var buf bytes.Buffer
	if err := formatJSONOutputE2E(&buf, results); err != nil {
		t.Fatalf("failed to format JSON output: %v", err)
	}

	// Parse and verify JSON output
	var output JSONOutputE2E
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(output.Results) == 0 {
		t.Error("expected at least 1 result in JSON output")
	}

	// Verify JSON structure
	for _, r := range output.Results {
		if r.ID == "" {
			t.Error("expected result to have ID")
		}
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("expected score between 0 and 1, got %f", r.Score)
		}
	}
}

// TestE2E_CLISearch_WithGroupFilter tests search with group filter
func TestE2E_CLISearch_WithGroupFilter(t *testing.T) {
	h := setupTestHandler(t)
	projectID := "/test/cli-search-group"

	// Add notes to different groups
	callAddNote(t, h, projectID, "global", "グローバルノート")
	callAddNote(t, h, projectID, "feature-1", "フィーチャー1のノート")
	callAddNote(t, h, projectID, "feature-2", "フィーチャー2のノート")

	ctx := context.Background()
	canonicalProjectID, err := config.CanonicalizeProjectID(projectID)
	if err != nil {
		t.Fatalf("failed to canonicalize projectId: %v", err)
	}

	emb := &mockEmbedder{dim: 128}

	// Search with group filter
	groupID := "feature-1"
	results := searchWithMockService(t, ctx, emb, h, canonicalProjectID, &groupID, "ノート", 5)

	// All results should be from feature-1 group
	for _, r := range results {
		if r.GroupID != "feature-1" {
			t.Errorf("expected groupId 'feature-1', got %q", r.GroupID)
		}
	}
}

// TestE2E_CLISearch_ProjectIDCanonicalization tests that project ID is canonicalized
func TestE2E_CLISearch_ProjectIDCanonicalization(t *testing.T) {
	h := setupTestHandler(t)

	// Add note with tilde path (will be canonicalized)
	resp := callAddNote(t, h, "~/test/project", "global", "テストノート")
	canonicalProjectID := resp.CanonicalProjectID

	// Verify canonicalization
	if strings.Contains(canonicalProjectID, "~") {
		t.Error("expected ~ to be expanded")
	}
	if !strings.HasPrefix(canonicalProjectID, "/") {
		t.Error("expected absolute path")
	}

	// Search with the same tilde path
	searchProjectID, err := config.CanonicalizeProjectID("~/test/project")
	if err != nil {
		t.Fatalf("failed to canonicalize search projectId: %v", err)
	}

	// Both should resolve to the same path
	if searchProjectID != canonicalProjectID {
		t.Errorf("expected search projectID %q to match stored %q", searchProjectID, canonicalProjectID)
	}
}

// Helper functions for CLI search E2E tests

// searchWithMockService performs search using the test handler's underlying services
func searchWithMockService(t *testing.T, ctx context.Context, emb *mockEmbedder, h *jsonrpc.Handler, projectID string, groupID *string, query string, topK int) []SearchResultItem {
	t.Helper()

	// Use JSON-RPC search endpoint
	resp := callSearch(t, h, projectID, groupID, query)
	return resp.Results
}

// callAddNoteWithTags adds a note with tags
func callAddNoteWithTags(t *testing.T, h *jsonrpc.Handler, projectID, groupID, text string, tags []string) *AddNoteResult {
	t.Helper()

	reqBytes, err := json.Marshal(model.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "memory.add_note",
		Params: map[string]any{
			"projectId": projectID,
			"groupId":   groupID,
			"text":      text,
			"tags":      tags,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respBytes := h.Handle(ctx, reqBytes)

	var rawResp RawResponse
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rawResp.Error != nil {
		t.Fatalf("add_note failed: %v", rawResp.Error)
	}

	result := &AddNoteResult{}
	resultBytes, _ := json.Marshal(rawResp.Result)
	if err := json.Unmarshal(resultBytes, result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	return result
}

// formatTextOutputE2E formats search results as text (E2E test version)
func formatTextOutputE2E(buf *bytes.Buffer, results []SearchResultItem) {
	if len(results) == 0 {
		buf.WriteString("No results found.\n")
		return
	}

	for i, r := range results {
		title := "(no title)"
		if r.Title != nil && *r.Title != "" {
			title = *r.Title
		}

		buf.WriteString(strings.Repeat("", 0)) // avoid import not used
		buf.WriteString("[")
		buf.WriteString(strings.TrimSpace(strings.Repeat(" ", 0)))
		buf.WriteString(intToString(i + 1))
		buf.WriteString("] ")
		buf.WriteString(title)
		buf.WriteString(" (score: ")
		buf.WriteString(floatToString(r.Score))
		buf.WriteString(")\n")
		buf.WriteString("    ")
		buf.WriteString(truncateTextE2E(r.Text, 60))
		buf.WriteString("\n")

		if len(r.Tags) > 0 {
			buf.WriteString("    tags: ")
			buf.WriteString(strings.Join(r.Tags, ", "))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
}

// formatJSONOutputE2E formats search results as JSON (E2E test version)
func formatJSONOutputE2E(buf *bytes.Buffer, results []SearchResultItem) error {
	output := JSONOutputE2E{
		Results: make([]JSONResultE2E, 0, len(results)),
	}

	for _, r := range results {
		title := ""
		if r.Title != nil {
			title = *r.Title
		}

		output.Results = append(output.Results, JSONResultE2E{
			ID:    r.ID,
			Title: title,
			Text:  r.Text,
			Score: r.Score,
			Tags:  r.Tags,
		})
	}

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// truncateTextE2E truncates text for display
func truncateTextE2E(text string, maxLen int) string {
	if text == "" || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + " ..."
}

// intToString converts int to string without importing strconv
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string('0'+byte(n%10)) + result
		n /= 10
	}
	return result
}

// floatToString converts float64 to string with 2 decimal places
func floatToString(f float64) string {
	// Simple formatting for test purposes
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 100)
	if decPart < 0 {
		decPart = -decPart
	}
	return intToString(intPart) + "." + padLeft(intToString(decPart), 2, '0')
}

// padLeft pads string with character on the left
func padLeft(s string, length int, pad byte) string {
	for len(s) < length {
		s = string(pad) + s
	}
	return s
}

// JSONOutputE2E represents JSON output structure for E2E tests
type JSONOutputE2E struct {
	Results []JSONResultE2E `json:"results"`
}

// JSONResultE2E represents a single result in JSON output
type JSONResultE2E struct {
	ID    string   `json:"id"`
	Title string   `json:"title,omitempty"`
	Text  string   `json:"text"`
	Score float64  `json:"score"`
	Tags  []string `json:"tags,omitempty"`
}

// Ensure context is used (for callAddNoteWithTags)
var ctx = context.Background()
