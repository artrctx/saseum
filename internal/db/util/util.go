package util

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"saseum/internal/embed"
	"strings"
)

type EmbeddingTable interface {
	Name() string
	DeleteWithTx(tx *sql.Tx) error
	Sync(ctx context.Context, emb *embed.Embedder) (count int, err error)
}

func MapToReadableStr(m map[string]any) string {
	rs := make([]string, len(m))
	idx := 0
	for k, v := range m {
		rs[idx] = fmt.Sprintf("%s:%s", k, anyToJSONString(v))
		idx++
	}
	return strings.Join(rs, "\n")
}

func anyToJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
