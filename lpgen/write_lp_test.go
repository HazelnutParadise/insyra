package lpgen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// lp.Solve writes a model through WriteLP, so what it solves and what
// GenerateLPFile saves must be the same text.
func TestWriteLPMatchesGenerateLPFile(t *testing.T) {
	m := NewLPModel().
		SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").
		AddConstraint("x + 3 y <= 6").
		AddBound("x <= 3").
		AddIntegerVar("x").
		AddBinaryVar("y")

	path := filepath.Join(t.TempDir(), "model.lp")
	m.GenerateLPFile(path)
	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := m.WriteLP(&buf); err != nil {
		t.Fatalf("WriteLP: %v", err)
	}
	if buf.String() != string(file) {
		t.Errorf("WriteLP wrote\n%s\nGenerateLPFile wrote\n%s", buf.String(), file)
	}
}

func TestWriteLPRejectsAnUnknownObjectiveType(t *testing.T) {
	var buf bytes.Buffer
	if err := NewLPModel().SetObjective("upwards", "x").WriteLP(&buf); err == nil {
		t.Errorf("an unknown objective type wrote:\n%s", buf.String())
	}
}
