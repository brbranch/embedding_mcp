package embedder

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockDimUpdater はテスト用のDimUpdater実装
type mockDimUpdater struct {
	updatedDim int
	callCount  int
	err        error
}

func (m *mockDimUpdater) UpdateDim(dim int) error {
	m.updatedDim = dim
	m.callCount++
	return m.err
}

// openAIResponse はOpenAI API応答の構造
type openAIResponse struct {
	Data  []openAIEmbeddingData `json:"data"`
	Model string                `json:"model"`
}

type openAIEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// newMockOpenAIServer はモックサーバーを作成
func newMockOpenAIServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// successHandler は正常応答を返すハンドラ
func successHandler(embedding []float32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Data: []openAIEmbeddingData{
				{Embedding: embedding, Index: 0},
			},
			Model: "text-embedding-3-small",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestOpenAIEmbedder_NewEmbedder_APIKeyRequired(t *testing.T) {
	_, err := NewOpenAIEmbedder("")
	if !errors.Is(err, ErrAPIKeyRequired) {
		t.Errorf("expected ErrAPIKeyRequired, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed_Success(t *testing.T) {
	expected := []float32{0.1, 0.2, 0.3}
	server := newMockOpenAIServer(successHandler(expected))
	defer server.Close()

	emb, err := NewOpenAIEmbedder("test-api-key", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	result, err := emb.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(result) != len(expected) {
		t.Errorf("expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("element %d: expected %f, got %f", i, expected[i], v)
		}
	}
}

func TestOpenAIEmbedder_Embed_DimUpdatedOnFirstCall(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	server := newMockOpenAIServer(successHandler(embedding))
	defer server.Close()

	updater := &mockDimUpdater{}
	emb, err := NewOpenAIEmbedder("test-api-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithDimUpdater(updater))
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	_, err = emb.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if updater.callCount != 1 {
		t.Errorf("expected UpdateDim to be called once, got %d", updater.callCount)
	}
	if updater.updatedDim != 5 {
		t.Errorf("expected dim 5, got %d", updater.updatedDim)
	}
}

func TestOpenAIEmbedder_Embed_DimUpdatedOnlyOnce(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3}
	server := newMockOpenAIServer(successHandler(embedding))
	defer server.Close()

	updater := &mockDimUpdater{}
	emb, err := NewOpenAIEmbedder("test-api-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithDimUpdater(updater))
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Call Embed twice
	_, _ = emb.Embed(context.Background(), "text 1")
	_, _ = emb.Embed(context.Background(), "text 2")

	if updater.callCount != 1 {
		t.Errorf("expected UpdateDim to be called once, got %d", updater.callCount)
	}
}

func TestOpenAIEmbedder_Embed_DimPreset(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3}
	server := newMockOpenAIServer(successHandler(embedding))
	defer server.Close()

	updater := &mockDimUpdater{}
	emb, err := NewOpenAIEmbedder("test-api-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithDim(3),
		WithDimUpdater(updater))
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	_, _ = emb.Embed(context.Background(), "text")

	// When dim is preset, UpdateDim should not be called
	if updater.callCount != 0 {
		t.Errorf("expected UpdateDim not to be called when dim is preset, got %d calls", updater.callCount)
	}
}

func TestOpenAIEmbedder_Embed_DimUpdaterError(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3}
	server := newMockOpenAIServer(successHandler(embedding))
	defer server.Close()

	updater := &mockDimUpdater{err: errors.New("update failed")}
	emb, err := NewOpenAIEmbedder("test-api-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithDimUpdater(updater))
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Embed should succeed even if UpdateDim fails
	result, err := emb.Embed(context.Background(), "text")
	if err != nil {
		t.Errorf("Embed should succeed even if UpdateDim fails, got %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result))
	}
}

func TestOpenAIEmbedder_Embed_APIError_4xx(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API Key"}}`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("invalid-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if !errors.Is(err, ErrAPIRequestFailed) {
		t.Error("expected error to match ErrAPIRequestFailed")
	}
}

func TestOpenAIEmbedder_Embed_APIError_RateLimit(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
}

func TestOpenAIEmbedder_Embed_APIError_5xx(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

func TestOpenAIEmbedder_Embed_InvalidJSON(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Errorf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed_EmptyData(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [], "model": "text-embedding-3-small"}`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if !errors.Is(err, ErrEmptyEmbedding) {
		t.Errorf("expected ErrEmptyEmbedding, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed_EmptyEmbedding(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"embedding": [], "index": 0}], "model": "text-embedding-3-small"}`))
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if !errors.Is(err, ErrEmptyEmbedding) {
		t.Errorf("expected ErrEmptyEmbedding, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed_ContextCanceled(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		successHandler([]float32{0.1})(w, r)
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := emb.Embed(ctx, "text")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestOpenAIEmbedder_Embed_ContextDeadlineExceeded(t *testing.T) {
	server := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		successHandler([]float32{0.1})(w, r)
	})
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := emb.Embed(ctx, "text")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestOpenAIEmbedder_GetDimension_BeforeEmbed(t *testing.T) {
	emb, _ := NewOpenAIEmbedder("test-key")
	if dim := emb.GetDimension(); dim != 0 {
		t.Errorf("expected 0, got %d", dim)
	}
}

func TestOpenAIEmbedder_GetDimension_AfterEmbed(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	server := newMockOpenAIServer(successHandler(embedding))
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, _ = emb.Embed(context.Background(), "text")

	if dim := emb.GetDimension(); dim != 4 {
		t.Errorf("expected 4, got %d", dim)
	}
}

func TestOpenAIEmbedder_WithBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request was made to this server
		successHandler([]float32{0.1})(w, r)
	}))
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()))

	_, err := emb.Embed(context.Background(), "text")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestOpenAIEmbedder_WithModel(t *testing.T) {
	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		receivedModel = req.Model
		successHandler([]float32{0.1})(w, r)
	}))
	defer server.Close()

	emb, _ := NewOpenAIEmbedder("test-key",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithModel("text-embedding-3-large"))

	_, _ = emb.Embed(context.Background(), "text")

	if receivedModel != "text-embedding-3-large" {
		t.Errorf("expected model text-embedding-3-large, got %s", receivedModel)
	}
}
