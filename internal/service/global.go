package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/store"
	"github.com/google/uuid"
)

// globalService はGlobalServiceの実装
type globalService struct {
	store     store.Store
	namespace string
}

// NewGlobalService はGlobalServiceの新しいインスタンスを作成
func NewGlobalService(s store.Store, namespace string) GlobalService {
	return &globalService{
		store:     s,
		namespace: namespace,
	}
}

// UpsertGlobal はグローバル設定をupsertする
func (s *globalService) UpsertGlobal(ctx context.Context, req *UpsertGlobalRequest) (*UpsertGlobalResponse, error) {
	// バリデーション
	if req.ProjectID == "" {
		return nil, ErrProjectIDRequired
	}

	// keyのプレフィックス検証
	if !strings.HasPrefix(req.Key, "global.") {
		return nil, ErrInvalidGlobalKey
	}

	// keyの完全性検証
	if err := model.ValidateGlobalKey(req.Key); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGlobalKey, err)
	}

	// UpdatedAtが指定されている場合はISO8601形式を検証
	if req.UpdatedAt != nil {
		if _, err := time.Parse(time.RFC3339, *req.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidTimeFormat, err)
		}
	}

	// updatedAtの生成
	updatedAt := req.UpdatedAt
	if updatedAt == nil {
		// RFC3339は秒までの精度なので、ナノ秒がある場合は次の秒に切り上げ
		now := time.Now().UTC()
		if now.Nanosecond() > 0 {
			now = now.Truncate(time.Second).Add(time.Second)
		}
		nowStr := now.Format(time.RFC3339)
		updatedAt = &nowStr
	}

	// 既存のエントリを確認してIDを取得または生成
	existing, found, err := s.store.GetGlobal(ctx, req.ProjectID, req.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing global: %w", err)
	}

	id := uuid.New().String()
	if found {
		id = existing.ID
	}

	// GlobalConfigモデルの作成
	globalConfig := &model.GlobalConfig{
		ID:        id,
		ProjectID: req.ProjectID,
		Key:       req.Key,
		Value:     req.Value,
		UpdatedAt: updatedAt,
	}

	// Storeにupsert
	if err := s.store.UpsertGlobal(ctx, globalConfig); err != nil {
		return nil, fmt.Errorf("failed to upsert global: %w", err)
	}

	return &UpsertGlobalResponse{
		OK:        true,
		ID:        id,
		Namespace: s.namespace,
	}, nil
}

// GetGlobal はグローバル設定を取得する
func (s *globalService) GetGlobal(ctx context.Context, projectID, key string) (*GetGlobalResponse, error) {
	// バリデーション
	if projectID == "" {
		return nil, ErrProjectIDRequired
	}

	// keyのプレフィックス検証（空文字は not found として扱う）
	if key != "" && !strings.HasPrefix(key, "global.") {
		return nil, ErrInvalidGlobalKey
	}

	// Storeから取得
	globalConfig, found, err := s.store.GetGlobal(ctx, projectID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get global: %w", err)
	}

	if !found {
		return &GetGlobalResponse{
			Namespace: s.namespace,
			Found:     false,
			ID:        nil,
			Value:     nil,
			UpdatedAt: nil,
		}, nil
	}

	return &GetGlobalResponse{
		Namespace: s.namespace,
		Found:     true,
		ID:        &globalConfig.ID,
		Value:     globalConfig.Value,
		UpdatedAt: globalConfig.UpdatedAt,
	}, nil
}
