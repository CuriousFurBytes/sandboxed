package install

import (
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// Requirement describes a single system dependency.
type Requirement struct {
	Name        string
	Description string
	Required    bool
	CheckFn     func() bool
}

// Result holds the outcome of a full dependency check.
type Result struct {
	Missing  []Requirement
	Warnings []Requirement
}

// OK returns true when no required dependencies are missing.
func (r Result) OK() bool { return len(r.Missing) == 0 }

// Checker verifies system requirements for sbx.
type Checker struct {
	lookPath func(string) (string, error)
	readFile func(string) ([]byte, error)
	username func() (string, error)
}

// New creates a Checker using real system calls.
func New() *Checker {
	return &Checker{
		lookPath: exec.LookPath,
		readFile: os.ReadFile,
		username: func() (string, error) {
			u, err := user.Current()
			if err != nil {
				return "", err
			}
			return u.Username, nil
		},
	}
}

// NewWithDeps creates a Checker with injectable dependencies for testing.
func NewWithDeps(
	lookPath func(string) (string, error),
	readFile func(string) ([]byte, error),
	username func() (string, error),
) *Checker {
	return &Checker{lookPath: lookPath, readFile: readFile, username: username}
}

// Check runs all requirement checks and returns a Result.
func (c *Checker) Check() Result {
	var result Result
	for _, r := range c.requirements() {
		if !r.CheckFn() {
			if r.Required {
				result.Missing = append(result.Missing, r)
			} else {
				result.Warnings = append(result.Warnings, r)
			}
		}
	}
	return result
}

func (c *Checker) requirements() []Requirement {
	return []Requirement{
		{
			Name:        "podman",
			Description: "Container runtime (required)",
			Required:    true,
			CheckFn:     func() bool { _, err := c.lookPath("podman"); return err == nil },
		},
		{
			Name:        "krun",
			Description: "VM-isolated OCI runtime (optional, falls back to crun)",
			Required:    false,
			CheckFn: func() bool {
				_, e1 := c.lookPath("krun")
				_, e2 := c.lookPath("crun-krun")
				return e1 == nil || e2 == nil
			},
		},
		{
			Name:        "fuse-overlayfs",
			Description: "Overlay filesystem for rootless containers (required)",
			Required:    true,
			CheckFn:     func() bool { _, err := c.lookPath("fuse-overlayfs"); return err == nil },
		},
		{
			Name:        "slirp4netns",
			Description: "User-space networking for rootless containers (required)",
			Required:    true,
			CheckFn:     func() bool { _, err := c.lookPath("slirp4netns"); return err == nil },
		},
		{
			Name:        "rsync",
			Description: "Required for 'sbx sync' (optional)",
			Required:    false,
			CheckFn:     func() bool { _, err := c.lookPath("rsync"); return err == nil },
		},
		{
			Name:        "subuid",
			Description: "User namespace mapping entry in /etc/subuid (required)",
			Required:    true,
			CheckFn:     func() bool { return c.hasSubEntry("/etc/subuid") },
		},
		{
			Name:        "subgid",
			Description: "User namespace mapping entry in /etc/subgid (required)",
			Required:    true,
			CheckFn:     func() bool { return c.hasSubEntry("/etc/subgid") },
		},
	}
}

func (c *Checker) hasSubEntry(path string) bool {
	username, err := c.username()
	if err != nil {
		return false
	}
	data, err := c.readFile(path)
	if err != nil {
		return false
	}
	prefix := username + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
