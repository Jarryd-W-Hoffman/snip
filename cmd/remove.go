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
// It requires a unique lookup name to find and delete a snippet from storage.
var RemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"}, // Provides familiar shorthand alternatives
	Short:   "Remove a saved snippet by its lookup name",
	Long: `Accepts a unique lookup name as a primary argument, searches the local 
JSON flat-file storage configuration, and deletes the matching record if found.`,
	Args: cobra.ExactArgs(1), // Enforces that exactly one argument (the snippet name) is supplied
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}

		snippets, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading existing records from database: %v\n", err)
			os.Exit(1)
		}

		var updatedSnippets []storage.Snippet
		found := false

		for _, s := range snippets {
			if s.Name == name {
				found = true
				continue // Skip appending this item to effectively remove it
			}
			updatedSnippets = append(updatedSnippets, s)
		}

		if !found {
			fmt.Printf("⚠️  No snippet found with the name '%s'.\n", name)
			return
		}

		err = store.Save(updatedSnippets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error committing record changes to local storage: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("🗑️  Successfully removed snippet '%s'!\n", name)
	},
}

func init() {
	RootCmd.AddCommand(RemoveCmd)
}