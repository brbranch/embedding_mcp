package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// Manager は設定の読み書きを管理する
type Manager struct {
	mu         sync.RWMutex
	config     *model.Config
	configPath string
}

// NewManager は新しいManagerを作成する
// configPathが空文字の場合、デフォルトパス（~/.local-mcp-memory/config.json）を使用
func NewManager(configPath string) (*Manager, error) {
	// configPathが空の場合はデフォルトパスを使用
	if configPath == "" {
		defaultPath, err := GetDefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default config path: %w", err)
		}
		configPath = defaultPath
	}

	// デフォルトのデータディレクトリを取得
	dataDir, err := GetDefaultDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get default data dir: %w", err)
	}

	// デフォルト設定で初期化
	config := DefaultConfig(configPath, dataDir)

	return &Manager{
		config:     config,
		configPath: configPath,
	}, nil
}

// Load は設定ファイルを読み込む
// ファイルが存在しない場合はデフォルト設定を使用（エラーなし）
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// ファイルが存在しない場合はデフォルト設定を使う
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// デフォルト設定は既に設定されているのでそのまま返す
		return nil
	}

	// ファイルを読み込み
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// JSONをパース
	var config model.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	m.config = &config
	return nil
}

// Save は設定ファイルを保存する
func (m *Manager) Save() error {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	// ディレクトリを作成
	configDir := filepath.Dir(m.configPath)
	if err := EnsureDir(configDir); err != nil {
		return err
	}

	// JSONにエンコード
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 一時ファイルに書き込み（atomicな保存のため）
	tmpFile := m.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	// 一時ファイルを本番ファイルにリネーム
	if err := os.Rename(tmpFile, m.configPath); err != nil {
		os.Remove(tmpFile) // クリーンアップ
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// GetConfig は現在の設定を返す（ロード済みの場合）
func (m *Manager) GetConfig() *model.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetConfigPath は設定ファイルパスを返す
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// UpdateEmbedder はembedder設定のみを更新する
// 他のフィールド（store/paths）は変更不可
func (m *Manager) UpdateEmbedder(embedder *model.EmbedderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 既存の設定を保持しつつ、embedderのみ更新
	if embedder.Provider != "" {
		m.config.Embedder.Provider = embedder.Provider
	}
	if embedder.Model != "" {
		m.config.Embedder.Model = embedder.Model
	}
	if embedder.Dim != 0 {
		m.config.Embedder.Dim = embedder.Dim
	}
	if embedder.BaseURL != nil {
		m.config.Embedder.BaseURL = embedder.BaseURL
	}
	if embedder.APIKey != nil {
		m.config.Embedder.APIKey = embedder.APIKey
	}

	return nil
}

// UpdateDim は埋め込み次元を更新する（初回埋め込み時に使用）
func (m *Manager) UpdateDim(dim int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Embedder.Dim = dim
	return nil
}

// NewManagerWithConfig は指定した設定でManagerを作成する（テスト用）
func NewManagerWithConfig(cfg *model.Config) *Manager {
	return &Manager{
		config:     cfg,
		configPath: "", // テスト用なので空
	}
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig(configPath, dataDir string) *model.Config {
	return &model.Config{
		TransportDefaults: model.TransportDefaults{
			DefaultTransport: model.TransportStdio,
		},
		Embedder: model.EmbedderConfig{
			Provider: model.ProviderOpenAI,
			Model:    "text-embedding-3-small",
			Dim:      0, // 初回埋め込み時に設定
			BaseURL:  nil,
			APIKey:   nil,
		},
		Store: model.StoreConfig{
			Type: model.StoreTypeChroma,
			Path: nil,
			URL:  nil,
		},
		Paths: model.PathsConfig{
			ConfigPath: configPath,
			DataDir:    dataDir,
		},
	}
}
