package main

import (
	"ecstatic/cmd/api"
	"ecstatic/cmd/git"
	"ecstatic/cmd/intake"
	"ecstatic/cmd/query"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ecstatic",
		Short: "ecstatic - parse, store, and query server access logs",
	}
	rootCmd.AddCommand(api.ApiCmd)
	rootCmd.AddCommand(git.GitCmd)
	rootCmd.AddCommand(intake.IntakeCmd)
	rootCmd.AddCommand(query.QueryCmd)
	rootCmd.Execute()
}
