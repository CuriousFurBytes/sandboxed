package sandbox_test

import (
	"errors"
	"os"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/config"
	"github.com/CuriousFurBytes/sandboxed/internal/runtime"
	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

// mockPodman implements sandbox.PodmanRunner for unit tests.
type mockPodman struct {
	containers map[string]string // name → state ("running", "stopped", etc.)
	images     map[string]bool
	runOut     []byte
	runErr     error
	runCalls   [][]string
}

func (m *mockPodman) Run(args ...string) ([]byte, error) {
	m.runCalls = append(m.runCalls, append([]string{}, args...))
	return m.runOut, m.runErr
}

func (m *mockPodman) ContainerExists(name string) bool {
	_, ok := m.containers[name]
	return ok
}

func (m *mockPodman) ContainerRunning(name string) bool {
	return m.containers[name] == "running"
}

func (m *mockPodman) ImageExists(image string) bool {
	return m.images[image]
}

func (m *mockPodman) Start(name string) error {
	if _, ok := m.containers[name]; ok {
		m.containers[name] = "running"
		return nil
	}
	return errors.New("container not found")
}

func (m *mockPodman) Pause(name string) error  { return nil }
func (m *mockPodman) Unpause(name string) error { return nil }

func newTestManager(t *testing.T, pm *mockPodman) *sandbox.Manager {
	t.Helper()
	cfg := config.Config{
		Image:      "localhost/sandbox-base:latest",
		Runtime:    "krun",
		Network:    "slirp4netns",
		Memory:     "4g",
		CPUs:       "4",
		StateDir:   t.TempDir(),
		OverlayDir: t.TempDir(),
	}
	det := runtime.NewWithLookPath(cfg, func(string) (string, error) {
		return "", errors.New("not found")
	})
	return sandbox.NewManagerWithRunner(cfg, pm, det)
}

func TestCreate_Success(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{"localhost/sandbox-base:latest": true},
	}
	mgr := newTestManager(t, pm)
	hostDir := t.TempDir()
	if err := mgr.Create(hostDir); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(pm.runCalls) == 0 {
		t.Error("expected podman run to be called")
	}
	// First call must be "run" to create the container
	if pm.runCalls[0][0] != "run" {
		t.Errorf("expected first call to be 'run', got %q", pm.runCalls[0][0])
	}
}

func TestCreate_AlreadyExists(t *testing.T) {
	hostDir := t.TempDir()
	name := sandbox.ID(hostDir)
	pm := &mockPodman{
		containers: map[string]string{name: "running"},
		images:     map[string]bool{"localhost/sandbox-base:latest": true},
	}
	mgr := newTestManager(t, pm)
	err := mgr.Create(hostDir)
	if err == nil {
		t.Error("expected error when container already exists")
	}
}

func TestCreate_ImageMissing(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	err := mgr.Create(t.TempDir())
	if err == nil {
		t.Error("expected error when base image is missing")
	}
}

func TestCreate_WritesMeta(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{"localhost/sandbox-base:latest": true},
	}
	mgr := newTestManager(t, pm)
	hostDir := t.TempDir()
	if err := mgr.Create(hostDir); err != nil {
		t.Fatalf("Create: %v", err)
	}
	name := sandbox.ID(hostDir)
	meta, err := mgr.GetMeta(name)
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if meta.HostPath != hostDir {
		t.Errorf("meta.HostPath: got %q want %q", meta.HostPath, hostDir)
	}
}

func TestRemove_Success(t *testing.T) {
	hostDir := t.TempDir()
	name := sandbox.ID(hostDir)
	pm := &mockPodman{
		containers: map[string]string{name: "running"},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	if err := mgr.Remove(hostDir); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestRemove_NotExists(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	err := mgr.Remove("/nonexistent/path")
	if err == nil {
		t.Error("expected error when sandbox does not exist")
	}
}

func TestList_Empty(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
		runOut:     []byte(""),
	}
	mgr := newTestManager(t, pm)
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 sandboxes, got %d", len(infos))
	}
}

func TestList_WithContainers(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
		runOut:     []byte("sbx-proj-abc123\trunning\t/home/user/proj\n"),
	}
	mgr := newTestManager(t, pm)
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 sandbox, got %d", len(infos))
	}
	if infos[0].Name != "sbx-proj-abc123" {
		t.Errorf("Name: got %q", infos[0].Name)
	}
	if infos[0].State != "running" {
		t.Errorf("State: got %q", infos[0].State)
	}
	if infos[0].HostPath != "/home/user/proj" {
		t.Errorf("HostPath: got %q", infos[0].HostPath)
	}
}

func TestEnsureRunning_StartsStoppedContainer(t *testing.T) {
	name := "sbx-foo-abc1234567"
	pm := &mockPodman{
		containers: map[string]string{name: "stopped"},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	if err := mgr.EnsureRunning(name); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if pm.containers[name] != "running" {
		t.Errorf("expected container to be running after EnsureRunning")
	}
}

func TestEnsureRunning_MissingContainer(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	err := mgr.EnsureRunning("nonexistent-container")
	if err == nil {
		t.Error("expected error for missing container")
	}
}

func TestEnsureRunning_AlreadyRunningNoOp(t *testing.T) {
	name := "sbx-foo-abc1234567"
	pm := &mockPodman{
		containers: map[string]string{name: "running"},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	if err := mgr.EnsureRunning(name); err != nil {
		t.Fatalf("EnsureRunning on already-running container: %v", err)
	}
}

func TestPrune_RemovesAllContainers(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
		runOut:     []byte("sbx-a-aaa\tstopped\t/tmp/a\nsbx-b-bbb\tstopped\t/tmp/b\n"),
	}
	mgr := newTestManager(t, pm)
	if err := mgr.Prune(); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	// List is called first (1 call), then rm -f for each container (2 calls)
	rmCalls := 0
	for _, call := range pm.runCalls {
		if len(call) >= 2 && call[0] == "rm" && call[1] == "-f" {
			rmCalls++
		}
	}
	if rmCalls != 2 {
		t.Errorf("expected 2 rm -f calls, got %d (all calls: %v)", rmCalls, pm.runCalls)
	}
}

func TestPrune_EmptyListIsNoOp(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
		runOut:     []byte(""),
	}
	mgr := newTestManager(t, pm)
	if err := mgr.Prune(); err != nil {
		t.Fatalf("Prune on empty list: %v", err)
	}
}

func TestManager_Cfg_ReturnsCfg(t *testing.T) {
	pm := &mockPodman{containers: map[string]string{}, images: map[string]bool{}}
	mgr := newTestManager(t, pm)
	cfg := mgr.Cfg()
	if cfg.Image != "localhost/sandbox-base:latest" {
		t.Errorf("Cfg().Image: got %q", cfg.Image)
	}
}

func TestManager_Podman_ReturnsPodman(t *testing.T) {
	pm := &mockPodman{containers: map[string]string{}, images: map[string]bool{}}
	mgr := newTestManager(t, pm)
	if mgr.Podman() == nil {
		t.Error("Podman() must not return nil")
	}
}

func TestList_MultipleContainers(t *testing.T) {
	pm := &mockPodman{
		runOut: []byte("sbx-a-111\trunning\t/home/user/a\nsbx-b-222\tstopped\t/home/user/b\n"),
	}
	mgr := newTestManager(t, pm)
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(infos))
	}
	if infos[1].State != "stopped" {
		t.Errorf("second container State: got %q want stopped", infos[1].State)
	}
}

func TestList_MalformedLineSkipped(t *testing.T) {
	// A line without proper tab separators should be silently skipped.
	pm := &mockPodman{
		runOut: []byte("malformed-line\nsbx-a-111\trunning\t/home/user/a\n"),
	}
	mgr := newTestManager(t, pm)
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("expected 1 valid container (malformed line skipped), got %d", len(infos))
	}
}

func TestList_RunError_PropagatesError(t *testing.T) {
	pm := &mockPodman{
		runErr: errors.New("podman error"),
	}
	mgr := newTestManager(t, pm)
	_, err := mgr.List()
	if err == nil {
		t.Error("expected error when podman.Run fails")
	}
}

func TestPrune_RunErrorPropagates(t *testing.T) {
	// First call (list) succeeds, second call (rm -f) fails.
	callCount := 0
	pm := &mockPodmanFn{
		runFn: func(args ...string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return []byte("sbx-a-111\tstopped\t/tmp/a\n"), nil
			}
			return nil, errors.New("podman rm failed")
		},
	}
	mgr := newTestManagerFn(t, pm)
	err := mgr.Prune()
	if err == nil {
		t.Error("expected error when podman rm fails during prune")
	}
}

// mockPodmanFn lets tests provide a custom Run function.
type mockPodmanFn struct {
	runFn      func(args ...string) ([]byte, error)
	containers map[string]string
}

func (m *mockPodmanFn) Run(args ...string) ([]byte, error) { return m.runFn(args...) }
func (m *mockPodmanFn) ContainerExists(name string) bool   { _, ok := m.containers[name]; return ok }
func (m *mockPodmanFn) ContainerRunning(name string) bool  { return m.containers[name] == "running" }
func (m *mockPodmanFn) ImageExists(image string) bool      { return false }
func (m *mockPodmanFn) Start(name string) error            { return nil }
func (m *mockPodmanFn) Pause(name string) error            { return nil }
func (m *mockPodmanFn) Unpause(name string) error          { return nil }

func newTestManagerFn(t *testing.T, pm sandbox.PodmanRunner) *sandbox.Manager {
	t.Helper()
	cfg := config.Config{
		Image:      "localhost/sandbox-base:latest",
		Runtime:    "crun",
		Network:    "slirp4netns",
		Memory:     "4g",
		CPUs:       "4",
		StateDir:   t.TempDir(),
		OverlayDir: t.TempDir(),
	}
	det := runtime.NewWithLookPath(cfg, func(string) (string, error) {
		return "", errors.New("not found")
	})
	return sandbox.NewManagerWithRunner(cfg, pm, det)
}

func TestCreate_OverlayDirsCreated(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{"localhost/sandbox-base:latest": true},
	}
	mgr := newTestManager(t, pm)
	hostDir := t.TempDir()
	if err := mgr.Create(hostDir); err != nil {
		t.Fatalf("Create: %v", err)
	}
	name := sandbox.ID(hostDir)
	// Overlay upper and work dirs must exist under cfg.OverlayDir/<name>/
	cfg := mgr.Cfg()
	upperPath := cfg.OverlayDir + "/" + name + "/upper"
	workPath := cfg.OverlayDir + "/" + name + "/work"
	if _, err := os.Stat(upperPath); os.IsNotExist(err) {
		t.Errorf("overlay upper dir not created: %s", upperPath)
	}
	if _, err := os.Stat(workPath); os.IsNotExist(err) {
		t.Errorf("overlay work dir not created: %s", workPath)
	}
}

