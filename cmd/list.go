// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"snip/storage"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// UI View Mode Enum States
type sessionMode int

const (
	modeBrowsing sessionMode = iota // Standard snippet navigation list state
	modeInput                       // Interactive contextual variable gathering state
)

// Structural UI component styling definitions using Lip Gloss.
var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#6200EE")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	statusMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE00AA")).Bold(true)
)

// item wraps a storage.Snippet to satisfy the bubbletea list.Item layout interface.
type item struct {
	title   string   // Maps to the Snippet lookup name
	desc    string   // Maps to the Snippet description
	command string   // Maps to the raw snippet terminal action block
	tags    []string // Store relational metadata tags inside item data definitions
}

// Title returns the main string to render in the interactive list layout row.
func (i item) Title() string { return i.title }

// Description returns the secondary text block rendered directly below the title row, appending metadata tags.
func (i item) Description() string {
	if len(i.tags) > 0 {
		return fmt.Sprintf("[%s] %s", strings.Join(i.tags, ", "), i.desc)
	}
	return i.desc
}

// FilterValue defines the searchable terms evaluated when filtering the TUI menu rows.
func (i item) FilterValue() string {
	return i.title + " " + i.desc + " " + strings.Join(i.tags, " ")
}

// model stores and encapsulates runtime state properties for the Bubble Tea view loop.
type model struct {
	list         list.Model       // The internal list bubble component tracking layout view ports
	store        *storage.Storage // Active SQLite database layer reference connection pool
	copiedStatus string           // A status message displayed upon successful system actions

	// Variable parsing engine state properties
	mode         sessionMode       // Tracking active display focus context splits
	textInput    textinput.Model   // Native focused dynamic text entry box bubble
	targetItem   item              // Stashed target command reference being evaluated
	variables    []string          // Deduplicated array slice tracking variable labels to satisfy
	varIndex     int               // Current lookup element tracking cursor pointer location
	replacements map[string]string // Key-value placeholder tracking container mappings
}

// Init triggers initial asynchronous processes upon TUI application startup.
func (m model) Init() tea.Cmd {
	return nil
}

// Update intercepts runtime messages, windows size recalculations, and keyboard actions,
// adjusting internal model data state properties appropriately based on active modes.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == modeInput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.mode = modeBrowsing
				return m, nil

			case "enter":
				currentVar := m.variables[m.varIndex]
				m.replacements["{{"+currentVar+"}}"] = m.textInput.Value()

				if m.varIndex < len(m.variables)-1 {
					m.varIndex++
					m.textInput.SetValue("")
					m.textInput.Prompt = fmt.Sprintf("➡️ Enter value for [%s]: ", m.variables[m.varIndex])
					return m, nil
				}

				finalCommand := m.targetItem.command
				for placeholder, val := range m.replacements {
					finalCommand = strings.ReplaceAll(finalCommand, placeholder, val)
				}

				var c *exec.Cmd
				if runtime.GOOS == "windows" {
					c = exec.Command("cmd", "/c", finalCommand)
				} else {
					c = exec.Command("bash", "-c", finalCommand)
				}

				return m, tea.Sequence(
					tea.ExecProcess(c, func(err error) tea.Msg { return nil }),
					tea.Quit,
				)
			}
		}

		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

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
				re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
				matches := re.FindAllStringSubmatch(i.command, -1)

				if len(matches) > 0 {
					var vars []string
					seen := make(map[string]bool)
					for _, match := range matches {
						if !seen[match[1]] {
							seen[match[1]] = true
							vars = append(vars, match[1])
						}
					}

					ti := textinput.New()
					ti.Focus()
					ti.Prompt = fmt.Sprintf("➡️ Enter value for [%s]: ", vars[0])
					ti.CharLimit = 156
					ti.Width = 40

					m.mode = modeInput
					m.textInput = ti
					m.targetItem = i
					m.variables = vars
					m.varIndex = 0
					m.replacements = make(map[string]string)

					return m, textinput.Blink
				}

				var c *exec.Cmd
				if runtime.GOOS == "windows" {
					c = exec.Command("cmd", "/c", i.command)
				} else {
					c = exec.Command("bash", "-c", i.command)
				}

				return m, tea.Sequence(
					tea.ExecProcess(c, func(err error) tea.Msg { return nil }),
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

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View evaluates structural properties and outputs standard formatted strings to the terminal.
func (m model) View() string {
	if m.mode == modeInput {
		return docStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render(" Configure Variables "),
			promptStyle.Render(fmt.Sprintf("Snippet: %s", m.targetItem.title)),
			m.textInput.View(),
		))
	}

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
				tags:    s.Tags,
			})
		}

		m := model{
			list:  list.New(items, list.NewDefaultDelegate(), 0, 0),
			store: store,
			mode:  modeBrowsing,
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