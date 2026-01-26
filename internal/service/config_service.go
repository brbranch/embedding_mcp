package service

import (
	"context"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/model"
)

// configService はConfigServiceの実装
type configService struct {
	manager *config.Manager
}

// NewConfigService はConfigServiceの新しいインスタンスを作成
func NewConfigService(mgr *config.Manager) ConfigService {
	return &configService{
		manager: mgr,
	}
}

// GetConfig は現在の設定を取得する
func (s *configService) GetConfig(ctx context.Context) (*GetConfigResponse, error) {
	cfg := s.manager.GetConfig()

	return &GetConfigResponse{
		TransportDefaults: cfg.TransportDefaults,
		Embedder:          cfg.Embedder,
		Store:             cfg.Store,
		Paths:             cfg.Paths,
	}, nil
}

// SetConfig は設定を変更する（embedderのみ変更可能）
func (s *configService) SetConfig(ctx context.Context, req *SetConfigRequest) (*SetConfigResponse, error) {
	if req.Embedder == nil {
		// 変更がない場合は現在のnamespaceを返す
		cfg := s.manager.GetConfig()
		namespace := config.GenerateNamespace(cfg.Embedder.Provider, cfg.Embedder.Model, cfg.Embedder.Dim)
		return &SetConfigResponse{
			OK:                 true,
			EffectiveNamespace: namespace,
		}, nil
	}

	// 現在の設定を取得
	cfg := s.manager.GetConfig()

	// provider/model変更時はdimをリセット
	dimReset := false
	if req.Embedder.Provider != nil && *req.Embedder.Provider != cfg.Embedder.Provider {
		dimReset = true
	}
	if req.Embedder.Model != nil && *req.Embedder.Model != cfg.Embedder.Model {
		dimReset = true
	}

	// パッチを適用
	updatedEmbedder := &model.EmbedderConfig{
		Provider: cfg.Embedder.Provider,
		Model:    cfg.Embedder.Model,
		Dim:      cfg.Embedder.Dim,
		BaseURL:  cfg.Embedder.BaseURL,
		APIKey:   cfg.Embedder.APIKey,
	}

	if req.Embedder.Provider != nil {
		updatedEmbedder.Provider = *req.Embedder.Provider
	}
	if req.Embedder.Model != nil {
		updatedEmbedder.Model = *req.Embedder.Model
	}
	if req.Embedder.BaseURL != nil {
		updatedEmbedder.BaseURL = req.Embedder.BaseURL
	}
	if req.Embedder.APIKey != nil {
		updatedEmbedder.APIKey = req.Embedder.APIKey
	}

	// dimリセットが必要な場合
	if dimReset {
		updatedEmbedder.Dim = 0
	}

	// 設定を更新
	if err := s.manager.UpdateEmbedder(updatedEmbedder); err != nil {
		return nil, err
	}

	// dimリセットが必要な場合は再度0に設定
	if dimReset {
		if err := s.manager.UpdateDim(0); err != nil {
			return nil, err
		}
		updatedEmbedder.Dim = 0
	}

	// 新しいnamespaceを生成
	namespace := config.GenerateNamespace(updatedEmbedder.Provider, updatedEmbedder.Model, updatedEmbedder.Dim)

	return &SetConfigResponse{
		OK:                 true,
		EffectiveNamespace: namespace,
	}, nil
}
