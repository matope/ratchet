package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"log"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

type row []string
type result struct {
	cols []string
	rows []row
}

func execCmd() *cobra.Command {
	var flgFile string
	var flgForce bool
	execCmd := &cobra.Command{
		Use:   "exec [flags] [SQL]",
		Short: "Throw specified SQL(s) to Cloud Spanner.",
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

			admin, cli := createClients(context.Background(), dbName)
			for _, sql := range sqls {
				if err := exec(context.Background(), os.Stdout, admin, cli, dbName, sql); err != nil {
					if flgForce {
						log.Printf("exec error: %s\n", err.Error())
					} else {
						return err
					}
				}
			}
			return nil
		},
	}
	execCmd.Flags().StringVarP(&flgFile, "file", "f", "", "filepath which contains SQL(s). If '-', read from STDIN.")
	execCmd.Flags().BoolVarP(&flgForce, "force", "", false, "ignore execution error.")
	return execCmd
}

func exec(ctx context.Context, w io.Writer, admin *database.DatabaseAdminClient, cli *spanner.Client, dbName, sql string) error {
	log.Printf("sql:%s\n", sql)
	defer log.Println()

	switch token := strings.ToUpper(strings.Split(sql, " ")[0]); token {
	case "SELECT":
		res, err := query(ctx, cli, sql)
		if err != nil {
			return err
		}
		printTable(w, res)
		return nil
	case "INSERT", "DELETE", "UPDATE":
		return dml(ctx, w, cli, sql)
	case "CREATE", "ALTER", "DROP":
		return updateDatabaseDdl(ctx, w, admin, dbName, sql)
	default:
		return fmt.Errorf("unsupported SQL statement: [%s]", sql)
	}
}

func updateDatabaseDdl(ctx context.Context, w io.Writer, adminClient *database.DatabaseAdminClient, dbName, sql string) error {
	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   dbName,
		Statements: []string{sql},
	})
	if err != nil {
		return err
	}
	if err := op.Wait(ctx); err != nil {
		return err
	}
	fmt.Fprintf(w, "finished.\n")
	return nil
}

func dml(ctx context.Context, w io.Writer, client *spanner.Client, sql string) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{SQL: sql}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) affected.\n", rowCount)
		return nil
	})
	return err
}

func query(ctx context.Context, client *spanner.Client, sql string) (*result, error) {
	stmt := spanner.Statement{SQL: sql}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var res result
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		res.cols = row.ColumnNames()
		size := row.Size()
		fields := make([]string, 0, size)
		for i := 0; i < size; i++ {
			var v spanner.GenericColumnValue
			if err := row.Column(i, &v); err != nil {
				return nil, err
			}
			fields = append(fields, v.Value.GetStringValue())
		}
		res.rows = append(res.rows, fields)
	}
	return &res, nil
}

func printTable(w io.Writer, res *result) {
	if len(res.rows) == 0 {
		return
	}
	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(res.cols)
	for _, row := range res.rows {
		cols := make([]string, 0, len(row))
		for _, row := range row {
			cols = append(cols, runewidth.Wrap(row, tablewriter.MAX_ROW_WIDTH))
		}
		table.Append(cols)
	}
	table.Render()
	log.Printf("%d record(s) found.\n", len(res.rows))
}
