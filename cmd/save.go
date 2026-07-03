package cmd

import (
	"fmt"
	"snip/storage"
	"strings"

	"github.com/spf13/cobra"
)

var (
	commandStr string
	descStr    string
	tagsSlice  []string
)

var SaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save a command snippet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return fmt.Errorf("snippet name cannot be empty")
		}

		store, err := storage.NewStorage()
		if err != nil {
			return fmt.Errorf("error initializing storage: %w", err)
		}
		defer store.Close()

		snippet := storage.Snippet{
			Name:        name,
			Command:     commandStr,
			Description: descStr,
			Tags:        tagsSlice,
		}

		if err := store.Upsert(snippet); err != nil {
			return fmt.Errorf("error saving snippet: %w", err)
		}

		fmt.Printf("✅ Snippet '%s' saved successfully with %d tag(s)!\n", name, len(tagsSlice))
		return nil
	},
}

func init() {
	SaveCmd.Flags().StringVarP(&commandStr, "command", "c", "", "The terminal command block to store")
	SaveCmd.Flags().StringVarP(&descStr, "desc", "d", "", "A short description tracking execution behavior context summaries")
	SaveCmd.Flags().StringSliceVarP(&tagsSlice, "tag", "t", []string{}, "Category labels or project stack groupings to associate with snippet")
	SaveCmd.MarkFlagRequired("command")

	RootCmd.AddCommand(SaveCmd)
}