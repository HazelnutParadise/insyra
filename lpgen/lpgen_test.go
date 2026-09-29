package lpgen

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The package had no test file at all: neither the model builder, the LP-file
// writer, nor the LINGO parser had ever been run by a test.

// quiet silences the warnings these tests provoke and restores the global
// config afterwards.
func quiet(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
}

func TestLPModel_Builders(t *testing.T) {
	m := NewLPModel().
		SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").
		AddConstraint("x + 3 y <= 6").
		AddBound("x <= 3").
		AddIntegerVar("x").
		AddBinaryVar("y")

	if m.ObjectiveType != "Maximize" || m.Objective != "3 x + 2 y" {
		t.Errorf("objective: got %q / %q", m.ObjectiveType, m.Objective)
	}
	if want := []string{"x + y <= 4", "x + 3 y <= 6"}; !reflect.DeepEqual(m.Constraints, want) {
		t.Errorf("constraints: got %v, want %v", m.Constraints, want)
	}
	if want := []string{"x <= 3"}; !reflect.DeepEqual(m.Bounds, want) {
		t.Errorf("bounds: got %v, want %v", m.Bounds, want)
	}
	if want := []string{"x"}; !reflect.DeepEqual(m.IntegerVars, want) {
		t.Errorf("integer vars: got %v, want %v", m.IntegerVars, want)
	}
	if want := []string{"y"}; !reflect.DeepEqual(m.BinaryVars, want) {
		t.Errorf("binary vars: got %v, want %v", m.BinaryVars, want)
	}
}

// A variable cannot be integer and binary at once: declaring it one way takes
// it off the other list.
func TestLPModel_BinaryAndIntegerAreExclusive(t *testing.T) {
	m := NewLPModel().AddIntegerVar("x").AddBinaryVar("x")
	if len(m.IntegerVars) != 0 {
		t.Errorf("x stayed in IntegerVars: %v", m.IntegerVars)
	}
	if want := []string{"x"}; !reflect.DeepEqual(m.BinaryVars, want) {
		t.Errorf("binary vars: got %v, want %v", m.BinaryVars, want)
	}

	m = NewLPModel().AddBinaryVar("y").AddIntegerVar("y")
	if len(m.BinaryVars) != 0 {
		t.Errorf("y stayed in BinaryVars: %v", m.BinaryVars)
	}
	if want := []string{"y"}; !reflect.DeepEqual(m.IntegerVars, want) {
		t.Errorf("integer vars: got %v, want %v", m.IntegerVars, want)
	}
}

func TestGenerateLPFile(t *testing.T) {
	quiet(t)

	m := NewLPModel().
		SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").
		AddConstraint("x + 3 y <= 6").
		AddBound("x <= 3").
		AddIntegerVar("x").
		AddBinaryVar("y")

	path := filepath.Join(t.TempDir(), "model.lp")
	if err := m.GenerateLPFile(path); err != nil {
		t.Fatalf("GenerateLPFile: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the generated file: %v", err)
	}
	got := string(b)

	// The sections in the order CPLEX LP expects them.
	wantInOrder := []string{
		"MAX\n",
		"  obj: 3 x + 2 y\n",
		"Subject To\n",
		"  c1: x + y <= 4\n",
		"  c2: x + 3 y <= 6\n",
		"Bounds\n",
		"  x <= 3\n",
		"General\n",
		"  x\n",
		"Binary\n",
		"  y\n",
		"End\n",
	}
	at := 0
	for _, want := range wantInOrder {
		i := strings.Index(got[at:], want)
		if i < 0 {
			t.Fatalf("the generated file is missing %q, or it is out of order:\n%s", want, got)
		}
		at += i + len(want)
	}
}

// Minimize, and a model with no bounds or variable declarations, so the
// optional sections are left out entirely.
func TestGenerateLPFile_MinimalModel(t *testing.T) {
	quiet(t)

	m := NewLPModel().SetObjective("min", "x").AddConstraint("x >= 1")
	path := filepath.Join(t.TempDir(), "model.lp")
	if err := m.GenerateLPFile(path); err != nil {
		t.Fatalf("GenerateLPFile: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the generated file: %v", err)
	}
	got := string(b)

	if !strings.Contains(got, "MIN\n") {
		t.Errorf("the file does not say MIN:\n%s", got)
	}
	for _, absent := range []string{"Bounds", "General", "Binary"} {
		if strings.Contains(got, absent+"\n") {
			t.Errorf("the file has an empty %s section:\n%s", absent, got)
		}
	}
}

// The objective type is matched case-insensitively and with the three spellings
// the code accepts.
func TestGenerateLPFile_ObjectiveTypeSpellings(t *testing.T) {
	quiet(t)

	tests := []struct {
		objType string
		want    string
	}{
		{objType: "Minimize", want: "MIN"},
		{objType: "min", want: "MIN"},
		{objType: "MINIMUM", want: "MIN"},
		{objType: " Maximize ", want: "MAX"},
		{objType: "max", want: "MAX"},
		{objType: "maximum", want: "MAX"},
	}
	for _, tt := range tests {
		t.Run(tt.objType, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "model.lp")
			if err := NewLPModel().SetObjective(tt.objType, "x").GenerateLPFile(path); err != nil {
				t.Fatalf("GenerateLPFile: %v", err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading: %v", err)
			}
			if !strings.Contains(string(b), tt.want+"\n") {
				t.Errorf("objective type %q did not produce %s", tt.objType, tt.want)
			}
		})
	}
}

// An objective type the writer does not recognise is an error, and it leaves no
// file behind at all: neither the target nor a temporary one. It used to leave a
// file holding only the banner, because the writer opened the file before it
// checked the type.
func TestGenerateLPFile_UnknownObjectiveType(t *testing.T) {
	quiet(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "model.lp")
	if err := NewLPModel().SetObjective("sideways", "x").GenerateLPFile(path); err == nil {
		t.Error("an unknown objective type was accepted")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("a failed call left files behind: %v", dirEntryNames(entries))
	}
}

// A failed save leaves an existing file at the path exactly as it was, and no
// temporary file beside it.
func TestGenerateLPFile_FailedSaveKeepsTheExistingFile(t *testing.T) {
	quiet(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "model.lp")
	if err := os.WriteFile(path, []byte("\\ keep me\n"), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if err := NewLPModel().SetObjective("sideways", "x").GenerateLPFile(path); err == nil {
		t.Error("an unknown objective type was accepted")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the existing file: %v", err)
	}
	if want := "\\ keep me\n"; string(b) != want {
		t.Errorf("the existing file was changed: got %q, want %q", b, want)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "model.lp" {
		t.Errorf("the directory holds %v, want only model.lp", dirEntryNames(entries))
	}
}

// A path that cannot be created is an error naming the path, and nothing is
// written; the call returns rather than panicking.
func TestGenerateLPFile_UnwritablePath(t *testing.T) {
	quiet(t)

	path := filepath.Join(t.TempDir(), "no", "such", "dir", "model.lp")
	err := NewLPModel().SetObjective("min", "x").GenerateLPFile(path)
	if err == nil {
		t.Fatal("an unwritable path was accepted")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not name the path %q: %v", path, err)
	}

	if _, err := os.Stat(path); err == nil {
		t.Error("a file appeared at an unwritable path")
	}
}

// dirEntryNames lists a directory's entries by name, for the failure messages
// that have to say what was left behind.
func dirEntryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

const lingoModel = `MODEL:
MAX= 3 * X1 + 2 * X2;
[_1] X1 + X2 <= 4;
[_2] 3 X1 + X2 <= 6;
X1 >= 1;
0 <= X2 <= 10;
@BIN(X3);
@INT(X4);
END`

func TestParseLingoModel_str(t *testing.T) {
	quiet(t)

	m := ParseLingoModel_str(lingoModel)
	if m == nil {
		t.Fatal("ParseLingoModel_str returned nil")
	}

	if m.ObjectiveType != "Maximize" {
		t.Errorf("objective type: got %q, want \"Maximize\"", m.ObjectiveType)
	}
	// The `*` becomes a space and the row label is stripped.
	if want := "3 X1 + 2 X2"; m.Objective != want {
		t.Errorf("objective: got %q, want %q", m.Objective, want)
	}
	// A single bare variable against constants is a bound; anything with more
	// terms or a coefficient is a constraint.
	if want := []string{"X1 + X2 <= 4", "3 X1 + X2 <= 6"}; !reflect.DeepEqual(m.Constraints, want) {
		t.Errorf("constraints: got %v, want %v", m.Constraints, want)
	}
	if want := []string{"X1 >= 1", "0 <= X2 <= 10"}; !reflect.DeepEqual(m.Bounds, want) {
		t.Errorf("bounds: got %v, want %v", m.Bounds, want)
	}
	if want := []string{"X3"}; !reflect.DeepEqual(m.BinaryVars, want) {
		t.Errorf("binary vars: got %v, want %v", m.BinaryVars, want)
	}
	if want := []string{"X4"}; !reflect.DeepEqual(m.IntegerVars, want) {
		t.Errorf("integer vars: got %v, want %v", m.IntegerVars, want)
	}
}

// An expression split over several lines is joined before it is read.
func TestParseLingoModel_str_MultiLineExpression(t *testing.T) {
	quiet(t)

	m := ParseLingoModel_str("MODEL:\nMIN= X1\n + X2\n + X3;\nEND")
	if m == nil {
		t.Fatal("ParseLingoModel_str returned nil")
	}
	if want := "X1 + X2 + X3"; m.Objective != want {
		t.Errorf("objective: got %q, want %q", m.Objective, want)
	}
}

// The parser reads the whole thing and keeps nothing it did not recognise. An
// expression that never ends with a semicolon is never flushed, so it is
// dropped — worth pinning, because it looks like data loss and is by design.
func TestParseLingoModel_str_EdgeCases(t *testing.T) {
	quiet(t)

	t.Run("empty input", func(t *testing.T) {
		m := ParseLingoModel_str("")
		if m == nil {
			t.Fatal("empty input gave nil")
		}
		if m.Objective != "" || len(m.Constraints) != 0 || len(m.Bounds) != 0 {
			t.Errorf("empty input produced %+v", m)
		}
	})

	t.Run("unterminated expression is dropped", func(t *testing.T) {
		m := ParseLingoModel_str("MODEL:\nMIN= X1 + X2\nEND")
		if m == nil {
			t.Fatal("returned nil")
		}
		if m.Objective != "" {
			t.Errorf("an expression with no semicolon was kept: %q", m.Objective)
		}
	})

	t.Run("unrecognised expression is dropped", func(t *testing.T) {
		m := ParseLingoModel_str("MODEL:\nhello world;\nEND")
		if m == nil {
			t.Fatal("returned nil")
		}
		if len(m.Constraints) != 0 || len(m.Bounds) != 0 {
			t.Errorf("an unrecognised expression was kept: %+v", m)
		}
	})

	t.Run("lowercase min and max", func(t *testing.T) {
		m := ParseLingoModel_str("min= X;")
		if m.ObjectiveType != "Minimize" {
			t.Errorf("objective type: got %q, want \"Minimize\"", m.ObjectiveType)
		}
	})
}

func TestParseLingoModel_txt(t *testing.T) {
	quiet(t)

	path := filepath.Join(t.TempDir(), "model.lng")
	if err := os.WriteFile(path, []byte(lingoModel), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	fromFile := ParseLingoModel_txt(path)
	fromString := ParseLingoModel_str(lingoModel)
	if fromFile == nil {
		t.Fatal("ParseLingoModel_txt returned nil for a readable file")
	}
	if !reflect.DeepEqual(fromFile, fromString) {
		t.Errorf("the file and string parsers disagree:\n file:   %+v\n string: %+v", fromFile, fromString)
	}
}

// A file that cannot be opened gives nil, which is the only signal the caller
// gets — there is no error return.
func TestParseLingoModel_txt_MissingFile(t *testing.T) {
	quiet(t)

	if m := ParseLingoModel_txt(filepath.Join(t.TempDir(), "nope.lng")); m != nil {
		t.Errorf("a missing file gave %+v, want nil", m)
	}
}

// lingoIsBound is what decides whether a relation lands in Bounds or in
// Constraints, and the four cases in its doc comment had no test.
func TestLingoIsBound(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{expr: "X1 >= -5", want: true},
		{expr: "0 <= X1 <= 10", want: true},
		{expr: "X1 = 3", want: true},
		{expr: "3 X1 <= 10", want: false},
		{expr: "X1 + X2 <= 10", want: false},
		{expr: "X1 <= X2", want: false},
		{expr: "10 >= 5", want: false}, // no variable at all
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			if got := lingoIsBound(tt.expr); got != tt.want {
				t.Errorf("lingoIsBound(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
