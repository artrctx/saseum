package util

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Table struct {
	name string
	db   *sqlx.DB
}

func NewTable(name string, db *sqlx.DB) *Table {
	return &Table{name, db}
}

func (t *Table) Name() string {
	return t.name
}

func (t *Table) Delete(tx *sql.Tx) error {
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", t.name)); err != nil {
		return err
	}
	return nil
}
