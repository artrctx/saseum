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
		tSlice[idx] = rand.Intn(1000) + 100
	}

	chunkSize := rand.Intn(30) + 10
	overlap := int(float32(chunkSize) * 0.2)

	beginID, endID, padID := rand.Intn(40)+10, rand.Intn(50)+50, 0
	chunks, err := chunkWithOverlap(tSlice, chunkSize, overlap, beginID, endID, padID)

	if err != nil {
		t.Fatal(err)
	}

	if len(chunks[0]) != chunkSize || len(chunks[len(chunks)-1]) != chunkSize {
		t.Fatalf("chunked slice size not equal chunk size expected=%d, got=%d", chunkSize, len(chunks[0]))
	}

	cLen := len(chunks)
	for idx := 1; idx < cLen-1; idx++ {
		c1, c2 := chunks[idx-1], chunks[idx]
		if c1[0] != beginID || c1[len(c1)-1] != endID {
			t.Fatalf("Expected chunk to start and end with valid begin and end token expected begin,end=%d,%d | got begin,end=%d,%d", beginID, endID, c1[0], c1[len(c1)-1])
		}
		co1, co2 := c1[chunkSize-overlap-1:chunkSize-1], c2[1:overlap+1]
		if slices.Equal(co1, co2) {
			continue
		}
		t.Fatalf("Expected overlapping chunks to be equal but received different chunks, chunk1=%v, chunk2=%v", c1, c2)
	}

	// verify last chunk
	lc := chunks[len(chunks)-1]
	padIdx := slices.Index(lc, endID)
	if padIdx == -1 {
		padIdx = len(lc) - 1
	}
	lc = lc[:padIdx+1]
	if lc[0] != beginID || lc[len(lc)-1] != endID {
		t.Fatalf("Expected last chunk to start and end with valida begin and end token expected begin,end=%d,%d | got begin,end=%d,%d", beginID, endID, lc[0], lc[len(lc)-1])
	}
	if len(lc) == 0 {
		t.Fatalf("Expected final chunk to contain some tokens besides bos and eos but got empty")
	}

	lc = lc[1 : len(lc)-1]
	if len(lc) < overlap {
		overlap = len(lc)
	}

	overlapLc := lc[:overlap]
	llc := chunks[len(chunks)-2]
	llc = llc[len(llc)-len(overlapLc)-1 : len(llc)-1]
	if !slices.Equal(overlapLc, llc) {
		t.Fatalf("Expected last overlapping chunks to be equal but received different chunks, last chunk=%v, last last chunk=%v", overlapLc, llc)
	}
}

func TestSemanticChunk(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"     line1     ", 1},
		{"line1\r\nline2\n\n\n\nline3    ", 2},
		{"line1     \ngarret\nline2    \n\n\n\nline3", 2},
		{"line1line2\n\n\n\nline3", 2},
		{"line1\nline2\nline3", 1},
	}

	for idx, tc := range tests {
		sc := semanticChunks(tc.value)
		if len(sc) == tc.expected {
			continue
		}
		t.Errorf("test case %d failed expected len=%d got=%d", idx+1, tc.expected, len(sc))
	}

}
