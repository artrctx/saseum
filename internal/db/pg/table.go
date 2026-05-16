package pg

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type EmbeddingTable struct {
	// src table name
	src  string
	name string
	db   *sqlx.DB
}

func (et *EmbeddingTable) Name() string {
	return et.name
}

func (et *EmbeddingTable) SrcName() string {
	return et.src
}

func (et *EmbeddingTable) DeleteWithTx(tx *sql.Tx) error {
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", et.name)); err != nil {
		return err
	}
	return nil
}

func (et *EmbeddingTable) Migrate() error {
	return nil
}
