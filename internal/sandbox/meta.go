package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrNotFound is returned when no metadata file exists for a sandbox.
var ErrNotFound = errors.New("sandbox not found")

// Meta holds the persistent state written for each sandbox container.
type Meta struct {
	Name     string    `json:"name"`
	HostPath string    `json:"host_path"`
	Image    string    `json:"image"`
	Runtime  string    `json:"runtime"`
	Upper    string    `json:"upper"`
	Created  time.Time `json:"created"`
}

// WriteMeta persists m to stateDir/<m.Name>.json.
func WriteMeta(stateDir string, m Meta) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stateDir, m.Name+".json"), data, 0o644)
}

// ReadMeta loads sandbox metadata from stateDir/<name>.json.
// Returns ErrNotFound when no such file exists.
func ReadMeta(stateDir, name string) (Meta, error) {
	path := filepath.Join(stateDir, name+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Meta{}, ErrNotFound
	}
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("corrupt meta file %s: %w", path, err)
	}
	return m, nil
}

// DeleteMeta removes the metadata file for name.
func DeleteMeta(stateDir, name string) error {
	err := os.Remove(filepath.Join(stateDir, name+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
