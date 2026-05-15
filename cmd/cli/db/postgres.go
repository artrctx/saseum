package db

import (
	"fmt"
	"saseum/internal/db"
	"saseum/internal/embed"

	"github.com/spf13/cobra"
)

var pgCmd = &cobra.Command{
	Use:   "pg",
	Short: "Postgres pgvector impl",
	Long: `Postgres vectorization implementation.

Utilizes pgvector to created vectordb.
This command will vectorize target table and create 
new table with embedding that will reference original table's PK.

e.g.) {OG_TABLE}_vector_`,
	Run: func(cmd *cobra.Command, args []string) {
		connStr, err := connStrFromFlags(cmd)
		cobra.CheckErr(err)
		vecCfg, err := vectorizationConfigFromFlags(cmd)
		cobra.CheckErr(err)

		serv, err := db.New(connStr)
		cobra.CheckErr(err)
		defer serv.Close()

		embedder, err := embed.New(vecCfg.modelID)

		vecTable, err := serv.Prepare(vecCfg.target, embedder.Dim(), vecCfg.clean)
		cobra.CheckErr(err)

		fmt.Println(vecTable.Name(), "table name")
	},
}

func RegisterPostgresCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(pgCmd)

	registerDatabaseFlags(pgCmd)
	registerVectorizationFlags(pgCmd)
}
