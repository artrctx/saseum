package pg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"saseum/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// https://github.com/pgvector/pgvector/tree/master
type Client struct {
	db *sql.DB
}

func New(connStr string) (*Client, error) {
	conn, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed closing postgres db conn after ping fail", slog.Any("error", err))
		}
		return nil, err
	}

	return &Client{conn}, nil
}

func ConnStrFromConfig(c config.DatabaseConfig) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s", c.Username, c.Password, c.Host, c.Port, c.Database, c.SslMode, c.Schema)
}

func (c *Client) Close() error {
	return c.db.Close()
}
