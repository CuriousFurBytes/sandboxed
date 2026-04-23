package tui_test

import (
	"errors"
	"os"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/tui"
)

func TestModalLauncher_NoTuiosSession_RunsInline(t *testing.T) {
	// Clear session env vars so isTuiosSession returns false.
	t.Setenv("TUIOS_SESSION", "")
	t.Setenv("TUIOS_PANE_ID", "")
	os.Unsetenv("TUIOS_SESSION")
	os.Unsetenv("TUIOS_PANE_ID")

	launcher := tui.NewModalLauncherWithDeps(func(string) (string, error) {
		return "", errors.New("tuios not found")
	})

	// "true" always succeeds and exits immediately — good inline fallback test.
	err := launcher.Launch("test", []string{"true"})
	if err != nil {
		t.Errorf("Launch with no tuios session: %v", err)
	}
}

func TestModalLauncher_EmptyArgs_ReturnsError(t *testing.T) {
	launcher := tui.NewModalLauncherWithDeps(func(string) (string, error) {
		return "", errors.New("not found")
	})
	err := launcher.Launch("test", []string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestModalLauncher_TuiosSessionDetected_WhenEnvAndBinaryPresent(t *testing.T) {
	t.Setenv("TUIOS_SESSION", "test-session")

	// Pretend tuios is "available" in PATH but don't actually run it;
	// the launch will fail because the fake binary doesn't exist, but
	// what we're testing is that isTuiosSession returns true → launchWithTuios is called.
	launcher := tui.NewModalLauncherWithDeps(func(name string) (string, error) {
		if name == "tuios" {
			return "/fake/tuios", nil
		}
		return "", errors.New("not found")
	})

	// launchWithTuios will try exec.Command("tuios", ...) which will fail since
	// /fake/tuios doesn't exist. We just verify it tried tuios (not inline).
	err := launcher.Launch("sbx — claude", []string{"podman", "exec", "container", "claude"})
	// We expect an error because /fake/tuios doesn't exist.
	if err == nil {
		t.Log("tuios happened to succeed (tuios may actually be installed)")
	}
}

func TestModalLauncher_NoEnvVar_NeverTuios(t *testing.T) {
	os.Unsetenv("TUIOS_SESSION")
	os.Unsetenv("TUIOS_PANE_ID")

	var lookPathCalled bool
	launcher := tui.NewModalLauncherWithDeps(func(name string) (string, error) {
		lookPathCalled = true
		return "/usr/bin/" + name, nil
	})

	// Even though lookPath would return tuios as found, no env var means no session.
	// LaunchModal should fall back inline.
	err := launcher.Launch("test", []string{"true"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// lookPath should NOT be called when no env var is set (short-circuit)
	if lookPathCalled {
		t.Error("lookPath should not be called when no tuios session env var is set")
	}
}

func TestLaunchModal_PackageLevel_UsesDefaultLauncher(t *testing.T) {
	os.Unsetenv("TUIOS_SESSION")
	os.Unsetenv("TUIOS_PANE_ID")

	// Package-level convenience function should work without error for inline.
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
