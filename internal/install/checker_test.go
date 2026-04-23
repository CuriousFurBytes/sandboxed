package install_test

import (
	"errors"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/install"
)

func TestCheck_AllPresent(t *testing.T) {
	c := install.NewWithDeps(
		func(name string) (string, error) { return "/usr/bin/" + name, nil },
		func(path string) ([]byte, error) { return []byte("testuser:100000:65536\n"), nil },
		func() (string, error) { return "testuser", nil },
	)
	result := c.Check()
	if !result.OK() {
		t.Errorf("expected OK, but missing: %v", result.Missing)
	}
}

func TestCheck_MissingPodman(t *testing.T) {
	c := install.NewWithDeps(
		func(name string) (string, error) {
			if name == "podman" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
		func(path string) ([]byte, error) { return []byte("testuser:100000:65536\n"), nil },
		func() (string, error) { return "testuser", nil },
	)
	result := c.Check()
	if result.OK() {
		t.Error("expected not OK when podman is missing")
	}
	found := false
	for _, m := range result.Missing {
		if m.Name == "podman" {
			found = true
		}
	}
	if !found {
		t.Error("expected podman in Missing list")
	}
}

func TestCheck_MissingKrunIsWarning(t *testing.T) {
	c := install.NewWithDeps(
		func(name string) (string, error) {
			if name == "krun" || name == "crun-krun" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
		func(path string) ([]byte, error) { return []byte("testuser:100000:65536\n"), nil },
		func() (string, error) { return "testuser", nil },
	)
	result := c.Check()
	if !result.OK() {
		t.Errorf("missing krun should not fail required check, missing: %v", result.Missing)
	}
	found := false
	for _, w := range result.Warnings {
		if w.Name == "krun" {
			found = true
		}
	}
	if !found {
		t.Error("expected krun in Warnings list")
	}
}

func TestCheck_MissingSubUID(t *testing.T) {
	c := install.NewWithDeps(
		func(name string) (string, error) { return "/usr/bin/" + name, nil },
		func(path string) ([]byte, error) {
			if path == "/etc/subuid" {
				return []byte("otheruser:100000:65536\n"), nil
			}
			return []byte("testuser:100000:65536\n"), nil
		},
		func() (string, error) { return "testuser", nil },
	)
	result := c.Check()
	if result.OK() {
		t.Error("expected not OK when subuid entry is missing for current user")
	}
}

func TestCheck_MissingRsyncIsWarning(t *testing.T) {
	c := install.NewWithDeps(
		func(name string) (string, error) {
			if name == "rsync" {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + name, nil
		},
		func(path string) ([]byte, error) { return []byte("testuser:100000:65536\n"), nil },
		func() (string, error) { return "testuser", nil },
	)
	result := c.Check()
	if !result.OK() {
		t.Errorf("missing rsync should not fail required check, missing: %v", result.Missing)
	}
}
