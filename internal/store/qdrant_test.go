package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

const (
	testQdrantNamespace = "test-qdrant-namespace"
	testQdrantProjectID = "/test/project"
	testQdrantGroupID   = "test-group"
)

// getQdrantURL は環境変数からQdrant URLを取得、未設定時はデフォルトを返す
func getQdrantURL() string {
	if url := os.Getenv("QDRANT_URL"); url != "" {
		return url
	}
	return "http://localhost:6333"
}

func setupQdrantTestStore(t *testing.T) *QdrantStore {
	t.Helper()

	store, err := NewQdrantStore(getQdrantURL())
	if err != nil {
		if err == ErrConnectionFailed {
			t.Skip("Qdrant is not available, skipping test")
		}
		t.Fatalf("Failed to create QdrantStore: %v", err)
	}

	return store
}

func setupInitializedQdrantStore(t *testing.T) *QdrantStore {
	t.Helper()
	store := setupQdrantTestStore(t)

	ctx := context.Background()
	if err := store.Initialize(ctx, testQdrantNamespace); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	return store
}

func newQdrantTestNote(id, projectID, groupID, text string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      []string{},
	}
}

func newQdrantTestNoteWithTags(id, projectID, groupID, text string, tags []string) *model.Note {
	return &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
		Tags:      tags,
	}
}

func dummyQdrantEmbedding(dim int) []float32 {
	embedding := make([]float32, dim)
	for i := range embedding {
		embedding[i] = float32(i) / float32(dim)
	}
	return embedding
}

// TestQdrantStore_NewStore はインスタンス作成をテスト
func TestQdrantStore_NewStore(t *testing.T) {
	store := setupQdrantTestStore(t)
	defer store.Close()

	if store == nil {
		t.Error("QdrantStore instance should not be nil")
	}
}

// TestQdrantStore_Initialize はコレクション作成とnamespace設定をテスト
func TestQdrantStore_Initialize(t *testing.T) {
	store := setupQdrantTestStore(t)
	defer store.Close()

	ctx := context.Background()
	if err := store.Initialize(ctx, testQdrantNamespace); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 2回目の初期化も成功すること（冪等性）
	if err := store.Initialize(ctx, testQdrantNamespace); err != nil {
		t.Fatalf("Second Initialize failed: %v", err)
	}
}

// TestQdrantStore_NotInitialized はInitialize前の操作がErrNotInitializedを返すことをテスト
func TestQdrantStore_NotInitialized(t *testing.T) {
	store := setupQdrantTestStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("test-id", testQdrantProjectID, testQdrantGroupID, "test")
	embedding := dummyQdrantEmbedding(1536)

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
}

// TestQdrantStore_AddNote_Basic は基本的なノート追加をテスト
func TestQdrantStore_AddNote_Basic(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("note-1", testQdrantProjectID, testQdrantGroupID, "Hello World")
	embedding := dummyQdrantEmbedding(1536)

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

// TestQdrantStore_AddNote_WithAllFields は全フィールド指定でのノート追加をテスト
func TestQdrantStore_AddNote_WithAllFields(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	title := "Test Title"
	source := "test-source"
	createdAt := "2024-01-15T10:30:00Z"
	note := &model.Note{
		ID:        "note-full",
		ProjectID: testQdrantProjectID,
		GroupID:   testQdrantGroupID,
		Title:     &title,
		Text:      "Full content",
		Tags:      []string{"tag1", "tag2"},
		Source:    &source,
		CreatedAt: &createdAt,
		Metadata:  map[string]any{"key": "value"},
	}
	embedding := dummyQdrantEmbedding(1536)

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

// TestQdrantStore_AddNote_CreatedAtFormat はcreatedAtがnullの場合の補完をテスト
func TestQdrantStore_AddNote_CreatedAtFormat(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("note-time", testQdrantProjectID, testQdrantGroupID, "Time test")
	embedding := dummyQdrantEmbedding(1536)

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

// TestQdrantStore_Get_Found は存在するノート取得をテスト
func TestQdrantStore_Get_Found(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("get-test", testQdrantProjectID, testQdrantGroupID, "Get test")
	embedding := dummyQdrantEmbedding(1536)

	store.AddNote(ctx, note, embedding)

	retrieved, err := store.Get(ctx, "get-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != "get-test" {
		t.Errorf("Expected ID 'get-test', got '%s'", retrieved.ID)
	}
}

// TestQdrantStore_Get_NotFound は存在しないノート取得をテスト
func TestQdrantStore_Get_NotFound(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	_, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestQdrantStore_Update_Basic はノート更新をテスト
func TestQdrantStore_Update_Basic(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("update-test", testQdrantProjectID, testQdrantGroupID, "Original")
	embedding := dummyQdrantEmbedding(1536)

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

// TestQdrantStore_Update_NotFound は存在しないノート更新をテスト
func TestQdrantStore_Update_NotFound(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("nonexistent", testQdrantProjectID, testQdrantGroupID, "Test")
	embedding := dummyQdrantEmbedding(1536)

	err := store.Update(ctx, note, embedding)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestQdrantStore_Delete_Basic はノート削除をテスト
func TestQdrantStore_Delete_Basic(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	note := newQdrantTestNote("delete-test", testQdrantProjectID, testQdrantGroupID, "To delete")
	embedding := dummyQdrantEmbedding(1536)

	store.AddNote(ctx, note, embedding)

	if err := store.Delete(ctx, "delete-test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(ctx, "delete-test")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

// TestQdrantStore_Delete_NotFound は存在しないノート削除をテスト
func TestQdrantStore_Delete_NotFound(t *testing.T) {
	store := setupInitializedQdrantStore(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Delete(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestQdrantStore_Close はクローズをテスト
func TestQdrantStore_Close(t *testing.T) {
	store := setupInitializedQdrantStore(t)

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
