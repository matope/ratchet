package command

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/spf13/cobra"
)

func dropAllTablesCmd() *cobra.Command {
	var flgDryrun bool
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
			root, err := resolveTableDependency(ctx, cli)
			if err != nil {
				return err
			}
			walkCtx := newWalkContext()
			root.DropStatements(walkCtx)
			stmts := walkCtx.stmts
			for _, v := range stmts {
				fmt.Println(v)
			}
			if !flgDryrun {
				fmt.Printf("Updating Database with %d DDL(s)...\n", len(stmts))
				return updateDatabaseDdls(ctx, os.Stdout, admin, dbName, stmts...)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&flgDryrun, "dry-run", "", false, "Print DDLs to execute instead of executing them")
	return cmd
}

type foreignkey struct {
	name     string
	table    string
	refTable string
}

type table struct {
	name        string
	parent      string
	children    []*table
	indexes     []string
	references  []*foreignkey
	foreignKeys []*foreignkey
}

func (t *table) addChild(c *table)            { t.children = append(t.children, c) }
func (t *table) addIndex(i string)            { t.indexes = append(t.indexes, i) }
func (t *table) addReference(fk *foreignkey)  { t.references = append(t.references, fk) }
func (t *table) addForeignKey(fk *foreignkey) { t.foreignKeys = append(t.foreignKeys, fk) }

type walkCtx struct {
	droppedFKs map[*foreignkey]struct{}
	stmts      []string
}

func (w *walkCtx) addStmt(s string) { w.stmts = append(w.stmts, s) }

func newWalkContext() *walkCtx { return &walkCtx{droppedFKs: make(map[*foreignkey]struct{})} }

func (t *table) DropStatements(wc *walkCtx) {
	for _, v := range t.children {
		v.DropStatements(wc)
	}
	if t.name != "" {
		t.dropStatements(wc)
	}
}

func (t *table) dropStatements(wc *walkCtx) {
	dropFKs(wc, t.foreignKeys)
	dropFKs(wc, t.references)

	for _, v := range t.indexes {
		wc.addStmt(fmt.Sprintf("DROP INDEX %s", v))
	}
	wc.addStmt(fmt.Sprintf("DROP TABLE %s", t.name))
}

func dropFKs(wc *walkCtx, fks []*foreignkey) {
	for _, v := range fks {
		if _, dropped := wc.droppedFKs[v]; !dropped {
			wc.addStmt(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", v.table, v.name))
			wc.droppedFKs[v] = struct{}{}
		}
	}
}

func resolveTableDependency(ctx context.Context, cli *spanner.Client) (*table, error) {
	tables, err := queryTables(ctx, cli)
	if err != nil {
		return nil, err
	}

	root := &table{}
	tableMap := map[string]*table{
		"": root,
	}
	for _, v := range tables {
		tableMap[v.name] = v
	}

	if err := queryIndexes(ctx, cli, tableMap); err != nil {
		return nil, err
	}

	if err := queryConstraints(ctx, cli, tableMap); err != nil {
		return nil, err
	}

	for _, v := range tables {
		tableMap[v.parent].addChild(v)
	}
	return root, err
}

func queryTables(ctx context.Context, cli *spanner.Client) ([]*table, error) {
	sql := `SELECT TABLE_NAME, PARENT_TABLE_NAME FROM information_schema.tables WHERE TABLE_CATALOG = '' AND TABLE_SCHEMA = ''`
	iter := cli.Single().Query(ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	tables := make([]*table, 0)
	err := iter.Do(func(row *spanner.Row) error {
		t := table{}
		row.Column(0, &t.name)
		row.Column(1, &t.parent)
		tables = append(tables, &t)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tables, nil
}

func queryIndexes(ctx context.Context, cli *spanner.Client, tables map[string]*table) error {
	sql := "SELECT TABLE_NAME, INDEX_NAME FROM information_schema.indexes WHERE TABLE_SCHEMA = '' AND INDEX_TYPE = 'INDEX'"
	iter := cli.Single().Query(ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	return iter.Do(func(row *spanner.Row) error {
		var tableName, indexName string
		row.Column(0, &tableName)
		row.Column(1, &indexName)
		tables[tableName].addIndex(indexName)
		return nil
	})
}

func queryConstraints(ctx context.Context, cli *spanner.Client, tables map[string]*table) error {
	sql := `SELECT c.CONSTRAINT_NAME, c.TABLE_NAME, u.TABLE_NAME AS REF_TABLE_NAME FROM ` +
		`information_schema.table_constraints as c JOIN information_schema.CONSTRAINT_TABLE_USAGE as u ` +
		`ON c.CONSTRAINT_NAME=u.CONSTRAINT_NAME ` +
		`WHERE CONSTRAINT_TYPE='FOREIGN KEY'`

	iter := cli.Single().Query(ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	return iter.Do(func(row *spanner.Row) error {
		fk := foreignkey{}
		row.Column(0, &fk.name)
		row.Column(1, &fk.table)
		row.Column(2, &fk.refTable)
		tables[fk.table].addForeignKey(&fk)
		tables[fk.refTable].addReference(&fk)
		return nil
	})
}
