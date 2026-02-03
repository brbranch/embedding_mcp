package service

import (
	"context"
	"fmt"
	"time"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/store"
	"github.com/google/uuid"
)

// groupService はGroupServiceの実装
type groupService struct {
	store     store.Store
	namespace string
}

// NewGroupService はGroupServiceの新しいインスタンスを作成
func NewGroupService(s store.Store, namespace string) GroupService {
	return &groupService{
		store:     s,
		namespace: namespace,
	}
}

// CreateGroup はグループを作成する
func (s *groupService) CreateGroup(ctx context.Context, req *CreateGroupRequest) (*CreateGroupResponse, error) {
	// バリデーション
	if req.ProjectID == "" {
		return nil, ErrProjectIDRequired
	}
	if req.GroupKey == "" {
		return nil, ErrGroupKeyRequired
	}
	if req.Title == "" {
		return nil, ErrTitleRequired
	}

	// GroupKeyのバリデーション（"global" 禁止 + フォーマット）
	if err := model.ValidateGroupKeyForCreate(req.GroupKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGroupKey, err)
	}

	// ProjectIDを正規化
	canonicalProjectID, err := config.CanonicalizeProjectID(req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize projectId: %w", err)
	}

	// 重複チェック（同一プロジェクト内でgroupKeyが一意）
	_, err = s.store.GetGroupByKey(ctx, canonicalProjectID, req.GroupKey)
	if err == nil {
		// 既に存在する
		return nil, ErrGroupKeyExists
	}
	if err != store.ErrNotFound {
		return nil, fmt.Errorf("failed to check group key uniqueness: %w", err)
	}

	// IDと時刻の生成
	id := uuid.New().String()
	now := time.Now().UTC()

	// Groupモデルの作成
	group := &model.Group{
		ID:          id,
		ProjectID:   canonicalProjectID,
		GroupKey:    req.GroupKey,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Storeに保存
	if err := s.store.AddGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to add group to store: %w", err)
	}

	return &CreateGroupResponse{
		ID:        id,
		Namespace: s.namespace,
	}, nil
}

// GetGroup はグループを取得する
func (s *groupService) GetGroup(ctx context.Context, id string) (*GetGroupResponse, error) {
	// バリデーション
	if id == "" {
		return nil, ErrIDRequired
	}

	// Storeから取得
	group, err := s.store.GetGroup(ctx, id)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return &GetGroupResponse{
		ID:          group.ID,
		ProjectID:   group.ProjectID,
		GroupKey:    group.GroupKey,
		Title:       group.Title,
		Description: group.Description,
		CreatedAt:   group.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   group.UpdatedAt.Format(time.RFC3339),
		Namespace:   s.namespace,
	}, nil
}

// UpdateGroup はグループを更新する
func (s *groupService) UpdateGroup(ctx context.Context, req *UpdateGroupRequest) error {
	// バリデーション
	if req.ID == "" {
		return ErrIDRequired
	}

	// 既存グループを取得
	group, err := s.store.GetGroup(ctx, req.ID)
	if err != nil {
		if err == store.ErrNotFound {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	// パッチを適用
	if req.Patch.Title != nil {
		group.Title = *req.Patch.Title
	}
	if req.Patch.Description != nil {
		group.Description = *req.Patch.Description
	}

	// UpdatedAtを更新
	group.UpdatedAt = time.Now().UTC()

	// Storeを更新
	if err := s.store.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// DeleteGroup はグループを削除する
func (s *groupService) DeleteGroup(ctx context.Context, id string) error {
	// バリデーション
	if id == "" {
		return ErrIDRequired
	}

	// Storeから削除
	if err := s.store.DeleteGroup(ctx, id); err != nil {
		if err == store.ErrNotFound {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// ListGroups はプロジェクト内の全グループを取得する
func (s *groupService) ListGroups(ctx context.Context, projectID string) (*ListGroupsResponse, error) {
	// バリデーション
	if projectID == "" {
		return nil, ErrProjectIDRequired
	}

	// ProjectIDを正規化
	canonicalProjectID, err := config.CanonicalizeProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize projectId: %w", err)
	}

	// Storeから取得
	groups, err := s.store.ListGroups(ctx, canonicalProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// レスポンスの構築
	items := make([]ListGroupItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, ListGroupItem{
			ID:          group.ID,
			ProjectID:   group.ProjectID,
			GroupKey:    group.GroupKey,
			Title:       group.Title,
			Description: group.Description,
			CreatedAt:   group.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   group.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &ListGroupsResponse{
		Namespace: s.namespace,
		Groups:    items,
	}, nil
}
