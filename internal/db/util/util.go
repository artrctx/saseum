package util

import (
	"database/sql"
)

type EmbeddingTable interface {
	Name() string
	DeleteWithTx(tx *sql.Tx) error
}
