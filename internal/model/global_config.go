package model

import (
	"fmt"
	"regexp"
	"strings"
)

// GlobalConfig はプロジェクト単位のグローバル設定を表す
type GlobalConfig struct {
	ID        string  `json:"id"`        // UUID形式
	ProjectID string  `json:"projectId"` // 正規化済みパス
	Key       string  `json:"key"`       // "global."プレフィックス必須
	Value     any     `json:"value"`     // 任意のJSON値
	UpdatedAt *string `json:"updatedAt"` // ISO8601 UTC形式、nullable（nullならサーバー側で現在時刻設定）
}

// 標準キー定数
const (
	GlobalKeyEmbedderProvider   = "global.memory.embedder.provider"
	GlobalKeyEmbedderModel      = "global.memory.embedder.model"
	GlobalKeyGroupDefaults      = "global.memory.groupDefaults"
	GlobalKeyProjectConventions = "global.project.conventions"
)

var globalKeyPattern = regexp.MustCompile(`^global\.[a-zA-Z0-9._-]+$`)

// Validate はGlobalConfigのバリデーションを実行する
func (g *GlobalConfig) Validate() error {
	if err := ValidateGlobalKey(g.Key); err != nil {
		return err
	}

	return nil
}

// ValidateGlobalKey はグローバルキーのバリデーションを実行する
func ValidateGlobalKey(key string) error {
	if key == "" {
		return fmt.Errorf("Key must not be empty")
	}

	if !strings.HasPrefix(key, "global.") {
		return fmt.Errorf("Key must have 'global.' prefix, got %q", key)
	}

	// "global."の後に何か文字が必要
	if len(key) <= len("global.") {
		return fmt.Errorf("Key must have content after 'global.' prefix, got %q", key)
	}

	// 全体パターンチェック（スペースや特殊文字を除外）
	if !globalKeyPattern.MatchString(key) {
		return fmt.Errorf("Key must match pattern ^global\\.[a-zA-Z0-9._-]+$, got %q", key)
	}

	return nil
}
