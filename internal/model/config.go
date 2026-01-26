package model

// Config はサーバー全体の設定を表す
type Config struct {
	TransportDefaults TransportDefaults `json:"transportDefaults"`
	Embedder          EmbedderConfig    `json:"embedder"`
	Store             StoreConfig       `json:"store"`
	Paths             PathsConfig       `json:"paths"`
}

// TransportDefaults はtransportのデフォルト設定
type TransportDefaults struct {
	DefaultTransport string `json:"defaultTransport"` // "stdio" | "http"
}

// EmbedderConfig はembedder設定
type EmbedderConfig struct {
	Provider string  `json:"provider"`          // "openai" | "ollama" | "local"
	Model    string  `json:"model"`             // モデル名
	Dim      int     `json:"dim"`               // ベクトル次元（0は未設定）
	BaseURL  *string `json:"baseUrl,omitempty"` // nullable、省略可
	APIKey   *string `json:"apiKey,omitempty"`  // nullable、省略可（セキュリティ注意）
}

// StoreConfig はvector store設定
type StoreConfig struct {
	Type string  `json:"type"`           // "chroma" | "sqlite" | "qdrant" | "faiss"
	Path *string `json:"path,omitempty"` // nullable（SQLite用）
	URL  *string `json:"url,omitempty"`  // nullable（Chroma/Qdrant用）
}

// PathsConfig はファイルパス設定
type PathsConfig struct {
	ConfigPath string `json:"configPath"` // 設定ファイルパス
	DataDir    string `json:"dataDir"`    // データディレクトリ
}

// Transport定数
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// Embedder Provider定数
const (
	ProviderOpenAI = "openai"
	ProviderOllama = "ollama"
	ProviderLocal  = "local"
)

// Store Type定数
const (
	StoreTypeChroma = "chroma"
	StoreTypeSQLite = "sqlite"
	StoreTypeQdrant = "qdrant"
	StoreTypeFAISS  = "faiss"
)
