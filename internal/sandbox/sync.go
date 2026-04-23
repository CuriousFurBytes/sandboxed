package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Sync rsyncs changes from the sandbox overlay upperdir back to hostPath.
func Sync(podman PodmanRunner, overlayDir, containerName, hostPath string) error {
	return SyncWith(podman, overlayDir, containerName, hostPath, rsync)
}

// SyncWith rsyncs using a custom sync function (injectable for tests).
func SyncWith(podman PodmanRunner, overlayDir, containerName, hostPath string, syncFn func(upper, dst string) error) error {
	upper := filepath.Join(overlayDir, containerName, "upper")
	if _, err := os.Stat(upper); os.IsNotExist(err) {
		return fmt.Errorf("overlay upperdir missing at %s", upper)
	}

	wasRunning := podman.ContainerRunning(containerName)
	if wasRunning {
		if err := podman.Pause(containerName); err != nil {
			return fmt.Errorf("pause container: %w", err)
		}
	}

	err := syncFn(upper, hostPath)

	if wasRunning {
		_ = podman.Unpause(containerName)
	}
	return err
}

func rsync(upper, hostPath string) error {
	cmd := exec.Command("rsync", "-aHAX",
		"--info=stats1,progress2",
		"--exclude=.wh..wh.*",
		upper+"/",
		hostPath+"/",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
