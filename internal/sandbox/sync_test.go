package sandbox_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

func TestSyncWith_CallsSyncFn(t *testing.T) {
	overlayDir := t.TempDir()
	hostDir := t.TempDir()
	container := "sbx-test-abc1234567"

	upper := filepath.Join(overlayDir, container, "upper")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}

	pm := &mockPodman{containers: map[string]string{container: "stopped"}}

	var gotUpper, gotDst string
	fakeSyncFn := func(upper, dst string) error {
		gotUpper = upper
		gotDst = dst
		return nil
	}

	if err := sandbox.SyncWith(pm, overlayDir, container, hostDir, fakeSyncFn); err != nil {
		t.Fatalf("SyncWith: %v", err)
	}

	if gotUpper != upper {
		t.Errorf("syncFn upper: got %q want %q", gotUpper, upper)
	}
	if gotDst != hostDir {
		t.Errorf("syncFn dst: got %q want %q", gotDst, hostDir)
	}
}

func TestSyncWith_PausesRunningContainer(t *testing.T) {
	overlayDir := t.TempDir()
	hostDir := t.TempDir()
	container := "sbx-test-abc1234567"

	upper := filepath.Join(overlayDir, container, "upper")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}

	var pauseCalls, unpauseCalls int
	pm := &mockPodmanTracked{
		containers: map[string]string{container: "running"},
		pauseFn:    func(string) error { pauseCalls++; return nil },
		unpauseFn:  func(string) error { unpauseCalls++; return nil },
	}

	if err := sandbox.SyncWith(pm, overlayDir, container, hostDir, func(_, _ string) error { return nil }); err != nil {
		t.Fatalf("SyncWith: %v", err)
	}

	if pauseCalls != 1 {
		t.Errorf("expected 1 pause call, got %d", pauseCalls)
	}
	if unpauseCalls != 1 {
		t.Errorf("expected 1 unpause call, got %d", unpauseCalls)
	}
}

func TestSyncWith_DoesNotPauseStoppedContainer(t *testing.T) {
	overlayDir := t.TempDir()
	hostDir := t.TempDir()
	container := "sbx-test-abc1234567"

	upper := filepath.Join(overlayDir, container, "upper")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}

	var pauseCalls int
	pm := &mockPodmanTracked{
		containers: map[string]string{container: "stopped"},
		pauseFn:    func(string) error { pauseCalls++; return nil },
		unpauseFn:  func(string) error { return nil },
	}

	if err := sandbox.SyncWith(pm, overlayDir, container, hostDir, func(_, _ string) error { return nil }); err != nil {
		t.Fatalf("SyncWith: %v", err)
	}

	if pauseCalls != 0 {
		t.Errorf("expected 0 pause calls for stopped container, got %d", pauseCalls)
	}
}

func TestSyncWith_MissingUpperdirReturnsError(t *testing.T) {
	overlayDir := t.TempDir()
	pm := &mockPodman{containers: map[string]string{}}

	err := sandbox.SyncWith(pm, overlayDir, "no-such-container", t.TempDir(), func(_, _ string) error { return nil })
	if err == nil {
		t.Error("expected error when upperdir does not exist")
	}
}

func TestSyncWith_UnpausesAfterSyncError(t *testing.T) {
	overlayDir := t.TempDir()
	hostDir := t.TempDir()
	container := "sbx-test-abc1234567"

	upper := filepath.Join(overlayDir, container, "upper")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}

	var unpauseCalls int
	pm := &mockPodmanTracked{
		containers: map[string]string{container: "running"},
		pauseFn:    func(string) error { return nil },
		unpauseFn:  func(string) error { unpauseCalls++; return nil },
	}

	syncErr := fmt.Errorf("rsync failed")
	err := sandbox.SyncWith(pm, overlayDir, container, hostDir, func(_, _ string) error {
		return syncErr
	})

	if err == nil {
		t.Error("expected sync error to be propagated")
	}
	if unpauseCalls != 1 {
		t.Errorf("expected unpause even after sync error, got %d calls", unpauseCalls)
	}
}

func TestSyncWith_PauseError_ReturnsError(t *testing.T) {
	overlayDir := t.TempDir()
	hostDir := t.TempDir()
	container := "sbx-test-abc1234567"

	upper := filepath.Join(overlayDir, container, "upper")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}

	pm := &mockPodmanTracked{
		containers: map[string]string{container: "running"},
		pauseFn:    func(string) error { return fmt.Errorf("pause failed") },
		unpauseFn:  func(string) error { return nil },
	}

	err := sandbox.SyncWith(pm, overlayDir, container, hostDir, func(_, _ string) error { return nil })
	if err == nil {
		t.Error("expected error when pause fails")
	}
}

// mockPodmanTracked lets sync tests inspect pause/unpause calls via callbacks.
type mockPodmanTracked struct {
	containers map[string]string
	pauseFn    func(string) error
	unpauseFn  func(string) error
}

func (m *mockPodmanTracked) Run(args ...string) ([]byte, error)   { return nil, nil }
func (m *mockPodmanTracked) ContainerExists(name string) bool     { _, ok := m.containers[name]; return ok }
func (m *mockPodmanTracked) ContainerRunning(name string) bool    { return m.containers[name] == "running" }
func (m *mockPodmanTracked) ImageExists(image string) bool        { return false }
func (m *mockPodmanTracked) Start(name string) error              { return nil }
func (m *mockPodmanTracked) Pause(name string) error              { return m.pauseFn(name) }
func (m *mockPodmanTracked) Unpause(name string) error            { return m.unpauseFn(name) }
