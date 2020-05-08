package main

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func dropAllTablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop-all-tables",
		Short: "Drop all tables",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, err := dbNameFromFlagEnv()
			if err != nil {
				return err
			}
			log.Printf("db: %s\n", dbName)

			ctx := context.Background()

			admin, cli := createClients(ctx, dbName)
			tables, err := query(ctx, cli, "SELECT TABLE_NAME FROM information_schema.tables WHERE TABLE_CATALOG = '' AND TABLE_SCHEMA = ''")
			if err != nil {
				return err
			}
			for _, t := range tables.rows {
				indexes, err := query(ctx, cli, "SELECT INDEX_NAME FROM information_schema.indexes WHERE INDEX_NAME != 'PRIMARY_KEY' AND TABLE_NAME='"+t[0]+"'")
				if err != nil {
					return err
				}
				for _, i := range indexes.rows {
					log.Printf("Dropping index: %s\n", i[0])
					if err := updateDatabaseDdl(ctx, os.Stdout, admin, dbName, "DROP INDEX "+i[0]); err != nil {
						return err
					}
				}
				log.Printf("Dropping table: %s\n", t[0])
				if err := updateDatabaseDdl(context.Background(), os.Stdout, admin, dbName, "DROP TABLE "+t[0]); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
