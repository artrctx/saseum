package db

import (
	"fmt"
	"log/slog"
	"os"
	"saseum/internal/db"

	"github.com/spf13/cobra"
)

var pgCmd = &cobra.Command{
	Use:   "postgres",
	Short: "Postgres vector implementation",
	Long:  `Implement vector implementation with pgvector.`,
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		connStr, _ := flags.GetString(ConnStr)
		var cfg db.Config
		if connStr != "" {
			cfg = db.ConfigFromConnStr(connStr)
		} else {
			database, _ := flags.GetString(Database)
			username, _ := flags.GetString(Username)
			host, _ := flags.GetString(Host)
			port, _ := flags.GetUint16(Port)
			schema, _ := flags.GetString(Schema)
			sslMode, _ := flags.GetString(SslMode)

			if database == "" || username == "" || host == "" || port == 0 {
				slog.Error("Invalid database configuration", slog.String("database", database), slog.String("username", username), slog.String("host", host), slog.Int("port", int(port)))
				os.Exit(1)
			}
			// ask password

			cfg = db.Config{
				Database: database,
				Username: username,
				// add password
				Host:    host,
				Port:    port,
				Schema:  schema,
				SslMode: sslMode,
			}
		}

		fmt.Println(cfg)
	},
}

func RegisterPostgresCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(pgCmd)

	registerDatabaseConfig(pgCmd)
}
