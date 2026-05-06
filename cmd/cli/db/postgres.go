package db

import (
	"saseum/internal/db"

	"github.com/spf13/cobra"
)

var pgCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres vector implementation",
	Long:  `Implement vector implementation with pgvector.`,
	Run: func(cmd *cobra.Command, args []string) {
		connStr, err := connStrFromFlag(cmd)
		cobra.CheckErr(err)

		serv, err := db.New(connStr)
		cobra.CheckErr(err)
		defer serv.Close()
	},
}

func RegisterPostgresCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(pgCmd)

	registerDatabaseConfig(pgCmd)
}
