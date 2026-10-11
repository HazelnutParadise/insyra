package lpgen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// A model with an unknown objective type is refused before anything is
// written: GenerateLPFile leaves an existing file as it was, and WriteLP
// leaves the writer empty.
func TestAnUnknownObjectiveWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.lp")
	if err := os.WriteFile(path, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	model := NewLPModel()
	model.ObjectiveType = "sideways"
	model.Objective = "x"
	model.GenerateLPFile(path)
	if got, _ := os.ReadFile(path); string(got) != "keep me\n" {
		t.Errorf("GenerateLPFile changed the file to %q", got)
	}
	var buf bytes.Buffer
	if err := model.WriteLP(&buf); err == nil {
		t.Error("WriteLP accepted an unknown objective")
	}
	if buf.Len() != 0 {
		t.Errorf("WriteLP wrote %q before refusing", buf.String())
	}
}
