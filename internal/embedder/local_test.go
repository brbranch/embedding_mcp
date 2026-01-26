package embedder

import (
	"context"
	"errors"
	"testing"
)

func TestLocalEmbedder_Embed_NotImplemented(t *testing.T) {
	emb := NewLocalEmbedder()

	_, err := emb.Embed(context.Background(), "test text")
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestLocalEmbedder_GetDimension_Zero(t *testing.T) {
	emb := NewLocalEmbedder()

	if dim := emb.GetDimension(); dim != 0 {
		t.Errorf("expected 0, got %d", dim)
	}
}
