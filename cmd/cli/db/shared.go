package db

import (
	"fmt"
	"saseum/internal/config"
	"saseum/internal/db/pg"
	"saseum/internal/embed"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type DBFlag = string

const (
	ConnStr  DBFlag = "connStr"
	Database DBFlag = "database"
	Username DBFlag = "username"
	Host     DBFlag = "host"
	Port     DBFlag = "port"
	Schema   DBFlag = "schema"
	SslMode  DBFlag = "sslMode"
)

// made it postgres specific might want to change when mysql gets implemnetd
func registerDatabaseFlags(cmd *cobra.Command) {
	// connStr will take priority
	cmd.Flags().String(ConnStr, "", "Connection string to postgres database")

	cmd.Flags().StringP(Database, "d", "postgres", "Database name")
	cmd.Flags().StringP(Username, "u", "postgres", "Database username")
	// cmd.MarkFlagRequired(Username)
	cmd.Flags().StringP(Host, "h", "localhost", "Database host")
	cmd.Flags().Uint16P(Port, "p", 5432, "Database port")
	cmd.Flags().StringP(Schema, "s", "public", "Database port")
	cmd.Flags().String(SslMode, "prefer", "Database ssl mode")
}

func connStrFromFlags(cmd *cobra.Command) (string, error) {
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
			return "", fmt.Errorf("Invalid database configuration. host=%s, port=%d, username=%s, database=%s", database, port, username, database)
		}

		// ask password
		fmt.Print("Enter Password:")
		pswd, err := term.ReadPassword(syscall.Stdin)
		// adding line break after password enter
		fmt.Print("\n")
		if err != nil {
			return "", fmt.Errorf("Failed reading password error=%s", err)
		}

		connStr = pg.ConnStrFromConfig(
			config.DatabaseConfig{
				Database: database,
				Username: username,
				Password: string(pswd),
				Host:     host,
				Port:     port,
				Schema:   schema,
				SslMode:  sslMode,
			})
	}

	return connStr, nil
}

type ExecOpt struct {
	modelID   embed.ModelID
	workers   uint8
	target    string
	clean     bool
	watch     bool
	limit     uint8
	batchSize int
	threshold float32
}

func registerVectorizationFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("model", "m", embed.E5BaseV2.ID, fmt.Sprintf("Embedding model to use. (supports: %s | %s | %s)", embed.E5BaseV2.ID, embed.E5LargeV2.ID, embed.AllMiniLM.ID))
	cmd.Flags().Uint8("workers", 4, "inference worker count")

	cmd.Flags().StringP("target", "t", "", "Target database table to be vectorized")
	cmd.MarkFlagRequired("target")

	cmd.Flags().BoolP("clean", "c", false, "Truncate and recreate table if already exists")
	cmd.Flags().BoolP("watch", "w", false, "Start testing interface after processing")
	cmd.Flags().Uint8("limit", 3, "Resuilt limit while watching")
	cmd.Flags().Int("batchSize", 20, "Processing batch size")
	cmd.Flags().Float32("threshold", 0, "Threshold to be used while querying (between 0 to 1)")
}

func vectorizationConfigFromFlags(cmd *cobra.Command) (*ExecOpt, error) {
	flags := cmd.Flags()

	model, err := flags.GetString("model")
	if err != nil {
		return nil, err
	}

	var modelID embed.ModelID
	switch model {
	case embed.E5BaseV2.ID:
		modelID = embed.E5BaseV2
	case embed.E5LargeV2.ID:
		modelID = embed.E5LargeV2
	case embed.AllMiniLM.ID:
		modelID = embed.AllMiniLM
	default:
		return nil, fmt.Errorf("provided model (%s) is not supported", model)
	}

	workers, err := flags.GetUint8("workers")
	if err != nil {
		return nil, err
	}

	target, err := flags.GetString("target")
	if target == "" {
		return nil, fmt.Errorf("expected target to be provided but got=%s", target)
	}

	clean, err := flags.GetBool("clean")
	if err != nil {
		return nil, err
	}

	watch, err := flags.GetBool("watch")
	if err != nil {
		return nil, err
	}
	limit, err := flags.GetUint8("limit")
	if err != nil {
		return nil, err
	}
	batchSize, err := flags.GetInt("batchSize")
	if err != nil {
		return nil, err
	}
	threshold, err := flags.GetFloat32("threshold")
	if err != nil {
		return nil, err
	}

	return &ExecOpt{modelID, workers, target, clean, watch, limit, batchSize, threshold}, nil
}
