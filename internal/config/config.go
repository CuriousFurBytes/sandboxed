package config

import (
	"os"
	"path/filepath"
)

// Config holds all runtime configuration for sbx, sourced from environment
// variables with sensible defaults matching the original shell implementation.
type Config struct {
	Image   string
	Runtime string
	Network string
	Memory  string
	CPUs    string

	StateDir   string
	DataDir    string
	OverlayDir string
}

// Load reads configuration from environment variables, falling back to defaults.
func Load() Config {
	home, _ := os.UserHomeDir()
	xdgState := envOr("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	xdgData := envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	dataDir := filepath.Join(xdgData, "sandbox")

	return Config{
		Image:   envOr("SANDBOX_IMAGE", "localhost/sandbox-base:latest"),
		Runtime: envOr("SANDBOX_RUNTIME", "krun"),
		Network: envOr("SANDBOX_NET", "slirp4netns:enable_ipv6=true"),
		Memory:  envOr("SANDBOX_MEMORY", "4g"),
		CPUs:    envOr("SANDBOX_CPUS", "4"),

		StateDir:   filepath.Join(xdgState, "sandbox"),
		DataDir:    dataDir,
		OverlayDir: filepath.Join(dataDir, "overlays"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
