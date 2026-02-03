// Package bootstrap provides common initialization logic for mcp-memory.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/embedder"
	"github.com/brbranch/embedding_mcp/internal/model"
	"github.com/brbranch/embedding_mcp/internal/service"
	"github.com/brbranch/embedding_mcp/internal/store"
)

// Services は初期化されたサービス群を保持
type Services struct {
	NoteService   service.NoteService
	ConfigService service.ConfigService
	GlobalService service.GlobalService
	GroupService  service.GroupService
	Config        *model.Config
	Namespace     string
}

// Initialize は設定を読み込み、必要なサービスを初期化する
func Initialize(ctx context.Context, configPath string) (*Services, func(), error) {
	// 設定マネージャーの作成
	configManager, err := config.NewManager(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	// 設定ファイルの読み込み
	if err := configManager.Load(); err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := configManager.GetConfig()

	// namespace生成
	namespace := config.GenerateNamespace(cfg.Embedder.Provider, cfg.Embedder.Model, cfg.Embedder.Dim)

	// 1. Embedder初期化
	emb, err := embedder.NewEmbedder(&cfg.Embedder, os.Getenv("OPENAI_API_KEY"), configManager)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 2. Store初期化
	var st store.Store
	switch cfg.Store.Type {
	case "chroma":
		url := "http://localhost:8000"
		if cfg.Store.URL != nil && *cfg.Store.URL != "" {
			url = *cfg.Store.URL
		}
		st, err = store.NewChromaStore(url)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create store: %w", err)
		}
	case "sqlite":
		// SQLiteのDBパスを決定
		dbPath := cfg.Paths.DataDir + "/memory.db"
		if cfg.Store.Path != nil && *cfg.Store.Path != "" {
			dbPath = *cfg.Store.Path
		}
		// DBファイルの親ディレクトリを作成
		if err := config.EnsureDir(filepath.Dir(dbPath)); err != nil {
			return nil, nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		st, err = store.NewSQLiteStore(dbPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create sqlite store: %w", err)
		}
	case "qdrant":
		url := "http://localhost:6333"
		if cfg.Store.URL != nil && *cfg.Store.URL != "" {
			url = *cfg.Store.URL
		}
		st, err = store.NewQdrantStore(url)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create qdrant store: %w", err)
		}
	default:
		st = store.NewMemoryStore()
	}

	// 3. Store初期化（namespace設定）
	if err := st.Initialize(ctx, namespace); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	// 4. Services初期化
	noteService := service.NewNoteService(emb, st, namespace)
	configService := service.NewConfigService(configManager)
	globalService := service.NewGlobalService(st, namespace)
	groupService := service.NewGroupService(st, namespace)

	cleanup := func() {
		st.Close()
	}

	return &Services{
		NoteService:   noteService,
		ConfigService: configService,
		GlobalService: globalService,
		GroupService:  groupService,
		Config:        cfg,
		Namespace:     namespace,
	}, cleanup, nil
}
