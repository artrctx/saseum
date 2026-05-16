package pg

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ColumnInfo struct {
	Name       string `db:"column_name"`
	DataType   string `db:"data_type"`
	IsNullable bool   `db:"is_nullable"`
	IsPK       bool   `db:"is_pk"`
}

func getColumnInfos(db *sqlx.DB, table string) ([]ColumnInfo, error) {
	schema, err := currentSchema(db)
	if err != nil {
		return nil, err
	}
	rows, err := db.Queryx(
		`SELECT c.column_name, c.data_type, (c.is_nullable::boolean) as is_nullable,
        (pk.column_name IS NOT NULL) as is_pk
    FROM information_schema.columns c
    LEFT JOIN (
        SELECT ku.table_schema, ku.table_name, ku.column_name
        FROM information_schema.key_column_usage ku
        JOIN information_schema.table_constraints tc 
          ON ku.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'PRIMARY KEY'
    ) pk ON c.table_schema = pk.table_schema 
        AND c.table_name = pk.table_name 
        AND c.column_name = pk.column_name
    WHERE c.table_schema = $1 AND c.table_name = $2
    ORDER BY c.ordinal_position;`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		err := rows.StructScan(&col)
		if err != nil {
			return nil, err
		}

		cols = append(cols, col)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("target table does not exists")
	}
	return cols, nil
}

func getPrimaryColumnInfos(db *sqlx.DB, table string) ([]ColumnInfo, error) {
	schema, err := currentSchema(db)
	if err != nil {
		return nil, err
	}
	rows, err := db.Queryx(
		`SELECT c.column_name, c.data_type, (c.is_nullable::boolean) as is_nullable,
        (pk.column_name IS NOT NULL) as is_pk
    FROM information_schema.columns c
    LEFT JOIN (
        SELECT ku.table_schema, ku.table_name, ku.column_name
        FROM information_schema.key_column_usage ku 
        JOIN information_schema.table_constraints tc 
          ON ku.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'PRIMARY KEY'
    ) pk ON c.table_schema = pk.table_schema 
        AND c.table_name = pk.table_name 
        AND c.column_name = pk.column_name
    WHERE pk.column_name IS NOT NULL AND c.table_schema = $1 AND c.table_name = $2
    ORDER BY c.ordinal_position;`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	primaryCols := []ColumnInfo{}
	for rows.Next() {
		var col ColumnInfo
		err := rows.StructScan(&col)
		if err != nil {
			return nil, err
		}
		primaryCols = append(primaryCols, col)
	}

	if len(primaryCols) == 0 {
		return nil, fmt.Errorf("target table does not exists or does not have primary keys")
	}

	return primaryCols, nil
}

func currentSchema(db *sqlx.DB) (string, error) {
	var schema string
	if err := db.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		return "", err
	}
	return schema, nil
}
