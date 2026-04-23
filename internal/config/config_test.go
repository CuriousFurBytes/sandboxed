package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	clearSandboxEnv(t)
	cfg := config.Load()

	if cfg.Image != "localhost/sandbox-base:latest" {
		t.Errorf("Image: got %q want localhost/sandbox-base:latest", cfg.Image)
	}
	if cfg.Runtime != "krun" {
		t.Errorf("Runtime: got %q want krun", cfg.Runtime)
	}
	if cfg.Network != "slirp4netns:enable_ipv6=true" {
		t.Errorf("Network: got %q want slirp4netns:enable_ipv6=true", cfg.Network)
	}
	if cfg.Memory != "4g" {
		t.Errorf("Memory: got %q want 4g", cfg.Memory)
	}
	if cfg.CPUs != "4" {
		t.Errorf("CPUs: got %q want 4", cfg.CPUs)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearSandboxEnv(t)
	t.Setenv("SANDBOX_IMAGE", "myrepo/myimage:v1")
	t.Setenv("SANDBOX_RUNTIME", "crun")
	t.Setenv("SANDBOX_NET", "bridge")
	t.Setenv("SANDBOX_MEMORY", "8g")
	t.Setenv("SANDBOX_CPUS", "8")

	cfg := config.Load()

	if cfg.Image != "myrepo/myimage:v1" {
		t.Errorf("Image: got %q", cfg.Image)
	}
	if cfg.Runtime != "crun" {
		t.Errorf("Runtime: got %q", cfg.Runtime)
	}
	if cfg.Network != "bridge" {
		t.Errorf("Network: got %q", cfg.Network)
	}
	if cfg.Memory != "8g" {
		t.Errorf("Memory: got %q", cfg.Memory)
	}
	if cfg.CPUs != "8" {
		t.Errorf("CPUs: got %q", cfg.CPUs)
	}
}

func TestLoad_XDGStatePath(t *testing.T) {
	clearSandboxEnv(t)
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	cfg := config.Load()
	want := filepath.Join(tmp, "sandbox")
	if cfg.StateDir != want {
		t.Errorf("StateDir: got %q want %q", cfg.StateDir, want)
	}
}

func TestLoad_XDGDataPath(t *testing.T) {
	clearSandboxEnv(t)
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	cfg := config.Load()
	wantData := filepath.Join(tmp, "sandbox")
	wantOverlay := filepath.Join(tmp, "sandbox", "overlays")

	if cfg.DataDir != wantData {
		t.Errorf("DataDir: got %q want %q", cfg.DataDir, wantData)
	}
	if cfg.OverlayDir != wantOverlay {
		t.Errorf("OverlayDir: got %q want %q", cfg.OverlayDir, wantOverlay)
	}
}

func TestLoad_DefaultXDGUsesHome(t *testing.T) {
	clearSandboxEnv(t)
	home, _ := os.UserHomeDir()

	cfg := config.Load()

	wantState := filepath.Join(home, ".local", "state", "sandbox")
	wantData := filepath.Join(home, ".local", "share", "sandbox")

	if cfg.StateDir != wantState {
		t.Errorf("StateDir: got %q want %q", cfg.StateDir, wantState)
	}
	if cfg.DataDir != wantData {
		t.Errorf("DataDir: got %q want %q", cfg.DataDir, wantData)
	}
}

// clearSandboxEnv removes all sandbox-related env vars for the test duration.
func clearSandboxEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SANDBOX_IMAGE", "SANDBOX_RUNTIME", "SANDBOX_NET",
		"SANDBOX_MEMORY", "SANDBOX_CPUS",
		"XDG_STATE_HOME", "XDG_DATA_HOME",
	} {
		t.Setenv(key, "") // t.Setenv restores on cleanup; empty string → envOr uses fallback
		os.Unsetenv(key)
	}
}
