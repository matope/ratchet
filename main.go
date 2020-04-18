package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/spanner"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

var (
	flgProject  = flag.String("project", "fake", "project name")
	flgInstance = flag.String("instance", "fake", "instance name")
	flgDatabase = flag.String("database", "fake", "database name")
	flgSql      = flag.String("sql", "", "SQL statement")
	flgFile     = flag.String("file", "", "filepath which contains sqls")
)

func main() {
	flag.Parse()

	if *flgFile != "" && *flgSql != "" {
		log.Fatal("You can't specify both flags of -sql and -file.")
	}
	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", *flgProject, *flgInstance, *flgDatabase)

	sqls := []string{*flgSql}

	if *flgFile != "" {
		content, err := ioutil.ReadFile(*flgFile)
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range strings.Split(string(content), ";") {
			sqls = append(sqls, strings.TrimSpace(v))
		}
	}

	fmt.Printf("db:[%s]\n", dbName)
	fmt.Printf("file:[%s]\n", *flgFile)
	fmt.Printf("SPANNER_EMULATOR_HOST:[%s]\n", os.Getenv("SPANNER_EMULATOR_HOST"))

	admin, cli := createClients(context.Background(), dbName)

	for _, sql := range sqls {
		if sql == "" {
			continue
		}
		if err := executeSQL(context.Background(), os.Stdout, admin, cli, dbName, sql); err != nil {
			log.Fatal(err)
		}
	}
}

func executeSQL(ctx context.Context, w io.Writer, admin *database.DatabaseAdminClient, cli *spanner.Client, dbName, sql string) error {
	fmt.Printf("sql:[%s]\n", sql)
	fmt.Println("")

	switch token := strings.ToUpper(strings.Split(sql, " ")[0]); token {
	case "SELECT":
		return query(ctx, w, cli, sql)
	case
		"INSERT",
		"DELETE",
		"UPDATE":
		return dml(ctx, w, cli, sql)
	case
		"CREATE",
		"ALTER",
		"DROP":
		return updateDatabaseDdl(ctx, w, admin, dbName, sql)
	case
		"DESCRIBE":
		return describe(ctx, w, admin, dbName)
	default:
		return fmt.Errorf("unsupported SQL statement: [%s]", sql)
	}
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
	fmt.Fprintf(w, "Updatad database.\n")
	return nil
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

func dml(ctx context.Context, w io.Writer, client *spanner.Client, sql string) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{SQL: sql}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
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
	printTableExpand(w, colNames, rows)
	return nil
}

func printTable(w io.Writer, colNames []string, rows [][]string) {
	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(colNames)
	table.AppendBulk(rows)
	table.Render()
}

func printTableExpand(w io.Writer, colNames []string, rows [][]string) {
	for i, row := range rows {
		fmt.Fprintf(w, "@ Row %d\n", i)

		// Observe how the b's and the d's, despite appearing in the
		// second cell of each line, belong to different columns.
		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.Debug)
		for j := range row {
			fmt.Fprintf(tw, "  %s\t%s\n", colNames[j], row[j])
		}
		tw.Flush()
	}
}
