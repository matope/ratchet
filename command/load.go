package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func loadCmd() *cobra.Command {
	var flgFile string
	var flgForce bool
	loadCmd := &cobra.Command{
		Use:   "load [flag] [SQL]",
		Short: "Load applies DDLs to Cloud Spanner using a single transaction. If you apply many DDLs, 'load' would be way much faster than 'exec'.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, err := dbNameFromFlagEnv()
			if err != nil {
				return err
			}

			var sqls []string
			if len(args) > 2 && flgFile != "" {
				return errors.New("Specify eather arg(sql) or --file")
			} else if flgFile != "" {
				if sqls, err = parseFile(flgFile); err != nil {
					return err
				}
			} else if len(args) > 0 {
				sqls = parseSqls(args[0])
			} else {
				return errors.New("Specify either arg(sql) or --file")
			}

			log.Printf("db: %s\n", dbName)
			if v := os.Getenv("SPANNER_EMULATOR_HOST"); v != "" {
				log.Printf("SPANNER_EMULATOR_HOST: %s\n", v)
			}
			log.Println()

			for _, sql := range sqls {
				switch token := strings.ToUpper(strings.Split(sql, " ")[0]); token {
				case "CREATE", "ALTER", "DROP":
				default:
					return fmt.Errorf("All SQL should be DDL for load [%s]", sql)
				}
			}

			admin, _ := createClients(context.Background(), dbName)
			return updateDatabaseDdls(context.Background(), os.Stdout, admin, dbName, sqls...)
		},
	}
	loadCmd.Flags().StringVarP(&flgFile, "file", "f", "", "filepath which contains SQL(s). If '-', read from STDIN.")
	loadCmd.Flags().BoolVarP(&flgForce, "force", "", false, "ignore execution error.")
	return loadCmd
}
