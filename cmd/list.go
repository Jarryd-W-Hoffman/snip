// Package cmd handles terminal subcommands. This specific file implements
// the interactive terminal user interface (TUI) for listing snippets.
package cmd

import (
	"fmt"
	"os"
	"snip/storage"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Define structural UI component styling definitions using Lip Gloss.
var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#6200EE")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	statusMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
)

// item wraps a storage.Snippet to satisfy the bubbletea list.Item layout interface.
type item struct {
	title   string // Maps to the Snippet lookup name
	desc    string // Maps to the Snippet description
	command string // Maps to the raw snippet terminal action block
}

// Title returns the main string to render in the interactive list layout row.
func (i item) Title() string { return i.title }

// Description returns the secondary text block rendered directly below the title row.
func (i item) Description() string { return i.desc }

// FilterValue defines the searchable terms evaluated when filtering the TUI menu rows.
func (i item) FilterValue() string { return i.title + " " + i.desc }

// model stores and encapsulates runtime state properties for the Bubble Tea view loop.
type model struct {
	list         list.Model // The internal list bubble component tracking layout view ports
	copiedStatus string     // A status message displayed upon successful system clipboard copy actions
}

// Init triggers initial asynchronous processes upon TUI application startup.
// Returning nil indicates no background operations or commands are required.
func (m model) Init() tea.Cmd {
	return nil
}

// Update intercepts runtime messages, windows size recalculations, and keyboard actions,
// adjusting internal model data state properties appropriately.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				if err := clipboard.WriteAll(i.command); err != nil {
					m.copiedStatus = fmt.Sprintf("❌ Error writing to clipboard: %v", err)
				} else {
					m.copiedStatus = fmt.Sprintf("✓ Copied '%s' to clipboard!", i.title)
				}
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View evaluates structural properties and outputs standard formatted strings to the terminal.
func (m model) View() string {
	viewStr := docStyle.Render(m.list.View())
	if m.copiedStatus != "" {
		viewStr += "\n" + statusMessage.Render(m.copiedStatus)
	}
	return viewStr
}

// ListCmd maps out configuration, usage guides, and launch behaviors for 'snip list'.
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively view, filter, and copy all snippets",
	Long:  `Launches a full-screen Terminal User Interface (TUI) allowing rapid navigation and selection of your saved command bank.`,
	Args:  cobra.NoArgs, // Restricts command to execute only when no stray arguments are provided
	Run: func(cmd *cobra.Command, args []string) {
		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}

		storedSnippets, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error loading records from database file: %v\n", err)
			os.Exit(1)
		}

		var items []list.Item
		for _, s := range storedSnippets {
			items = append(items, item{
				title:   s.Name,
				desc:    s.Description,
				command: s.Command,
			})
		}

		delegate := list.NewDefaultDelegate()
		m := model{
			list: list.New(items, delegate, 0, 0),
		}
		m.list.Title = " Your Snippets "
		m.list.Styles.Title = titleStyle

		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing user interface loop: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(ListCmd)
}