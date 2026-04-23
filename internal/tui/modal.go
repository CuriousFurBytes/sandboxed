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

// LaunchModal runs args in a modal sub-terminal.
//
// When tuios is available and a tuios session is active, the command is opened
// as a new floating pane above the current terminal. Otherwise it runs inline
// with a styled header banner.
func LaunchModal(title string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command to run")
	}
	if isTuiosSession() {
		return launchWithTuios(title, args)
	}
	return launchInline(title, args)
}

// isTuiosSession returns true when running inside an active tuios session.
func isTuiosSession() bool {
	if _, err := exec.LookPath("tuios"); err != nil {
		return false
	}
	// tuios sets TUIOS_SESSION or similar; fall back to checking the env var
	return os.Getenv("TUIOS_SESSION") != "" || os.Getenv("TUIOS_PANE_ID") != ""
}

// launchWithTuios spawns a floating pane in the current tuios session.
func launchWithTuios(title string, args []string) error {
	tuiosArgs := []string{"spawn", "--title", title, "--float", "--"}
	tuiosArgs = append(tuiosArgs, args...)
	cmd := exec.Command("tuios", tuiosArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// launchInline runs args in the current terminal with a styled header.
func launchInline(title string, args []string) error {
	header := modalHeaderStyle.Render(fmt.Sprintf("  %s  ", title))
	fmt.Fprintln(os.Stderr, header)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
