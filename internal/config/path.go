package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultConfigDir はデフォルトの設定ディレクトリ名
	DefaultConfigDir = ".local-mcp-memory"
	// DefaultConfigFile はデフォルトの設定ファイル名
	DefaultConfigFile = "config.json"
	// DefaultDataSubDir はデフォルトのデータサブディレクトリ名
	DefaultDataSubDir = "data"
)

// CanonicalizeProjectID はprojectIdを正規化する
// 1. "~" をホームディレクトリに展開
// 2. 絶対パス化（filepath.Abs）
// 3. シンボリックリンク解決（filepath.EvalSymlinks）※失敗時はAbsまで
func CanonicalizeProjectID(projectID string) (string, error) {
	// 1. "~" をホームに展開
	expanded, err := ExpandTilde(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to expand tilde: %w", err)
	}

	// 2. 絶対パス化
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 3. シンボリックリンク解決（失敗時はAbsまで）
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// EvalSymlinks が失敗した場合は Abs の結果を使用
		return abs, nil
	}

	return canonical, nil
}

// ExpandTilde は"~"をホームディレクトリに展開する
// "~/" で始まる場合のみ展開し、それ以外はそのまま返す
func ExpandTilde(path string) (string, error) {
	// "~" のみ、または "~/" で始まる場合のみ展開
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}

	// それ以外（"~user" など）はそのまま返す
	return path, nil
}

// GetDefaultConfigPath はデフォルトの設定ファイルパスを返す
// ~/.local-mcp-memory/config.json
func GetDefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile), nil
}

// GetDefaultDataDir はデフォルトのデータディレクトリを返す
// ~/.local-mcp-memory/data
func GetDefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultDataSubDir), nil
}

// EnsureDir はディレクトリが存在することを確認し、なければ作成する
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}
