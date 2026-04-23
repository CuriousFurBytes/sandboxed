package install_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CuriousFurBytes/sandboxed/internal/install"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintDoctorReport_AllOK(t *testing.T) {
	result := install.Result{}
	out := captureStdout(t, func() {
		install.PrintDoctorReport(result)
	})
	if !strings.Contains(out, "All dependencies present") {
		t.Errorf("expected 'All dependencies present' in output, got:\n%s", out)
	}
}

func TestPrintDoctorReport_ShowsMissing(t *testing.T) {
	result := install.Result{
		Missing: []install.Requirement{
			{Name: "podman", Description: "Container runtime (required)", Required: true},
		},
	}
	out := captureStdout(t, func() {
		install.PrintDoctorReport(result)
	})
	if !strings.Contains(out, "podman") {
		t.Errorf("expected 'podman' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "MISSING") {
		t.Errorf("expected 'MISSING' in output, got:\n%s", out)
	}
}

func TestPrintDoctorReport_ShowsWarnings(t *testing.T) {
	result := install.Result{
		Warnings: []install.Requirement{
			{Name: "krun", Description: "VM isolation (optional)", Required: false},
		},
	}
	out := captureStdout(t, func() {
		install.PrintDoctorReport(result)
	})
	if !strings.Contains(out, "krun") {
		t.Errorf("expected 'krun' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "OPTIONAL") {
		t.Errorf("expected 'OPTIONAL' in output, got:\n%s", out)
	}
}

func TestPrintDoctorReport_FixHintWhenMissing(t *testing.T) {
	result := install.Result{
		Missing: []install.Requirement{
			{Name: "podman", Description: "desc", Required: true},
		},
	}
	out := captureStdout(t, func() {
		install.PrintDoctorReport(result)
	})
	if !strings.Contains(out, "Install missing") {
		t.Errorf("expected install hint in output, got:\n%s", out)
	}
}

func TestResultOK_TrueWhenNoMissing(t *testing.T) {
	r := install.Result{
		Warnings: []install.Requirement{{Name: "krun"}},
	}
	if !r.OK() {
		t.Error("OK() should be true when no required deps are missing")
	}
}

func TestResultOK_FalseWhenMissing(t *testing.T) {
	r := install.Result{
		Missing: []install.Requirement{{Name: "podman"}},
	}
	if r.OK() {
		t.Error("OK() should be false when required deps are missing")
	}
}

// TestRequirementFields verifies the Requirement struct fields are accessible.
func TestRequirementFields(t *testing.T) {
	req := install.Requirement{
		Name:        "podman",
		Description: "Container runtime",
		Required:    true,
		CheckFn:     func() bool { return true },
	}
	if !req.CheckFn() {
		t.Error("CheckFn should return true")
	}
	if !req.Required {
		t.Error("Required should be true")
	}
	_ = fmt.Sprintf("%s %s", req.Name, req.Description) // ensure fields readable
}

func TestDoctorModel_Init_ReturnsNil(t *testing.T) {
	m := install.NewDoctorModel(install.Result{})
	if m.Init() != nil {
		t.Error("Init() should return nil Cmd")
	}
}

func TestDoctorModel_View_AllOK(t *testing.T) {
	m := install.NewDoctorModel(install.Result{})
	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty string when not done")
	}
	if !strings.Contains(view, "All dependencies present") {
		t.Errorf("View() should show all-OK message, got:\n%s", view)
	}
}

func TestDoctorModel_View_ShowsMissingAndWarnings(t *testing.T) {
	result := install.Result{
		Missing:  []install.Requirement{{Name: "podman", Description: "req"}},
		Warnings: []install.Requirement{{Name: "krun", Description: "opt"}},
	}
	m := install.NewDoctorModel(result)
	view := m.View()
	if !strings.Contains(view, "podman") {
		t.Errorf("View() missing 'podman', got:\n%s", view)
	}
	if !strings.Contains(view, "krun") {
		t.Errorf("View() missing 'krun', got:\n%s", view)
	}
}

func TestDoctorModel_Update_KeyMsgQuits(t *testing.T) {
	m := install.NewDoctorModel(install.Result{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if updated.View() != "" {
		t.Error("View() should return empty string after key press (done=true)")
	}
}

func TestDoctorModel_Update_NonKeyMsgNoOp(t *testing.T) {
	m := install.NewDoctorModel(install.Result{})
	before := m.View()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	after := updated.View()
	// Non-key messages should not change the model to done.
	if after == "" && before != "" {
		t.Error("non-key msg should not cause View() to return empty string")
	}
}
