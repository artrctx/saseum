package pg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"saseum/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Client struct {
	db *sql.DB
}

func New(connStr string) (*Client, error) {
	conn, err := sql.Open("pgx", connStr)
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

func ConnStrFromConfig(c config.DatabaseConfig) string {
	sslMode := c.SslMode
	if sslMode == "" {
		sslMode = "prefer"
	}

	var schema string
	if c.Schema != "" {
		schema = fmt.Sprintf("&search_path=%s", c.Schema)
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s%s", c.Username, c.Password, c.Host, c.Port, c.Database, c.SslMode, schema)
}

func (c *Client) Close() error {
	return c.db.Close()
}
