package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// MemoryStore はテスト用のインメモリStore実装
type MemoryStore struct {
	mu               sync.RWMutex
	notes            map[string]*noteEntry      // key: note.ID
	globalConfigs    map[string]*model.GlobalConfig // key: projectID + key
	initialized      bool
	namespace        string
}

type noteEntry struct {
	note      *model.Note
	embedding []float32
}

// NewMemoryStore はMemoryStoreを作成する
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		notes:         make(map[string]*noteEntry),
		globalConfigs: make(map[string]*model.GlobalConfig),
	}
}

// Initialize はストアを初期化する
func (s *MemoryStore) Initialize(ctx context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.namespace = namespace
	s.initialized = true
	return nil
}

// Close はストアをクローズする
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notes = make(map[string]*noteEntry)
	s.globalConfigs = make(map[string]*model.GlobalConfig)
	s.initialized = false
	return nil
}

// AddNote はノートを追加する
func (s *MemoryStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	// ディープコピー
	noteCopy := s.copyNote(note)
	embeddingCopy := make([]float32, len(embedding))
	copy(embeddingCopy, embedding)

	s.notes[note.ID] = &noteEntry{
		note:      noteCopy,
		embedding: embeddingCopy,
	}

	return nil
}

// Get はIDでノートを取得する
func (s *MemoryStore) Get(ctx context.Context, id string) (*model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	entry, ok := s.notes[id]
	if !ok {
		return nil, ErrNotFound
	}

	return s.copyNote(entry.note), nil
}

// Update はノートを更新する
func (s *MemoryStore) Update(ctx context.Context, note *model.Note, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	if _, ok := s.notes[note.ID]; !ok {
		return ErrNotFound
	}

	// ディープコピー
	noteCopy := s.copyNote(note)
	embeddingCopy := make([]float32, len(embedding))
	copy(embeddingCopy, embedding)

	s.notes[note.ID] = &noteEntry{
		note:      noteCopy,
		embedding: embeddingCopy,
	}

	return nil
}

// Delete はノートを削除する
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	if _, ok := s.notes[id]; !ok {
		return ErrNotFound
	}

	delete(s.notes, id)
	return nil
}

// Search はベクトル検索を実行する
func (s *MemoryStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	var results []SearchResult

	// 全ノートをスキャン
	for _, entry := range s.notes {
		// projectIDフィルタ
		if entry.note.ProjectID != opts.ProjectID {
			continue
		}

		// groupIDフィルタ
		if opts.GroupID != nil && entry.note.GroupID != *opts.GroupID {
			continue
		}

		// tagsフィルタ（AND検索）
		if len(opts.Tags) > 0 {
			if !containsAllTags(entry.note.Tags, opts.Tags) {
				continue
			}
		}

		// since/untilフィルタ
		if opts.Since != nil || opts.Until != nil {
			if entry.note.CreatedAt == nil {
				continue
			}
			createdAt, err := time.Parse(time.RFC3339, *entry.note.CreatedAt)
			if err != nil {
				continue
			}

			// since <= createdAt
			if opts.Since != nil && createdAt.Before(*opts.Since) {
				continue
			}

			// createdAt < until
			if opts.Until != nil && !createdAt.Before(*opts.Until) {
				continue
			}
		}

		// コサイン距離を計算してスコアに変換
		distance := cosineSimilarity(embedding, entry.embedding)
		score := 1.0 - (distance / 2.0) // 0-1に正規化

		results = append(results, SearchResult{
			Note:  s.copyNote(entry.note),
			Score: score,
		})
	}

	// スコア降順でソート
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// TopK制限
	if opts.TopK > 0 && len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	return results, nil
}

// ListRecent は最新ノート一覧を取得する
func (s *MemoryStore) ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	var notes []*model.Note

	// 全ノートをスキャン
	for _, entry := range s.notes {
		// projectIDフィルタ
		if entry.note.ProjectID != opts.ProjectID {
			continue
		}

		// groupIDフィルタ
		if opts.GroupID != nil && entry.note.GroupID != *opts.GroupID {
			continue
		}

		// tagsフィルタ（AND検索）
		if len(opts.Tags) > 0 {
			if !containsAllTags(entry.note.Tags, opts.Tags) {
				continue
			}
		}

		notes = append(notes, s.copyNote(entry.note))
	}

	// createdAt降順でソート
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].CreatedAt == nil || notes[j].CreatedAt == nil {
			return false
		}
		ti, _ := time.Parse(time.RFC3339, *notes[i].CreatedAt)
		tj, _ := time.Parse(time.RFC3339, *notes[j].CreatedAt)
		return ti.After(tj)
	})

	// Limit制限
	if opts.Limit > 0 && len(notes) > opts.Limit {
		notes = notes[:opts.Limit]
	}

	return notes, nil
}

// UpsertGlobal はグローバル設定を追加/更新する
func (s *MemoryStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	key := s.globalKey(config.ProjectID, config.Key)

	// ディープコピー
	configCopy := &model.GlobalConfig{
		ID:        config.ID,
		ProjectID: config.ProjectID,
		Key:       config.Key,
		Value:     s.copyValue(config.Value),
		UpdatedAt: config.UpdatedAt,
	}

	s.globalConfigs[key] = configCopy
	return nil
}

// GetGlobal はグローバル設定を取得する
func (s *MemoryStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, false, ErrNotInitialized
	}

	gkey := s.globalKey(projectID, key)
	config, ok := s.globalConfigs[gkey]
	if !ok {
		return nil, false, nil
	}

	// ディープコピー
	configCopy := &model.GlobalConfig{
		ID:        config.ID,
		ProjectID: config.ProjectID,
		Key:       config.Key,
		Value:     s.copyValue(config.Value),
		UpdatedAt: config.UpdatedAt,
	}

	return configCopy, true, nil
}

// Helper methods

func (s *MemoryStore) globalKey(projectID, key string) string {
	return fmt.Sprintf("%s:%s", projectID, key)
}

func (s *MemoryStore) copyNote(note *model.Note) *model.Note {
	noteCopy := &model.Note{
		ID:        note.ID,
		ProjectID: note.ProjectID,
		GroupID:   note.GroupID,
		Text:      note.Text,
		Tags:      make([]string, len(note.Tags)),
	}

	copy(noteCopy.Tags, note.Tags)

	if note.Title != nil {
		title := *note.Title
		noteCopy.Title = &title
	}

	if note.Source != nil {
		source := *note.Source
		noteCopy.Source = &source
	}

	if note.CreatedAt != nil {
		createdAt := *note.CreatedAt
		noteCopy.CreatedAt = &createdAt
	}

	if note.Metadata != nil {
		noteCopy.Metadata = s.copyValue(note.Metadata).(map[string]any)
	}

	return noteCopy
}

func (s *MemoryStore) copyValue(v any) any {
	if v == nil {
		return nil
	}

	// JSON経由でディープコピー
	b, _ := json.Marshal(v)
	var result any
	json.Unmarshal(b, &result)
	return result
}

func containsAllTags(tags []string, targets []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, target := range targets {
		if !tagSet[target] {
			return false
		}
	}

	return true
}

// cosineSimilarity はコサイン類似度を計算する（実際はcosine distanceを返す: 0=同一、2=正反対）
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 2.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 2.0
	}

	// cosine similarity: -1 to 1
	similarity := dotProduct / (normA * normB)

	// cosine distance: 0 to 2
	distance := 1.0 - similarity

	return distance
}
