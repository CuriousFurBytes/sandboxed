package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/CuriousFurBytes/sandboxed/internal/config"
	"github.com/CuriousFurBytes/sandboxed/internal/runtime"
)

// PodmanRunner abstracts podman command execution for testability.
type PodmanRunner interface {
	Run(args ...string) ([]byte, error)
	ContainerExists(name string) bool
	ContainerRunning(name string) bool
	ImageExists(image string) bool
	Start(name string) error
	Pause(name string) error
	Unpause(name string) error
}

// RuntimeDetector picks the OCI runtime to use.
type RuntimeDetector interface {
	Pick() (string, bool)
}

// SandboxInfo is a summary of a running or stopped sandbox container.
type SandboxInfo struct {
	Name     string
	State    string
	HostPath string
}

// Manager handles the full sandbox container lifecycle.
type Manager struct {
	cfg      config.Config
	podman   PodmanRunner
	detector RuntimeDetector
}

// NewManager creates a Manager using real podman and runtime detection.
func NewManager(cfg config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		podman:   &realPodman{},
		detector: runtime.New(cfg),
	}
}

// NewManagerWithRunner creates a Manager with injected dependencies (for tests).
func NewManagerWithRunner(cfg config.Config, podman PodmanRunner, detector RuntimeDetector) *Manager {
	return &Manager{cfg: cfg, podman: podman, detector: detector}
}

// Create creates a new sandbox container for hostPath.
func (m *Manager) Create(hostPath string) error {
	if !m.podman.ImageExists(m.cfg.Image) {
		return fmt.Errorf("base image %s not found; run 'sbx build' to create it", m.cfg.Image)
	}

	name := ID(hostPath)
	if m.podman.ContainerExists(name) {
		return fmt.Errorf("sandbox already exists for %s (container: %s)", hostPath, name)
	}

	upper := filepath.Join(m.cfg.OverlayDir, name, "upper")
	work := filepath.Join(m.cfg.OverlayDir, name, "work")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		return fmt.Errorf("create overlay dirs: %w", err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return fmt.Errorf("create overlay dirs: %w", err)
	}

	rt, _ := m.detector.Pick()

	if _, err := m.podman.Run(
		"run", "-d",
		"--name", name,
		"--runtime", rt,
		"--hostname", "sbx",
		"--network", m.cfg.Network,
		"--memory", m.cfg.Memory,
		"--cpus", m.cfg.CPUs,
		"--userns=keep-id",
		"--security-opt", "label=disable",
		"--mount", fmt.Sprintf("type=overlay,source=%s,destination=/workspace,upperdir=%s,workdir=%s", hostPath, upper, work),
		"-w", "/workspace",
		"--label", fmt.Sprintf("sandbox.host_path=%s", hostPath),
		"--label", fmt.Sprintf("sandbox.id=%s", name),
		m.cfg.Image,
		"sleep", "infinity",
	); err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	return WriteMeta(m.cfg.StateDir, Meta{
		Name:     name,
		HostPath: hostPath,
		Image:    m.cfg.Image,
		Runtime:  rt,
		Upper:    upper,
		Created:  time.Now(),
	})
}

// Remove stops and deletes the sandbox for hostPath.
func (m *Manager) Remove(hostPath string) error {
	name := ID(hostPath)
	if !m.podman.ContainerExists(name) {
		return fmt.Errorf("no sandbox for %s", hostPath)
	}
	if _, err := m.podman.Run("rm", "-f", name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	_ = os.RemoveAll(filepath.Join(m.cfg.OverlayDir, name))
	_ = DeleteMeta(m.cfg.StateDir, name)
	return nil
}

// EnsureRunning ensures name is running, starting it if stopped.
func (m *Manager) EnsureRunning(name string) error {
	if !m.podman.ContainerExists(name) {
		return fmt.Errorf("no sandbox container %s; run 'sbx create' first", name)
	}
	if m.podman.ContainerRunning(name) {
		return nil
	}
	return m.podman.Start(name)
}

// List returns all sandbox containers tracked by sbx labels.
func (m *Manager) List() ([]SandboxInfo, error) {
	out, err := m.podman.Run(
		"ps", "-a",
		"--filter", "label=sandbox.id",
		"--format", `{{.Names}}\t{{.State}}\t{{index .Labels "sandbox.host_path"}}`,
	)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	return parsePSOutput(string(out)), nil
}

// Prune removes all sbx-tracked sandbox containers and their state.
func (m *Manager) Prune() error {
	infos, err := m.List()
	if err != nil {
		return err
	}
	for _, info := range infos {
		if _, err := m.podman.Run("rm", "-f", info.Name); err != nil {
			return fmt.Errorf("remove %s: %w", info.Name, err)
		}
		_ = os.RemoveAll(filepath.Join(m.cfg.OverlayDir, info.Name))
		_ = DeleteMeta(m.cfg.StateDir, info.Name)
	}
	return nil
}

// GetMeta reads the stored metadata for a sandbox by container name.
func (m *Manager) GetMeta(name string) (Meta, error) {
	return ReadMeta(m.cfg.StateDir, name)
}

// Cfg returns the manager's configuration (used by runner/sync).
func (m *Manager) Cfg() config.Config { return m.cfg }

// Podman returns the manager's PodmanRunner (used by runner).
func (m *Manager) Podman() PodmanRunner { return m.podman }

func parsePSOutput(out string) []SandboxInfo {
	var infos []SandboxInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		infos = append(infos, SandboxInfo{
			Name:     parts[0],
			State:    parts[1],
			HostPath: parts[2],
		})
	}
	return infos
}

// realPodman implements PodmanRunner against the actual podman binary.
type realPodman struct{}

func (p *realPodman) Run(args ...string) ([]byte, error) {
	return exec.Command("podman", args...).Output()
}

func (p *realPodman) ContainerExists(name string) bool {
	return exec.Command("podman", "container", "exists", name).Run() == nil
}

func (p *realPodman) ContainerRunning(name string) bool {
	out, err := exec.Command("podman", "container", "inspect", "--format", "{{.State.Running}}", name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func (p *realPodman) ImageExists(image string) bool {
	return exec.Command("podman", "image", "exists", image).Run() == nil
}

func (p *realPodman) Start(name string) error {
	return exec.Command("podman", "start", name).Run()
}

func (p *realPodman) Pause(name string) error {
	return exec.Command("podman", "pause", name).Run()
}

func (p *realPodman) Unpause(name string) error {
	return exec.Command("podman", "unpause", name).Run()
}
