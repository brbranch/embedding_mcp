package model

// InitializeParams は initialize メソッドのパラメータ
type InitializeParams struct {
	ProtocolVersion string      `json:"protocolVersion"`
	ClientInfo      ClientInfo  `json:"clientInfo"`
	Capabilities    Capabilities `json:"capabilities,omitempty"`
}

// ClientInfo はクライアント情報
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo はサーバー情報
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities はクライアント/サーバーの機能
type Capabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

// ToolsCapability はツール機能の設定
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability はプロンプト機能の設定
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability はリソース機能の設定
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeResult は initialize メソッドの結果
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

// Tool はMCPツールの定義
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema JSONSchema `json:"inputSchema"`
}

// JSONSchema はJSON Schemaの定義
type JSONSchema struct {
	Type       string                `json:"type,omitempty"`
	Properties map[string]JSONSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Items      *JSONSchema           `json:"items,omitempty"`
	// 追加プロパティ
	Description string        `json:"description,omitempty"`
	Enum        []string      `json:"enum,omitempty"`
	Default     any           `json:"default,omitempty"`
	OneOf       []JSONSchema  `json:"oneOf,omitempty"`
}

// ToolsListResult は tools/list メソッドの結果
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolsCallParams は tools/call メソッドのパラメータ
type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolsCallResult は tools/call メソッドの結果
type ToolsCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem はコンテンツアイテム
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// NewTextContent はテキストコンテンツを生成
func NewTextContent(text string) ContentItem {
	return ContentItem{
		Type: "text",
		Text: text,
	}
}
