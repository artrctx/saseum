package db

import (
	"fmt"
	"log/slog"
	"os"
	"saseum/internal/db"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var pgCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres vector implementation",
	Long:  `Implement vector implementation with pgvector.`,
	Run: func(cmd *cobra.Command, args []string) {
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
				slog.Error("Invalid database configuration", slog.String("database", database), slog.String("username", username), slog.String("host", host), slog.Int("port", int(port)))
				os.Exit(1)
			}
			// ask password
			fmt.Print("Enter Password:")
			bs, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				slog.Error("Failed reading password", slog.Any("error", err))
				os.Exit(1)
			}

			cfg := db.Config{
				Database: database,
				Username: username,
				Password: string(bs),
				Host:     host,
				Port:     port,
				Schema:   schema,
				SslMode:  sslMode,
			}
			connStr = cfg.ConnStr()
		}

		conn, err := db.New(connStr)
		if err != nil {
			slog.Error("Failed creating db connection", slog.Any("error", err))
			os.Exit(1)
		}
		defer conn.Close()
	},
}

func RegisterPostgresCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(pgCmd)

	registerDatabaseConfig(pgCmd)
}
