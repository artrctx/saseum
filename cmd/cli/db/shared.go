package db

import (
	"fmt"
	"saseum/internal/config"
	"saseum/internal/db/pg"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type DBFlag = string

const (
	ConnStr  DBFlag = "connStr"
	Database DBFlag = "database"
	Username DBFlag = "username"
	Host     DBFlag = "host"
	Port     DBFlag = "port"
	Schema   DBFlag = "schema"
	SslMode  DBFlag = "sslMode"
)

func registerDatabaseConfig(cmd *cobra.Command) {
	// connStr will take priority
	cmd.Flags().String(ConnStr, "", "Connection string to postgres database")

	cmd.Flags().StringP(Database, "d", "", "Database name")
	cmd.Flags().StringP(Username, "u", "", "Database username")
	cmd.Flags().StringP(Host, "H", "", "Database host")
	cmd.Flags().Uint16P(Port, "p", 0, "Database port")
	cmd.Flags().StringP(Schema, "s", "", "Database port")
	cmd.Flags().String(SslMode, "prefer", "Database ssl mode")
}

func connStrFromFlag(cmd *cobra.Command) (string, error) {
	flags := cmd.Flags()

	connStr, _ := flags.GetString(ConnStr)
	if connStr == "" {
		database, _ := flags.GetString(Database)
		username, _ := flags.GetString(Username)
		host, _ := flags.GetString(Host)
		port, _ := flags.GetUint16(Port)
		schema, _ := flags.GetString(Schema)
		sslMode, _ := flags.GetString(SslMode)

		if database == "" || username == "" || host == "" || port == 0 {
			return "", fmt.Errorf("Invalid database configuration. host=%s, port=%d, username=%s, database=%s", database, port, username, database)
		}

		// ask password
		fmt.Print("Enter Password:")
		bs, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return "", fmt.Errorf("Failed reading password error=%s", err)
		}

		connStr = pg.ConnStrFromConfig(
			config.DatabaseConfig{
				Database: database,
				Username: username,
				Password: string(bs),
				Host:     host,
				Port:     port,
				Schema:   schema,
				SslMode:  sslMode,
			})
	}

	return connStr, nil
}
