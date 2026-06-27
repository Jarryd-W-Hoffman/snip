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
	modeInput                      // Interactive contextual variable gathering state
)

// Structural UI component styling definitions using Lip Gloss.
var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#6200EE")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1)
	statusMessage = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE00AA")).Bold(true)
)

type item struct {
	title   string
	desc    string
	command string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

// model stores and encapsulates runtime state properties for the Bubble Tea view loop.
type model struct {
	list          list.Model
	store         *storage.Storage
	copiedStatus  string
	
	// Variable parsing engine state properties
	mode          sessionMode
	textInput     textinput.Model
	targetItem    item
	variables     []string
	varIndex      int
	replacements  map[string]string
}

func (m model) Init() tea.Cmd {
	return nil
}

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

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively view, run, and manage all snippets",
	Long:  `Launches a full-screen Terminal User Interface (TUI) allowing rapid navigation, inline execution, and deletion of your saved command bank.`,
	Args:  cobra.NoArgs,
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