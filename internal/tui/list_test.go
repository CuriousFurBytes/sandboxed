package tui_test

import (
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
	"github.com/CuriousFurBytes/sandboxed/internal/tui"
)

func TestSbxItem_FilterValue_ContainsNameAndPath(t *testing.T) {
	item := tui.NewSbxItem(sandbox.SandboxInfo{
		Name:     "sbx-foo-abc123",
		State:    "running",
		HostPath: "/home/user/foo",
	})
	fv := item.FilterValue()
	if fv == "" {
		t.Error("FilterValue must not be empty")
	}
	// FilterValue should contain enough info to filter by name or path.
	for _, want := range []string{"sbx-foo-abc123", "/home/user/foo"} {
		found := false
		for i := 0; i <= len(fv)-len(want); i++ {
			if fv[i:i+len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FilterValue %q missing %q", fv, want)
		}
	}
}

func TestSbxItem_Title_ReturnsName(t *testing.T) {
	item := tui.NewSbxItem(sandbox.SandboxInfo{Name: "sbx-foo-abc123", State: "running"})
	if item.Title() != "sbx-foo-abc123" {
		t.Errorf("Title: got %q want sbx-foo-abc123", item.Title())
	}
}

func TestSbxItem_Description_ReturnsHostPath(t *testing.T) {
	item := tui.NewSbxItem(sandbox.SandboxInfo{Name: "sbx-foo-abc123", HostPath: "/home/user/foo"})
	if item.Description() != "/home/user/foo" {
		t.Errorf("Description: got %q want /home/user/foo", item.Description())
	}
}

func TestSbxDelegate_Height_IsTwo(t *testing.T) {
	d := tui.NewSbxDelegate()
	if d.Height() != 2 {
		t.Errorf("Height: got %d want 2", d.Height())
	}
}

func TestSbxDelegate_Spacing_IsOne(t *testing.T) {
	d := tui.NewSbxDelegate()
	if d.Spacing() != 1 {
		t.Errorf("Spacing: got %d want 1", d.Spacing())
	}
}

func TestNewListModel_WithItems(t *testing.T) {
	infos := []sandbox.SandboxInfo{
		{Name: "sbx-a-111", State: "running", HostPath: "/tmp/a"},
		{Name: "sbx-b-222", State: "stopped", HostPath: "/tmp/b"},
	}
	m := tui.NewListModel(infos)
	if m == nil {
		t.Fatal("NewListModel must not return nil")
	}
}

func TestListModel_View_ReturnsString(t *testing.T) {
	infos := []sandbox.SandboxInfo{
		{Name: "sbx-a-111", State: "running", HostPath: "/tmp/a"},
	}
	m := tui.NewListModel(infos)
	view := m.View()
	// When not done, View must return a non-empty string.
	if view == "" {
		t.Error("View() returned empty string before quit")
	}
}

func TestListModel_View_EmptyWhenDone(t *testing.T) {
	m := tui.NewDoneListModel()
	if m.View() != "" {
		t.Error("View() should return empty string when done=true")
	}
}

func TestSbxDelegate_Render_Running(t *testing.T) {
	item := tui.NewSbxItem(sandbox.SandboxInfo{
		Name:     "sbx-foo-abc123",
		State:    "running",
		HostPath: "/home/user/foo",
	})
	// Render via the list model so the delegate is called in context.
	m := tui.NewListModel([]sandbox.SandboxInfo{item.SandboxInfo})
	view := m.View()
	// The view should include the container name somewhere.
	if view == "" {
		t.Error("View() should include rendered items")
	}
}

func TestSbxDelegate_Render_Stopped(t *testing.T) {
	item := tui.NewSbxItem(sandbox.SandboxInfo{
		Name:     "sbx-bar-def456",
		State:    "stopped",
		HostPath: "/home/user/bar",
	})
	m := tui.NewListModel([]sandbox.SandboxInfo{item.SandboxInfo})
	view := m.View()
	if view == "" {
		t.Error("View() should include rendered stopped item")
	}
}

func TestListModel_WithNoItems(t *testing.T) {
	m := tui.NewListModel([]sandbox.SandboxInfo{})
	view := m.View()
	// Even with no items, view should return something (title bar etc.)
	if view == "" {
		t.Error("View() of empty list should not be empty string")
	}
}
