package db

import (
	"fmt"
	"io"
	"log/slog"
	"saseum/internal/db/pg"
	"saseum/internal/db/util"
	"strings"
)

type client interface {
	io.Closer
	Prepare(target string, vecSize int, clean bool) (util.EmbeddingTable, error)
}

type Service struct {
	client
}

func New(connStr string) (*Service, error) {
	connSrc := strings.Split(connStr, "://")[0]

	var client client
	var err error
	switch connSrc {
	case "postgresql", "postgres":
		client, err = pg.New(connStr)
		// TODO: Add in mysql support when ready
		// NOTE: Might be able to combine all in one service
		// "github.com/go-sql-driver/mysql"
	case "mysql":
	default:
		return nil, fmt.Errorf("%s is currently not supported", connSrc)
	}

	if err != nil {
		return nil, err
	}

	return &Service{client}, nil
}

func (s *Service) Close() {
	if err := s.client.Close(); err != nil {
		slog.Error("Failed to close db connection", slog.Any("error", err))
	}
}
