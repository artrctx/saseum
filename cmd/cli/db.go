package main

import "github.com/spf13/cobra"

func registerDatabaseConfig(cmd *cobra.Command) {
	// connStr will take priority
	cmd.Flags().String("connStr", "", "Connection string to postgres database")

	cmd.Flags().StringP("database", "d", "", "Database name")
	cmd.Flags().StringP("username", "u", "", "Database username")
	cmd.Flags().StringP("password", "p", "", "Database password")
	cmd.Flags().StringP("host", "h", "", "Database host")
}
