// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"github.com/Jarryd-W-Hoffman/snip/storage"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			return fmt.Errorf("❌ Error initializing storage configuration: %w", err)
		}
		defer store.Close()

		if err := store.Delete(name); err == storage.ErrNotFound {
			return fmt.Errorf("❌ Error: No snippet found with the name '%s'", name)
		} else if err != nil {
			return fmt.Errorf("❌ Error deleting snippet from database: %w", err)
		}

		fmt.Printf("🗑️ Snippet '%s' permanently removed successfully.\n", name)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(RemoveCmd)
}