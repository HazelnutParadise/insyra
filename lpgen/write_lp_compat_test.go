package lpgen

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGenerateLPFileMatchesPreMigrationBytes(t *testing.T) {
	m := NewLPModel().SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").AddConstraint("x + 3 y <= 6").
		AddBound("x >= 0").AddIntegerVar("y").AddBinaryVar("z")
	path := filepath.Join(t.TempDir(), "model.lp")
	m.GenerateLPFile(path)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "generate_lp_legacy.lp.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("GenerateLPFile differs from pre-migration bytes\ngot:  %q\nwant: %q", got, want)
	}
	var written bytes.Buffer
	if err := m.WriteLP(&written); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written.Bytes(), want) {
		t.Fatalf("WriteLP differs from GenerateLPFile\ngot:  %q\nwant: %q", written.Bytes(), want)
	}
}

func TestDeprecatedLingoParserStillIgnoresUnreadStatements(t *testing.T) {
	input := "MODEL:\nMIN= X + Y;\nthis statement is not supported;\nX + Y <= 10;\nEND"
	want := &LPModel{Objective: "X + Y", ObjectiveType: "Minimize", Constraints: []string{"X + Y <= 10"}, Bounds: []string{}, BinaryVars: []string{}, IntegerVars: []string{}}
	got := ParseLingoModel_str(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy parser result %#v, want %#v", got, want)
	}
	if _, err := ParseLingo(input); err == nil {
		t.Fatal("ParseLingo accepted the unread statement")
	}
}
