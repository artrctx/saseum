package main

import (
	"github.com/spf13/cobra"
)

// https://github.com/charmbracelet/bubbletea/tree/main
// https://github.com/spf13/cobra
// https://github.com/pgvector/pgvector/tree/master
// https://www.youtube.com/watch?v=a4HBKEda_F8&t=15s
// https://cobra.dev/docs/examples/02-task-manager/

var rootCmd = &cobra.Command{
	Use:   "saseum",
	Short: "Implement search vector for using your own thing",
	Long:  `Implement search vector for using your own thing`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
