package main

import (
	"cbnr/cmd/api"
	"cbnr/cmd/intake"
	"cbnr/cmd/query"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cbnr",
		Short: "cbnr - parse, store, and query server access logs",
	}
	rootCmd.AddCommand(api.ApiCmd)
	rootCmd.AddCommand(intake.IntakeCmd)
	rootCmd.AddCommand(query.QueryCmd)
	rootCmd.Execute()
}
