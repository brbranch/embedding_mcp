package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brbranch/embedding_mcp/internal/bootstrap"
	"github.com/brbranch/embedding_mcp/internal/config"
	"github.com/brbranch/embedding_mcp/internal/service"
)

// SearchOptions holds parsed search command options
type SearchOptions struct {
	ProjectID  string
	GroupID    string
	TopK       int
	Tags       string
	Format     string
	ConfigPath string
	UseStdin   bool
	Query      string
}

// JSONOutput represents the JSON output format
type JSONOutput struct {
	Results []JSONResult `json:"results"`
}

// JSONResult represents a single result in JSON output
type JSONResult struct {
	ID    string   `json:"id"`
	Title string   `json:"title,omitempty"`
	Text  string   `json:"text"`
	Score float64  `json:"score"`
	Tags  []string `json:"tags,omitempty"`
}

// parseSearchFlags parses command line arguments for search command
func parseSearchFlags(args []string) (*SearchOptions, error) {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress default error output

	opts := &SearchOptions{}

	// Long flags
	fs.StringVar(&opts.ProjectID, "project", "", "Project ID/path (required)")
	fs.StringVar(&opts.GroupID, "group", "", "Group ID (optional)")
	fs.IntVar(&opts.TopK, "top-k", 5, "Number of results")
	fs.StringVar(&opts.Tags, "tags", "", "Tag filter (comma-separated)")
	fs.StringVar(&opts.Format, "format", "text", "Output format: text|json")
	fs.StringVar(&opts.ConfigPath, "config", "", "Config file path")
	fs.BoolVar(&opts.UseStdin, "stdin", false, "Read query from stdin")

	// Short flags
	fs.StringVar(&opts.ProjectID, "p", "", "Project ID/path (required)")
	fs.StringVar(&opts.GroupID, "g", "", "Group ID (optional)")
	fs.IntVar(&opts.TopK, "k", 5, "Number of results")
	fs.StringVar(&opts.Format, "f", "text", "Output format: text|json")
	fs.StringVar(&opts.ConfigPath, "c", "", "Config file path")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Set default format if not specified
	if opts.Format == "" {
		opts.Format = "text"
	}

	// Get query from remaining args
	opts.Query = strings.Join(fs.Args(), " ")

	// Validation
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required (-p or --project)")
	}

	if !opts.UseStdin && opts.Query == "" {
		return nil, fmt.Errorf("query is required (or use --stdin)")
	}

	// Validate top-k
	if opts.TopK <= 0 {
		return nil, fmt.Errorf("top-k must be greater than 0")
	}

	// Validate format
	if opts.Format != "text" && opts.Format != "json" {
		return nil, fmt.Errorf("invalid format: %s (must be text or json)", opts.Format)
	}

	return opts, nil
}

// runSearchCmd is the entry point for search command
func runSearchCmd(args []string) error {
	opts, err := parseSearchFlags(args)
	if err != nil {
		return err
	}

	// Read query from stdin if requested
	if opts.UseStdin {
		query, err := readQueryFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read query from stdin: %w", err)
		}
		opts.Query = query
	}

	if opts.Query == "" {
		return fmt.Errorf("query is empty")
	}

	// Initialize services
	ctx := context.Background()
	services, cleanup, err := bootstrap.Initialize(ctx, opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer cleanup()

	// Canonicalize project ID
	canonicalProjectID, err := config.CanonicalizeProjectID(opts.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to canonicalize projectId: %w", err)
	}

	// Parse tags
	tags := parseTags(opts.Tags)

	// Prepare group ID pointer
	var groupID *string
	if opts.GroupID != "" {
		groupID = &opts.GroupID
	}

	// Execute search
	results, err := executeSearchWithService(ctx, services.NoteService, canonicalProjectID, groupID, opts.Query, opts.TopK, tags)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Output results
	switch opts.Format {
	case "json":
		if err := formatJSONOutput(os.Stdout, results); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	default:
		formatTextOutput(os.Stdout, results)
	}

	return nil
}

// executeSearchWithService executes search using the provided NoteService
func executeSearchWithService(ctx context.Context, noteService service.NoteService, projectID string, groupID *string, query string, topK int, tags []string) ([]service.SearchResult, error) {
	req := &service.SearchRequest{
		ProjectID: projectID,
		GroupID:   groupID,
		Query:     query,
		TopK:      &topK,
		Tags:      tags,
	}

	resp, err := noteService.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Results, nil
}

// readQueryFromStdin reads a single line query from stdin
func readQueryFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no input received")
}

// parseTags parses comma-separated tags into a slice
func parseTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}

	parts := strings.Split(tagsStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// formatTextOutput outputs results in human-readable text format
func formatTextOutput(w io.Writer, results []service.SearchResult) {
	if len(results) == 0 {
		fmt.Fprintln(w, "No results found.")
		return
	}

	for i, r := range results {
		// Title or truncated text as header
		title := "(no title)"
		if r.Title != nil && *r.Title != "" {
			title = *r.Title
		}

		fmt.Fprintf(w, "[%d] %s (score: %.2f)\n", i+1, title, r.Score)

		// Truncated text content
		text := truncateText(r.Text, 60)
		fmt.Fprintf(w, "    %s\n", text)

		// Tags
		if len(r.Tags) > 0 {
			fmt.Fprintf(w, "    tags: %s\n", strings.Join(r.Tags, ", "))
		}

		fmt.Fprintln(w)
	}
}

// formatJSONOutput outputs results in JSON format
func formatJSONOutput(w io.Writer, results []service.SearchResult) error {
	output := JSONOutput{
		Results: make([]JSONResult, 0, len(results)),
	}

	for _, r := range results {
		title := ""
		if r.Title != nil {
			title = *r.Title
		}

		output.Results = append(output.Results, JSONResult{
			ID:    r.ID,
			Title: title,
			Text:  r.Text,
			Score: r.Score,
			Tags:  r.Tags,
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// truncateText truncates text to maxLen and adds "..." if truncated
func truncateText(text string, maxLen int) string {
	if text == "" || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + " ..."
}
