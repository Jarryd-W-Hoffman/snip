// Package cmd contains the CLI command-line subcommands and routing
// logic using the Cobra library framework.
package cmd

import (
	"fmt"
	"regexp"
	"github.com/Jarryd-W-Hoffman/snip/storage"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput" // Native text input bubble component
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

// execCompleteMsg is sent back from ExecProcess when a snippet finishes running.
type execCompleteMsg struct {
	err error
}

// Structural UI component styling definitions using Lip Gloss.
var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#6200EE")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	statusMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE00AA")).Bold(true)

	varRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)
)

// item wraps a storage.Snippet to satisfy the bubbletea list.Item layout interface.
type item struct {
	title      string   // Maps to the Snippet lookup name
	desc       string   // Maps to the Snippet description
	command    string   // Maps to the raw snippet terminal action block
	tags       []string // Store relational metadata tags inside item data definitions
	usageCount int      // Tracks individual execution counts visually
}

// Title returns the main string to render in the interactive list layout row.
func (i item) Title() string { return i.title }

// Description returns the secondary text block rendered directly below the title row, appending metadata tags and usage counters.
func (i item) Description() string {
	var meta []string
	
	// Append usage metric badge dynamically if it has been executed
	if i.usageCount > 0 {
		meta = append(meta, fmt.Sprintf("🔥 %d", i.usageCount))
	}
	if len(i.tags) > 0 {
		meta = append(meta, fmt.Sprintf("[%s]", strings.Join(i.tags, ", ")))
	}

	if len(meta) > 0 {
		return fmt.Sprintf("%s • %s", strings.Join(meta, " "), i.desc)
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

	// ---- INTERACTIVE VARIABLE INPUT PROCESSING STATE MODE ----
	if m.mode == modeInput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			// Back out to browsing menu dashboard frame safely
			case "ctrl+c", "esc":
				m.mode = modeBrowsing
				return m, nil

			case "enter":
				// Capture current variable entry block value safely
				currentVar := m.variables[m.varIndex]
				m.replacements["{{"+currentVar+"}}"] = m.textInput.Value()

				// If there are more variables to gather, pivot input focuses instantly
				if m.varIndex < len(m.variables)-1 {
					m.varIndex++
					m.textInput.SetValue("")
					m.textInput.Prompt = fmt.Sprintf("➡️ Enter value for [%s]: ", m.variables[m.varIndex])
					return m, nil
				}

				// All variables gathered successfully -> Build final script execution string context
				finalCommand := substituteAll(m.targetItem.command, m.replacements)

				// 🚀 Record interactive execution tracking metrics natively prior to running
				_ = m.store.IncrementUsage(m.targetItem.title)

				return m, tea.ExecProcess(buildExecCommand(finalCommand), func(err error) tea.Msg {
					return execCompleteMsg{err: err}
				})
			}
		}

		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// ---- STANDARD BROWSING MODE ----
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If filtering the list, pass key inputs straight down to the filter input field
		if m.list.FilterState() == list.Filtering {
			break
		}

		m.copiedStatus = ""

		switch msg.String() {
		// Terminate and exit the runtime environment cleanly
		case "ctrl+c", "q":
			return m, tea.Quit

		// 🚀 INLINE RUN: Directly execute the highlighted command, evaluating placeholders
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				vars := extractVariables(i.command)

				// If placeholders exist, shift setup gears into the dynamic prompt input layout mode
				if len(vars) > 0 {
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

				// Standard execution path if command lacks template placeholders
				_ = m.store.IncrementUsage(i.title)

				return m, tea.ExecProcess(buildExecCommand(i.command), func(err error) tea.Msg {
					return execCompleteMsg{err: err}
				})
			}

		// 📋 INLINE COPY: Write the command string directly into the system clipboard
		case "c":
			if i, ok := m.list.SelectedItem().(item); ok {
				_ = clipboard.WriteAll(i.command)
				m.copiedStatus = fmt.Sprintf("✓ Copied '%s' to clipboard!", i.title)
				return m, nil
			}

		// 🗑️ INLINE DELETE: Permanently remove the item from SQLite and the UI view
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

	// Dynamically scale and recalculate terminal menu viewport dimensional constraints
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	// 🏁 Exec process finished — show error if any, then quit
	case execCompleteMsg:
		if msg.err != nil {
			m.copiedStatus = fmt.Sprintf("❌ Command failed: %v", msg.err)
		}
		return m, tea.Quit
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View evaluates structural properties and outputs standard formatted strings to the terminal.
func (m model) View() string {
	// If gathering context variable fields, render focused prompt panel view instead
	if m.mode == modeInput {
		return docStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render(" Configure Variables "),
			promptStyle.Render(fmt.Sprintf("Snippet: %s", m.targetItem.title)),
			m.textInput.View(),
		))
	}

	// Empty state: show a helpful message instead of a blank list
	if len(m.list.Items()) == 0 {
		return docStyle.Render(fmt.Sprintf(
			"%s\n\n✨ No snippets yet — use 'snip save --command \"your command\" <name>' to get started\n\nPress q or Ctrl+C to exit.",
			titleStyle.Render(" Manage Snippets "),
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Establish database storage instance reference links
		store, err := storage.NewStorage()
		if err != nil {
			return fmt.Errorf("❌ Error initializing storage configuration: %w", err)
		}
		defer store.Close()

		// 2. Load snippet rows from SQLite database
		storedSnippets, err := store.Load()
		if err != nil {
			return fmt.Errorf("❌ Error loading records from database file: %w", err)
		}

		// 3. Process records into list model data formats
		var items []list.Item
		for _, s := range storedSnippets {
			items = append(items, item{
				title:      s.Name,
				desc:       s.Description,
				command:    s.Command,
				tags:       s.Tags,
				usageCount: s.UsageCount, // Map the database metric down into UI elements
			})
		}

		// 4. Construct the default layout and inject text contents
		m := model{
			list:  list.New(items, list.NewDefaultDelegate(), 0, 0),
			store: store,
			mode:  modeBrowsing,
		}
		m.list.Title = " Manage Snippets "
		m.list.Styles.Title = titleStyle
		m.list.SetStatusBarItemName("snippet", "snippets")

		// Configure explicit shortcut help labels at the bottom of the viewport menu
		m.list.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
				key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
				key.NewBinding(key.WithKeys("x", "backspace"), key.WithHelp("x", "delete")),
			}
		}

		// 5. Initialize the bubble tea execution loop within full viewport alternative screens
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("Error executing user interface loop: %w", err)
		}

		return nil
	},
}

func init() {
	// Register ListCmd directly into the parent RootCmd execution hierarchy layout
	RootCmd.AddCommand(ListCmd)
}