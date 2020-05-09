package command

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"google.golang.org/api/option"
)

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
