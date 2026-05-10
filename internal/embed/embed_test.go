package embed

import (
	"math/rand"
	"slices"
	"testing"
)

func TestChuckWithOverlapShouldCreateChunksWithValidOverlap(t *testing.T) {
	// min 100 to max 1000 len
	tSliceLen := rand.Intn(900) + 100
	tSlice := make([]int, tSliceLen)

	for idx := range tSliceLen {
		tSlice[idx] = rand.Intn(1000)
	}

	chunkSize := rand.Intn(30) + 10
	overlap := int(float32(chunkSize) * 0.2)

	chunks, err := chunkWithOverlap(tSlice, chunkSize, overlap, 0)

	if err != nil {
		t.Fatal(err)
	}

	if len(chunks[0]) != chunkSize || len(chunks[len(chunks)-1]) != chunkSize {
		t.Fatalf("chunked slice size not equal chunk size expected=%d, got=%d", chunkSize, len(chunks[0]))
	}

	cLen := len(chunks)
	for idx := 1; idx < cLen-1; idx++ {
		c1, c2 := chunks[idx-1][chunkSize-overlap:], chunks[idx][:overlap]
		if slices.Equal(c1, c2) {
			continue
		}
		t.Fatalf("Expected overlapping chunks to be equal but received different chunks, chunk1=%v, chunk2=%v", c1, c2)
	}
}
