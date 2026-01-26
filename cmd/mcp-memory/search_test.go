package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/service"
)

// TestParseSearchFlags tests flag parsing for search command
func TestParseSearchFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProject string
		wantGroup   string
		wantTopK    int
		wantTags    string
		wantFormat  string
		wantStdin   bool
		wantQuery   string
		wantErr     bool
	}{
		{
			name:        "all flags",
			args:        []string{"-p", "/test/project", "-g", "global", "-k", "10", "--tags", "tag1,tag2", "-f", "json", "search query"},
			wantProject: "/test/project",
			wantGroup:   "global",
			wantTopK:    10,
			wantTags:    "tag1,tag2",
			wantFormat:  "json",
			wantStdin:   false,
			wantQuery:   "search query",
			wantErr:     false,
		},
		{
			name:        "long flags",
			args:        []string{"--project", "/test/project", "--group", "feature-1", "--top-k", "3", "--format", "text", "query"},
			wantProject: "/test/project",
			wantGroup:   "feature-1",
			wantTopK:    3,
			wantFormat:  "text",
			wantQuery:   "query",
			wantErr:     false,
		},
		{
			name:        "minimal required flags",
			args:        []string{"-p", "/test/project", "query"},
			wantProject: "/test/project",
			wantGroup:   "",
			wantTopK:    5, // default
			wantFormat:  "text",
			wantQuery:   "query",
			wantErr:     false,
		},
		{
			name:        "stdin flag",
			args:        []string{"-p", "/test/project", "--stdin"},
			wantProject: "/test/project",
			wantTopK:    5, // default
			wantFormat:  "text",
			wantStdin:   true,
			wantQuery:   "", // query from stdin
			wantErr:     false,
		},
		{
			name:        "multi-word query",
			args:        []string{"-p", "/test/project", "multi", "word", "query"},
			wantProject: "/test/project",
			wantTopK:    5, // default
			wantFormat:  "text",
			wantQuery:   "multi word query",
			wantErr:     false,
		},
		{
			name:    "missing project",
			args:    []string{"query"},
			wantErr: true,
		},
		{
			name:    "missing query without stdin",
			args:    []string{"-p", "/test/project"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseSearchFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if opts.ProjectID != tt.wantProject {
				t.Errorf("ProjectID = %q, want %q", opts.ProjectID, tt.wantProject)
			}
			if opts.GroupID != tt.wantGroup {
				t.Errorf("GroupID = %q, want %q", opts.GroupID, tt.wantGroup)
			}
			if opts.TopK != tt.wantTopK {
				t.Errorf("TopK = %d, want %d", opts.TopK, tt.wantTopK)
			}
			if opts.Tags != tt.wantTags {
				t.Errorf("Tags = %q, want %q", opts.Tags, tt.wantTags)
			}
			if opts.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", opts.Format, tt.wantFormat)
			}
			if opts.UseStdin != tt.wantStdin {
				t.Errorf("UseStdin = %v, want %v", opts.UseStdin, tt.wantStdin)
			}
			if opts.Query != tt.wantQuery {
				t.Errorf("Query = %q, want %q", opts.Query, tt.wantQuery)
			}
		})
	}
}

// TestFormatTextOutput tests text format output
func TestFormatTextOutput(t *testing.T) {
	title := "Test Title"
	results := []service.SearchResult{
		{
			ID:        "id1",
			ProjectID: "/test/project",
			GroupID:   "global",
			Title:     &title,
			Text:      "This is a test note content",
			Tags:      []string{"tag1", "tag2"},
			Score:     0.92,
		},
		{
			ID:        "id2",
			ProjectID: "/test/project",
			GroupID:   "feature-1",
			Title:     nil,
			Text:      "Another note without title",
			Tags:      nil,
			Score:     0.85,
		},
	}

	var buf bytes.Buffer
	formatTextOutput(&buf, results)
	output := buf.String()

	// Check result format
	if !strings.Contains(output, "[1]") {
		t.Error("expected output to contain [1]")
	}
	if !strings.Contains(output, "Test Title") {
		t.Error("expected output to contain title")
	}
	if !strings.Contains(output, "score: 0.92") {
		t.Error("expected output to contain score")
	}
	if !strings.Contains(output, "tag1, tag2") {
		t.Error("expected output to contain tags")
	}
	if !strings.Contains(output, "[2]") {
		t.Error("expected output to contain [2]")
	}
}

// TestFormatJSONOutput tests JSON format output
func TestFormatJSONOutput(t *testing.T) {
	title := "Test Title"
	results := []service.SearchResult{
		{
			ID:        "id1",
			ProjectID: "/test/project",
			GroupID:   "global",
			Title:     &title,
			Text:      "Test content",
			Tags:      []string{"tag1"},
			Score:     0.92,
		},
	}

	var buf bytes.Buffer
	if err := formatJSONOutput(&buf, results); err != nil {
		t.Fatalf("formatJSONOutput failed: %v", err)
	}

	// Parse JSON output
	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(output.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(output.Results))
	}
	if output.Results[0].ID != "id1" {
		t.Errorf("expected id 'id1', got %q", output.Results[0].ID)
	}
	if output.Results[0].Score != 0.92 {
		t.Errorf("expected score 0.92, got %f", output.Results[0].Score)
	}
}

// TestReadQueryFromStdin tests reading query from stdin
func TestReadQueryFromStdin(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create pipe for mock stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r

	// Write test query to pipe
	testQuery := "test query from stdin"
	go func() {
		w.WriteString(testQuery + "\n")
		w.Close()
	}()

	query, err := readQueryFromStdin()
	if err != nil {
		t.Fatalf("readQueryFromStdin failed: %v", err)
	}
	if query != testQuery {
		t.Errorf("expected %q, got %q", testQuery, query)
	}
}

// mockNoteService is a mock implementation for testing
type mockNoteService struct {
	searchFunc func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error)
}

func (m *mockNoteService) AddNote(ctx context.Context, req *service.AddNoteRequest) (*service.AddNoteResponse, error) {
	return nil, nil
}

func (m *mockNoteService) Search(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	return &service.SearchResponse{
		Namespace: "test",
		Results:   []service.SearchResult{},
	}, nil
}

func (m *mockNoteService) Get(ctx context.Context, id string) (*service.GetResponse, error) {
	return nil, nil
}

func (m *mockNoteService) Update(ctx context.Context, req *service.UpdateRequest) error {
	return nil
}

func (m *mockNoteService) ListRecent(ctx context.Context, req *service.ListRecentRequest) (*service.ListRecentResponse, error) {
	return nil, nil
}

// TestExecuteSearch tests the search execution logic
func TestExecuteSearch(t *testing.T) {
	title := "Test Note"
	mockService := &mockNoteService{
		searchFunc: func(ctx context.Context, req *service.SearchRequest) (*service.SearchResponse, error) {
			// Verify request parameters
			if req.ProjectID != "/test/canonical/project" {
				t.Errorf("expected canonicalized project ID, got %q", req.ProjectID)
			}
			if req.Query != "test query" {
				t.Errorf("expected query 'test query', got %q", req.Query)
			}
			return &service.SearchResponse{
				Namespace: "test",
				Results: []service.SearchResult{
					{
						ID:        "id1",
						ProjectID: req.ProjectID,
						GroupID:   "global",
						Title:     &title,
						Text:      "Test content",
						Score:     0.9,
					},
				},
			}, nil
		},
	}

	ctx := context.Background()
	results, err := executeSearchWithService(ctx, mockService, "/test/canonical/project", nil, "test query", 5, nil)
	if err != nil {
		t.Fatalf("executeSearchWithService failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// TestParseTags tests tag parsing
func TestParseTags(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"tag1,tag2,tag3", []string{"tag1", "tag2", "tag3"}},
		{"tag1", []string{"tag1"}},
		{"", nil},
		{"tag1, tag2, tag3", []string{"tag1", "tag2", "tag3"}}, // with spaces
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTags(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseTags(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestTruncateText tests text truncation
func TestTruncateText(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short text", 100, "short text"},
		{"this is a very long text that should be truncated", 20, "this is a very long  ..."},
		{"exactly twenty chars", 20, "exactly twenty chars"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateText(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// Dummy test to ensure the file compiles
func TestSearchCompiles(t *testing.T) {
	// This test just ensures the search.go file compiles correctly
	_ = io.Discard
}

// TestParseSearchFlags_TopKValidation tests top-k validation
func TestParseSearchFlags_TopKValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "negative top-k",
			args:    []string{"-p", "/test/project", "-k", "-1", "query"},
			wantErr: true,
			errMsg:  "top-k must be greater than 0",
		},
		{
			name:    "zero top-k",
			args:    []string{"-p", "/test/project", "-k", "0", "query"},
			wantErr: true,
			errMsg:  "top-k must be greater than 0",
		},
		{
			name:    "valid positive top-k",
			args:    []string{"-p", "/test/project", "-k", "10", "query"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSearchFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestParseSearchFlags_FormatValidation tests format validation
func TestParseSearchFlags_FormatValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid format",
			args:    []string{"-p", "/test/project", "-f", "yaml", "query"},
			wantErr: true,
			errMsg:  "invalid format: yaml (must be text or json)",
		},
		{
			name:    "another invalid format",
			args:    []string{"-p", "/test/project", "--format", "xml", "query"},
			wantErr: true,
			errMsg:  "invalid format: xml (must be text or json)",
		},
		{
			name:    "valid text format",
			args:    []string{"-p", "/test/project", "-f", "text", "query"},
			wantErr: false,
		},
		{
			name:    "valid json format",
			args:    []string{"-p", "/test/project", "-f", "json", "query"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSearchFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
