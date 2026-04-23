package sandbox_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

func TestMeta_WriteRead(t *testing.T) {
	dir := t.TempDir()
	want := sandbox.Meta{
		Name:     "sbx-myproject-abc1234567",
		HostPath: "/home/user/myproject",
		Image:    "localhost/sandbox-base:latest",
		Runtime:  "krun",
		Upper:    "/overlays/sbx-myproject-abc1234567/upper",
		Created:  time.Now().Truncate(time.Second),
	}
	if err := sandbox.WriteMeta(dir, want); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	got, err := sandbox.ReadMeta(dir, want.Name)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if got.Name != want.Name {
		t.Errorf("Name: got %q want %q", got.Name, want.Name)
	}
	if got.HostPath != want.HostPath {
		t.Errorf("HostPath: got %q want %q", got.HostPath, want.HostPath)
	}
	if got.Runtime != want.Runtime {
		t.Errorf("Runtime: got %q want %q", got.Runtime, want.Runtime)
	}
}

func TestMeta_ReadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := sandbox.ReadMeta(dir, "nonexistent-sandbox")
	if !errors.Is(err, sandbox.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMeta_ReadCorrupt(t *testing.T) {
	dir := t.TempDir()
	name := "sbx-test-abc123"
	if err := os.WriteFile(dir+"/"+name+".json", []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := sandbox.ReadMeta(dir, name)
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
	if errors.Is(err, sandbox.ErrNotFound) {
		t.Error("corrupt file should not return ErrNotFound")
	}
}

func TestMeta_Delete(t *testing.T) {
	dir := t.TempDir()
	m := sandbox.Meta{Name: "sbx-test-abc123", HostPath: "/tmp/test"}
	if err := sandbox.WriteMeta(dir, m); err != nil {
		t.Fatal(err)
	}
	if err := sandbox.DeleteMeta(dir, m.Name); err != nil {
		t.Fatalf("DeleteMeta: %v", err)
	}
	_, err := sandbox.ReadMeta(dir, m.Name)
	if !errors.Is(err, sandbox.ErrNotFound) {
		t.Errorf("after delete expected ErrNotFound, got %v", err)
	}
}
