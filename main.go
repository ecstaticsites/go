package main

import (
	"cbnr/cmd/intake"
	"cbnr/cmd/serve"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cbnr",
		Short: "cbnr - parse, store, and query server access logs",
	}
	rootCmd.AddCommand(intake.IntakeCmd)
	rootCmd.AddCommand(serve.ServeCmd)
	rootCmd.Execute()
}
