package runtime_test

import (
	"errors"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/config"
	"github.com/CuriousFurBytes/sandboxed/internal/runtime"
)

func TestPick_UsesKrunWhenAvailable(t *testing.T) {
	d := newDetector("krun", map[string]bool{"krun": true})
	got, isVM := d.Pick()
	if got != "krun" {
		t.Errorf("got %q, want krun", got)
	}
	if !isVM {
		t.Error("expected isVM=true when krun is available")
	}
}

func TestPick_FallsBackToCrunWhenKrunMissing(t *testing.T) {
	d := newDetector("krun", map[string]bool{})
	got, isVM := d.Pick()
	if got != "crun" {
		t.Errorf("got %q, want crun", got)
	}
	if isVM {
		t.Error("expected isVM=false when krun is missing")
	}
}

func TestPick_UsesCrunKrunAlternative(t *testing.T) {
	d := newDetector("krun", map[string]bool{"crun-krun": true})
	got, isVM := d.Pick()
	if got != "krun" {
		t.Errorf("got %q, want krun", got)
	}
	if !isVM {
		t.Error("expected isVM=true when crun-krun is available")
	}
}

func TestPick_RespectsExplicitCrunOverride(t *testing.T) {
	d := newDetector("crun", map[string]bool{"krun": true, "crun": true})
	got, isVM := d.Pick()
	if got != "crun" {
		t.Errorf("got %q, want crun", got)
	}
	if isVM {
		t.Error("expected isVM=false when explicitly using crun")
	}
}

func TestAvailable_ReturnsInstalledRuntimes(t *testing.T) {
	d := newDetector("krun", map[string]bool{"krun": true, "crun": true})
	avail := d.Available()
	if len(avail) < 2 {
		t.Errorf("expected at least 2 available runtimes, got %v", avail)
	}
}

func newDetector(runtimePref string, installed map[string]bool) *runtime.Detector {
	cfg := config.Config{Runtime: runtimePref}
	return runtime.NewWithLookPath(cfg, func(name string) (string, error) {
		if installed[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	})
}
