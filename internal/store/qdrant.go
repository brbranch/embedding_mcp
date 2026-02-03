package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantStore はQdrantを使用したStore実装
type QdrantStore struct {
	client      *qdrant.Client
	url         string
	namespace   string
	initialized bool
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

// Initialize はストアを初期化する
func (s *QdrantStore) Initialize(ctx context.Context, namespace string) error {
	if s.client == nil {
		return ErrConnectionFailed
	}

	// コレクション存在確認
	exists, err := s.client.CollectionExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	// コレクションが存在しない場合は作成
	if !exists {
		err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: namespace,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1536, // デフォルトの埋め込み次元数
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}

	s.namespace = namespace
	s.initialized = true
	return nil
}

// Close はストアをクローズする
func (s *QdrantStore) Close() error {
	s.initialized = false
	if s.client != nil {
		s.client.Close()
	}
	return nil
}

// AddNote はノートを追加する
func (s *QdrantStore) AddNote(ctx context.Context, note *model.Note, embedding []float32) error {
	if !s.initialized {
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
		CollectionName: s.namespace,
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
	if !s.initialized {
		return nil, ErrNotInitialized
	}

	// IDでポイントを取得
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.namespace,
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
	if !s.initialized {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.namespace,
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
		CollectionName: s.namespace,
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
	if !s.initialized {
		return ErrNotInitialized
	}

	// 存在確認
	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.namespace,
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
		CollectionName: s.namespace,
		Points:         qdrant.NewPointsSelector(qdrant.NewIDNum(hashID(id))),
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	return nil
}

// Search はベクトル検索を実行する
func (s *QdrantStore) Search(ctx context.Context, embedding []float32, opts SearchOptions) ([]SearchResult, error) {
	if !s.initialized {
		return nil, ErrNotInitialized
	}

	// フィルタを構築
	filter := buildSearchFilter(opts)

	// Qdrantで検索実行
	queryResp, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.namespace,
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
	if !s.initialized {
		return nil, ErrNotInitialized
	}

	// フィルタを構築
	filter := buildListFilter(opts)

	// Qdrantで全ポイントを取得
	scrollResp, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: s.namespace,
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(opts.Limit * 2)), // 余裕を持って取得
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

// Helper functions

// hashID は文字列IDを数値IDに変換する（簡易実装）
func hashID(id string) uint64 {
	var hash uint64 = 0
	for i := 0; i < len(id); i++ {
		hash = hash*31 + uint64(id[i])
	}
	return hash
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

	// metadataの取得
	if v, ok := payload["metadata"]; ok && v.GetStructValue() != nil {
		metadata := make(map[string]any)
		structVal := v.GetStructValue()
		if structVal != nil && structVal.Fields != nil {
			// structValueをJSONに変換してから戻す
			jsonBytes, err := json.Marshal(structVal.Fields)
			if err == nil {
				json.Unmarshal(jsonBytes, &metadata)
				note.Metadata = metadata
			}
		}
	}

	return note, nil
}

// GlobalConfig操作（スタブ）

// UpsertGlobal はGlobalConfigの新規作成または更新を行う（未実装）
func (s *QdrantStore) UpsertGlobal(ctx context.Context, config *model.GlobalConfig) error {
	return fmt.Errorf("UpsertGlobal is not implemented yet")
}

// GetGlobal はProjectIDとKeyでGlobalConfigを取得する（未実装）
func (s *QdrantStore) GetGlobal(ctx context.Context, projectID, key string) (*model.GlobalConfig, bool, error) {
	return nil, false, fmt.Errorf("GetGlobal is not implemented yet")
}

// GetGlobalByID はIDでGlobalConfigを取得する（未実装）
func (s *QdrantStore) GetGlobalByID(ctx context.Context, id string) (*model.GlobalConfig, error) {
	return nil, fmt.Errorf("GetGlobalByID is not implemented yet")
}

// DeleteGlobalByID はIDでGlobalConfigを削除する（未実装）
func (s *QdrantStore) DeleteGlobalByID(ctx context.Context, id string) error {
	return fmt.Errorf("DeleteGlobalByID is not implemented yet")
}

// Group操作（スタブ）

// AddGroup はグループを追加する（未実装）
func (s *QdrantStore) AddGroup(ctx context.Context, group *model.Group) error {
	return fmt.Errorf("AddGroup is not implemented yet")
}

// GetGroup はIDでグループを取得する（未実装）
func (s *QdrantStore) GetGroup(ctx context.Context, id string) (*model.Group, error) {
	return nil, fmt.Errorf("GetGroup is not implemented yet")
}

// GetGroupByKey はProjectIDとGroupKeyでグループを取得する（未実装）
func (s *QdrantStore) GetGroupByKey(ctx context.Context, projectID, groupKey string) (*model.Group, error) {
	return nil, fmt.Errorf("GetGroupByKey is not implemented yet")
}

// UpdateGroup はグループを更新する（未実装）
func (s *QdrantStore) UpdateGroup(ctx context.Context, group *model.Group) error {
	return fmt.Errorf("UpdateGroup is not implemented yet")
}

// DeleteGroup はグループを削除する（未実装）
func (s *QdrantStore) DeleteGroup(ctx context.Context, id string) error {
	return fmt.Errorf("DeleteGroup is not implemented yet")
}

// ListGroups はプロジェクト内のグループ一覧を取得する（未実装）
func (s *QdrantStore) ListGroups(ctx context.Context, projectID string) ([]*model.Group, error) {
	return nil, fmt.Errorf("ListGroups is not implemented yet")
}
