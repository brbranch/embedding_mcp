package embedder

import "github.com/brbranch/embedding_mcp/internal/model"

// NewEmbedder はEmbedderConfigからEmbedderを作成
func NewEmbedder(cfg *model.EmbedderConfig, envAPIKey string, dimUpdater DimUpdater) (Embedder, error) {
	switch cfg.Provider {
	case "openai":
		// APIKey解決: cfg.APIKey > envAPIKey
		apiKey := envAPIKey
		if cfg.APIKey != nil && *cfg.APIKey != "" {
			apiKey = *cfg.APIKey
		}

		opts := []OpenAIOption{}

		// BaseURL適用
		if cfg.BaseURL != nil && *cfg.BaseURL != "" {
			opts = append(opts, WithBaseURL(*cfg.BaseURL))
		}

		// Model適用
		if cfg.Model != "" {
			opts = append(opts, WithModel(cfg.Model))
		}

		// Dim適用
		if cfg.Dim > 0 {
			opts = append(opts, WithDim(cfg.Dim))
		}

		// DimUpdater適用
		if dimUpdater != nil {
			opts = append(opts, WithDimUpdater(dimUpdater))
		}

		return NewOpenAIEmbedder(apiKey, opts...)

	case "ollama":
		baseURL := DefaultOllamaBaseURL
		if cfg.BaseURL != nil && *cfg.BaseURL != "" {
			baseURL = *cfg.BaseURL
		}
		return NewOllamaEmbedder(baseURL, cfg.Model), nil

	case "local":
		return NewLocalEmbedder(), nil

	default:
		return nil, ErrUnknownProvider
	}
}
