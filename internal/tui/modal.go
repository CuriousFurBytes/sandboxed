package tui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/lipgloss"
)

var modalHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FFFDF5")).
	Background(lipgloss.Color("#25A065")).
	Padding(0, 2)

// ModalLauncher runs commands inline with a styled header banner.
type ModalLauncher struct{}

// NewModalLauncher creates a ModalLauncher.
func NewModalLauncher() *ModalLauncher {
	return &ModalLauncher{}
}

// Launch runs args inline with a styled header banner.
func (l *ModalLauncher) Launch(title string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command to run")
	}
	return launchInline(title, args)
}

func launchInline(title string, args []string) error {
	header := modalHeaderStyle.Render(fmt.Sprintf("  %s  ", title))
	fmt.Fprintln(os.Stderr, header)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// LaunchModal is a package-level convenience using the default launcher.
func LaunchModal(title string, args []string) error {
	return NewModalLauncher().Launch(title, args)
}
