package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/Jarryd-W-Hoffman/snip/storage"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UI View Mode Enum States
type sessionMode int

const (
	modeBrowsing sessionMode = iota // Standard snippet navigation list state
	modeInput                       // Interactive contextual variable gathering state
)

// Panes are shown side by side once the terminal is at least this wide.
const splitMinWidth = 90

type statusKind int

const (
	statusInfo statusKind = iota
	statusSuccess
	statusError
	statusConfirm
)

// execCompleteMsg is sent back from Exec when a snippet finishes running.
type execCompleteMsg struct {
	err error
}

// clearStatusMsg expires a status message; id guards against clearing a newer one.
type clearStatusMsg struct{ id int }

type keyMap struct {
	run, copy, del, confirm, next, prev, cancel key.Binding
}

var keys = keyMap{
	run:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
	copy:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
	del:     key.NewBinding(key.WithKeys("x", "backspace"), key.WithHelp("x", "delete")),
	confirm: key.NewBinding(key.WithKeys("y", "Y")),
	next:    key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("tab", "next")),
	prev:    key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("shift+tab", "prev")),
	cancel:  key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
}

// item wraps a storage.Snippet to satisfy the bubbletea list.Item layout interface.
type item struct {
	title      string   // Maps to the Snippet lookup name
	desc       string   // Maps to the Snippet description
	command    string   // Maps to the raw snippet terminal action block
	tags       []string // Store relational metadata tags inside item data definitions
	usageCount int      // Tracks individual execution counts visually
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

// FilterValue defines the searchable terms evaluated when filtering the TUI menu rows.
func (i item) FilterValue() string {
	return i.title + " " + i.desc + " " + strings.Join(i.tags, " ")
}

// model stores and encapsulates runtime state properties for the Bubble Tea view loop.
type model struct {
	list          list.Model       // The internal list bubble component tracking layout view ports
	store         *storage.Storage // Active SQLite database layer reference connection pool
	width, height int              // Last known terminal size

	status     string     // Transient message shown in the footer
	statusKind statusKind // Controls the status colour
	statusID   int        // Incremented per message so stale clear ticks are ignored
	confirmDel bool       // Awaiting y/n before deleting the selected snippet

	// Variable form state
	mode       sessionMode       // Tracking active display focus context splits
	targetItem item              // Snippet being configured
	variables  []string          // Deduplicated variable names in placeholder order
	inputs     []textinput.Model // One input per variable
	varIndex   int               // Focused input

	execErr error // Error from the executed snippet, reported after the TUI exits
}

func newModel(items []list.Item, store *storage.Storage) model {
	l := list.New(items, snippetDelegate{}, 0, 0)
	l.Title = "snip"
	l.Styles.Title = titleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 2)
	l.Styles.StatusBar = mutedStyle.Padding(0, 0, 1, 2)
	l.Styles.FilterPrompt = accentStyle
	l.Styles.FilterCursor = accentStyle
	l.FilterInput.Prompt = "/ "
	l.SetStatusBarItemName("snippet", "snippets")
	l.SetShowHelp(false) // rendered full-width in the footer instead
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.run, keys.copy, keys.del}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys
	l.Help.Styles.ShortKey = accentStyle.UnsetBold()
	l.Help.Styles.FullKey = accentStyle.UnsetBold()
	l.Help.Styles.ShortDesc = mutedStyle
	l.Help.Styles.FullDesc = mutedStyle

	return model{list: l, store: store, mode: modeBrowsing}
}

// Init triggers initial asynchronous processes upon TUI application startup.
func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) setStatus(kind statusKind, format string, args ...any) tea.Cmd {
	m.statusID++
	m.status = fmt.Sprintf(format, args...)
	m.statusKind = kind
	if kind == statusConfirm {
		return nil // stays until answered
	}
	id := m.statusID
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{id} })
}

// Update routes messages to the active mode, keeping layout in sync with the window.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case clearStatusMsg:
		if msg.id == m.statusID && m.statusKind != statusConfirm {
			m.status = ""
		}
		return m, nil

	case execCompleteMsg:
		m.execErr = msg.err
		return m, tea.Quit
	}

	if m.mode == modeInput {
		return m.updateInput(msg)
	}
	return m.updateBrowsing(msg)
}

func (m model) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && m.list.FilterState() != list.Filtering {
		if m.confirmDel {
			m.confirmDel = false
			m.status = ""
			if key.Matches(msg, keys.confirm) {
				return m, m.deleteSelected()
			}
			return m, nil
		}

		switch {
		case msg.String() == "ctrl+c" || msg.String() == "q":
			return m, tea.Quit

		case key.Matches(msg, keys.run):
			if i, ok := m.list.SelectedItem().(item); ok {
				return m.startRun(i)
			}

		case key.Matches(msg, keys.copy):
			if i, ok := m.list.SelectedItem().(item); ok {
				if err := clipboard.WriteAll(i.command); err != nil {
					return m, m.setStatus(statusError, "✗ Clipboard unavailable: %v", err)
				}
				return m, m.setStatus(statusSuccess, "✓ Copied %s to clipboard", i.title)
			}

		case key.Matches(msg, keys.del):
			if i, ok := m.list.SelectedItem().(item); ok {
				m.confirmDel = true
				return m, m.setStatus(statusConfirm, "Delete %s? y to confirm, any other key to cancel", i.title)
			}
		}
	}

	helpWasOpen := m.list.Help.ShowAll
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.list.Help.ShowAll != helpWasOpen {
		m.resize() // footer height changed
	}
	return m, cmd
}

func (m *model) deleteSelected() tea.Cmd {
	i, ok := m.list.SelectedItem().(item)
	if !ok {
		return nil
	}
	if err := m.store.Delete(i.title); err != nil {
		return m.setStatus(statusError, "✗ Failed to delete %s: %v", i.title, err)
	}
	// list.RemoveItem mixes up filtered and unfiltered indices, so rebuild the items instead.
	var remaining []list.Item
	for _, li := range m.list.Items() {
		if li.(item).title != i.title {
			remaining = append(remaining, li)
		}
	}
	if refilter := m.list.SetItems(remaining); refilter != nil {
		m.list, _ = m.list.Update(refilter()) // apply the new matches now rather than a frame later
	}
	if n := len(m.list.VisibleItems()); m.list.Index() >= n && n > 0 {
		m.list.Select(n - 1)
	}
	m.resize() // refreshes pagination for the new item count
	return m.setStatus(statusInfo, "Removed %s", i.title)
}

// startRun executes the snippet immediately, or opens the variable form if it has placeholders.
func (m model) startRun(i item) (tea.Model, tea.Cmd) {
	vars := extractVariables(i.command)
	if len(vars) == 0 {
		return m, m.exec(i, i.command)
	}

	m.mode = modeInput
	m.targetItem = i
	m.variables = vars
	m.varIndex = 0
	m.inputs = make([]textinput.Model, len(vars))
	for n, v := range vars {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = v
		ti.CharLimit = 256
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorFaint)
		ti.Cursor.Style = accentStyle
		m.inputs[n] = ti
	}
	m.status = ""
	m.resize()
	return m, m.focusInput(0)
}

func (m *model) focusInput(n int) tea.Cmd {
	m.inputs[m.varIndex].Blur()
	m.varIndex = n
	return m.inputs[n].Focus()
}

func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keys.cancel):
			m.mode = modeBrowsing
			return m, nil

		case key.Matches(msg, keys.next):
			return m, m.focusInput((m.varIndex + 1) % len(m.inputs))

		case key.Matches(msg, keys.prev):
			return m, m.focusInput((m.varIndex - 1 + len(m.inputs)) % len(m.inputs))

		case key.Matches(msg, keys.run):
			if m.varIndex < len(m.inputs)-1 {
				return m, m.focusInput(m.varIndex + 1)
			}
			return m, m.exec(m.targetItem, substituteAll(m.targetItem.command, m.replacements()))
		}
	}

	var cmd tea.Cmd
	m.inputs[m.varIndex], cmd = m.inputs[m.varIndex].Update(msg)
	return m, cmd
}

// values maps variable names to the text currently typed into their inputs.
func (m model) values() map[string]string {
	vals := make(map[string]string, len(m.variables))
	for n, v := range m.variables {
		vals[v] = m.inputs[n].Value()
	}
	return vals
}

func (m model) replacements() map[string]string {
	r := make(map[string]string, len(m.variables))
	for v, val := range m.values() {
		r["{{"+v+"}}"] = val
	}
	return r
}

func (m model) exec(i item, command string) tea.Cmd {
	_ = m.store.IncrementUsage(i.title)
	return tea.Exec(&echoExec{Cmd: buildExecCommand(command), name: i.title, command: command}, func(err error) tea.Msg {
		return execCompleteMsg{err: err}
	})
}

// echoExec prints which snippet is running before handing the terminal to it,
// so the scrollback shows what produced the output.
type echoExec struct {
	*exec.Cmd
	name, command string
}

func (e *echoExec) SetStdin(r io.Reader)  { e.Stdin = r }
func (e *echoExec) SetStdout(w io.Writer) { e.Stdout = w }
func (e *echoExec) SetStderr(w io.Writer) { e.Stderr = w }

func (e *echoExec) Run() error {
	if e.Stdout != nil {
		fmt.Fprintf(e.Stdout, "%s %s\n\n", titleStyle.Render("snip ▸ "+e.name), mutedStyle.Render("$ "+e.command))
	}
	return e.Cmd.Run()
}

// resize distributes the window between the list, the detail pane and the footer.
func (m *model) resize() {
	h, v := docStyle.GetFrameSize()
	w, height := m.width-h, m.height-v-lipgloss.Height(m.footerView())
	listW := w
	if w >= splitMinWidth {
		listW = w * 45 / 100
	}
	m.list.SetSize(listW, max(height, 0))
	m.list.Help.Width = w // SetSize narrows help to the list; the footer spans the window
	for n := range m.inputs {
		m.inputs[n].Width = min(48, max(w-30, 10))
	}
}

// View renders the active mode inside the document margins.
func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	h, v := docStyle.GetFrameSize()
	w, height := m.width-h, m.height-v

	var body string
	switch {
	case m.mode == modeInput:
		body = lipgloss.Place(w, height-lipgloss.Height(m.footerView()), lipgloss.Center, lipgloss.Center, m.formView(w))
	case len(m.list.Items()) == 0:
		body = lipgloss.Place(w, height-lipgloss.Height(m.footerView()), lipgloss.Center, lipgloss.Center, emptyView())
	case w >= splitMinWidth:
		listView := m.list.View()
		detailW := w - m.list.Width() - 2
		body = lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", m.detailView(detailW, m.list.Height()))
	default:
		body = m.list.View()
	}

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, body, m.footerView()))
}

func (m model) footerView() string {
	var help string
	switch {
	case m.mode == modeInput:
		help = m.list.Help.ShortHelpView([]key.Binding{keys.next, keys.prev, keys.run, keys.cancel})
	case len(m.list.Items()) == 0:
		help = m.list.Help.ShortHelpView([]key.Binding{m.list.KeyMap.Quit})
	default:
		help = m.list.Help.View(m.list)
	}
	status := " "
	if m.status != "" {
		status = statusStyles[m.statusKind].PaddingLeft(2).Render(m.status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, status, lipgloss.NewStyle().PaddingLeft(2).Render(help))
}

func (m model) detailView(width, height int) string {
	style := panelStyle.Width(width - 2).Height(height - 2)
	i, ok := m.list.SelectedItem().(item)
	if !ok {
		return style.Render(mutedStyle.Render("No matching snippets"))
	}
	inner := width - 2 - panelStyle.GetHorizontalPadding()

	sections := []string{accentStyle.Render(i.title)}
	if i.desc != "" {
		sections = append(sections, lipgloss.NewStyle().Width(inner).Render(i.desc))
	}
	sections = append(sections, "", labelStyle.Render("COMMAND"), renderCommand(i.command, nil, "", inner), "")

	field := func(label, value string) string {
		return labelStyle.Width(11).Render(label) + value
	}
	if vars := extractVariables(i.command); len(vars) > 0 {
		sections = append(sections, field("VARIABLES", lipgloss.NewStyle().Foreground(colorWarn).Render(strings.Join(vars, ", "))))
	}
	if len(i.tags) > 0 {
		var tags []string
		for _, t := range i.tags {
			tags = append(tags, "#"+t)
		}
		sections = append(sections, field("TAGS", tagStyle.Render(strings.Join(tags, " "))))
	}
	runs := "never run"
	if i.usageCount == 1 {
		runs = "1 time"
	} else if i.usageCount > 1 {
		runs = fmt.Sprintf("%d times", i.usageCount)
	}
	sections = append(sections, field("RUNS", mutedStyle.Render(runs)))

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

func (m model) formView(maxWidth int) string {
	labelW := 0
	for _, v := range m.variables {
		labelW = max(labelW, lipgloss.Width(v))
	}

	header := titleStyle.Render("Run") + " " + accentStyle.Render(m.targetItem.title) +
		mutedStyle.Render(fmt.Sprintf("  %d/%d", m.varIndex+1, len(m.variables)))
	rows := []string{header}
	if m.targetItem.desc != "" {
		rows = append(rows, mutedStyle.Render(m.targetItem.desc))
	}
	rows = append(rows, "")

	for n, v := range m.variables {
		marker, label := "  ", mutedStyle.Width(labelW).Render(v)
		if n == m.varIndex {
			marker, label = accentStyle.Render("› "), accentStyle.Width(labelW).Render(v)
		}
		rows = append(rows, marker+label+"  "+m.inputs[n].View())
	}

	previewW := min(maxWidth-cardStyle.GetHorizontalFrameSize(), 72)
	rows = append(rows, "", labelStyle.Render("PREVIEW"),
		renderCommand(m.targetItem.command, m.values(), m.variables[m.varIndex], previewW))

	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func emptyView() string {
	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("snip"),
		"",
		"No snippets yet",
		"",
		mutedStyle.Render("Save your first one with"),
		codeStyle.Render(`snip save --command "your command" <name>`),
	))
}
