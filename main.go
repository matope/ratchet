package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

var (
	flgProject  string
	flgInstance string
	flgDatabase string
)

func main() {
	log.SetFlags(0)
	rootCmd().Execute()
}

func rootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "ratchet",
		Long: `A simple client CLI for Cloud Spanner

ratchet is a simple CLI tool for accessing Cloud Spanner. This tool allows you to
throw queries to the Cloud Spanner (or an emulator of that).  If SPANNER_EMULATOR_HOST
env is set, ratchet uses it.`,
		SilenceUsage: true,
	}
	rootCmd.PersistentFlags().StringVarP(&flgInstance, "instance", "i", "", "your-instance-id. (you can also set by `SPANNER_INSTANCE_ID`)")
	rootCmd.PersistentFlags().StringVarP(&flgProject, "project", "p", "", "your-project-id. (you can also set by `SPANNER_PROJECT_ID`)")
	rootCmd.PersistentFlags().StringVarP(&flgDatabase, "database", "d", "", "your-database-id. (you can also set by `SPANNER_DATABASE_ID`)")
	rootCmd.AddCommand(execCmd())
	rootCmd.AddCommand(describeCmd())
	return rootCmd
}

func execCmd() *cobra.Command {
	var flgFile string
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
					return err
				}
			}
			return nil
		},
	}
	execCmd.Flags().StringVarP(&flgFile, "file", "f", "", "filepath which contains SQL(s). If '-', read from STDIN.")
	return execCmd
}

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

func dbNameFromFlagEnv() (string, error) {
	flgOrEnv := func(flg, env string) string {
		if flg != "" {
			return flg
		}
		return env
	}
	var (
		project  = flgOrEnv(flgProject, os.Getenv("SPANNER_PROJECT_ID"))
		instance = flgOrEnv(flgInstance, os.Getenv("SPANNER_INSTANCE_ID"))
		database = flgOrEnv(flgDatabase, os.Getenv("SPANNER_DATABASE_ID"))
	)
	if project == "" {
		return "", errors.New("Please specify project by -p, --project or SPANNER_PROJECT_ID")
	}
	if instance == "" {
		return "", errors.New("Please specify instance by -i, --instance or SPANNER_INSTANCE_ID")
	}
	if database == "" {
		return "", errors.New("Please specify database by -s, --database or SPANNER_DATABASE_ID")
	}
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database), nil
}

func parseFile(path string) ([]string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(path)
		fmt.Println(os.Getwd())
		return nil, err
	}
	return parseSqls(string(content)), nil
}

func parseSqls(s string) []string {
	sqls := make([]string, 0)
	for _, v := range strings.Split(s, ";") {
		if sql := strings.TrimSpace(v); sql != "" {
			sqls = append(sqls, sql)
		}
	}
	return sqls
}

func createClients(ctx context.Context, db string, opts ...option.ClientOption) (*database.DatabaseAdminClient, *spanner.Client) {
	adminClient, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		log.Fatal(err)
	}

	dataClient, err := spanner.NewClient(ctx, db, opts...)
	if err != nil {
		log.Fatal(err)
	}
	return adminClient, dataClient
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

func exec(ctx context.Context, w io.Writer, admin *database.DatabaseAdminClient, cli *spanner.Client, dbName, sql string) error {
	log.Printf("sql:%s\n", sql)
	defer log.Println()

	switch token := strings.ToUpper(strings.Split(sql, " ")[0]); token {
	case "SELECT":
		return query(ctx, w, cli, sql)
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
	fmt.Fprintf(w, "updatad database.\n")
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

func query(ctx context.Context, w io.Writer, client *spanner.Client, sql string) error {
	stmt := spanner.Statement{SQL: sql}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var colNames []string
	var rows [][]string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		colNames = row.ColumnNames()
		size := row.Size()
		cols := make([]string, 0, size)
		for i := 0; i < size; i++ {
			var v spanner.GenericColumnValue
			if err := row.Column(i, &v); err != nil {
				return err
			}
			cols = append(cols, v.Value.GetStringValue())
		}
		rows = append(rows, cols)
	}
	printTable(w, colNames, rows)
	log.Printf("%d record(s) found.\n", len(rows))
	return nil
}

func printTable(w io.Writer, colNames []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(colNames)
	for _, row := range rows {
		cols := make([]string, 0, len(row))
		for _, row := range row {
			cols = append(cols, runewidth.Wrap(row, tablewriter.MAX_ROW_WIDTH))
		}
		table.Append(cols)
	}
	table.Render()
}
