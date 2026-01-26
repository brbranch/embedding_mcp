package store

import (
	"context"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestStoreInterface_NoteOperations はStore interfaceのNote操作をテスト
func TestStoreInterface_NoteOperations(t *testing.T) {
	// このテストはモック実装やインメモリ実装を使用してインターフェースの基本動作を検証する
	// 具体的な実装テストは chroma_test.go で行う
	t.Skip("TODO: implement after mock or in-memory store is available")
}

// TestStoreInterface_SearchOperations はStore interfaceの検索操作をテスト
func TestStoreInterface_SearchOperations(t *testing.T) {
	t.Skip("TODO: implement after mock or in-memory store is available")
}

// TestStoreInterface_GlobalConfigOperations はStore interfaceのGlobalConfig操作をテスト
func TestStoreInterface_GlobalConfigOperations(t *testing.T) {
	t.Skip("TODO: implement after mock or in-memory store is available")
}

// Helper functions for test data

// newTestNote は基本的なテスト用Noteを生成
func newTestNote(id, projectID, groupID, text string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      []string{},
	}
}

// newTestNoteWithTitle はタイトル付きのテスト用Noteを生成
func newTestNoteWithTitle(id, projectID, groupID, title, text string) *model.Note {
	titlePtr := &title
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Title:     titlePtr,
		Text:      text,
		Tags:      []string{},
	}
}

// newTestNoteWithTags はタグ付きのテスト用Noteを生成
func newTestNoteWithTags(id, projectID, groupID, text string, tags []string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      tags,
	}
}

// newTestNoteWithCreatedAt は作成日時付きのテスト用Noteを生成
func newTestNoteWithCreatedAt(id, projectID, groupID, text string, createdAt time.Time) *model.Note {
	createdAtStr := createdAt.Format(time.RFC3339)
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      []string{},
		CreatedAt: &createdAtStr,
	}
}

// newTestNoteAllFields は全フィールドを含むテスト用Noteを生成
func newTestNoteAllFields(id, projectID, groupID string) *model.Note {
	title := "Test Note Title"
	source := "test-source"
	createdAt := "2024-01-15T10:30:00Z"
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Title:     &title,
		Text:      "This is a test note with all fields",
		Tags:      []string{"tag1", "tag2", "tag3"},
		Source:    &source,
		CreatedAt: &createdAt,
		Metadata:  map[string]any{"key1": "value1", "key2": 123, "nested": map[string]any{"inner": "data"}},
	}
}

// newTestGlobalConfig はテスト用GlobalConfigを生成
func newTestGlobalConfig(projectID, key, value string) *model.GlobalConfig {
	return &model.GlobalConfig{
		ProjectID: projectID,
		Key:       key,
		Value:     value,
	}
}

// stringPtr は文字列のポインタを返すヘルパー関数
func stringPtr(s string) *string {
	return &s
}

// timePtr は時刻のポインタを返すヘルパー関数
func timePtr(t time.Time) *time.Time {
	return &t
}

// dummyEmbedding はテスト用の固定embeddingベクトルを生成
func dummyEmbedding(dim int) []float32 {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = float32(i) / float32(dim)
	}
	return vec
}

// assertNoteEquals はNoteが期待値と一致することを検証
func assertNoteEquals(t *testing.T, expected, actual *model.Note) {
	t.Helper()
	if actual.ID != expected.ID {
		t.Errorf("ID mismatch: expected=%q, actual=%q", expected.ID, actual.ID)
	}
	if actual.ProjectID != expected.ProjectID {
		t.Errorf("ProjectID mismatch: expected=%q, actual=%q", expected.ProjectID, actual.ProjectID)
	}
	if actual.GroupID != expected.GroupID {
		t.Errorf("GroupID mismatch: expected=%q, actual=%q", expected.GroupID, actual.GroupID)
	}
	if (expected.Title == nil && actual.Title != nil) || (expected.Title != nil && actual.Title == nil) {
		t.Errorf("Title nil mismatch: expected=%v, actual=%v", expected.Title, actual.Title)
	} else if expected.Title != nil && actual.Title != nil && *expected.Title != *actual.Title {
		t.Errorf("Title mismatch: expected=%q, actual=%q", *expected.Title, *actual.Title)
	}
	if actual.Text != expected.Text {
		t.Errorf("Text mismatch: expected=%q, actual=%q", expected.Text, actual.Text)
	}
	if len(actual.Tags) != len(expected.Tags) {
		t.Errorf("Tags length mismatch: expected=%d, actual=%d", len(expected.Tags), len(actual.Tags))
	} else {
		for i := range expected.Tags {
			if actual.Tags[i] != expected.Tags[i] {
				t.Errorf("Tags[%d] mismatch: expected=%q, actual=%q", i, expected.Tags[i], actual.Tags[i])
			}
		}
	}
}

// assertSearchResultsOrdered は検索結果がスコア降順になっていることを検証
func assertSearchResultsOrdered(t *testing.T, results []SearchResult) {
	t.Helper()
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not ordered by score: results[%d].Score=%f > results[%d].Score=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

// assertNotesOrderedByCreatedAt はNoteリストがCreatedAt降順になっていることを検証
func assertNotesOrderedByCreatedAt(t *testing.T, notes []*model.Note) {
	t.Helper()
	for i := 1; i < len(notes); i++ {
		if notes[i].CreatedAt == nil || notes[i-1].CreatedAt == nil {
			continue
		}
		prevTime, _ := time.Parse(time.RFC3339, *notes[i-1].CreatedAt)
		currTime, _ := time.Parse(time.RFC3339, *notes[i].CreatedAt)
		if currTime.After(prevTime) {
			t.Errorf("Notes not ordered by CreatedAt: notes[%d].CreatedAt=%s > notes[%d].CreatedAt=%s",
				i, *notes[i].CreatedAt, i-1, *notes[i-1].CreatedAt)
		}
	}
}

// containsTag はタグ配列に指定されたタグが含まれているかチェック
func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

// containsAllTags はタグ配列に全ての指定タグが含まれているかチェック（AND検索）
func containsAllTags(tags []string, targets []string) bool {
	for _, target := range targets {
		if !containsTag(tags, target) {
			return false
		}
	}
	return true
}

// assertErrorIs はエラーが期待値と一致することを検証
func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if err != target {
		t.Errorf("Error mismatch: expected=%v, actual=%v", target, err)
	}
}

// assertNoError はエラーがnilであることを検証
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// setupTestContext はテスト用のコンテキストを生成
func setupTestContext() context.Context {
	return context.Background()
}
