package db

import (
	"github.com/spf13/cobra"
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

	cmd.Flags().StringP(Database, "D", "", "Database name")
	cmd.Flags().StringP(Username, "U", "", "Database username")
	cmd.Flags().StringP(Host, "H", "", "Database host")
	cmd.Flags().Uint16P(Port, "P", 0, "Database port")
	cmd.Flags().StringP(Schema, "S", "", "Database port")
	cmd.Flags().String(SslMode, "prefer", "Database ssl mode")
}
