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

type TableMeta struct {
}

// returns created vector table name or error
func (c *Client) Prepare(target string) (string, error) {
	pkQuery := `
    SELECT 
        c.column_name, 
        c.data_type, 
        c.is_nullable,
        CASE WHEN pk.column_name IS NOT NULL THEN 'YES' ELSE 'NO' END as is_pk
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
    ORDER BY c.ordinal_position;`

	rows, err := c.db.Query(pkQuery, "postgres", target)
	if err != nil {
		return "", err
	}

	fmt.Println("\nROW----")
	for rows.Next() {
		rowVal := make(map[string]any)
		err := rows.Scan(rowVal)
		if err != nil {
			return "", err
		}
		fmt.Println(rowVal, "no count?")
	}
	return "", nil
}

func (c *Client) Close() error {
	return c.db.Close()
}
