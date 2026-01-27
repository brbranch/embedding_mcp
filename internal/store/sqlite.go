package store

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
	_ "modernc.org/sqlite"
)

const (
	// noteCountWarningThreshold は警告を出すノート件数の閾値
	noteCountWarningThreshold = 5000
)

// SQLiteStore はSQLiteを使用したStore実装
type SQLiteStore struct {
	mu          sync.RWMutex
	db          *sql.DB
	dbPath      string
	namespace   string
	initialized bool
}

// NewSQLiteStore はSQLiteStoreを作成する
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// WALモードを有効化
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	return &SQLiteStore{
		db:     db,
		dbPath: dbPath,
	}, nil
}

// Initialize はストアを初期化する
func (s *SQLiteStore) Initialize(ctx context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// notesテーブル作成
	notesSQL := `
	CREATE TABLE IF NOT EXISTS notes (
		id TEXT PRIMARY KEY,
		namespace TEXT NOT NULL,
		project_id TEXT NOT NULL,
		group_id TEXT NOT NULL,
		title TEXT,
		text TEXT NOT NULL,
		tags TEXT,
		source TEXT,
		created_at TEXT,
		metadata TEXT,
		embedding BLOB
	);
	CREATE INDEX IF NOT EXISTS idx_notes_namespace ON notes(namespace);
	CREATE INDEX IF NOT EXISTS idx_notes_project_id ON notes(namespace, project_id);
	CREATE INDEX IF NOT EXISTS idx_notes_group_id ON notes(namespace, group_id);
	CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(namespace, created_at);
	`

	if _, err := s.db.ExecContext(ctx, notesSQL); err != nil {
		return fmt.Errorf("failed to create notes table: %w", err)
	}

	// global_configsテーブル作成
	globalSQL := `
	CREATE TABLE IF NOT EXISTS global_configs (
		id TEXT PRIMARY KEY,
		namespace TEXT NOT NULL,
		project_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		updated_at TEXT,
		UNIQUE(namespace, project_id, key)
	);
	CREATE INDEX IF NOT EXISTS idx_global_configs_namespace ON global_configs(namespace);
	CREATE INDEX IF NOT EXISTS idx_global_configs_project_key ON global_configs(namespace, project_id, key);
	`

	if _, err := s.db.ExecContext(ctx, globalSQL); err != nil {
		return fmt.Errorf("failed to create global_configs table: %w", err)
	}

	s.namespace = namespace
	s.initialized = true
	return nil
}

// Close はストアをクローズする
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = false
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// AddNote はノートを追加する
func (s *SQLiteStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	// createdAtがnilの場合は現在時刻を設定
	if note.CreatedAt == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		note.CreatedAt = &now
	}

	tagsJSON, err := json.Marshal(note.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	var metadataJSON []byte
	if note.Metadata != nil {
		metadataJSON, err = json.Marshal(note.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	embeddingBlob := encodeEmbedding(embedding)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO notes (id, namespace, project_id, group_id, title, text, tags, source, created_at, metadata, embedding)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, note.ID, s.namespace, note.ProjectID, note.GroupID, note.Title, note.Text,
		string(tagsJSON), note.Source, note.CreatedAt, metadataJSON, embeddingBlob)

	if err != nil {
		return fmt.Errorf("failed to insert note: %w", err)
	}

	// 件数チェックと警告
	count, _ := s.countNotes(ctx)
	if count >= noteCountWarningThreshold {
		slog.Warn("note count exceeded threshold",
			"count", count,
			"threshold", noteCountWarningThreshold,
			"recommendation", "consider using ChromaStore for better performance")
	}

	return nil
}

// Get はIDでノートを取得する
func (s *SQLiteStore) Get(ctx context.Context, id string) (*model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, group_id, title, text, tags, source, created_at, metadata
		FROM notes
		WHERE id = ? AND namespace = ?
	`, id, s.namespace)

	note, err := s.scanNote(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	return note, nil
}

// Update はノートを更新する
func (s *SQLiteStore) Update(ctx context.Context, note *model.Note, embedding []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	// 存在確認
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM notes WHERE id = ? AND namespace = ?
	`, note.ID, s.namespace).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check note existence: %w", err)
	}

	tagsJSON, err := json.Marshal(note.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	var metadataJSON []byte
	if note.Metadata != nil {
		metadataJSON, err = json.Marshal(note.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	embeddingBlob := encodeEmbedding(embedding)

	_, err = s.db.ExecContext(ctx, `
		UPDATE notes
		SET project_id = ?, group_id = ?, title = ?, text = ?, tags = ?, source = ?, created_at = ?, metadata = ?, embedding = ?
		WHERE id = ? AND namespace = ?
	`, note.ProjectID, note.GroupID, note.Title, note.Text, string(tagsJSON),
		note.Source, note.CreatedAt, metadataJSON, embeddingBlob, note.ID, s.namespace)

	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

// Delete はノートを削除する
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM notes WHERE id = ? AND namespace = ?
	`, id, s.namespace)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Search はベクトル検索を実行する
func (s *SQLiteStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	// 全件取得（namespace + projectIDフィルタ）
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, group_id, title, text, tags, source, created_at, metadata, embedding
		FROM notes
		WHERE namespace = ? AND project_id = ?
	`, s.namespace, opts.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	for rows.Next() {
		var (
			id, projectID, groupID, text string
			title, source, createdAt     sql.NullString
			tagsJSON, metadataJSON       sql.NullString
			embeddingBlob                []byte
		)

		if err := rows.Scan(&id, &projectID, &groupID, &title, &text, &tagsJSON, &source, &createdAt, &metadataJSON, &embeddingBlob); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		note := &model.Note{
			ID:        id,
			ProjectID: projectID,
			GroupID:   groupID,
			Text:      text,
		}

		if title.Valid {
			note.Title = &title.String
		}
		if source.Valid {
			note.Source = &source.String
		}
		if createdAt.Valid {
			note.CreatedAt = &createdAt.String
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &note.Tags)
		}
		if note.Tags == nil {
			note.Tags = []string{}
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &note.Metadata)
		}

		// groupIDフィルタ
		if opts.GroupID != nil && note.GroupID != *opts.GroupID {
			continue
		}

		// tagsフィルタ（AND検索）
		if len(opts.Tags) > 0 {
			if !ContainsAllTags(note.Tags, opts.Tags) {
				continue
			}
		}

		// since/untilフィルタ
		if opts.Since != nil || opts.Until != nil {
			if note.CreatedAt == nil {
				continue
			}
			createdTime, err := time.Parse(time.RFC3339, *note.CreatedAt)
			if err != nil {
				continue
			}

			// since <= createdAt
			if opts.Since != nil && createdTime.Before(*opts.Since) {
				continue
			}

			// createdAt < until
			if opts.Until != nil && !createdTime.Before(*opts.Until) {
				continue
			}
		}

		// cosine類似度計算
		noteEmbedding := decodeEmbedding(embeddingBlob)
		distance := CosineSimilarity(embedding, noteEmbedding)
		score := 1.0 - (distance / 2.0) // 0-1に正規化

		results = append(results, SearchResult{
			Note:  note,
			Score: score,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
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
func (s *SQLiteStore) ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, ErrNotInitialized
	}

	// 全件取得（namespace + projectIDフィルタ、createdAt降順）
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, group_id, title, text, tags, source, created_at, metadata
		FROM notes
		WHERE namespace = ? AND project_id = ?
		ORDER BY created_at DESC NULLS LAST
	`, s.namespace, opts.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	var notes []*model.Note

	for rows.Next() {
		var (
			id, projectID, groupID, text string
			title, source, createdAt     sql.NullString
			tagsJSON, metadataJSON       sql.NullString
		)

		if err := rows.Scan(&id, &projectID, &groupID, &title, &text, &tagsJSON, &source, &createdAt, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		note := &model.Note{
			ID:        id,
			ProjectID: projectID,
			GroupID:   groupID,
			Text:      text,
		}

		if title.Valid {
			note.Title = &title.String
		}
		if source.Valid {
			note.Source = &source.String
		}
		if createdAt.Valid {
			note.CreatedAt = &createdAt.String
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &note.Tags)
		}
		if note.Tags == nil {
			note.Tags = []string{}
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &note.Metadata)
		}

		// groupIDフィルタ
		if opts.GroupID != nil && note.GroupID != *opts.GroupID {
			continue
		}

		// tagsフィルタ（AND検索）
		if len(opts.Tags) > 0 {
			if !ContainsAllTags(note.Tags, opts.Tags) {
				continue
			}
		}

		notes = append(notes, note)

		// Limit制限
		if opts.Limit > 0 && len(notes) >= opts.Limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return notes, nil
}

// UpsertGlobal はグローバル設定を追加/更新する
func (s *SQLiteStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return ErrNotInitialized
	}

	var valueJSON []byte
	var err error
	if config.Value != nil {
		valueJSON, err = json.Marshal(config.Value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO global_configs (id, namespace, project_id, key, value, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(namespace, project_id, key) DO UPDATE SET
			id = excluded.id,
			value = excluded.value,
			updated_at = excluded.updated_at
	`, config.ID, s.namespace, config.ProjectID, config.Key, string(valueJSON), config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert global config: %w", err)
	}

	return nil
}

// GetGlobal はグローバル設定を取得する
func (s *SQLiteStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return nil, false, ErrNotInitialized
	}

	var (
		id, pID, k   string
		valueJSON    sql.NullString
		updatedAt    sql.NullString
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, key, value, updated_at
		FROM global_configs
		WHERE namespace = ? AND project_id = ? AND key = ?
	`, s.namespace, projectID, key).Scan(&id, &pID, &k, &valueJSON, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get global config: %w", err)
	}

	config := &model.GlobalConfig{
		ID:        id,
		ProjectID: pID,
		Key:       k,
	}

	if valueJSON.Valid && valueJSON.String != "" {
		var value any
		if err := json.Unmarshal([]byte(valueJSON.String), &value); err == nil {
			config.Value = value
		} else {
			config.Value = valueJSON.String
		}
	}

	if updatedAt.Valid {
		ua := updatedAt.String
		config.UpdatedAt = &ua
	}

	return config, true, nil
}

// Helper functions

func (s *SQLiteStore) countNotes(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notes WHERE namespace = ?
	`, s.namespace).Scan(&count)
	return count, err
}

func (s *SQLiteStore) scanNote(row *sql.Row) (*model.Note, error) {
	var (
		id, projectID, groupID, text string
		title, source, createdAt     sql.NullString
		tagsJSON, metadataJSON       sql.NullString
	)

	if err := row.Scan(&id, &projectID, &groupID, &title, &text, &tagsJSON, &source, &createdAt, &metadataJSON); err != nil {
		return nil, err
	}

	note := &model.Note{
		ID:        id,
		ProjectID: projectID,
		GroupID:   groupID,
		Text:      text,
	}

	if title.Valid {
		note.Title = &title.String
	}
	if source.Valid {
		note.Source = &source.String
	}
	if createdAt.Valid {
		note.CreatedAt = &createdAt.String
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &note.Tags)
	}
	if note.Tags == nil {
		note.Tags = []string{}
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		json.Unmarshal([]byte(metadataJSON.String), &note.Metadata)
	}

	return note, nil
}

// encodeEmbedding はfloat32配列をバイト配列に変換する
func encodeEmbedding(embedding []float32) []byte {
	buf := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// decodeEmbedding はバイト配列をfloat32配列に変換する
func decodeEmbedding(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	embedding := make([]float32, len(data)/4)
	for i := range embedding {
		embedding[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return embedding
}
