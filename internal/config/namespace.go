package config

import (
	"fmt"
	"strconv"
	"strings"
)

// GenerateNamespace はembedder設定からnamespaceを生成する
// 形式: "{provider}:{model}:{dim}"
// dimが0（未設定）の場合も0をそのまま使用（仕様準拠）
func GenerateNamespace(provider, model string, dim int) string {
	return fmt.Sprintf("%s:%s:%d", provider, model, dim)
}

// ParseNamespace はnamespaceをprovider, model, dimに分解する
// 不正な形式の場合はエラーを返す
// dimは0以上の整数であること（負数の場合はエラー）
func ParseNamespace(namespace string) (provider, model string, dim int, err error) {
	parts := strings.Split(namespace, ":")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("invalid namespace format: expected 'provider:model:dim', got %q", namespace)
	}

	provider = parts[0]
	model = parts[1]

	dim, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid dim in namespace %q: %w", namespace, err)
	}

	if dim < 0 {
		return "", "", 0, fmt.Errorf("invalid dim in namespace %q: dim must be non-negative, got %d", namespace, dim)
	}

	return provider, model, dim, nil
}
