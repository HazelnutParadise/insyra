package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvNameCannotEscape(t *testing.T) {
	base := filepath.Join(t.TempDir(), "base")
	m := NewManager(base, "")
	for _, bad := range []string{"../outside", "a/../../outside", "/abs", "..", "a/..", "   "} {
		if _, err := m.ResolveEnvPath(bad); err == nil {
			t.Fatalf("ResolveEnvPath(%q) should fail", bad)
		}
		if err := m.Create(bad); err == nil {
			t.Fatalf("Create(%q) should fail", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "outside")); err == nil {
		t.Fatal("directory escaped the envs root")
	}
}

func TestEnvNameAllowsExistingShapes(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base, "")
	envs, err := m.EnvsPath()
	if err != nil {
		t.Fatal(err)
	}
	for _, good := range []string{"default", "proj-1.test", "my env", "中文", ".hidden"} {
		if err := m.Create(good); err != nil {
			t.Fatalf("Create(%q): %v", good, err)
		}
		p, err := m.ResolveEnvPath(good)
		if err != nil {
			t.Fatalf("ResolveEnvPath(%q): %v", good, err)
		}
		if p != filepath.Join(envs, good) || !strings.HasPrefix(p, envs+string(filepath.Separator)) {
			t.Fatalf("ResolveEnvPath(%q) = %q, want inside %q", good, p, envs)
		}
		if !m.Exists(good) {
			t.Fatalf("Exists(%q) = false after Create", good)
		}
	}
}
