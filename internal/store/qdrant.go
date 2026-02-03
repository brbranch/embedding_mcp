package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

// sanitizeCollectionName はQdrantのコレクション名として使用できる文字列に変換する
// Qdrantは ":" などの特殊文字をコレクション名に使用できないため
func sanitizeCollectionName(name string) string {
	return strings.ReplaceAll(name, ":", "_")
}

// noteCollection はNote用コレクション名を返す
func (s *QdrantStore) noteCollection() string {
	return sanitizeCollectionName(s.namespace)
}

// globalConfigCollection はGlobalConfig用コレクション名を返す
func (s *QdrantStore) globalConfigCollection() string {
	return sanitizeCollectionName(s.namespace) + "_global_configs"
}

// groupCollection はGroup用コレクション名を返す
func (s *QdrantStore) groupCollection() string {
	return sanitizeCollectionName(s.namespace) + "_groups"
}

// QdrantStore はQdrantを使用したStore実装
type QdrantStore struct {
	client      *qdrant.Client
	url         string
	namespace   string
	vectorDim   uint64       // ベクトル次元数（namespaceから取得）
	initialized bool
	mu          sync.RWMutex // initializedフラグの保護
}

// NewQdrantStore はQdrantStoreを作成する
func NewQdrantStore(urlStr string) (*QdrantStore, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	host := parsedURL.Hostname()
	portStr := parsedURL.Port()
	// Qdrant gRPCポートはデフォルト6334（HTTPは6333）
	port := 6334
	if portStr != "" {
		// URLにポートが明示されている場合は、gRPCポートに変換
		// 例: http://localhost:6333 -> 6334
		if p, err := strconv.Atoi(portStr); err == nil {
			if p == 6333 {
				port = 6334 // HTTPポート指定の場合はgRPCポートに変換
			} else {
				port = p
			}
		}
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
		SkipCompatibilityCheck: true, // バージョンチェックをスキップ
	})
	if err != nil {
		return nil, ErrConnectionFailed
	}

	// 接続確認
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.HealthCheck(ctx); err != nil {
		return nil, ErrConnectionFailed
	}

	return &QdrantStore{
		client: client,
		url:    urlStr,
	}, nil
}

// parseVectorDim はnamespaceからベクトル次元数を取得する
// namespaceは "provider:model:dim" の形式（例: "openai:text-embedding-3-small:1536"）
func parseVectorDim(namespace string) uint64 {
	parts := strings.Split(namespace, ":")
	if len(parts) >= 3 {
		if dim, err := strconv.ParseUint(parts[len(parts)-1], 10, 64); err == nil {
			return dim
		}
	}
	return 1536 // デフォルト
}

// Initialize はストアを初期化する
func (s *QdrantStore) Initialize(ctx context.Context, namespace string) error {
	if s.client == nil {
		return ErrConnectionFailed
	}

	// namespaceからベクトル次元数を取得
	vectorDim := parseVectorDim(namespace)

	// コレクション名をサニタイズ（Qdrantは ":" を許可しない）
	collectionName := sanitizeCollectionName(namespace)

	// Note用コレクション存在確認
	exists, err := s.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	// Note用コレクションが存在しない場合は作成
	if !exists {
		err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     vectorDim,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// GlobalConfig用コレクション作成
	globalConfigCollection := collectionName + "_global_configs"
	exists, err = s.client.CollectionExists(ctx, globalConfigCollection)
	if err != nil {
		return fmt.Errorf("failed to check global_configs collection existence: %w", err)
	}
	if !exists {
		err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: globalConfigCollection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1, // ダミーベクトル（1次元）
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create global_configs collection: %w", err)
		}
	}

	// Group用コレクション作成
	groupCollection := collectionName + "_groups"
	exists, err = s.client.CollectionExists(ctx, groupCollection)
	if err != nil {
		return fmt.Errorf("failed to check groups collection existence: %w", err)
	}
	if !exists {
		err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: groupCollection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1, // ダミーベクトル（1次元）
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create groups collection: %w", err)
		}
	}

	s.mu.Lock()
	s.namespace = namespace
	s.vectorDim = vectorDim
	s.initialized = true
	s.mu.Unlock()
	return nil
}

// Close はストアをクローズする
func (s *QdrantStore) Close() error {
	s.mu.Lock()
	s.initialized = false
	s.mu.Unlock()
	if s.client != nil {
		s.client.Close()
	}
	return nil
}

// isInitialized は初期化状態を安全に取得する
func (s *QdrantStore) isInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// AddNote はノートを追加する
func (s *QdrantStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// createdAtがnilの場合は現在時刻を設定
	if note.CreatedAt == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		note.CreatedAt = &now
	}

	// tagsがnilの場合は空配列を設定
	if note.Tags == nil {
		note.Tags = []string{}
	}

	// payloadを構築
	payload := buildPayload(note)

	// ポイントを追加
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.noteCollection(),
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(hashID(note.ID)),
				Vectors: qdrant.NewVectors(embedding...),
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// Get はIDでノートを取得する
func (s *QdrantStore) Get(ctx context.Context, id string) (*model.Note, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// IDでポイントを取得
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.noteCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get point: %w", err)
	}

	if len(points) == 0 {
		return nil, ErrNotFound
	}

	// payloadからNoteを構築
	note, err := payloadToNote(points[0].Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload to note: %w", err)
	}

	return note, nil
}

// Update はノートを更新する
func (s *QdrantStore) Update(ctx context.Context, note *model.Note, embedding []float32) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.noteCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(note.ID))},
		WithPayload:    qdrant.NewWithPayload(false),
	})

	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if len(points) == 0 {
		return ErrNotFound
	}

	// tagsがnilの場合は空配列を設定
	if note.Tags == nil {
		note.Tags = []string{}
	}

	// payloadを構築
	payload := buildPayload(note)

	// ポイントを更新（Upsertで上書き）
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.noteCollection(),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(hashID(note.ID)),
				Vectors: qdrant.NewVectors(embedding...),
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update point: %w", err)
	}

	return nil
}

// Delete はノートを削除する
func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.noteCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(false),
	})

	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if len(points) == 0 {
		return ErrNotFound
	}

	// ポイントを削除
	_, err = s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.noteCollection(),
		Points:         qdrant.NewPointsSelector(qdrant.NewIDNum(hashID(id))),
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	return nil
}

// Search はベクトル検索を実行する
func (s *QdrantStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// フィルタを構築
	filter := buildSearchFilter(opts)

	// Qdrantで検索実行
	queryResp, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.noteCollection(),
		Query:          qdrant.NewQuery(embedding...),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint64(opts.TopK)),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query points: %w", err)
	}

	// 結果を変換
	var results []SearchResult
	for _, point := range queryResp {
		note, err := payloadToNote(point.Payload)
		if err != nil {
			continue
		}

		// スコアを0-1に正規化 (Qdrantのcosine距離は-1〜1なので (score+1)/2)
		score := float64((point.Score + 1.0) / 2.0)

		results = append(results, SearchResult{
			Note:  note,
			Score: score,
		})
	}

	// スコア降順でソート（念のため）
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// ListRecent は最新のノートをリストする
func (s *QdrantStore) ListRecent(ctx context.Context, opts ListOptions) ([]*model.Note, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// フィルタを構築
	filter := buildListFilter(opts)

	// Qdrant Scrollは順序保証がないため、十分な件数を取得してソートする
	// 最低1000件、またはlimit*10のうち大きい方を取得
	fetchLimit := opts.Limit * 10
	if fetchLimit < 1000 {
		fetchLimit = 1000
	}

	// Qdrantで全ポイントを取得
	scrollResp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.noteCollection(),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(fetchLimit)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scroll points: %w", err)
	}

	// payloadからNoteに変換
	var notes []*model.Note
	for _, point := range scrollResp {
		note, err := payloadToNote(point.Payload)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	// createdAt降順でソート
	// パースエラーの場合はzero timeとして末尾にソート
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].CreatedAt == nil {
			return false // nilは末尾
		}
		if notes[j].CreatedAt == nil {
			return true // nilは末尾
		}
		ti, erri := time.Parse(time.RFC3339, *notes[i].CreatedAt)
		tj, errj := time.Parse(time.RFC3339, *notes[j].CreatedAt)
		if erri != nil {
			log.Printf("warning: failed to parse createdAt for note %s: %v", notes[i].ID, erri)
			return false // パースエラーは末尾
		}
		if errj != nil {
			log.Printf("warning: failed to parse createdAt for note %s: %v", notes[j].ID, errj)
			return true // パースエラーは末尾
		}
		return ti.After(tj)
	})

	// Limit制限
	if opts.Limit > 0 && len(notes) > opts.Limit {
		notes = notes[:opts.Limit]
	}

	return notes, nil
}

// Helper functions

// hashID は文字列IDを数値IDに変換する（簡易実装）
func hashID(id string) uint64 {
	// SHA256ハッシュの先頭8バイトを使用して衝突耐性を向上
	h := sha256.Sum256([]byte(id))
	return binary.BigEndian.Uint64(h[:8])
}

// buildSearchFilter はSearchOptionsからQdrantのフィルタを構築する
func buildSearchFilter(opts SearchOptions) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// projectIDフィルタ（必須）
	conditions = append(conditions, qdrant.NewMatch("projectId", opts.ProjectID))

	// groupIDフィルタ（オプション）
	if opts.GroupID != nil {
		conditions = append(conditions, qdrant.NewMatch("groupId", *opts.GroupID))
	}

	// tagsフィルタ（AND検索）
	for _, tag := range opts.Tags {
		conditions = append(conditions, qdrant.NewMatch("tags", tag))
	}

	// 時間範囲フィルタ
	if opts.Since != nil || opts.Until != nil {
		rangeCondition := &qdrant.Range{}
		if opts.Since != nil {
			// since <= createdAt
			sinceTimestamp := float64(opts.Since.Unix())
			rangeCondition.Gte = &sinceTimestamp
		}
		if opts.Until != nil {
			// createdAt < until
			untilTimestamp := float64(opts.Until.Unix())
			rangeCondition.Lt = &untilTimestamp
		}
		conditions = append(conditions, qdrant.NewRange("createdAtTimestamp", rangeCondition))
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// buildListFilter はListOptionsからQdrantのフィルタを構築する
func buildListFilter(opts ListOptions) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// projectIDフィルタ（必須）
	conditions = append(conditions, qdrant.NewMatch("projectId", opts.ProjectID))

	// groupIDフィルタ（オプション）
	if opts.GroupID != nil {
		conditions = append(conditions, qdrant.NewMatch("groupId", *opts.GroupID))
	}

	// tagsフィルタ（AND検索）
	for _, tag := range opts.Tags {
		conditions = append(conditions, qdrant.NewMatch("tags", tag))
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// buildPayload はNoteからQdrantのpayloadを構築する
func buildPayload(note *model.Note) map[string]*qdrant.Value {
	payload := make(map[string]*qdrant.Value)

	payload["id"], _ = qdrant.NewValue(note.ID)
	payload["projectId"], _ = qdrant.NewValue(note.ProjectID)
	payload["groupId"], _ = qdrant.NewValue(note.GroupID)
	payload["text"], _ = qdrant.NewValue(note.Text)

	if note.Title != nil {
		payload["title"], _ = qdrant.NewValue(*note.Title)
	}
	if note.Source != nil {
		payload["source"], _ = qdrant.NewValue(*note.Source)
	}
	if note.CreatedAt != nil {
		payload["createdAt"], _ = qdrant.NewValue(*note.CreatedAt)
		// 時間フィルタ用にタイムスタンプも保存
		if t, err := time.Parse(time.RFC3339, *note.CreatedAt); err == nil {
			payload["createdAtTimestamp"], _ = qdrant.NewValue(float64(t.Unix()))
		}
	}

	// tags を *qdrant.Value のリストに変換
	tagValues := make([]*qdrant.Value, len(note.Tags))
	for i, tag := range note.Tags {
		tagValues[i], _ = qdrant.NewValue(tag)
	}
	payload["tags"] = qdrant.NewValueList(&qdrant.ListValue{Values: tagValues})

	// metadata をJSON経由で変換
	if note.Metadata != nil {
		jsonBytes, err := json.Marshal(note.Metadata)
		if err == nil {
			var metadataMap map[string]any
			if err := json.Unmarshal(jsonBytes, &metadataMap); err == nil {
				valueMap := qdrant.NewValueMap(metadataMap)
				payload["metadata"], _ = qdrant.NewValue(valueMap)
			}
		}
	}

	return payload
}

// payloadToNote はQdrantのpayloadからNoteを構築する
func payloadToNote(payload map[string]*qdrant.Value) (*model.Note, error) {
	note := &model.Note{}

	if v, ok := payload["id"]; ok && v.GetStringValue() != "" {
		note.ID = v.GetStringValue()
	}
	if v, ok := payload["projectId"]; ok && v.GetStringValue() != "" {
		note.ProjectID = v.GetStringValue()
	}
	if v, ok := payload["groupId"]; ok && v.GetStringValue() != "" {
		note.GroupID = v.GetStringValue()
	}
	if v, ok := payload["text"]; ok && v.GetStringValue() != "" {
		note.Text = v.GetStringValue()
	}

	if v, ok := payload["title"]; ok && v.GetStringValue() != "" {
		title := v.GetStringValue()
		note.Title = &title
	}
	if v, ok := payload["source"]; ok && v.GetStringValue() != "" {
		source := v.GetStringValue()
		note.Source = &source
	}
	if v, ok := payload["createdAt"]; ok && v.GetStringValue() != "" {
		createdAt := v.GetStringValue()
		note.CreatedAt = &createdAt
	}

	// tagsの取得
	if v, ok := payload["tags"]; ok && v.GetListValue() != nil {
		tags := []string{}
		for _, item := range v.GetListValue().Values {
			if s := item.GetStringValue(); s != "" {
				tags = append(tags, s)
			}
		}
		note.Tags = tags
	} else {
		note.Tags = []string{}
	}

	// metadataの取得（convertQdrantValueを使用して型を正確に復元）
	if v, ok := payload["metadata"]; ok && v != nil {
		converted := convertQdrantValue(v)
		if metadata, ok := converted.(map[string]any); ok {
			note.Metadata = metadata
		}
	}

	return note, nil
}

// GlobalConfig操作

// globalConfigID はprojectIDとkeyから決定論的なIDを生成する
// これにより同一projectID+keyの競合を防ぐ
func globalConfigID(projectID, key string) string {
	return fmt.Sprintf("global:%s:%s", projectID, key)
}

// UpsertGlobal はGlobalConfigの新規作成または更新を行う
func (s *QdrantStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// updatedAtがnilの場合は現在時刻を設定
	if config.UpdatedAt == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		config.UpdatedAt = &now
	}

	// projectID + keyから決定論的なIDを生成（競合を防ぐ）
	// クライアントが指定したIDは無視し、常にprojectID+keyからIDを導出する
	config.ID = globalConfigID(config.ProjectID, config.Key)

	// payloadを構築
	payload := make(map[string]*qdrant.Value)
	payload["id"], _ = qdrant.NewValue(config.ID)
	payload["projectId"], _ = qdrant.NewValue(config.ProjectID)
	payload["key"], _ = qdrant.NewValue(config.Key)
	payload["updatedAt"], _ = qdrant.NewValue(*config.UpdatedAt)
	payload["type"], _ = qdrant.NewValue("global_config")

	// Valueをそのまま保存（JSON経由）
	jsonBytes, err := json.Marshal(config.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	var valueAny any
	if err := json.Unmarshal(jsonBytes, &valueAny); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	payload["value"], _ = qdrant.NewValue(valueAny)

	// ダミーベクトル（1次元）
	dummyVector := []float32{1.0}

	// ポイントを追加（Upsert）
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.globalConfigCollection(),
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(hashID(config.ID)),
				Vectors: qdrant.NewVectors(dummyVector...),
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert global config: %w", err)
	}

	return nil
}

// GetGlobal はProjectIDとKeyでGlobalConfigを取得する
func (s *QdrantStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error) {
	if !s.isInitialized() {
		return nil, false, ErrNotInitialized
	}

	// projectID + keyでフィルタ検索
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("projectId", projectID),
			qdrant.NewMatch("key", key),
			qdrant.NewMatch("type", "global_config"),
		},
	}

	scrollResp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.globalConfigCollection(),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(1)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})

	if err != nil {
		return nil, false, fmt.Errorf("failed to scroll global configs: %w", err)
	}

	if len(scrollResp) == 0 {
		return nil, false, nil
	}

	// payloadからGlobalConfigを構築
	config, err := payloadToGlobalConfig(scrollResp[0].Payload)
	if err != nil {
		return nil, false, fmt.Errorf("failed to convert payload to global config: %w", err)
	}

	return config, true, nil
}

// GetGlobalByID はIDでGlobalConfigを取得する
func (s *QdrantStore) GetGlobalByID(ctx context.Context, id string) (*model.GlobalConfig, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// IDでポイントを取得
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.globalConfigCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get global config: %w", err)
	}

	if len(points) == 0 {
		return nil, ErrNotFound
	}

	// payloadからGlobalConfigを構築
	config, err := payloadToGlobalConfig(points[0].Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload to global config: %w", err)
	}

	return config, nil
}

// DeleteGlobalByID はIDでGlobalConfigを削除する
func (s *QdrantStore) DeleteGlobalByID(ctx context.Context, id string) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.globalConfigCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(false),
	})

	if err != nil {
		return fmt.Errorf("failed to check global config existence: %w", err)
	}

	if len(points) == 0 {
		return ErrNotFound
	}

	// ポイントを削除
	_, err = s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.globalConfigCollection(),
		Points:         qdrant.NewPointsSelector(qdrant.NewIDNum(hashID(id))),
	})

	if err != nil {
		return fmt.Errorf("failed to delete global config: %w", err)
	}

	return nil
}

// Group操作

// AddGroup はグループを追加する
func (s *QdrantStore) AddGroup(ctx context.Context, group *model.Group) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// payloadを構築
	payload := make(map[string]*qdrant.Value)
	payload["id"], _ = qdrant.NewValue(group.ID)
	payload["projectId"], _ = qdrant.NewValue(group.ProjectID)
	payload["groupKey"], _ = qdrant.NewValue(group.GroupKey)
	payload["title"], _ = qdrant.NewValue(group.Title)
	payload["description"], _ = qdrant.NewValue(group.Description)
	payload["createdAt"], _ = qdrant.NewValue(group.CreatedAt.Format(time.RFC3339))
	payload["updatedAt"], _ = qdrant.NewValue(group.UpdatedAt.Format(time.RFC3339))
	payload["type"], _ = qdrant.NewValue("group")

	// ダミーベクトル（1次元）
	dummyVector := []float32{1.0}

	// ポイントを追加
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.groupCollection(),
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(hashID(group.ID)),
				Vectors: qdrant.NewVectors(dummyVector...),
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to add group: %w", err)
	}

	return nil
}

// GetGroup はIDでグループを取得する
func (s *QdrantStore) GetGroup(ctx context.Context, id string) (*model.Group, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// IDでポイントを取得
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.groupCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	if len(points) == 0 {
		return nil, ErrNotFound
	}

	// payloadからGroupを構築
	group, err := payloadToGroup(points[0].Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload to group: %w", err)
	}

	return group, nil
}

// GetGroupByKey はProjectIDとGroupKeyでグループを取得する
func (s *QdrantStore) GetGroupByKey(ctx context.Context, projectID, groupKey string) (*model.Group, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// projectID + groupKeyでフィルタ検索
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("projectId", projectID),
			qdrant.NewMatch("groupKey", groupKey),
			qdrant.NewMatch("type", "group"),
		},
	}

	scrollResp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.groupCollection(),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(1)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scroll groups: %w", err)
	}

	if len(scrollResp) == 0 {
		return nil, ErrNotFound
	}

	// payloadからGroupを構築
	group, err := payloadToGroup(scrollResp[0].Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to convert payload to group: %w", err)
	}

	return group, nil
}

// UpdateGroup はグループを更新する
func (s *QdrantStore) UpdateGroup(ctx context.Context, group *model.Group) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.groupCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(group.ID))},
		WithPayload:    qdrant.NewWithPayload(false),
	})

	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	if len(points) == 0 {
		return ErrNotFound
	}

	// payloadを構築
	payload := make(map[string]*qdrant.Value)
	payload["id"], _ = qdrant.NewValue(group.ID)
	payload["projectId"], _ = qdrant.NewValue(group.ProjectID)
	payload["groupKey"], _ = qdrant.NewValue(group.GroupKey)
	payload["title"], _ = qdrant.NewValue(group.Title)
	payload["description"], _ = qdrant.NewValue(group.Description)
	payload["createdAt"], _ = qdrant.NewValue(group.CreatedAt.Format(time.RFC3339))
	payload["updatedAt"], _ = qdrant.NewValue(group.UpdatedAt.Format(time.RFC3339))
	payload["type"], _ = qdrant.NewValue("group")

	// ダミーベクトル（1次元）
	dummyVector := []float32{1.0}

	// ポイントを更新（Upsert）
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.groupCollection(),
		Wait:           qdrant.PtrOf(true),
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(hashID(group.ID)),
				Vectors: qdrant.NewVectors(dummyVector...),
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// DeleteGroup はグループを削除する
func (s *QdrantStore) DeleteGroup(ctx context.Context, id string) error {
	if !s.isInitialized() {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.groupCollection(),
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(hashID(id))},
		WithPayload:    qdrant.NewWithPayload(false),
	})

	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}

	if len(points) == 0 {
		return ErrNotFound
	}

	// ポイントを削除
	_, err = s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.groupCollection(),
		Points:         qdrant.NewPointsSelector(qdrant.NewIDNum(hashID(id))),
	})

	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// ListGroups はプロジェクト内のグループ一覧を取得する
func (s *QdrantStore) ListGroups(ctx context.Context, projectID string) ([]*model.Group, error) {
	if !s.isInitialized() {
		return nil, ErrNotInitialized
	}

	// projectIDでフィルタ検索
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("projectId", projectID),
			qdrant.NewMatch("type", "group"),
		},
	}

	scrollResp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.groupCollection(),
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(1000)), // 十分大きな値
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scroll groups: %w", err)
	}

	// payloadからGroupに変換
	var groups []*model.Group
	for _, point := range scrollResp {
		group, err := payloadToGroup(point.Payload)
		if err != nil {
			continue
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// payloadToGlobalConfig はQdrantのpayloadからGlobalConfigを構築する
// convertQdrantValue はQdrantのValueをGoの値に変換する
func convertQdrantValue(v *qdrant.Value) any {
	if v == nil {
		return nil
	}
	switch v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return v.GetStringValue()
	case *qdrant.Value_IntegerValue:
		return v.GetIntegerValue()
	case *qdrant.Value_DoubleValue:
		return v.GetDoubleValue()
	case *qdrant.Value_BoolValue:
		return v.GetBoolValue()
	case *qdrant.Value_StructValue:
		structVal := v.GetStructValue()
		if structVal != nil && structVal.Fields != nil {
			result := make(map[string]any)
			for key, field := range structVal.Fields {
				result[key] = convertQdrantValue(field)
			}
			return result
		}
	case *qdrant.Value_ListValue:
		listVal := v.GetListValue()
		if listVal != nil {
			var values []any
			for _, item := range listVal.Values {
				values = append(values, convertQdrantValue(item))
			}
			return values
		}
	}
	return nil
}

func payloadToGlobalConfig(payload map[string]*qdrant.Value) (*model.GlobalConfig, error) {
	config := &model.GlobalConfig{}

	if v, ok := payload["id"]; ok && v.GetStringValue() != "" {
		config.ID = v.GetStringValue()
	}
	if v, ok := payload["projectId"]; ok && v.GetStringValue() != "" {
		config.ProjectID = v.GetStringValue()
	}
	if v, ok := payload["key"]; ok && v.GetStringValue() != "" {
		config.Key = v.GetStringValue()
	}
	if v, ok := payload["updatedAt"]; ok && v.GetStringValue() != "" {
		updatedAt := v.GetStringValue()
		config.UpdatedAt = &updatedAt
	}

	// Valueの取得（型スイッチで正確に判定）
	if v, ok := payload["value"]; ok && v != nil {
		switch v.Kind.(type) {
		case *qdrant.Value_StringValue:
			config.Value = v.GetStringValue()
		case *qdrant.Value_IntegerValue:
			config.Value = v.GetIntegerValue()
		case *qdrant.Value_DoubleValue:
			config.Value = v.GetDoubleValue()
		case *qdrant.Value_BoolValue:
			config.Value = v.GetBoolValue()
		case *qdrant.Value_StructValue:
			structVal := v.GetStructValue()
			if structVal != nil && structVal.Fields != nil {
				// structValueをmap[string]anyに変換
				result := make(map[string]any)
				for key, field := range structVal.Fields {
					result[key] = convertQdrantValue(field)
				}
				config.Value = result
			}
		case *qdrant.Value_ListValue:
			listVal := v.GetListValue()
			if listVal != nil {
				var values []any
				for _, item := range listVal.Values {
					values = append(values, convertQdrantValue(item))
				}
				config.Value = values
			}
		}
	}

	return config, nil
}

// payloadToGroup はQdrantのpayloadからGroupを構築する
func payloadToGroup(payload map[string]*qdrant.Value) (*model.Group, error) {
	group := &model.Group{}

	if v, ok := payload["id"]; ok && v.GetStringValue() != "" {
		group.ID = v.GetStringValue()
	}
	if v, ok := payload["projectId"]; ok && v.GetStringValue() != "" {
		group.ProjectID = v.GetStringValue()
	}
	if v, ok := payload["groupKey"]; ok && v.GetStringValue() != "" {
		group.GroupKey = v.GetStringValue()
	}
	if v, ok := payload["title"]; ok && v.GetStringValue() != "" {
		group.Title = v.GetStringValue()
	}
	if v, ok := payload["description"]; ok && v.GetStringValue() != "" {
		group.Description = v.GetStringValue()
	}
	if v, ok := payload["createdAt"]; ok && v.GetStringValue() != "" {
		if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil {
			group.CreatedAt = t
		}
	}
	if v, ok := payload["updatedAt"]; ok && v.GetStringValue() != "" {
		if t, err := time.Parse(time.RFC3339, v.GetStringValue()); err == nil {
			group.UpdatedAt = t
		}
	}

	return group, nil
}
