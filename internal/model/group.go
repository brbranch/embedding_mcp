package model

import (
	"fmt"
	"regexp"
	"time"
)

// Group はグループを表す（内部データモデル）
type Group struct {
	ID          string    `json:"id"`          // UUID形式
	ProjectID   string    `json:"projectId"`   // 正規化済みパス
	GroupKey    string    `json:"groupKey"`    // 英数字、-、_のみ。"global"は予約値
	Title       string    `json:"title"`       // 必須
	Description string    `json:"description"` // 空文字列可
	CreatedAt   time.Time `json:"createdAt"`   // UTC
	UpdatedAt   time.Time `json:"updatedAt"`   // UTC
}

var groupKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate はGroupのバリデーションを実行する
func (g *Group) Validate() error {
	if g.ID == "" {
		return fmt.Errorf("ID must not be empty")
	}

	if g.ProjectID == "" {
		return fmt.Errorf("ProjectID must not be empty")
	}

	if err := ValidateGroupKeyForCreate(g.GroupKey); err != nil {
		return err
	}

	if g.Title == "" {
		return fmt.Errorf("Title must not be empty")
	}

	return nil
}

// ValidateGroupKeyForCreate はGroupKeyのバリデーションを実行する
// "global" は予約されているため登録禁止
func ValidateGroupKeyForCreate(groupKey string) error {
	if groupKey == "" {
		return fmt.Errorf("GroupKey must not be empty")
	}

	if groupKey == "global" {
		return fmt.Errorf("GroupKey 'global' is reserved and cannot be used")
	}

	if !groupKeyPattern.MatchString(groupKey) {
		return fmt.Errorf("GroupKey must match pattern ^[a-zA-Z0-9_-]+$, got %q", groupKey)
	}

	return nil
}
