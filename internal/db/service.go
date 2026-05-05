package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Source = string

const (
	Postgres Source = "postgres"
	MySQL    Source = "mysql"
)

type Service struct {
	src Source
	db  *sql.DB
}

func New(connStr string) (*Service, error) {
	connSrc := strings.Split(connStr, "://")[0]
	var driver string
	var source Source
	switch connSrc {
	case "postgresql":
		driver = "pgx"
		source = Postgres

	// TODO: Add in mysql support when ready
	// "github.com/go-sql-driver/mysql"
	// case "mysql":
	// 	adapter = "mysql"
	default:
		return nil, fmt.Errorf("%s is currently not supported", connSrc)
	}

	conn, err := sql.Open(driver, connStr)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed closing db conn after ping fail", slog.Any("error", err))
		}
		return nil, err
	}

	return &Service{source, conn}, nil
}

func (s *Service) Close() {
	if err := s.db.Close(); err != nil {
		slog.Error("Failed to close db connection", slog.Any("error", err))
	}
}
