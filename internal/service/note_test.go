package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/embedder"
	"github.com/brbranch/embedding_mcp/internal/store"
)

// mockEmbedder はテスト用のEmbedder
type mockEmbedder struct {
	embedFunc func(ctx context.Context, text string) ([]float32, error)
	dim       int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbedder) GetDimension() int {
	return m.dim
}

func newTestNoteService(emb embedder.Embedder, s store.Store, namespace string) *noteService {
	return &noteService{
		embedder:  emb,
		store:     s,
		namespace: namespace,
	}
}

func TestNoteService_AddNote_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "test note",
	}

	resp, err := svc.AddNote(context.Background(), req)
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Namespace != "openai:test:3" {
		t.Errorf("expected namespace openai:test:3, got %s", resp.Namespace)
	}
}

func TestNoteService_AddNote_ProjectIDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		GroupID: "global",
		Text:    "test note",
	}

	_, err := svc.AddNote(context.Background(), req)
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestNoteService_AddNote_GroupIDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		ProjectID: "/test/project",
		Text:      "test note",
	}

	_, err := svc.AddNote(context.Background(), req)
	if !errors.Is(err, ErrGroupIDRequired) {
		t.Errorf("expected ErrGroupIDRequired, got %v", err)
	}
}

func TestNoteService_AddNote_InvalidGroupID(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "invalid group!", // contains space and !
		Text:      "test note",
	}

	_, err := svc.AddNote(context.Background(), req)
	if !errors.Is(err, ErrInvalidGroupID) {
		t.Errorf("expected ErrInvalidGroupID, got %v", err)
	}
}

func TestNoteService_AddNote_TextRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
	}

	_, err := svc.AddNote(context.Background(), req)
	if !errors.Is(err, ErrTextRequired) {
		t.Errorf("expected ErrTextRequired, got %v", err)
	}
}

func TestNoteService_AddNote_CreatedAtDefault(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	before := time.Now().UTC()
	req := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "test note",
	}

	resp, err := svc.AddNote(context.Background(), req)
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	// Check that createdAt was set
	note, err := memStore.Get(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if note.CreatedAt == nil {
		t.Fatal("expected createdAt to be set")
	}

	createdAt, err := time.Parse(time.RFC3339, *note.CreatedAt)
	if err != nil {
		t.Fatalf("failed to parse createdAt: %v", err)
	}

	if createdAt.Before(before) {
		t.Error("createdAt should be after test start time")
	}
}

func TestNoteService_AddNote_EmbedderError(t *testing.T) {
	memStore := store.NewMemoryStore()
	embedErr := errors.New("embed failed")
	emb := &mockEmbedder{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return nil, embedErr
		},
	}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	req := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "test note",
	}

	_, err := svc.AddNote(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, embedErr) {
		t.Errorf("expected embedErr, got %v", err)
	}
}

func TestNoteService_Search_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add a note first
	addReq := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "test note about Go programming",
	}
	_, _ = svc.AddNote(context.Background(), addReq)

	// Search
	searchReq := &SearchRequest{
		ProjectID: "/test/project",
		Query:     "Go programming",
	}

	resp, err := svc.Search(context.Background(), searchReq)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if resp.Namespace != "openai:test:3" {
		t.Errorf("expected namespace openai:test:3, got %s", resp.Namespace)
	}
	if len(resp.Results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestNoteService_Search_ProjectIDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	searchReq := &SearchRequest{
		Query: "test",
	}

	_, err := svc.Search(context.Background(), searchReq)
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestNoteService_Search_QueryRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	searchReq := &SearchRequest{
		ProjectID: "/test/project",
	}

	_, err := svc.Search(context.Background(), searchReq)
	if !errors.Is(err, ErrQueryRequired) {
		t.Errorf("expected ErrQueryRequired, got %v", err)
	}
}

func TestNoteService_Search_WithGroupID(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add notes with different groups
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "feature-1",
		Text:      "feature note",
	})
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "global note",
	})

	// Search with group filter
	groupID := "feature-1"
	searchReq := &SearchRequest{
		ProjectID: "/test/project",
		GroupID:   &groupID,
		Query:     "note",
	}

	resp, err := svc.Search(context.Background(), searchReq)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// All results should be from feature-1 group
	for _, r := range resp.Results {
		if r.GroupID != "feature-1" {
			t.Errorf("expected groupId feature-1, got %s", r.GroupID)
		}
	}
}

func TestNoteService_Get_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add a note first
	addReq := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "test note",
	}
	addResp, _ := svc.AddNote(context.Background(), addReq)

	// Get the note
	resp, err := svc.Get(context.Background(), addResp.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.ID != addResp.ID {
		t.Errorf("expected ID %s, got %s", addResp.ID, resp.ID)
	}
	if resp.Text != "test note" {
		t.Errorf("expected text 'test note', got %s", resp.Text)
	}
}

func TestNoteService_Get_NotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	_, err := svc.Get(context.Background(), "non-existent-id")
	if !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestNoteService_Get_IDRequired(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	_, err := svc.Get(context.Background(), "")
	if !errors.Is(err, ErrIDRequired) {
		t.Errorf("expected ErrIDRequired, got %v", err)
	}
}

func TestNoteService_Update_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add a note first
	addReq := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "original text",
	}
	addResp, _ := svc.AddNote(context.Background(), addReq)

	// Update title only (no re-embed)
	newTitle := "updated title"
	updateReq := &UpdateRequest{
		ID: addResp.ID,
		Patch: NotePatch{
			Title: &newTitle,
		},
	}

	err := svc.Update(context.Background(), updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	note, _ := svc.Get(context.Background(), addResp.ID)
	if note.Title == nil || *note.Title != "updated title" {
		t.Error("title was not updated")
	}
}

func TestNoteService_Update_TextReembed(t *testing.T) {
	memStore := store.NewMemoryStore()
	embedCalls := 0
	emb := &mockEmbedder{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCalls++
			return []float32{0.1, 0.2, 0.3}, nil
		},
		dim: 3,
	}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add a note
	addReq := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "original text",
	}
	addResp, _ := svc.AddNote(context.Background(), addReq)
	initialCalls := embedCalls

	// Update text
	newText := "updated text"
	updateReq := &UpdateRequest{
		ID: addResp.ID,
		Patch: NotePatch{
			Text: &newText,
		},
	}

	err := svc.Update(context.Background(), updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify re-embed was called
	if embedCalls != initialCalls+1 {
		t.Errorf("expected embed to be called once more, calls: initial=%d, after=%d", initialCalls, embedCalls)
	}
}

func TestNoteService_Update_NoTextReembed(t *testing.T) {
	memStore := store.NewMemoryStore()
	embedCalls := 0
	emb := &mockEmbedder{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCalls++
			return []float32{0.1, 0.2, 0.3}, nil
		},
		dim: 3,
	}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add a note
	addReq := &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "original text",
	}
	addResp, _ := svc.AddNote(context.Background(), addReq)
	initialCalls := embedCalls

	// Update title only (no text change)
	newTitle := "updated title"
	updateReq := &UpdateRequest{
		ID: addResp.ID,
		Patch: NotePatch{
			Title: &newTitle,
		},
	}

	err := svc.Update(context.Background(), updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify re-embed was NOT called
	if embedCalls != initialCalls {
		t.Errorf("expected no additional embed calls, calls: initial=%d, after=%d", initialCalls, embedCalls)
	}
}

func TestNoteService_Update_NotFound(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	newText := "text"
	updateReq := &UpdateRequest{
		ID: "non-existent-id",
		Patch: NotePatch{
			Text: &newText,
		},
	}

	err := svc.Update(context.Background(), updateReq)
	if !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestNoteService_ListRecent_Success(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add notes
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "note 1",
	})
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "note 2",
	})

	// List recent
	listReq := &ListRecentRequest{
		ProjectID: "/test/project",
	}

	resp, err := svc.ListRecent(context.Background(), listReq)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Namespace != "openai:test:3" {
		t.Errorf("expected namespace openai:test:3, got %s", resp.Namespace)
	}
}

func TestNoteService_ListRecent_WithGroupID(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add notes with different groups
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "feature-1",
		Text:      "feature note",
	})
	_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
		ProjectID: "/test/project",
		GroupID:   "global",
		Text:      "global note",
	})

	// List with group filter
	groupID := "feature-1"
	listReq := &ListRecentRequest{
		ProjectID: "/test/project",
		GroupID:   &groupID,
	}

	resp, err := svc.ListRecent(context.Background(), listReq)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
	if len(resp.Items) > 0 && resp.Items[0].GroupID != "feature-1" {
		t.Errorf("expected groupId feature-1, got %s", resp.Items[0].GroupID)
	}
}

func TestNoteService_ListRecent_WithLimit(t *testing.T) {
	memStore := store.NewMemoryStore()
	emb := &mockEmbedder{dim: 3}
	svc := newTestNoteService(emb, memStore, "openai:test:3")

	// Add 5 notes
	for i := 0; i < 5; i++ {
		_, _ = svc.AddNote(context.Background(), &AddNoteRequest{
			ProjectID: "/test/project",
			GroupID:   "global",
			Text:      "note",
		})
	}

	// List with limit
	limit := 2
	listReq := &ListRecentRequest{
		ProjectID: "/test/project",
		Limit:     &limit,
	}

	resp, err := svc.ListRecent(context.Background(), listReq)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}
}

// Stub implementation for tests to compile
type noteService struct {
	embedder  embedder.Embedder
	store     store.Store
	namespace string
}

func (s *noteService) AddNote(ctx context.Context, req *AddNoteRequest) (*AddNoteResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *noteService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *noteService) Get(ctx context.Context, id string) (*GetResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *noteService) Update(ctx context.Context, req *UpdateRequest) error {
	return errors.New("not implemented")
}

func (s *noteService) ListRecent(ctx context.Context, req *ListRecentRequest) (*ListRecentResponse, error) {
	return nil, errors.New("not implemented")
}
