package db

import "fmt"

type Config struct {
	Database string
	Username string
	Password string
	Host     string
	Port     uint16
	Schema   string
	SslMode  string
}

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s", c.Username, c.Password, c.Host, c.Port, c.Database, c.SslMode, c.Schema)
}
