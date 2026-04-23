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

// SbxItem wraps a SandboxInfo for display in the list.
type SbxItem struct {
	sandbox.SandboxInfo
}

// NewSbxItem creates an SbxItem from a SandboxInfo.
func NewSbxItem(info sandbox.SandboxInfo) SbxItem {
	return SbxItem{info}
}

// FilterValue implements list.Item. Includes name and path for fuzzy filtering.
func (i SbxItem) FilterValue() string { return i.Name + " " + i.HostPath }

// Title implements list.DefaultItem.
func (i SbxItem) Title() string { return i.Name }

// Description implements list.DefaultItem.
func (i SbxItem) Description() string { return i.HostPath }

// SbxDelegate renders each SbxItem in the list.
type SbxDelegate struct{}

// NewSbxDelegate creates a SbxDelegate.
func NewSbxDelegate() SbxDelegate { return SbxDelegate{} }

func (d SbxDelegate) Height() int                             { return 2 }
func (d SbxDelegate) Spacing() int                            { return 1 }
func (d SbxDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d SbxDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(SbxItem)
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

// NewListModel constructs a listModel for the given sandbox infos.
// Returns tea.Model so external test code can call View() without knowing the concrete type.
func NewListModel(infos []sandbox.SandboxInfo) tea.Model {
	items := make([]list.Item, len(infos))
	for i, info := range infos {
		items[i] = SbxItem{info}
	}
	l := list.New(items, SbxDelegate{}, 80, 24)
	l.Title = "sbx — sandboxes"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	return listModel{list: l}
}

// NewDoneListModel returns a listModel already in the quit state (View returns "").
func NewDoneListModel() tea.Model {
	return listModel{done: true}
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
func RunList(infos []sandbox.SandboxInfo) error {
	_, err := tea.NewProgram(NewListModel(infos), tea.WithAltScreen()).Run()
	return err
}
