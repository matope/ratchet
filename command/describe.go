package command

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/spf13/cobra"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

func describeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe",
		Short: "Show Database DDLs.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, err := dbNameFromFlagEnv()
			if err != nil {
				return err
			}
			log.Printf("db: %s\n", dbName)
			if v := os.Getenv("SPANNER_EMULATOR_HOST"); v != "" {
				log.Printf("SPANNER_EMULATOR_HOST: %s\n", v)
			}
			log.Println()

			admin, _ := createClients(context.Background(), dbName)
			return describe(context.Background(), os.Stdout, admin, dbName)
		},
	}
}

func describe(ctx context.Context, w io.Writer, adminClient *database.DatabaseAdminClient, dbName string) error {
	ddl, err := adminClient.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
		Database: dbName,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Found %d DDL(s)\n", len(ddl.Statements))

	for _, d := range ddl.Statements {
		fmt.Fprintf(w, "\n%s\n", d)
	}
	return nil
}
