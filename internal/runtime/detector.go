package runtime

import (
	"os/exec"

	"github.com/CuriousFurBytes/sandboxed/internal/config"
)

// Detector picks the best available OCI runtime.
type Detector struct {
	cfg      config.Config
	lookPath func(string) (string, error)
}

// New creates a Detector using the real exec.LookPath.
func New(cfg config.Config) *Detector {
	return &Detector{cfg: cfg, lookPath: exec.LookPath}
}

// NewWithLookPath creates a Detector with a custom resolver, used in tests.
func NewWithLookPath(cfg config.Config, lookPath func(string) (string, error)) *Detector {
	return &Detector{cfg: cfg, lookPath: lookPath}
}

// Pick returns the runtime name to use and whether it provides VM isolation.
// Prefers krun when available; falls back to crun.
func (d *Detector) Pick() (runtime string, isVM bool) {
	if d.cfg.Runtime != "krun" {
		return d.cfg.Runtime, false
	}
	if d.hasKrun() {
		return "krun", true
	}
	return "crun", false
}

// Available returns every installed OCI runtime binary name.
func (d *Detector) Available() []string {
	var found []string
	for _, name := range []string{"krun", "crun-krun", "crun"} {
		if _, err := d.lookPath(name); err == nil {
			found = append(found, name)
		}
	}
	return found
}

func (d *Detector) hasKrun() bool {
	if _, err := d.lookPath("krun"); err == nil {
		return true
	}
	if _, err := d.lookPath("crun-krun"); err == nil {
		return true
	}
	return false
}
