package pg

import (
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Client struct {
	db *sqlx.DB
}

func New(connStr string) (*Client, error) {
	conn, err := sqlx.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	// https://github.com/pgvector/pgvector/tree/master
	// this checks connection and if pgvector is installed
	if _, err := conn.Exec("CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed closing postgres db conn after ping fail", slog.Any("error", err))
		}
		return nil, err
	}

	return &Client{conn}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

type ColumnInfo struct {
	Name       string `db:"column_name"`
	DataType   string `db:"data_type"`
	IsNullable bool   `db:"is_nullable"`
	IsPK       bool   `db:"is_pk"`
}

// returns created vector table name or error
func (c *Client) Prepare(target string, vecSize int) (string, error) {
	schema, err := c.getCurrentSchema()
	if err != nil {
		return "", err
	}
	rows, err := c.db.Queryx(
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
    ORDER BY c.ordinal_position;`, schema, target)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		err := rows.StructScan(&col)
		if err != nil {
			return "", err
		}
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("target table does not exists")
	}
	fmt.Println(cols)
	// vecTableName := fmt.Sprintf("%s_vector_", target)
	// vecTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (embedding vector(%d))`, vecTableName, vecSize)
	return "", nil
}

func (c *Client) getCurrentSchema() (string, error) {
	var schema string
	if err := c.db.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		return "", err
	}
	return schema, nil
}
