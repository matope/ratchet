package command

import "github.com/spf13/cobra"

var (
	flgProject  string
	flgInstance string
	flgDatabase string
)

func RootCmd() *cobra.Command {
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
	rootCmd.AddCommand(dropAllTablesCmd())
	return rootCmd
}
