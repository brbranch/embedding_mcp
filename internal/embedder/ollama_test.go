package embedder

import (
	"context"
	"errors"
	"testing"
)

func TestOllamaEmbedder_Embed_NotImplemented(t *testing.T) {
	emb := NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")

	_, err := emb.Embed(context.Background(), "test text")
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestOllamaEmbedder_GetDimension_Zero(t *testing.T) {
	emb := NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")

	if dim := emb.GetDimension(); dim != 0 {
		t.Errorf("expected 0, got %d", dim)
	}
}
