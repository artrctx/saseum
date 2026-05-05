package db

type Config struct {
	Database string
	Username string
	Password string
	Host     string
	Port     uint16
	Schema   string
	SslMode  string
}

func ConfigFromConnStr(connStr string) Config {
	return Config{}
}
