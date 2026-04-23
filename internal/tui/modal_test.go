package tui_test

import (
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/tui"
)

func TestModalLauncher_RunsInline(t *testing.T) {
	launcher := tui.NewModalLauncher()
	err := launcher.Launch("test", []string{"true"})
	if err != nil {
		t.Errorf("Launch: %v", err)
	}
}

func TestModalLauncher_EmptyArgs_ReturnsError(t *testing.T) {
	launcher := tui.NewModalLauncher()
	err := launcher.Launch("test", []string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestLaunchModal_PackageLevel_UsesDefaultLauncher(t *testing.T) {
	err := tui.LaunchModal("test title", []string{"true"})
	if err != nil {
		t.Errorf("LaunchModal: %v", err)
	}
}

func TestNewModalLauncher_IsNotNil(t *testing.T) {
	l := tui.NewModalLauncher()
	if l == nil {
		t.Error("NewModalLauncher() must not return nil")
	}
}
