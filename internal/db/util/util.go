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
	Sync(ctx context.Context, emb *embed.Embedder) (count int64, err error)
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

func ToJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
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
	case string, int, int64, bool:
		return v, nil

	case []byte:
		return string(v), nil

	case nil:
		return sql.NullString{}, nil

	default:
		return nil, fmt.Errorf("unsupported database datatype: %T", v)
	}
}
func ToValidInsertValue(val any) (any, error) {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", v), nil
	case int, int64, bool:
		return v, nil
	case nil:
		return sql.NullString{}, nil
	default:
		return nil, fmt.Errorf("unsupported database datatype: %T", v)
	}
}
