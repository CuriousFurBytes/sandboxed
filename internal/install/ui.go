package install

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	uiTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#25A065"))
	uiOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#25A065")).Render("✓")
	uiFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F57")).Render("✗")
	uiWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("!")
	uiSubtext = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

type doctorModel struct {
	result Result
	done   bool
}

// NewDoctorModel creates a doctorModel for testing without running the full TUI.
func NewDoctorModel(result Result) tea.Model {
	return doctorModel{result: result}
}

func (m doctorModel) Init() tea.Cmd { return nil }

func (m doctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m doctorModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(uiTitle.Render("sbx doctor — dependency check") + "\n\n")

	if m.result.OK() && len(m.result.Warnings) == 0 {
		b.WriteString(uiOK + "  All dependencies present. Ready to use sbx.\n")
	} else {
		for _, r := range m.result.Missing {
			b.WriteString(fmt.Sprintf("%s  %-20s %s\n", uiFail, r.Name, uiSubtext.Render(r.Description)))
		}
		for _, r := range m.result.Warnings {
			b.WriteString(fmt.Sprintf("%s  %-20s %s\n", uiWarn, r.Name, uiSubtext.Render(r.Description)))
		}
	}

	b.WriteString("\n" + uiSubtext.Render("Press any key to exit.") + "\n")
	return b.String()
}

// RunDoctorUI displays an interactive dependency check using bubbletea.
func RunDoctorUI(result Result) error {
	_, err := tea.NewProgram(doctorModel{result: result}).Run()
	return err
}

// PrintDoctorReport prints a plain-text dependency report to stdout.
func PrintDoctorReport(result Result) {
	fmt.Println("sbx doctor — dependency check")
	fmt.Println()
	if result.OK() && len(result.Warnings) == 0 {
		fmt.Println("✓  All dependencies present. Ready to use sbx.")
		return
	}
	for _, r := range result.Missing {
		fmt.Printf("✗  MISSING  %-20s %s\n", r.Name, r.Description)
	}
	for _, r := range result.Warnings {
		fmt.Printf("!  OPTIONAL %-20s %s\n", r.Name, r.Description)
	}
	if !result.OK() {
		fmt.Println("\nInstall missing dependencies, then re-run 'sbx doctor'.")
	}
}
