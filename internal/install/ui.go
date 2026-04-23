package install

import (
	"fmt"
	"strings"

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
}

// NewDoctorModel creates a doctorModel for use in tests or direct rendering.
func NewDoctorModel(result Result) *doctorModel {
	return &doctorModel{result: result}
}

func (m doctorModel) View() string {
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

	return b.String()
}

// RunDoctorUI prints a colored dependency check report to stdout.
func RunDoctorUI(result Result) error {
	fmt.Print(NewDoctorModel(result).View())
	return nil
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
