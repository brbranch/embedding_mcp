package service

import (
	"context"
	"errors"
	"regexp"
)

// NoteService はノートのCRUD + 検索を提供
type NoteService interface {
	AddNote(ctx context.Context, req *AddNoteRequest) (*AddNoteResponse, error)
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	Get(ctx context.Context, id string) (*GetResponse, error)
	Update(ctx context.Context, req *UpdateRequest) error
	Delete(ctx context.Context, id string) error
	ListRecent(ctx context.Context, req *ListRecentRequest) (*ListRecentResponse, error)
}

// ConfigService は設定の取得・変更を提供
type ConfigService interface {
	GetConfig(ctx context.Context) (*GetConfigResponse, error)
	SetConfig(ctx context.Context, req *SetConfigRequest) (*SetConfigResponse, error)
}

// GlobalService はグローバル設定のUpsert/Get/Deleteを提供
type GlobalService interface {
	UpsertGlobal(ctx context.Context, req *UpsertGlobalRequest) (*UpsertGlobalResponse, error)
	GetGlobal(ctx context.Context, projectID, key string) (*GetGlobalResponse, error)
	DeleteByID(ctx context.Context, id string) error
}

// GroupService はグループのCRUDを提供
type GroupService interface {
	CreateGroup(ctx context.Context, req *CreateGroupRequest) (*CreateGroupResponse, error)
	GetGroup(ctx context.Context, id string) (*GetGroupResponse, error)
	UpdateGroup(ctx context.Context, req *UpdateGroupRequest) error
	DeleteGroup(ctx context.Context, id string) error
	ListGroups(ctx context.Context, projectID string) (*ListGroupsResponse, error)
}

// エラー定義
var (
	ErrNoteNotFound         = errors.New("note not found")
	ErrGlobalConfigNotFound = errors.New("global config not found")
	ErrGroupNotFound        = errors.New("group not found")
	ErrInvalidGlobalKey     = errors.New("key must start with 'global.'")
	ErrProjectIDRequired    = errors.New("projectId is required")
	ErrGroupIDRequired      = errors.New("groupId is required")
	ErrGroupKeyRequired     = errors.New("groupKey is required")
	ErrTitleRequired        = errors.New("title is required")
	ErrInvalidGroupID       = errors.New("groupId contains invalid characters")
	ErrInvalidGroupKey      = errors.New("groupKey contains invalid characters or is reserved")
	ErrGroupKeyExists       = errors.New("groupKey already exists in this project")
	ErrTextRequired         = errors.New("text is required")
	ErrQueryRequired        = errors.New("query is required")
	ErrIDRequired           = errors.New("id is required")
	ErrInvalidTimeFormat    = errors.New("invalid time format (expected ISO8601 UTC)")
)

// groupIDRegex はgroupIdの文字制約を検証
// 許容: 英数字、ハイフン、アンダースコア
var groupIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateGroupID はgroupIdの文字制約を検証
func ValidateGroupID(groupID string) error {
	if groupID == "" {
		return ErrGroupIDRequired
	}
	if !groupIDRegex.MatchString(groupID) {
		return ErrInvalidGroupID
	}
	return nil
}
