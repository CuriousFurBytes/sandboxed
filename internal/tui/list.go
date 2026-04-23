package tui

import (
	"fmt"
	"io"
	"strings"

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

// FilterValue includes name and path for filtering.
func (i SbxItem) FilterValue() string { return i.Name + " " + i.HostPath }

// Title returns the sandbox name.
func (i SbxItem) Title() string { return i.Name }

// Description returns the host path.
func (i SbxItem) Description() string { return i.HostPath }

// SbxDelegate renders each SbxItem.
type SbxDelegate struct{}

// NewSbxDelegate creates a SbxDelegate.
func NewSbxDelegate() SbxDelegate { return SbxDelegate{} }

func (d SbxDelegate) Height() int  { return 2 }
func (d SbxDelegate) Spacing() int { return 1 }

func (d SbxDelegate) Render(w io.Writer, index int, selected bool, item SbxItem) {
	var stateStr string
	if item.State == "running" {
		stateStr = stateRunning.Render("● " + item.State)
	} else {
		stateStr = stateStopped.Render("○ " + item.State)
	}

	nameStr := item.Name
	pathStr := item.HostPath

	if selected {
		nameStr = itemSelected.Render(nameStr)
		pathStr = itemSelected.Render(pathStr)
	} else {
		nameStr = itemNormal.Render(nameStr)
		pathStr = itemNormal.Render(pathStr)
	}

	fmt.Fprintf(w, "  %s  %s\n  %s\n", stateStr, nameStr, pathStr)
}

type listModel struct {
	infos []sandbox.SandboxInfo
	done  bool
}

// NewListModel constructs a listModel for the given sandbox infos.
func NewListModel(infos []sandbox.SandboxInfo) *listModel {
	return &listModel{infos: infos}
}

// NewDoneListModel returns a listModel already in the quit state (View returns "").
func NewDoneListModel() *listModel {
	return &listModel{done: true}
}

func (m *listModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("sbx — sandboxes") + "\n\n")
	d := SbxDelegate{}
	for i, info := range m.infos {
		d.Render(&b, i, false, SbxItem{info})
		b.WriteString("\n")
	}
	return b.String()
}

// RunList prints a list of sandboxes to stdout.
func RunList(infos []sandbox.SandboxInfo) error {
	fmt.Print(NewListModel(infos).View())
	return nil
}
