package sandbox

import (
	"fmt"
	"os"
	"os/exec"
)

// Runner executes commands inside a sandbox container.
type Runner struct {
	mgr   *Manager
	cmdFn func(name string, args ...string) *exec.Cmd
}

// NewRunner creates a Runner using the real exec.Command.
func NewRunner(mgr *Manager) *Runner {
	return &Runner{mgr: mgr, cmdFn: exec.Command}
}

// NewRunnerWithCmd creates a Runner with an injectable command factory (for tests).
func NewRunnerWithCmd(mgr *Manager, cmdFn func(string, ...string) *exec.Cmd) *Runner {
	return &Runner{mgr: mgr, cmdFn: cmdFn}
}

// Shell launches an interactive bash login shell inside the sandbox for hostPath.
func (r *Runner) Shell(hostPath string) error {
	name := ID(hostPath)
	if err := r.mgr.EnsureRunning(name); err != nil {
		return err
	}
	return r.execPodman(name, []string{"/bin/bash", "-l"}, true)
}

// Run executes args inside the sandbox for hostPath with a TTY when stdin is a terminal.
func (r *Runner) Run(hostPath string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sbx run <cmd...>")
	}
	name := ID(hostPath)
	if err := r.mgr.EnsureRunning(name); err != nil {
		return err
	}
	return r.execPodman(name, args, true)
}

// RunNonInteractive executes args without a TTY (for scripts / piped output).
func (r *Runner) RunNonInteractive(hostPath string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sbx run <cmd...>")
	}
	name := ID(hostPath)
	if err := r.mgr.EnsureRunning(name); err != nil {
		return err
	}
	return r.execPodman(name, args, false)
}

func (r *Runner) execPodman(containerName string, args []string, tty bool) error {
	pArgs := []string{"exec"}
	if tty {
		pArgs = append(pArgs, "-it")
	}
	pArgs = append(pArgs,
		"-e", "TERM="+envOr("TERM", "xterm-256color"),
		"-e", "COLORTERM="+os.Getenv("COLORTERM"),
		containerName,
	)
	pArgs = append(pArgs, args...)

	cmd := r.cmdFn("podman", pArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
