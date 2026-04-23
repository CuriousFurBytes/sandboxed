package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	stateRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("#25A065"))
	stateStopped = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F57"))
	itemSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFDF5"))
	itemNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))
)

type sbxItem struct {
	sandbox.SandboxInfo
}

func (i sbxItem) FilterValue() string { return i.Name + " " + i.HostPath }
func (i sbxItem) Title() string       { return i.Name }
func (i sbxItem) Description() string { return i.HostPath }

type sbxDelegate struct{}

func (d sbxDelegate) Height() int                             { return 2 }
func (d sbxDelegate) Spacing() int                            { return 1 }
func (d sbxDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sbxDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(sbxItem)
	if !ok {
		return
	}

	var stateStr string
	if item.State == "running" {
		stateStr = stateRunning.Render("● " + item.State)
	} else {
		stateStr = stateStopped.Render("○ " + item.State)
	}

	nameStr := item.Name
	pathStr := item.HostPath

	if index == m.Index() {
		nameStr = itemSelected.Render(nameStr)
		pathStr = itemSelected.Render(pathStr)
	} else {
		nameStr = itemNormal.Render(nameStr)
		pathStr = itemNormal.Render(pathStr)
	}

	fmt.Fprintf(w, "  %s  %s\n  %s\n", stateStr, nameStr, pathStr)
}

type listModel struct {
	list list.Model
	done bool
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	if m.done {
		return ""
	}
	return m.list.View()
}

// RunList displays an interactive filterable list of sandboxes.
// Returns nil when the user quits normally.
func RunList(infos []sandbox.SandboxInfo) error {
	items := make([]list.Item, len(infos))
	for i, info := range infos {
		items[i] = sbxItem{info}
	}

	l := list.New(items, sbxDelegate{}, 80, 24)
	l.Title = "sbx — sandboxes"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	m := listModel{list: l}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
