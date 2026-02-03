package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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

// Helper functions

// hashID は文字列IDを数値IDに変換する（簡易実装）
func hashID(id string) uint64 {
	var hash uint64 = 0
	for i := 0; i < len(id); i++ {
		hash = hash*31 + uint64(id[i])
	}
	return hash
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
