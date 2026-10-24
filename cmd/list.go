// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"

	"github.com/Jarryd-W-Hoffman/snip/storage"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// ListCmd defines the configuration and behavior of the 'snip list' command.
// It launches a full-screen interactive TUI dashboard layout to manage snippets.
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively view, run, and manage all snippets",
	Long:  `Launches a full-screen Terminal User Interface (TUI) allowing rapid navigation, inline execution, and deletion of your saved command bank.`,
	Args:  cobra.NoArgs, // Restricts command to execute only when no stray arguments are provided
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true // failures past this point aren't usage mistakes

		store, err := storage.NewStorage()
		if err != nil {
			return fmt.Errorf("❌ Error initializing storage configuration: %w", err)
		}
		defer store.Close()

		storedSnippets, err := store.Load()
		if err != nil {
			return fmt.Errorf("❌ Error loading records from database file: %w", err)
		}

		var items []list.Item
		for _, s := range storedSnippets {
			items = append(items, item{
				title:      s.Name,
				desc:       s.Description,
				command:    s.Command,
				tags:       s.Tags,
				usageCount: s.UsageCount,
			})
		}

		p := tea.NewProgram(newModel(items, store), tea.WithAltScreen())
		final, err := p.Run()
		if err != nil {
			return fmt.Errorf("❌ Error executing user interface loop: %w", err)
		}

		// The TUI has exited by now, so a failed snippet's error lands in the normal terminal.
		if fm, ok := final.(model); ok && fm.execErr != nil {
			return fmt.Errorf("❌ Command failed: %w", fm.execErr)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(ListCmd)
}
