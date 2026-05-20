package db

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"saseum/internal/db"
	"saseum/internal/db/util"
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

		embedder, err := embed.New(vecCfg.modelID, vecCfg.workers)
		cobra.CheckErr(err)

		embTable, err := serv.Prepare(vecCfg.target, embedder.Dim(), vecCfg.clean)
		cobra.CheckErr(err)

		fmt.Printf("Processing %s table to %s embedding table.\n", vecCfg.target, embTable.Name())

		// TODO: SUPPORT TIME OUT
		count, err := embTable.Sync(context.Background(), embedder, vecCfg.batchSize)
		cobra.CheckErr(err)

		fmt.Printf("Syncing concluded with %d entry.\n", count)
		// if no watch end of execution
		if !vecCfg.watch {
			return
		}
		fmt.Println("Type in your query(Ctrl + C or type q to quit):")
		scanner := bufio.NewScanner(os.Stdin)

		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}

			input := scanner.Text()
			if input == "q" {
				break
			}

			results, err := embTable.Query(embedder, input, vecCfg.limit, vecCfg.threshold)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if len(results) == 0 {
				fmt.Println("------ NO RESULT ------")
			} else {
				for idx, r := range results {
					fmt.Printf("--------RESULTS %d----------\n\n", idx+1)
					fmt.Println(util.MapToReadableStr(r))
					fmt.Print("\n\n-------END RESIULT--------\n")
				}
			}
		}
	},
}

func RegisterPostgresCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(pgCmd)

	registerDatabaseFlags(pgCmd)
	registerVectorizationFlags(pgCmd)
}
