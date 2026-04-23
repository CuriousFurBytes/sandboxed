package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/CuriousFurBytes/sandboxed/internal/assets"
	"github.com/CuriousFurBytes/sandboxed/internal/config"
	"github.com/CuriousFurBytes/sandboxed/internal/install"
	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
	"github.com/CuriousFurBytes/sandboxed/internal/tui"
)

// version is set by the build system via -ldflags; falls back to the module
// version embedded by `go install` at build time.
var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok &&
			info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

// sandboxSubcmds is the set of subcommands handled by the Manager directly.
// Anything not in this set is treated as a tool shortcut (sbx <tool> [args]).
var sandboxSubcmds = map[string]bool{
	"create": true, "sh": true, "shell": true, "run": true,
	"list": true, "ls": true, "rm": true, "remove": true, "delete": true,
	"prune": true, "sync": true, "id": true, "path": true, "name": true,
	"build": true, "rebuild": true, "doctor": true,
	"help": true, "-h": true, "--help": true,
	"version": true, "--version": true, "-v": true,
}

// toolsInModal opens in a tuios floating pane instead of inline.
var toolsInModal = map[string]bool{
	"claude": true, "copilot": true, "codex": true,
}

func main() {
	cfg := config.Load()
	mustMkdir(cfg.StateDir)
	mustMkdir(cfg.OverlayDir)

	sub := argAt(1)

	switch sub {
	case "", "help", "-h", "--help":
		printUsage()

	case "version", "--version", "-v":
		fmt.Printf("sbx %s\n", version)

	case "doctor":
		checker := install.New()
		result := checker.Check()
		if isTerminal() {
			_ = install.RunDoctorUI(result)
		} else {
			install.PrintDoctorReport(result)
		}
		if !result.OK() {
			os.Exit(1)
		}

	case "create":
		mgr := sandbox.NewManager(cfg)
		if err := mgr.Create(mustPWD()); err != nil {
			die("%v", err)
		}
		okf("sandbox ready (host: %s)", mustPWD())

	case "sh", "shell":
		abs := mustPWD()
		name := sandbox.ID(abs)
		mgr := sandbox.NewManager(cfg)
		if err := mgr.EnsureRunning(name); err != nil {
			die("%v", err)
		}
		podmanArgs := []string{
			"podman", "exec", "-it",
			"-e", "TERM=" + envOr("TERM", "xterm-256color"),
			"-e", "COLORTERM=" + os.Getenv("COLORTERM"),
			name, "/bin/bash", "-l",
		}
		if err := tui.LaunchModal("sbx — shell", podmanArgs); err != nil {
			die("%v", err)
		}

	case "run":
		args := os.Args[2:]
		if len(args) == 0 {
			die("usage: sbx run <cmd...>")
		}
		mgr := sandbox.NewManager(cfg)
		runner := sandbox.NewRunner(mgr)
		if err := runner.Run(mustPWD(), args); err != nil {
			die("%v", err)
		}

	case "list", "ls":
		mgr := sandbox.NewManager(cfg)
		infos, err := mgr.List()
		if err != nil {
			die("%v", err)
		}
		if len(infos) == 0 {
			infof("no sandboxes")
			return
		}
		if isTerminal() {
			if err := tui.RunList(infos); err != nil {
				printInfosPlain(infos)
			}
		} else {
			printInfosPlain(infos)
		}

	case "rm", "remove", "delete":
		mgr := sandbox.NewManager(cfg)
		if err := mgr.Remove(mustPWD()); err != nil {
			die("%v", err)
		}
		okf("removed sandbox for %s", mustPWD())

	case "prune":
		mgr := sandbox.NewManager(cfg)
		if err := mgr.Prune(); err != nil {
			die("%v", err)
		}
		okf("pruned all sandboxes")

	case "sync":
		abs := mustPWD()
		name := sandbox.ID(abs)
		mgr := sandbox.NewManager(cfg)
		if err := sandbox.Sync(mgr.Podman(), cfg.OverlayDir, name, abs); err != nil {
			die("%v", err)
		}
		okf("synced sandbox changes to %s", abs)

	case "id":
		fmt.Println(sandbox.ID(mustPWD()))

	case "path", "name":
		fmt.Println(sandbox.ID(mustPWD()))

	case "build":
		if err := podmanBuildBase(cfg); err != nil {
			die("%v", err)
		}
		okf("built %s", cfg.Image)

	case "rebuild":
		if err := podmanBuildBase(cfg); err != nil {
			die("%v", err)
		}
		if err := podmanRebuildInteractive(cfg); err != nil {
			die("%v", err)
		}

	default:
		// Tool shortcut: sbx <tool> [args...] → sandbox run <tool> [args...]
		tool := sub
		toolArgs := os.Args[2:]
		if tool == "commit" {
			// sbx commit → gitscribe commit
			toolArgs = append([]string{"gitscribe", "commit"}, toolArgs...)
			runToolInSandbox(cfg, toolArgs, false)
			return
		}
		runToolInSandbox(cfg, append([]string{tool}, toolArgs...), toolsInModal[tool])
	}
}

// runToolInSandbox runs a tool inside the current directory's sandbox,
// optionally opening it in a tuios modal.
func runToolInSandbox(cfg config.Config, args []string, modal bool) {
	abs := mustPWD()
	name := sandbox.ID(abs)

	podmanArgs := append([]string{
		"podman", "exec", "-it",
		"-e", "TERM=" + envOr("TERM", "xterm-256color"),
		"-e", "COLORTERM=" + os.Getenv("COLORTERM"),
		name,
	}, args...)

	if modal {
		title := fmt.Sprintf("sbx — %s", args[0])
		if err := tui.LaunchModal(title, podmanArgs); err != nil {
			die("%v", err)
		}
		return
	}

	mgr := sandbox.NewManager(cfg)
	runner := sandbox.NewRunner(mgr)
	if err := runner.Run(abs, args); err != nil {
		die("%v", err)
	}
}

func podmanBuildBase(cfg config.Config) error {
	containerFile := filepath.Join(cfg.DataDir, "Containerfile")
	if _, err := os.Stat(containerFile); os.IsNotExist(err) {
		infof("no Containerfile found at %s — extracting default", containerFile)
		if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
			return fmt.Errorf("create data dir: %w", err)
		}
		if err := os.WriteFile(containerFile, assets.DefaultContainerfile, 0o644); err != nil {
			return fmt.Errorf("write Containerfile: %w", err)
		}
		okf("wrote default Containerfile to %s", containerFile)
	}
	infof("building %s from %s", cfg.Image, containerFile)
	cmd := exec.Command("podman", "build", "-f", containerFile, "-t", cfg.Image, cfg.DataDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func podmanRebuildInteractive(cfg config.Config) error {
	fmt.Fprint(os.Stderr, `
sbx rebuild: opening a scratch shell to authenticate tools.
Log in to each tool you use (claude, codex, gh, ...), then exit.
The authenticated state will be committed back into the base image.

  gh auth login
  claude      ← prompts on first run
  codex       ← prompts on first run

Press Ctrl-D or type 'exit' when done.
`)
	tmp := fmt.Sprintf("sandbox-base-rebuild-%d", os.Getpid())

	cmd := exec.Command("podman", "run", "--name", tmp,
		"--network", cfg.Network,
		"-it",
		"-e", "TERM="+envOr("TERM", "xterm-256color"),
		cfg.Image, "/bin/bash", "-l",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // non-fatal: user may Ctrl-D

	infof("committing %s → %s", tmp, cfg.Image)
	commit := exec.Command("podman", "commit",
		"--change", "CMD [\"/bin/bash\"]",
		"--change", "WORKDIR /workspace",
		tmp, cfg.Image,
	)
	commit.Stdout = os.Stdout
	commit.Stderr = os.Stderr
	if err := commit.Run(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	_ = exec.Command("podman", "rm", tmp).Run()
	okf("updated %s with authenticated state", cfg.Image)
	return nil
}

func printInfosPlain(infos []sandbox.SandboxInfo) {
	fmt.Printf("%-50s %-10s %s\n", "NAME", "STATE", "HOST PATH")
	for _, i := range infos {
		fmt.Printf("%-50s %-10s %s\n", i.Name, i.State, i.HostPath)
	}
}

func printUsage() {
	fmt.Print(`sbx — podman-backed per-project sandboxes with krun VM isolation

Usage:
  sbx create              Create a sandbox for the current directory
  sbx sh                  Open an interactive shell inside the sandbox
  sbx run <cmd...>        Run a command inside the sandbox
  sbx list                List all sandboxes (interactive TUI)
  sbx rm                  Remove the sandbox for the current directory
  sbx prune               Remove all sandboxes and their state
  sbx sync                Sync overlay changes back to the host filesystem
  sbx id                  Print the sandbox ID for the current directory
  sbx build               Build the base image from Containerfile
  sbx rebuild             Rebuild base image and re-authenticate tools
  sbx doctor              Check system dependencies (podman, krun, etc.)
  sbx version             Print sbx version

Tool shortcuts (run tool inside current sandbox):
  sbx claude [args]       Open Claude Code in a modal terminal
  sbx copilot [args]      Open GitHub Copilot in a modal terminal
  sbx codex [args]        Open Codex in a modal terminal
  sbx lazygit [args]      Run lazygit inside the sandbox
  sbx git [args]          Run git inside the sandbox
  sbx uv [args]           Run uv inside the sandbox
  sbx commit [args]       Run gitscribe commit inside the sandbox
  sbx <tool> [args]       Run any tool inside the sandbox

Environment:
  SANDBOX_IMAGE     Base image (default: localhost/sandbox-base:latest)
  SANDBOX_RUNTIME   OCI runtime (default: krun, falls back to crun)
  SANDBOX_NET       --network value (default: slirp4netns:enable_ipv6=true)
  SANDBOX_MEMORY    --memory limit (default: 4g)
  SANDBOX_CPUS      --cpus limit (default: 4)

Paths:
  State:    $XDG_STATE_HOME/sandbox/   (~/.local/state/sandbox/)
  Overlays: $XDG_DATA_HOME/sandbox/overlays/
  Image:    $XDG_DATA_HOME/sandbox/Containerfile
`)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func mustPWD() string {
	abs, err := filepath.Abs(".")
	if err != nil {
		die("resolve working directory: %v", err)
	}
	return abs
}

func mustMkdir(path string) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		die("create directory %s: %v", path, err)
	}
}

func argAt(i int) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return ""
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[0;31m[sbx]\033[0m "+format+"\n", args...)
	os.Exit(1)
}

func infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[0;34m[sbx]\033[0m "+format+"\n", args...)
}

func okf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[0;32m[sbx]\033[0m "+format+"\n", args...)
}
