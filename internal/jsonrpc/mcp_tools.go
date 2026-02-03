package jsonrpc

import "github.com/brbranch/embedding_mcp/internal/model"

// mcpTools はMCPプロトコルで公開するツールのリスト
var mcpTools = []model.Tool{
	{
		Name:        "memory_add_note",
		Description: "Add a new note to the memory store",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID (usually the working directory path)",
				},
				"groupId": {
					Type:        "string",
					Description: "Group ID for categorizing notes (e.g., 'global', 'reviews', 'learning')",
				},
				"text": {
					Type:        "string",
					Description: "The note content",
				},
				"title": {
					Type:        "string",
					Description: "Optional title for the note",
				},
				"tags": {
					Type:        "array",
					Description: "Optional tags for the note",
					Items: &model.JSONSchema{
						Type: "string",
					},
				},
				"source": {
					Type:        "string",
					Description: "Optional source reference",
				},
				"createdAt": {
					Type:        "string",
					Description: "Optional ISO8601 timestamp for the note creation time",
				},
				"metadata": {
					Type:        "object",
					Description: "Optional metadata as key-value pairs",
				},
			},
			Required: []string{"projectId", "groupId", "text"},
		},
	},
	{
		Name:        "memory_search",
		Description: "Search notes by semantic similarity",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID to search within",
				},
				"query": {
					Type:        "string",
					Description: "Search query text",
				},
				"groupId": {
					Type:        "string",
					Description: "Optional group ID to filter results",
				},
				"topK": {
					Type:        "integer",
					Description: "Maximum number of results to return (default: 5)",
					Default:     5,
				},
				"tags": {
					Type:        "array",
					Description: "Optional tags to filter results",
					Items: &model.JSONSchema{
						Type: "string",
					},
				},
				"since": {
					Type:        "string",
					Description: "Optional ISO8601 timestamp to filter notes created after this time",
				},
				"until": {
					Type:        "string",
					Description: "Optional ISO8601 timestamp to filter notes created before this time",
				},
			},
			Required: []string{"projectId", "query"},
		},
	},
	{
		Name:        "memory_get",
		Description: "Get a note by its ID",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The note ID",
				},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        "memory_update",
		Description: "Update an existing note",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The note ID to update",
				},
				"patch": {
					Type:        "object",
					Description: "Fields to update",
					Properties: map[string]model.JSONSchema{
						"title": {
							Description: "New title (null to clear)",
							OneOf: []model.JSONSchema{
								{Type: "string"},
								{Type: "null"},
							},
						},
						"text": {
							Type:        "string",
							Description: "New text content",
						},
						"tags": {
							Type:        "array",
							Description: "New tags array",
							Items: &model.JSONSchema{
								Type: "string",
							},
						},
						"source": {
							Description: "New source (null to clear)",
							OneOf: []model.JSONSchema{
								{Type: "string"},
								{Type: "null"},
							},
						},
						"groupId": {
							Type:        "string",
							Description: "New group ID",
						},
						"metadata": {
							Description: "New metadata (null to clear)",
							OneOf: []model.JSONSchema{
								{Type: "object"},
								{Type: "null"},
							},
						},
					},
				},
			},
			Required: []string{"id", "patch"},
		},
	},
	{
		Name:        "memory_delete",
		Description: "Delete a note or global config by ID",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The ID of the note or global config to delete",
				},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        "memory_list_recent",
		Description: "List recent notes",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID to list notes from",
				},
				"groupId": {
					Type:        "string",
					Description: "Optional group ID to filter results",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of notes to return (default: 10)",
					Default:     10,
				},
				"tags": {
					Type:        "array",
					Description: "Optional tags to filter results",
					Items: &model.JSONSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"projectId"},
		},
	},
	{
		Name:        "memory_get_config",
		Description: "Get the current server configuration",
		InputSchema: model.JSONSchema{
			Type:       "object",
			Properties: map[string]model.JSONSchema{},
		},
	},
	{
		Name:        "memory_set_config",
		Description: "Update server configuration",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"embedder": {
					Type:        "object",
					Description: "Embedder configuration",
					Properties: map[string]model.JSONSchema{
						"provider": {
							Type:        "string",
							Description: "Embedding provider (openai, anthropic, etc.)",
						},
						"model": {
							Type:        "string",
							Description: "Model name for embedding",
						},
						"baseUrl": {
							Type:        "string",
							Description: "Optional custom API base URL",
						},
						"apiKey": {
							Type:        "string",
							Description: "API key for the provider",
						},
					},
				},
			},
		},
	},
	{
		Name:        "memory_upsert_global",
		Description: "Create or update a global configuration value",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID",
				},
				"key": {
					Type:        "string",
					Description: "Configuration key (must start with 'global.')",
				},
				"value": {
					Description: "Configuration value (any JSON-serializable value: string, number, boolean, object, array)",
				},
				"updatedAt": {
					Type:        "string",
					Description: "Optional ISO8601 timestamp",
				},
			},
			Required: []string{"projectId", "key", "value"},
		},
	},
	{
		Name:        "memory_get_global",
		Description: "Get a global configuration value",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID",
				},
				"key": {
					Type:        "string",
					Description: "Configuration key",
				},
			},
			Required: []string{"projectId", "key"},
		},
	},
	{
		Name:        "memory_group_create",
		Description: "Create a new group for organizing notes",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID",
				},
				"groupKey": {
					Type:        "string",
					Description: "Unique key for the group (alphanumeric, hyphen, underscore only; 'global' is reserved)",
				},
				"title": {
					Type:        "string",
					Description: "Display title for the group",
				},
				"description": {
					Type:        "string",
					Description: "Optional description for the group",
				},
			},
			Required: []string{"projectId", "groupKey", "title"},
		},
	},
	{
		Name:        "memory_group_get",
		Description: "Get a group by its ID",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The group ID",
				},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        "memory_group_update",
		Description: "Update an existing group (groupKey cannot be changed)",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The group ID to update",
				},
				"patch": {
					Type:        "object",
					Description: "Fields to update",
					Properties: map[string]model.JSONSchema{
						"title": {
							Type:        "string",
							Description: "New title",
						},
						"description": {
							Type:        "string",
							Description: "New description",
						},
					},
				},
			},
			Required: []string{"id", "patch"},
		},
	},
	{
		Name:        "memory_group_delete",
		Description: "Delete a group by ID",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"id": {
					Type:        "string",
					Description: "The group ID to delete",
				},
			},
			Required: []string{"id"},
		},
	},
	{
		Name:        "memory_group_list",
		Description: "List all groups in a project",
		InputSchema: model.JSONSchema{
			Type: "object",
			Properties: map[string]model.JSONSchema{
				"projectId": {
					Type:        "string",
					Description: "Project ID to list groups from",
				},
			},
			Required: []string{"projectId"},
		},
	},
}

// toolNameToMethod はMCPツール名から内部メソッド名へのマッピング
var toolNameToMethod = map[string]string{
	"memory_add_note":      "memory.add_note",
	"memory_search":        "memory.search",
	"memory_get":           "memory.get",
	"memory_update":        "memory.update",
	"memory_delete":        "memory.delete",
	"memory_list_recent":   "memory.list_recent",
	"memory_get_config":    "memory.get_config",
	"memory_set_config":    "memory.set_config",
	"memory_upsert_global": "memory.upsert_global",
	"memory_get_global":    "memory.get_global",
	"memory_group_create":  "memory.group_create",
	"memory_group_get":     "memory.group_get",
	"memory_group_update":  "memory.group_update",
	"memory_group_delete":  "memory.group_delete",
	"memory_group_list":    "memory.group_list",
}
