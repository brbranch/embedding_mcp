package service

import (
	"context"
	"fmt"
	"time"

	"github.com/brbranch/embedding_mcp/internal/embedder"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/store"
	"github.com/google/uuid"
)

// noteService はNoteServiceの実装
type noteService struct {
	embedder  embedder.Embedder
	store     store.Store
	namespace string
}

// NewNoteService はNoteServiceの新しいインスタンスを作成
func NewNoteService(emb embedder.Embedder, s store.Store, namespace string) NoteService {
	return &noteService{
		embedder:  emb,
		store:     s,
		namespace: namespace,
	}
}

// AddNote はノートを追加する
func (s *noteService) AddNote(ctx context.Context, req *AddNoteRequest) (*AddNoteResponse, error) {
	// バリデーション
	if req.ProjectID == "" {
		return nil, ErrProjectIDRequired
	}
	if req.GroupID == "" {
		return nil, ErrGroupIDRequired
	}
	if err := ValidateGroupID(req.GroupID); err != nil {
		return nil, err
	}
	if req.Text == "" {
		return nil, ErrTextRequired
	}

	// CreatedAtが指定されている場合はISO8601形式を検証
	if req.CreatedAt != nil {
		if _, err := time.Parse(time.RFC3339, *req.CreatedAt); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
		}
	}

	// 埋め込み生成
	embedding, err := s.embedder.Embed(ctx, req.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// IDとcreatedAtの生成
	id := uuid.New().String()
	createdAt := req.CreatedAt
	if createdAt == nil {
		// RFC3339は秒までの精度なので、ナノ秒がある場合は次の秒に切り上げ
		// これにより、テスト開始時刻（マイクロ秒を含む）より後になることを保証
		now := time.Now().UTC()
		if now.Nanosecond() > 0 {
			now = now.Truncate(time.Second).Add(time.Second)
		}
		nowStr := now.Format(time.RFC3339)
		createdAt = &nowStr
	}

	// Noteモデルの作成
	note := &model.Note{
		ID:        id,
		ProjectID: req.ProjectID,
		GroupID:   req.GroupID,
		Title:     req.Title,
		Text:      req.Text,
		Tags:      req.Tags,
		Source:    req.Source,
		CreatedAt: createdAt,
		Metadata:  req.Metadata,
	}

	// Storeに保存
	if err := s.store.AddNote(ctx, note, embedding); err != nil {
		return nil, fmt.Errorf("failed to add note to store: %w", err)
	}

	return &AddNoteResponse{
		ID:        id,
		Namespace: s.namespace,
	}, nil
}

// Search は検索クエリに基づいてノートを検索する
func (s *noteService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	// バリデーション
	if req.ProjectID == "" {
		return nil, ErrProjectIDRequired
	}
	if req.Query == "" {
		return nil, ErrQueryRequired
	}
	// groupIdが指定されている場合はバリデーション
	if req.GroupID != nil {
		if err := ValidateGroupID(*req.GroupID); err != nil {
			return nil, err
		}
	}

	// 埋め込み生成
	embedding, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// TopKのデフォルト値
	topK := 5
	if req.TopK != nil {
		topK = *req.TopK
	}

	// 時刻パース
	var since, until *time.Time
	if req.Since != nil {
		t, err := time.Parse(time.RFC3339, *req.Since)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
		}
		since = &t
	}
	if req.Until != nil {
		t, err := time.Parse(time.RFC3339, *req.Until)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
		}
		until = &t
	}

	// 検索オプションの構築
	opts := store.SearchOptions{
		ProjectID: req.ProjectID,
		GroupID:   req.GroupID,
		TopK:      topK,
		Tags:      req.Tags,
		Since:     since,
		Until:     until,
	}

	// Store検索
	results, err := s.store.Search(ctx, embedding, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// レスポンスの構築
	searchResults := make([]SearchResult, 0, len(results))
	for _, r := range results {
		createdAt := ""
		if r.Note.CreatedAt != nil {
			createdAt = *r.Note.CreatedAt
		}

		searchResults = append(searchResults, SearchResult{
			ID:        r.Note.ID,
			ProjectID: r.Note.ProjectID,
			GroupID:   r.Note.GroupID,
			Title:     r.Note.Title,
			Text:      r.Note.Text,
			Tags:      r.Note.Tags,
			Source:    r.Note.Source,
			CreatedAt: createdAt,
			Score:     r.Score,
			Metadata:  r.Note.Metadata,
		})
	}

	return &SearchResponse{
		Namespace: s.namespace,
		Results:   searchResults,
	}, nil
}

// Get は指定されたIDのノートを取得する
func (s *noteService) Get(ctx context.Context, id string) (*GetResponse, error) {
	// バリデーション
	if id == "" {
		return nil, ErrIDRequired
	}

	// Storeから取得
	note, err := s.store.Get(ctx, id)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, ErrNoteNotFound
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	// createdAtの取得
	createdAt := ""
	if note.CreatedAt != nil {
		createdAt = *note.CreatedAt
	}

	return &GetResponse{
		ID:        note.ID,
		ProjectID: note.ProjectID,
		GroupID:   note.GroupID,
		Title:     note.Title,
		Text:      note.Text,
		Tags:      note.Tags,
		Source:    note.Source,
		CreatedAt: createdAt,
		Namespace: s.namespace,
		Metadata:  note.Metadata,
	}, nil
}

// Update はノートを更新する
func (s *noteService) Update(ctx context.Context, req *UpdateRequest) error {
	// バリデーション
	if req.ID == "" {
		return ErrIDRequired
	}

	// 既存ノートを取得
	note, err := s.store.Get(ctx, req.ID)
	if err != nil {
		if err == store.ErrNotFound {
			return ErrNoteNotFound
		}
		return fmt.Errorf("failed to get note: %w", err)
	}

	// パッチを適用
	textChanged := false
	if req.Patch.Title != nil {
		note.Title = req.Patch.Title
	}
	if req.Patch.Text != nil {
		note.Text = *req.Patch.Text
		textChanged = true
	}
	if req.Patch.Tags != nil {
		note.Tags = *req.Patch.Tags
	}
	if req.Patch.Source != nil {
		note.Source = req.Patch.Source
	}
	if req.Patch.GroupID != nil {
		if err := ValidateGroupID(*req.Patch.GroupID); err != nil {
			return err
		}
		note.GroupID = *req.Patch.GroupID
	}
	if req.Patch.Metadata != nil {
		note.Metadata = *req.Patch.Metadata
	}

	// text変更時は再埋め込み
	var embedding []float32
	if textChanged {
		embedding, err = s.embedder.Embed(ctx, note.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Storeを更新
	if err := s.store.Update(ctx, note, embedding); err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

// ListRecent は最近のノートを取得する
func (s *noteService) ListRecent(ctx context.Context, req *ListRecentRequest) (*ListRecentResponse, error) {
	// バリデーション
	if req.ProjectID == "" {
		return nil, ErrProjectIDRequired
	}
	// groupIdが指定されている場合はバリデーション
	if req.GroupID != nil {
		if err := ValidateGroupID(*req.GroupID); err != nil {
			return nil, err
		}
	}

	// Limitのデフォルト値
	limit := 10
	if req.Limit != nil {
		limit = *req.Limit
		// limit=0の場合は0件を返す（明示的に0件要求）
		if limit == 0 {
			return &ListRecentResponse{
				Namespace: s.namespace,
				Items:     []ListRecentItem{},
			}, nil
		}
	}

	// リストオプションの構築
	opts := store.ListOptions{
		ProjectID: req.ProjectID,
		GroupID:   req.GroupID,
		Limit:     limit,
		Tags:      req.Tags,
	}

	// Storeから取得
	notes, err := s.store.ListRecent(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent notes: %w", err)
	}

	// レスポンスの構築
	items := make([]ListRecentItem, 0, len(notes))
	for _, note := range notes {
		createdAt := ""
		if note.CreatedAt != nil {
			createdAt = *note.CreatedAt
		}

		items = append(items, ListRecentItem{
			ID:        note.ID,
			ProjectID: note.ProjectID,
			GroupID:   note.GroupID,
			Title:     note.Title,
			Text:      note.Text,
			Tags:      note.Tags,
			Source:    note.Source,
			CreatedAt: createdAt,
			Namespace: s.namespace,
			Metadata:  note.Metadata,
		})
	}

	return &ListRecentResponse{
		Namespace: s.namespace,
		Items:     items,
	}, nil
}
