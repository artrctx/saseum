package config

type DatabaseConfig struct {
	Database string
	Username string
	Password string
	Host     string
	Port     uint16
	Schema   string
	SslMode  string
}
