package pg

import (
	"fmt"
	"saseum/internal/config"
)

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
