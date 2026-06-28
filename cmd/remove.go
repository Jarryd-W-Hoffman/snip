// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"os"
	"snip/storage"

	"github.com/spf13/cobra"
)

// RemoveCmd defines the configuration and behavior of the 'snip remove' command.
// It accepts a single unique lookup name argument and deletes it from disk.
var RemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"}, // Helpful fallback command aliases
	Short:   "Permanently remove a command snippet by name",
	Long:    `Deletes a specified command snippet row entirely from the local SQLite relational storage database index layer.`,
	Args:    cobra.ExactArgs(1), // Enforces that exactly one lookup target argument must be supplied
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		if err := store.Delete(name); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error deleting snippet from database: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("🗑️ Snippet '%s' permanently removed successfully.\n", name)
	},
}

func init() {
	RootCmd.AddCommand(RemoveCmd)
}