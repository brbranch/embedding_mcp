package model

import (
	"fmt"
	"regexp"
)

// Note はメモリノートを表す（内部データモデル）
// 注: レスポンス用のDTO（namespaceを含む）はjsonrpc層で別途定義する
type Note struct {
	ID        string         `json:"id"`                  // UUID形式
	ProjectID string         `json:"projectId"`           // 正規化済みパス
	GroupID   string         `json:"groupId"`             // 英数字、-、_のみ。"global"は予約値
	Title     *string        `json:"title"`               // nullable
	Text      string         `json:"text"`                // 必須
	Tags      []string       `json:"tags"`                // 空配列可
	Source    *string        `json:"source"`              // nullable
	CreatedAt *string        `json:"createdAt"`           // ISO8601 UTC形式、nullable（nullならサーバー側で現在時刻設定）
	Metadata  map[string]any `json:"metadata,omitempty"`  // nullable（JSON null許容）、省略可
}

var groupIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate はNoteのバリデーションを実行する
func (n *Note) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("ID must not be empty")
	}

	if n.ProjectID == "" {
		return fmt.Errorf("ProjectID must not be empty")
	}

	if err := ValidateGroupID(n.GroupID); err != nil {
		return err
	}

	if n.Text == "" {
		return fmt.Errorf("Text must not be empty")
	}

	return nil
}

// ValidateGroupID はGroupIDのバリデーションを実行する
func ValidateGroupID(groupID string) error {
	if groupID == "" {
		return fmt.Errorf("GroupID must not be empty")
	}

	if !groupIDPattern.MatchString(groupID) {
		return fmt.Errorf("GroupID must match pattern ^[a-zA-Z0-9_-]+$, got %q", groupID)
	}

	return nil
}
