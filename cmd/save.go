package cmd

import (
	"fmt"
	"os"
	"snip/storage"

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
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if commandStr == "" {
			fmt.Println("❌ Error: --command (-c) flag is required.")
			os.Exit(1)
		}

		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		snippet := storage.Snippet{
			Name:        name,
			Command:     commandStr,
			Description: descStr,
			Tags:        tagsSlice,
		}

		if err := store.Upsert(snippet); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error saving snippet: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Snippet '%s' saved successfully with %d tag(s)!\n", name, len(tagsSlice))
	},
}

func init() {
	SaveCmd.Flags().StringVarP(&commandStr, "command", "c", "", "The terminal command block to store (required)")
	SaveCmd.Flags().StringVarP(&descStr, "desc", "d", "", "A short description tracking execution behavior context summaries")
	SaveCmd.Flags().StringSliceVarP(&tagsSlice, "tag", "t", []string{}, "Category labels or project stack groupings to associate with snippet")

	RootCmd.AddCommand(SaveCmd)
}