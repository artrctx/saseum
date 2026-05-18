package util

import (
	"context"
	"database/sql"
	"fmt"
	"saseum/internal/embed"
	"strings"
)

type EmbeddingTable interface {
	Name() string
	DeleteWithTx(tx *sql.Tx) error
	Sync(ctx context.Context, emb *embed.Embedder) (count int64, err error)
	Query(emb *embed.Embedder, text string, limit uint8) ([]map[string]any, error)
}

func MapToReadableStr(m map[string]any) string {
	rs := make([]string, len(m))
	idx := 0
	for k, v := range m {
		rs[idx] = fmt.Sprintf("%s:%v", k, ToReadableDBValue(v))
		idx++
	}
	return strings.Join(rs, "\n")
}

func ToReadableDBValue(val any) any {
	switch v := val.(type) {
	case string, int, int64, bool:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func ToValidDBValue(val any) (any, error) {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", v), nil
	case []byte:
		return string(v), nil
	case int, int64, bool:
		return v, nil
	case nil:
		return sql.NullString{}, nil
	default:
		return nil, fmt.Errorf("unsupported database datatype: %T", v)
	}
}

func ToEmbeddingStr(e []float32) string {
	var es strings.Builder
	for idx, v := range e {
		if idx > 0 {
			es.WriteString(",")
		}
		es.WriteString(fmt.Sprintf("%f", v))
	}
	return fmt.Sprintf("[%s]", es.String())
}
