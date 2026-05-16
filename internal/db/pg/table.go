package pg

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type EmbeddingTable struct {
	// src table name
	srcName string
	name    string
	db      *sqlx.DB
}

func (et *EmbeddingTable) Name() string {
	return et.name
}

func (et *EmbeddingTable) DeleteWithTx(tx *sql.Tx) error {
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", et.name)); err != nil {
		return err
	}
	return nil
}

// syncs embedding between src
func (et *EmbeddingTable) Sync() error {
	etMap, err := et.GetSourceTableMap()
	if err != nil {
		return err
	}
	if len(etMap) == 0 {
		return fmt.Errorf("embedding table(%s) and src table(%s) doesn't have any foreign relation", et.name, et.srcName)
	}
	//TODO: pull src table rows should chunk and insert into embedding with embedder

	return nil
}

type EmbeddingTableMap struct {
	ForeignKeyName string `db:"foreign_key_name"`
	// embedding table
	SrcTable  string `db:"source_table"`
	SrcColumn string `db:"source_column"`
	// source table
	PrimaryTable  string `db:"primary_table"`
	PrimaryColumn string `db:"primary_column"`
}

func (et *EmbeddingTable) GetSourceTableMap() ([]EmbeddingTableMap, error) {
	rows, err := et.db.Queryx(`SELECT 
    rc.constraint_name AS foreign_key_name,
    kcu1.table_name AS source_table,
    kcu1.column_name AS source_column,
    kcu2.table_name AS primary_table,
    kcu2.column_name AS primary_column
FROM information_schema.referential_constraints rc
JOIN information_schema.key_column_usage kcu1 ON rc.constraint_name = kcu1.constraint_name
JOIN information_schema.key_column_usage kcu2 ON rc.unique_constraint_name = kcu2.constraint_name
    AND kcu1.ordinal_position = kcu2.ordinal_position
WHERE kcu1.table_name = $1;`, et.name)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	embeddingTableMap := []EmbeddingTableMap{}
	for rows.Next() {
		var eMap EmbeddingTableMap
		if err := rows.StructScan(&eMap); err != nil {
			return nil, err
		}

		if eMap.PrimaryTable != et.srcName {
			continue
		}

		embeddingTableMap = append(embeddingTableMap, eMap)
	}
	return embeddingTableMap, nil
}
