package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvNameCannotEscape(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base, "")
	for _, bad := range []string{"../outside", "a/b", "..", ".hidden", "with space", "x\\y"} {
		if err := m.Create(bad); err == nil {
			t.Fatalf("Create(%q) should fail", bad)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(base), "outside")); err == nil {
		t.Fatal("directory escaped the envs root")
	}
	for _, good := range []string{"default", "proj-1.test", "A_b"} {
		if err := m.Create(good); err != nil {
			t.Fatalf("Create(%q): %v", good, err)
		}
	}
}
