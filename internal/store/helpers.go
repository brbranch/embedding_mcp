package store

import (
	"math"
)

// CosineSimilarity はコサイン類似度を計算する（実際はcosine distanceを返す: 0=同一、2=正反対）
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 2.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 2.0
	}

	// cosine similarity: -1 to 1
	similarity := dotProduct / (normA * normB)

	// cosine distance: 0 to 2
	distance := 1.0 - similarity

	return distance
}

// ContainsAllTags はtargets内の全てのタグがtagsに含まれているかをチェックする（AND検索）
func ContainsAllTags(tags []string, targets []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, target := range targets {
		if !tagSet[target] {
			return false
		}
	}

	return true
}
