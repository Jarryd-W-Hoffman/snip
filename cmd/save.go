package cmd

import (
	"fmt"
	"os"
	"snip/storage"

	"github.com/spf13/cobra"
)

// Internal storage tracking for command-line flag inputs.
var (
	commandStr string
	descStr    string
)

// SaveCmd defines the configuration, flag definitions, and execution behavior 
// of the 'snip save' command.
var SaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save or update a designated code snippet or command",
	Long:  `Accepts a unique lookup name as an argument along with required utility flags to persist a command to disk.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage: %v\n", err)
			os.Exit(1)
		}

		snippets, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error reading existing database record: %v\n", err)
			os.Exit(1)
		}

		newSnippet := storage.Snippet{
			Name:        name,
			Command:     commandStr,
			Description: descStr,
		}

		updated := false
		for i, s := range snippets {
			if s.Name == name {
				snippets[i] = newSnippet
				updated = true
				break
			}
		}

		if !updated {
			snippets = append(snippets, newSnippet)
		}

		err = store.Save(snippets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error committing record to local storage: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✨ Successfully saved snippet '%s'!\n", name)
	},
}

func init() {
	SaveCmd.Flags().StringVarP(&commandStr, "command", "c", "", "The executable command string to preserve (Required)")
	SaveCmd.Flags().StringVarP(&descStr, "desc", "d", "", "A short functional summary or description of the snippet")

	SaveCmd.MarkFlagRequired("command")

	RootCmd.AddCommand(SaveCmd)
}