package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Adaptive palette so the TUI reads well on both light and dark terminals.
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "#5A3FD1", Dark: "#9D7CFF"}
	colorText    = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E6E6E6"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#8A8A8A"}
	colorFaint   = lipgloss.AdaptiveColor{Light: "#B8B8B8", Dark: "#4A4A4A"}
	colorCodeBg  = lipgloss.AdaptiveColor{Light: "#F2F0FA", Dark: "#24212E"}
	colorTag     = lipgloss.AdaptiveColor{Light: "#0F7B8A", Dark: "#5FD7E0"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#08875D", Dark: "#04D98B"}
	colorWarn    = lipgloss.AdaptiveColor{Light: "#B25E00", Dark: "#FFB454"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#C4204A", Dark: "#FF6B8B"}
)

var (
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
	titleStyle = lipgloss.NewStyle().Background(colorAccent).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Padding(0, 1)

	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	accentStyle  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tagStyle     = lipgloss.NewStyle().Foreground(colorTag)
	labelStyle   = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	codeStyle    = lipgloss.NewStyle().Background(colorCodeBg).Foreground(colorText).Padding(0, 1)
	varSetStyle  = lipgloss.NewStyle().Background(colorCodeBg).Foreground(colorSuccess).Bold(true)
	varOpenStyle = lipgloss.NewStyle().Background(colorCodeBg).Foreground(colorWarn)
	varFocus     = lipgloss.NewStyle().Background(colorWarn).Foreground(lipgloss.Color("#1A1A1A")).Bold(true)

	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorFaint).Padding(0, 1)
	cardStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(1, 2)

	statusStyles = map[statusKind]lipgloss.Style{
		statusInfo:    lipgloss.NewStyle().Foreground(colorAccent),
		statusSuccess: lipgloss.NewStyle().Foreground(colorSuccess).Bold(true),
		statusError:   lipgloss.NewStyle().Foreground(colorDanger).Bold(true),
		statusConfirm: lipgloss.NewStyle().Foreground(colorDanger).Bold(true),
	}
)

// snippetDelegate renders each snippet as a two-line row with an accent bar on the selection.
type snippetDelegate struct{}

func (d snippetDelegate) Height() int                             { return 2 }
func (d snippetDelegate) Spacing() int                            { return 1 }
func (d snippetDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d snippetDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	it, ok := li.(item)
	if !ok || m.Width() <= 0 {
		return
	}

	selected := index == m.Index()
	width := m.Width() - 2 // room for the selection bar

	bar := "  "
	nameStyle := lipgloss.NewStyle().Foreground(colorText)
	subStyle := mutedStyle
	if selected {
		bar = accentStyle.Render("▌ ")
		nameStyle = accentStyle
		subStyle = lipgloss.NewStyle().Foreground(colorText)
	}

	// Underline the characters that match the active filter (they index into FilterValue,
	// which begins with the title).
	var titleMatches []int
	for _, i := range m.MatchesForItem(index) {
		if i < len([]rune(it.title)) {
			titleMatches = append(titleMatches, i)
		}
	}
	name := lipgloss.StyleRunes(it.title, titleMatches, nameStyle.Underline(true), nameStyle)

	var tags []string
	for _, t := range it.tags {
		tags = append(tags, "#"+t)
	}
	left := name
	if len(tags) > 0 {
		left += "  " + tagStyle.Render(strings.Join(tags, " "))
	}

	right := ""
	if it.usageCount > 0 {
		right = mutedStyle.Render(fmt.Sprintf("%d×", it.usageCount))
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		left = ansi.Truncate(left, width-lipgloss.Width(right)-2, "…")
		gap = width - lipgloss.Width(left) - lipgloss.Width(right)
	}
	line1 := left + strings.Repeat(" ", max(gap, 1)) + right

	sub := it.desc
	if sub == "" {
		sub = "$ " + firstLine(it.command)
	}
	line2 := subStyle.Render(ansi.Truncate(sub, width, "…"))

	fmt.Fprintf(w, "%s%s\n%s%s", bar, line1, bar, line2)
}

// renderCommand highlights {{var}} placeholders, showing filled values in green,
// pending placeholders in amber, and the currently focused one inverted.
func renderCommand(command string, values map[string]string, focused string, width int) string {
	var b strings.Builder
	last := 0
	for _, loc := range varRe.FindAllStringSubmatchIndex(command, -1) {
		b.WriteString(codeStyle.UnsetPadding().Render(command[last:loc[0]]))
		name := command[loc[2]:loc[3]]
		switch v := values[name]; {
		case name == focused && v == "":
			b.WriteString(varFocus.Render(command[loc[0]:loc[1]]))
		case name == focused:
			b.WriteString(varFocus.Render(v))
		case v != "":
			b.WriteString(varSetStyle.Render(v))
		default:
			b.WriteString(varOpenStyle.Render(command[loc[0]:loc[1]]))
		}
		last = loc[1]
	}
	b.WriteString(codeStyle.UnsetPadding().Render(command[last:]))
	return codeStyle.Width(width).Render(b.String())
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}
