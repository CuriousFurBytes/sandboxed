package sandbox_test

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

// captureCmd records calls to the fake command factory.
type captureCmd struct {
	name string
	args []string
}

func newFakeCmdFn(calls *[]captureCmd) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, captureCmd{name: name, args: args})
		// Run a no-op command so cmd.Run() succeeds.
		return exec.Command("true")
	}
}

// runningManager builds a manager where containerName is already running.
func runningManager(t *testing.T, hostPath string) *sandbox.Manager {
	t.Helper()
	name := sandbox.ID(hostPath)
	pm := &mockPodman{
		containers: map[string]string{name: "running"},
		images:     map[string]bool{},
	}
	return newTestManager(t, pm)
}

func TestRunner_Run_PassesArgsToExec(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))

	if err := runner.Run(hostDir, []string{"echo", "hello"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(calls) == 0 {
		t.Fatal("expected exec to be called")
	}
	got := calls[0]
	if got.name != "podman" {
		t.Errorf("expected podman, got %q", got.name)
	}
	// args must include the tool args at the end
	last := got.args[len(got.args)-2:]
	if last[0] != "echo" || last[1] != "hello" {
		t.Errorf("expected [echo hello] at end of args, got %v", last)
	}
}

func TestRunner_Run_EmptyArgsReturnsError(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(nil))

	err := runner.Run(hostDir, []string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestRunner_Run_ContainerNotExistReturnsError(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(nil))

	err := runner.Run(t.TempDir(), []string{"echo"})
	if err == nil {
		t.Error("expected error when container does not exist")
	}
}

func TestRunner_Shell_PassesBashArgs(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))

	if err := runner.Shell(hostDir); err != nil {
		t.Fatalf("Shell: %v", err)
	}
	if len(calls) == 0 {
		t.Fatal("expected exec to be called")
	}
	got := calls[0].args
	// Must end with /bin/bash -l
	n := len(got)
	if got[n-2] != "/bin/bash" || got[n-1] != "-l" {
		t.Errorf("expected shell to end with [/bin/bash -l], got %v", got[n-2:])
	}
}

func TestRunner_Shell_ContainerNotExistReturnsError(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(nil))

	err := runner.Shell(t.TempDir())
	if err == nil {
		t.Error("expected error when container does not exist")
	}
}

func TestRunner_RunNonInteractive_NoTTYFlag(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))

	if err := runner.RunNonInteractive(hostDir, []string{"ls"}); err != nil {
		t.Fatalf("RunNonInteractive: %v", err)
	}
	if len(calls) == 0 {
		t.Fatal("expected exec to be called")
	}
	for _, arg := range calls[0].args {
		if arg == "-it" {
			t.Error("non-interactive run must not pass -it flag")
		}
	}
}

func TestRunner_RunNonInteractive_EmptyArgsReturnsError(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(nil))

	err := runner.RunNonInteractive(hostDir, []string{})
	if err == nil {
		t.Error("expected error for empty args")
	}
}

func TestRunner_Run_StartsStoppedContainer(t *testing.T) {
	hostDir := t.TempDir()
	name := sandbox.ID(hostDir)
	pm := &mockPodman{
		containers: map[string]string{name: "stopped"},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))

	if err := runner.Run(hostDir, []string{"echo"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if pm.containers[name] != "running" {
		t.Error("expected container to be started before running command")
	}
}

func TestRunner_ExecFails_PropagatesError(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	failCmd := func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	runner := sandbox.NewRunnerWithCmd(mgr, failCmd)

	err := runner.Run(hostDir, []string{"echo"})
	if err == nil {
		t.Error("expected error when exec command fails")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("expected ExitError, got %T: %v", err, err)
	}
}

func TestRunner_EnvOr_UsesEnvVarWhenSet(t *testing.T) {
	t.Setenv("TERM", "my-custom-term")
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))
	_ = runner.Run(hostDir, []string{"echo"})

	if len(calls) == 0 {
		t.Fatal("no exec call made")
	}
	found := false
	for _, arg := range calls[0].args {
		if arg == "TERM=my-custom-term" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TERM=my-custom-term in exec args, got: %v", calls[0].args)
	}
}

func TestRunner_EnvOr_UsesFallbackWhenEnvEmpty(t *testing.T) {
	t.Setenv("TERM", "")
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))
	_ = runner.Run(hostDir, []string{"echo"})

	if len(calls) == 0 {
		t.Fatal("no exec call made")
	}
	found := false
	for _, arg := range calls[0].args {
		if arg == "TERM=xterm-256color" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TERM=xterm-256color fallback in exec args, got: %v", calls[0].args)
	}
}

func TestRunner_RunNonInteractive_ContainerNotExistReturnsError(t *testing.T) {
	pm := &mockPodman{
		containers: map[string]string{},
		images:     map[string]bool{},
	}
	mgr := newTestManager(t, pm)
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(nil))

	err := runner.RunNonInteractive(t.TempDir(), []string{"ls"})
	if err == nil {
		t.Error("expected error when container does not exist")
	}
}

func TestRunner_Run_PassesTTYFlag(t *testing.T) {
	hostDir := t.TempDir()
	mgr := runningManager(t, hostDir)

	var calls []captureCmd
	runner := sandbox.NewRunnerWithCmd(mgr, newFakeCmdFn(&calls))
	_ = runner.Run(hostDir, []string{"echo"})

	if len(calls) == 0 {
		t.Fatal("no exec call made")
	}
	found := false
	for _, arg := range calls[0].args {
		if arg == "-it" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected -it flag in Run() args, got: %v", calls[0].args)
	}
}
