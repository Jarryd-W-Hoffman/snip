// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"snip/storage"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Structural UI component styling definitions using Lip Gloss.
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
	list         list.Model       // The internal list bubble component tracking layout view ports
	store        *storage.Storage // Active SQLite database layer reference connection pool
	copiedStatus string           // A status message displayed upon successful system actions
}

// Init triggers initial asynchronous processes upon TUI application startup.
func (m model) Init() tea.Cmd {
	return nil
}

// Update intercepts runtime messages, windows size recalculations, and keyboard actions,
// adjusting internal model data state properties appropriately.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				var c *exec.Cmd
				if runtime.GOOS == "windows" {
					c = exec.Command("cmd", "/c", i.command)
				} else {
					c = exec.Command("bash", "-c", i.command)
				}

				return m, tea.Sequence(
					tea.ExecProcess(c, func(err error) tea.Msg {
						if err != nil {
							return fmt.Errorf("command failed: %v", err)
						}
						return nil
					}),
					tea.Quit,
				)
			}

		case "c":
			if i, ok := m.list.SelectedItem().(item); ok {
				_ = clipboard.WriteAll(i.command)
				m.copiedStatus = fmt.Sprintf("✓ Copied '%s' to clipboard!", i.title)
				return m, nil
			}

		case "x", "backspace":
			if i, ok := m.list.SelectedItem().(item); ok {
				if err := m.store.Delete(i.title); err == nil {
					idx := m.list.Index()
					m.list.RemoveItem(idx)
					m.copiedStatus = fmt.Sprintf("🗑️ Removed '%s'", i.title)
				}
				return m, nil
			}
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

// ListCmd defines the configuration and behavior of the 'snip list' command.
// It launches a full-screen interactive TUI dashboard layout to manage snippets.
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively view, run, and manage all snippets",
	Long:  `Launches a full-screen Terminal User Interface (TUI) allowing rapid navigation, inline execution, and deletion of your saved command bank.`,
	Args:  cobra.NoArgs, // Restricts command to execute only when no stray arguments are provided
	Run: func(cmd *cobra.Command, args []string) {
		store, err := storage.NewStorage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error initializing storage configuration: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

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

		m := model{
			list:  list.New(items, list.NewDefaultDelegate(), 0, 0),
			store: store,
		}
		m.list.Title = " Manage Snippets "
		m.list.Styles.Title = titleStyle
		
		m.list.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
				key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
				key.NewBinding(key.WithKeys("x", "backspace"), key.WithHelp("x", "delete")),
			}
		}

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