package main

import (
	"fmt"
	"saseum/internal/db"

	"github.com/spf13/cobra"
)

var pgCmd = &cobra.Command{
	Use:   "postgres",
	Short: "Postgres vector implementation",
	Long:  `Implement vector implementation with pgvector.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pgCmd")
	},
}

func init() {
	rootCmd.AddCommand(pgCmd)

	defaultCfg := db.Config{
		Database: "postgres",
		Username: "postgres",
		Host:     "localshot",
		Port:     5432,
	}

	registerDatabaseConfig(pgCmd, defaultCfg)
}
