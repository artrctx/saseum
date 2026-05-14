package pg

import (
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
