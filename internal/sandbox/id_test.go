package sandbox_test

import (
	"strings"
	"testing"

	"github.com/CuriousFurBytes/sandboxed/internal/sandbox"
)

func TestID_StartsWithPrefix(t *testing.T) {
	id := sandbox.ID("/home/user/myproject")
	if !strings.HasPrefix(id, "sbx-") {
		t.Errorf("ID %q does not start with sbx-", id)
	}
}

func TestID_Deterministic(t *testing.T) {
	a := sandbox.ID("/home/user/myproject")
	b := sandbox.ID("/home/user/myproject")
	if a != b {
		t.Errorf("ID not deterministic: %q vs %q", a, b)
	}
}

func TestID_DifferentPathsGiveDifferentIDs(t *testing.T) {
	a := sandbox.ID("/home/user/project-a")
	b := sandbox.ID("/home/user/project-b")
	if a == b {
		t.Errorf("different paths produced the same ID: %q", a)
	}
}

func TestID_SanitizesSpecialChars(t *testing.T) {
	id := sandbox.ID("/home/user/my project!")
	// Only sbx- prefix, alphanumeric, dash, underscore are valid
	for _, ch := range id {
		valid := ch == '-' || ch == '_' ||
			('a' <= ch && ch <= 'z') ||
			('A' <= ch && ch <= 'Z') ||
			('0' <= ch && ch <= '9')
		if !valid {
			t.Errorf("ID %q contains invalid character %q", id, string(ch))
		}
	}
}

func TestID_TruncatesLongBasename(t *testing.T) {
	long := "/home/user/" + strings.Repeat("a", 100)
	id := sandbox.ID(long)
	// sbx-(4) + max 40 basename + -(1) + 10 hash = 55
	if len(id) > 60 {
		t.Errorf("ID %q is too long: %d chars", id, len(id))
	}
}

func TestID_ContainsBasename(t *testing.T) {
	id := sandbox.ID("/home/user/myproject")
	if !strings.Contains(id, "myproject") {
		t.Errorf("ID %q does not contain basename %q", id, "myproject")
	}
}

func TestID_MatchesShellAlgorithm(t *testing.T) {
	// Verified against: printf '%s' '/tmp/foo' | sha256sum | cut -c1-10
	// Result: e2e1dcd28f
	id := sandbox.ID("/tmp/foo")
	if !strings.HasSuffix(id, "-e2e1dcd28f") {
		t.Errorf("ID %q has wrong hash suffix, want -e2e1dcd28f", id)
	}
	if !strings.HasPrefix(id, "sbx-foo-") {
		t.Errorf("ID %q has wrong prefix, want sbx-foo-", id)
	}
}
