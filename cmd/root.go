// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command executed when 'snip' is invoked without subcommands.
var RootCmd = &cobra.Command{
	Use:   "snip",
	Short: "snip is a lightning-fast interactive CLI snippet manager",
	Long: `snip allows you to store, categorize, and quickly reference frequently
used shell commands, script blocks, and code snippets directly inside your terminal,
instantly copying selected entries back to your system clipboard.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.go and is the primary entry point for the CLI runtime application loop.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing application: %v\n", err)
		os.Exit(1)
	}
}