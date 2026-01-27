package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

const (
	testSQLiteNamespace = "test-namespace"
	testSQLiteProjectID = "/test/project"
	testSQLiteGroupID   = "test-group"
)

func setupSQLiteTestStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLiteStore: %v", err)
	}

	return store, dbPath
}

func setupInitializedSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, _ := setupSQLiteTestStore(t)

	ctx := context.Background()
	if err := store.Initialize(ctx, testSQLiteNamespace); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	return store
}

func newSQLiteTestNote(id, projectID, groupID, text string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      []string{},
	}
}

func newSQLiteTestNoteWithTags(id, projectID, groupID, text string, tags []string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      tags,
	}
}

func dummySQLiteEmbedding(dim int) []float32 {
	embedding := make([]float32, dim)
	for i := range embedding {
		embedding[i] = float32(i) / float32(dim)
	}
	return embedding
}

// TestSQLiteStore_NewStore はインスタンス作成をテスト
func TestSQLiteStore_NewStore(t *testing.T) {
	store, dbPath := setupSQLiteTestStore(t)
	defer store.Close()

	// DBファイルが作成されていることを確認
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("DB file should exist at %s", dbPath)
	}
}

// TestSQLiteStore_Initialize はテーブル作成とnamespace設定をテスト
func TestSQLiteStore_Initialize(t *testing.T) {
	store, _ := setupSQLiteTestStore(t)
	defer store.Close()

	ctx := context.Background()
	if err := store.Initialize(ctx, testSQLiteNamespace); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 2回目の初期化も成功すること（冪等性）
	if err := store.Initialize(ctx, testSQLiteNamespace); err != nil {
		t.Fatalf("Second Initialize failed: %v", err)
	}
}

// TestSQLiteStore_NotInitialized はInitialize前の操作がErrNotInitializedを返すことをテスト
func TestSQLiteStore_NotInitialized(t *testing.T) {
	store, _ := setupSQLiteTestStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("test-id", testSQLiteProjectID, testSQLiteGroupID, "test")
	embedding := dummySQLiteEmbedding(1536)

	// AddNote
	if err := store.AddNote(ctx, note, embedding); err != ErrNotInitialized {
		t.Errorf("AddNote should return ErrNotInitialized, got %v", err)
	}

	// Get
	if _, err := store.Get(ctx, "test-id"); err != ErrNotInitialized {
		t.Errorf("Get should return ErrNotInitialized, got %v", err)
	}

	// Update
	if err := store.Update(ctx, note, embedding); err != ErrNotInitialized {
		t.Errorf("Update should return ErrNotInitialized, got %v", err)
	}

	// Delete
	if err := store.Delete(ctx, "test-id"); err != ErrNotInitialized {
		t.Errorf("Delete should return ErrNotInitialized, got %v", err)
	}

	// Search
	opts := SearchOptions{ProjectID: testSQLiteProjectID}
	if _, err := store.Search(ctx, embedding, opts); err != ErrNotInitialized {
		t.Errorf("Search should return ErrNotInitialized, got %v", err)
	}

	// ListRecent
	listOpts := ListOptions{ProjectID: testSQLiteProjectID}
	if _, err := store.ListRecent(ctx, listOpts); err != ErrNotInitialized {
		t.Errorf("ListRecent should return ErrNotInitialized, got %v", err)
	}

	// UpsertGlobal
	config := &model.GlobalConfig{ID: "test-id", ProjectID: testSQLiteProjectID, Key: "global.test"}
	if err := store.UpsertGlobal(ctx, config); err != ErrNotInitialized {
		t.Errorf("UpsertGlobal should return ErrNotInitialized, got %v", err)
	}

	// GetGlobal
	if _, _, err := store.GetGlobal(ctx, testSQLiteProjectID, "global.test"); err != ErrNotInitialized {
		t.Errorf("GetGlobal should return ErrNotInitialized, got %v", err)
	}
}

// TestSQLiteStore_AddNote_Basic は基本的なノート追加をテスト
func TestSQLiteStore_AddNote_Basic(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("note-1", testSQLiteProjectID, testSQLiteGroupID, "Hello World")
	embedding := dummySQLiteEmbedding(1536)

	if err := store.AddNote(ctx, note, embedding); err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	// 取得して確認
	retrieved, err := store.Get(ctx, "note-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Text != "Hello World" {
		t.Errorf("Expected text 'Hello World', got '%s'", retrieved.Text)
	}
}

// TestSQLiteStore_AddNote_WithAllFields は全フィールド指定でのノート追加をテスト
func TestSQLiteStore_AddNote_WithAllFields(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	title := "Test Title"
	source := "test-source"
	createdAt := "2024-01-15T10:30:00Z"
	note := &model.Note{
		ID:        "note-full",
		ProjectID: testSQLiteProjectID,
		GroupID:   testSQLiteGroupID,
		Title:     &title,
		Text:      "Full content",
		Tags:      []string{"tag1", "tag2"},
		Source:    &source,
		CreatedAt: &createdAt,
		Metadata:  map[string]any{"key": "value"},
	}
	embedding := dummySQLiteEmbedding(1536)

	if err := store.AddNote(ctx, note, embedding); err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	retrieved, err := store.Get(ctx, "note-full")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Title == nil || *retrieved.Title != title {
		t.Errorf("Expected title '%s', got %v", title, retrieved.Title)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}
	if retrieved.Source == nil || *retrieved.Source != source {
		t.Errorf("Expected source '%s', got %v", source, retrieved.Source)
	}
	if retrieved.CreatedAt == nil || *retrieved.CreatedAt != createdAt {
		t.Errorf("Expected createdAt '%s', got %v", createdAt, retrieved.CreatedAt)
	}
}

// TestSQLiteStore_AddNote_CreatedAtFormat はcreatedAtがnullの場合の補完をテスト
func TestSQLiteStore_AddNote_CreatedAtFormat(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("note-time", testSQLiteProjectID, testSQLiteGroupID, "Time test")
	embedding := dummySQLiteEmbedding(1536)

	before := time.Now().UTC().Add(-time.Second) // 1秒の余裕
	if err := store.AddNote(ctx, note, embedding); err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}
	after := time.Now().UTC().Add(time.Second) // 1秒の余裕

	retrieved, err := store.Get(ctx, "note-time")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.CreatedAt == nil {
		t.Fatal("CreatedAt should be set")
	}

	createdTime, err := time.Parse(time.RFC3339, *retrieved.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to parse createdAt: %v", err)
	}

	if createdTime.Before(before) || createdTime.After(after) {
		t.Errorf("CreatedAt should be between %v and %v, got %v", before, after, createdTime)
	}
}

// TestSQLiteStore_Get_Found は存在するノート取得をテスト
func TestSQLiteStore_Get_Found(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("get-test", testSQLiteProjectID, testSQLiteGroupID, "Get test")
	embedding := dummySQLiteEmbedding(1536)

	store.AddNote(ctx, note, embedding)

	retrieved, err := store.Get(ctx, "get-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != "get-test" {
		t.Errorf("Expected ID 'get-test', got '%s'", retrieved.ID)
	}
}

// TestSQLiteStore_Get_NotFound は存在しないノート取得をテスト
func TestSQLiteStore_Get_NotFound(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	_, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestSQLiteStore_Update_Basic はノート更新をテスト
func TestSQLiteStore_Update_Basic(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("update-test", testSQLiteProjectID, testSQLiteGroupID, "Original")
	embedding := dummySQLiteEmbedding(1536)

	store.AddNote(ctx, note, embedding)

	// 更新
	note.Text = "Updated"
	if err := store.Update(ctx, note, embedding); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, _ := store.Get(ctx, "update-test")
	if retrieved.Text != "Updated" {
		t.Errorf("Expected text 'Updated', got '%s'", retrieved.Text)
	}
}

// TestSQLiteStore_Update_NotFound は存在しないノート更新をテスト
func TestSQLiteStore_Update_NotFound(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("nonexistent", testSQLiteProjectID, testSQLiteGroupID, "Test")
	embedding := dummySQLiteEmbedding(1536)

	err := store.Update(ctx, note, embedding)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestSQLiteStore_Delete_Basic はノート削除をテスト
func TestSQLiteStore_Delete_Basic(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newSQLiteTestNote("delete-test", testSQLiteProjectID, testSQLiteGroupID, "To delete")
	embedding := dummySQLiteEmbedding(1536)

	store.AddNote(ctx, note, embedding)

	if err := store.Delete(ctx, "delete-test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(ctx, "delete-test")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

// TestSQLiteStore_Delete_NotFound は存在しないノート削除をテスト
func TestSQLiteStore_Delete_NotFound(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Delete(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestSQLiteStore_Search_Basic は基本的なベクトル検索をテスト
func TestSQLiteStore_Search_Basic(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNote("search-1", testSQLiteProjectID, testSQLiteGroupID, "First note")
	note2 := newSQLiteTestNote("search-2", testSQLiteProjectID, testSQLiteGroupID, "Second note")

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// TestSQLiteStore_Search_ProjectIDFilter はprojectIDフィルタをテスト
func TestSQLiteStore_Search_ProjectIDFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNote("proj-1", testSQLiteProjectID, testSQLiteGroupID, "Project 1")
	note2 := newSQLiteTestNote("proj-2", "/other/project", testSQLiteGroupID, "Project 2")

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Note.ID != "proj-1" {
		t.Errorf("Expected note 'proj-1', got '%s'", results[0].Note.ID)
	}
}

// TestSQLiteStore_Search_WithGroupIDFilter はgroupIDフィルタをテスト
func TestSQLiteStore_Search_WithGroupIDFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNote("group-1", testSQLiteProjectID, "group-a", "Group A")
	note2 := newSQLiteTestNote("group-2", testSQLiteProjectID, "group-b", "Group B")

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	groupID := "group-a"
	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		GroupID:   &groupID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Note.GroupID != "group-a" {
		t.Errorf("Expected groupID 'group-a', got '%s'", results[0].Note.GroupID)
	}

	// nil時はフィルタなし
	opts.GroupID = nil
	results, err = store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results with nil groupID, got %d", len(results))
	}
}

// TestSQLiteStore_Search_WithTagsFilter はtagsフィルタをテスト
func TestSQLiteStore_Search_WithTagsFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNoteWithTags("tag-1", testSQLiteProjectID, testSQLiteGroupID, "Note 1", []string{"go", "test"})
	note2 := newSQLiteTestNoteWithTags("tag-2", testSQLiteProjectID, testSQLiteGroupID, "Note 2", []string{"go"})
	note3 := newSQLiteTestNoteWithTags("tag-3", testSQLiteProjectID, testSQLiteGroupID, "Note 3", []string{"python"})

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)
	store.AddNote(ctx, note3, embedding)

	// AND検索
	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		Tags:      []string{"go", "test"},
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Note.ID != "tag-1" {
		t.Errorf("Expected note 'tag-1', got '%s'", results[0].Note.ID)
	}

	// 大小文字区別
	opts.Tags = []string{"Go"}
	results, err = store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results with case-sensitive tag, got %d", len(results))
	}

	// 空配列はフィルタなし
	opts.Tags = []string{}
	results, err = store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results with empty tags, got %d", len(results))
	}
}

// TestSQLiteStore_Search_WithTimeFilter_Boundary はsince/untilフィルタの境界条件をテスト
func TestSQLiteStore_Search_WithTimeFilter_Boundary(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	t1 := "2024-01-10T00:00:00Z"
	t2 := "2024-01-15T00:00:00Z"
	t3 := "2024-01-20T00:00:00Z"

	note1 := &model.Note{ID: "time-1", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "T1", CreatedAt: &t1}
	note2 := &model.Note{ID: "time-2", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "T2", CreatedAt: &t2}
	note3 := &model.Note{ID: "time-3", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "T3", CreatedAt: &t3}

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)
	store.AddNote(ctx, note3, embedding)

	// since <= createdAt < until
	since, _ := time.Parse(time.RFC3339, "2024-01-15T00:00:00Z")
	until, _ := time.Parse(time.RFC3339, "2024-01-20T00:00:00Z")

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		Since:     &since,
		Until:     &until,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// note2 (2024-01-15) は含まれる, note3 (2024-01-20) は含まれない
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Note.ID != "time-2" {
		t.Errorf("Expected note 'time-2', got '%s'", results[0].Note.ID)
	}
}

// TestSQLiteStore_Search_TopK はTopK制限をテスト
func TestSQLiteStore_Search_TopK(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	for i := 0; i < 10; i++ {
		note := newSQLiteTestNote("topk-"+string(rune('0'+i)), testSQLiteProjectID, testSQLiteGroupID, "Note")
		store.AddNote(ctx, note, embedding)
	}

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      3,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

// TestSQLiteStore_Search_ScoreNormalization はスコア正規化をテスト
func TestSQLiteStore_Search_ScoreNormalization(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note := newSQLiteTestNote("score-test", testSQLiteProjectID, testSQLiteGroupID, "Score test")
	store.AddNote(ctx, note, embedding)

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// 同じembeddingで検索するのでスコアは1.0（または非常に近い値）
	if results[0].Score < 0.99 || results[0].Score > 1.0 {
		t.Errorf("Expected score close to 1.0, got %f", results[0].Score)
	}
}

// TestSQLiteStore_Search_SortOrder はスコア降順ソートをテスト
func TestSQLiteStore_Search_SortOrder(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()

	// 異なるembeddingを持つノートを追加
	embedding1 := dummySQLiteEmbedding(1536)
	embedding2 := make([]float32, 1536)
	for i := range embedding2 {
		embedding2[i] = 1.0 - embedding1[i]
	}

	note1 := newSQLiteTestNote("sort-1", testSQLiteProjectID, testSQLiteGroupID, "Similar")
	note2 := newSQLiteTestNote("sort-2", testSQLiteProjectID, testSQLiteGroupID, "Different")

	store.AddNote(ctx, note1, embedding1)
	store.AddNote(ctx, note2, embedding2)

	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding1, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// スコア降順
	if results[0].Score < results[1].Score {
		t.Errorf("Results should be sorted by score descending: %f < %f", results[0].Score, results[1].Score)
	}
}

// TestSQLiteStore_ListRecent_Basic は最新ノート一覧をテスト
func TestSQLiteStore_ListRecent_Basic(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	t1 := "2024-01-10T00:00:00Z"
	t2 := "2024-01-15T00:00:00Z"

	note1 := &model.Note{ID: "recent-1", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "Old", CreatedAt: &t1}
	note2 := &model.Note{ID: "recent-2", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "New", CreatedAt: &t2}

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		Limit:     10,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(notes))
	}
}

// TestSQLiteStore_ListRecent_SortOrder はcreatedAt降順ソートをテスト
func TestSQLiteStore_ListRecent_SortOrder(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	t1 := "2024-01-10T00:00:00Z"
	t2 := "2024-01-15T00:00:00Z"
	t3 := "2024-01-20T00:00:00Z"

	note1 := &model.Note{ID: "sort-1", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "Old", CreatedAt: &t1}
	note2 := &model.Note{ID: "sort-2", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "Mid", CreatedAt: &t2}
	note3 := &model.Note{ID: "sort-3", ProjectID: testSQLiteProjectID, GroupID: testSQLiteGroupID, Text: "New", CreatedAt: &t3}

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)
	store.AddNote(ctx, note3, embedding)

	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		Limit:     10,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("Expected 3 notes, got %d", len(notes))
	}

	// 降順: sort-3, sort-2, sort-1
	if notes[0].ID != "sort-3" || notes[1].ID != "sort-2" || notes[2].ID != "sort-1" {
		t.Errorf("Expected order [sort-3, sort-2, sort-1], got [%s, %s, %s]",
			notes[0].ID, notes[1].ID, notes[2].ID)
	}
}

// TestSQLiteStore_ListRecent_Limit はLimit制限をテスト
func TestSQLiteStore_ListRecent_Limit(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	for i := 0; i < 15; i++ {
		note := newSQLiteTestNote("limit-"+string(rune('0'+i)), testSQLiteProjectID, testSQLiteGroupID, "Note")
		store.AddNote(ctx, note, embedding)
	}

	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		Limit:     5,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 5 {
		t.Errorf("Expected 5 notes, got %d", len(notes))
	}
}

// TestSQLiteStore_ListRecent_WithGroupIDFilter はgroupIDフィルタをテスト
func TestSQLiteStore_ListRecent_WithGroupIDFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNote("lg-1", testSQLiteProjectID, "group-a", "Group A")
	note2 := newSQLiteTestNote("lg-2", testSQLiteProjectID, "group-b", "Group B")

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	groupID := "group-a"
	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		GroupID:   &groupID,
		Limit:     10,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}
}

// TestSQLiteStore_ListRecent_WithTagsFilter はtagsフィルタをテスト
func TestSQLiteStore_ListRecent_WithTagsFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNoteWithTags("lt-1", testSQLiteProjectID, testSQLiteGroupID, "Note 1", []string{"go", "test"})
	note2 := newSQLiteTestNoteWithTags("lt-2", testSQLiteProjectID, testSQLiteGroupID, "Note 2", []string{"go"})

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		Tags:      []string{"go", "test"},
		Limit:     10,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}
}

// TestSQLiteStore_UpsertGlobal_Insert は新規global config作成をテスト
func TestSQLiteStore_UpsertGlobal_Insert(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	updatedAt := "2024-01-15T10:30:00Z"
	config := &model.GlobalConfig{
		ID:        "config-1",
		ProjectID: testSQLiteProjectID,
		Key:       "global.test.key",
		Value:     "test-value",
		UpdatedAt: &updatedAt,
	}

	if err := store.UpsertGlobal(ctx, config); err != nil {
		t.Fatalf("UpsertGlobal failed: %v", err)
	}

	retrieved, found, err := store.GetGlobal(ctx, testSQLiteProjectID, "global.test.key")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}
	if !found {
		t.Fatal("Config should be found")
	}
	if retrieved.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%v'", retrieved.Value)
	}
}

// TestSQLiteStore_UpsertGlobal_Update はglobal config更新をテスト
func TestSQLiteStore_UpsertGlobal_Update(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	updatedAt := "2024-01-15T10:30:00Z"
	config := &model.GlobalConfig{
		ID:        "config-1",
		ProjectID: testSQLiteProjectID,
		Key:       "global.test.key",
		Value:     "original",
		UpdatedAt: &updatedAt,
	}

	store.UpsertGlobal(ctx, config)

	// 更新
	config.Value = "updated"
	updatedAt2 := "2024-01-16T10:30:00Z"
	config.UpdatedAt = &updatedAt2
	if err := store.UpsertGlobal(ctx, config); err != nil {
		t.Fatalf("UpsertGlobal update failed: %v", err)
	}

	retrieved, _, _ := store.GetGlobal(ctx, testSQLiteProjectID, "global.test.key")
	if retrieved.Value != "updated" {
		t.Errorf("Expected value 'updated', got '%v'", retrieved.Value)
	}
}

// TestSQLiteStore_GetGlobal_Found は存在するconfig取得をテスト
func TestSQLiteStore_GetGlobal_Found(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	updatedAt := "2024-01-15T10:30:00Z"
	config := &model.GlobalConfig{
		ID:        "config-1",
		ProjectID: testSQLiteProjectID,
		Key:       "global.test.key",
		Value:     map[string]any{"nested": "value"},
		UpdatedAt: &updatedAt,
	}

	store.UpsertGlobal(ctx, config)

	retrieved, found, err := store.GetGlobal(ctx, testSQLiteProjectID, "global.test.key")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}
	if !found {
		t.Fatal("Config should be found")
	}
	if retrieved.Key != "global.test.key" {
		t.Errorf("Expected key 'global.test.key', got '%s'", retrieved.Key)
	}
}

// TestSQLiteStore_GetGlobal_NotFound は存在しないconfig取得をテスト
func TestSQLiteStore_GetGlobal_NotFound(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	_, found, err := store.GetGlobal(ctx, testSQLiteProjectID, "nonexistent")
	if err != nil {
		t.Fatalf("GetGlobal should not return error: %v", err)
	}
	if found {
		t.Error("Config should not be found")
	}
}

// TestSQLiteStore_Close はDB接続クローズをテスト
func TestSQLiteStore_Close(t *testing.T) {
	store := setupInitializedSQLiteStore(t)

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// クローズ後の操作はエラーになる
	ctx := context.Background()
	_, err := store.Get(ctx, "test")
	if err == nil {
		t.Error("Expected error after close")
	}
}

// TestSQLiteStore_ListRecent_ProjectIDFilter はprojectIDフィルタをテスト
func TestSQLiteStore_ListRecent_ProjectIDFilter(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding := dummySQLiteEmbedding(1536)

	note1 := newSQLiteTestNote("lp-1", testSQLiteProjectID, testSQLiteGroupID, "Project 1")
	note2 := newSQLiteTestNote("lp-2", "/other/project", testSQLiteGroupID, "Project 2")

	store.AddNote(ctx, note1, embedding)
	store.AddNote(ctx, note2, embedding)

	opts := ListOptions{
		ProjectID: testSQLiteProjectID,
		Limit:     10,
	}

	notes, err := store.ListRecent(ctx, opts)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}
	if len(notes) > 0 && notes[0].ID != "lp-1" {
		t.Errorf("Expected note 'lp-1', got '%s'", notes[0].ID)
	}
}

// TestSQLiteStore_Update_WithReembedding はembedding変更を伴う更新をテスト
func TestSQLiteStore_Update_WithReembedding(t *testing.T) {
	store := setupInitializedSQLiteStore(t)
	defer store.Close()

	ctx := context.Background()
	embedding1 := dummySQLiteEmbedding(1536)

	note := newSQLiteTestNote("reembed-test", testSQLiteProjectID, testSQLiteGroupID, "Original text")
	store.AddNote(ctx, note, embedding1)

	// 新しいembeddingで更新
	embedding2 := make([]float32, 1536)
	for i := range embedding2 {
		embedding2[i] = 1.0 - embedding1[i]
	}
	note.Text = "Updated text"
	if err := store.Update(ctx, note, embedding2); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 更新されたテキストを確認
	retrieved, _ := store.Get(ctx, "reembed-test")
	if retrieved.Text != "Updated text" {
		t.Errorf("Expected text 'Updated text', got '%s'", retrieved.Text)
	}

	// 新しいembeddingで検索するとスコアが高くなることを確認
	opts := SearchOptions{
		ProjectID: testSQLiteProjectID,
		TopK:      5,
	}

	results, err := store.Search(ctx, embedding2, opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// embedding2で検索してスコアが高い（1.0に近い）ことを確認
	if results[0].Score < 0.99 {
		t.Errorf("Expected score close to 1.0 after re-embedding, got %f", results[0].Score)
	}
}
