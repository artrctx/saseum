package main

import (
	"saseum/cmd/cli/db"

	"github.com/spf13/cobra"
)

// https://github.com/charmbracelet/bubbletea/tree/main
// https://github.com/spf13/cobra
// https://github.com/pgvector/pgvector/tree/master
// https://www.youtube.com/watch?v=a4HBKEda_F8&t=15s
// https://cobra.dev/docs/examples/02-task-manager/

var rootCmd = &cobra.Command{
	Use:   "saseum",
	Short: "Saseum means Deer.",
	Long: `Saseum means Deer. 
It does what deer does. Eating grass and jumping over cars and stuff.`,
}

func init() {
	db.RegisterPostgresCommand(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
