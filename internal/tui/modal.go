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

// ModalLauncher runs commands in a tuios floating pane or inline.
type ModalLauncher struct {
	lookPath func(string) (string, error)
}

// NewModalLauncher creates a ModalLauncher using the real exec.LookPath.
func NewModalLauncher() *ModalLauncher {
	return &ModalLauncher{lookPath: exec.LookPath}
}

// NewModalLauncherWithDeps creates a ModalLauncher with an injectable path resolver (for tests).
func NewModalLauncherWithDeps(lookPath func(string) (string, error)) *ModalLauncher {
	return &ModalLauncher{lookPath: lookPath}
}

// Launch runs args in a tuios modal when inside a tuios session, otherwise inline.
func (l *ModalLauncher) Launch(title string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command to run")
	}
	if l.isTuiosSession() {
		return l.launchWithTuios(title, args)
	}
	return launchInline(title, args)
}

// isTuiosSession returns true when running inside an active tuios session
// and the tuios binary is available.
func (l *ModalLauncher) isTuiosSession() bool {
	if os.Getenv("TUIOS_SESSION") == "" && os.Getenv("TUIOS_PANE_ID") == "" {
		return false
	}
	_, err := l.lookPath("tuios")
	return err == nil
}

// launchWithTuios spawns a floating pane in the current tuios session.
func (l *ModalLauncher) launchWithTuios(title string, args []string) error {
	tuiosArgs := []string{"spawn", "--title", title, "--float", "--"}
	tuiosArgs = append(tuiosArgs, args...)
	cmd := exec.Command("tuios", tuiosArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// launchInline runs args in the current terminal with a styled header banner.
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
