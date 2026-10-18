package insyra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// The column index conversions are named for both ends, ColIndexToNumber and
// ColNumberToIndex, and the coloured-output getter mirrors its setter (#212).
// The names they replace stay one release, marked Deprecated, with their old
// results.

func TestColIndexToNumberAndBack(t *testing.T) {
	known := []struct {
		index  string
		number int
	}{
		{"A", 0}, {"Z", 25}, {"AA", 26}, {"AZ", 51}, {"BA", 52}, {"ZZ", 701}, {"AAA", 702},
	}
	for _, k := range known {
		if got, ok := ColIndexToNumber(k.index); !ok || got != k.number {
			t.Errorf("ColIndexToNumber(%q) = %d, %v; want %d, true", k.index, got, ok, k.number)
		}
		if got, ok := ColIndexToNumber(strings.ToLower(k.index)); !ok || got != k.number {
			t.Errorf("ColIndexToNumber(%q) = %d, %v; want %d, true", strings.ToLower(k.index), got, ok, k.number)
		}
		if got, ok := ColNumberToIndex(k.number); !ok || got != k.index {
			t.Errorf("ColNumberToIndex(%d) = %q, %v; want %q, true", k.number, got, ok, k.index)
		}
	}
	for n := 0; n <= 20000; n++ {
		index, ok := ColNumberToIndex(n)
		if !ok {
			t.Fatalf("ColNumberToIndex(%d) reported false", n)
		}
		if back, ok := ColIndexToNumber(index); !ok || back != n {
			t.Fatalf("ColIndexToNumber(ColNumberToIndex(%d) = %q) = %d, %v", n, index, back, ok)
		}
	}
}

func TestColIndexToNumberRefusesWhatIsNotAColumn(t *testing.T) {
	for _, in := range []string{"", "A1", "1", "A-B", "Ä"} {
		if got, ok := ColIndexToNumber(in); ok || got != -1 {
			t.Errorf("ColIndexToNumber(%q) = %d, %v; want -1, false", in, got, ok)
		}
	}
	if got, ok := ColNumberToIndex(-1); ok || got != "" {
		t.Errorf("ColNumberToIndex(-1) = %q, %v; want \"\", false", got, ok)
	}
}

func TestGetUseColoredOutputMirrorsItsSetter(t *testing.T) {
	defer SetDefaultConfig()
	Config.SetUseColoredOutput(false)
	if Config.GetUseColoredOutput() {
		t.Fatal("GetUseColoredOutput() = true after SetUseColoredOutput(false)")
	}
	Config.SetUseColoredOutput(true)
	if !Config.GetUseColoredOutput() {
		t.Fatal("GetUseColoredOutput() = false after SetUseColoredOutput(true)")
	}
	Config.SetUseColoredOutput(false)
	SetDefaultConfig()
	if !Config.GetUseColoredOutput() {
		t.Fatal("GetUseColoredOutput() = false after SetDefaultConfig()")
	}
}

// rootDocComments returns the doc comment of each function and method declared
// in the named files, keyed by name, on one line.
func rootDocComments(t *testing.T, files ...string) map[string]string {
	t.Helper()
	docs := make(map[string]string)
	fset := token.NewFileSet()
	for _, name := range files {
		file, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
				docs[fn.Name.Name] = strings.Join(strings.Fields(fn.Doc.Text()), " ")
			}
		}
	}
	return docs
}

func TestThreadSafetySwitchSaysItsUnderscoreIsDeliberate(t *testing.T) {
	doc := rootDocComments(t, "config.go")["Dangerously_TurnOffThreadSafety"]
	if !strings.Contains(doc, "The underscore is deliberate") {
		t.Errorf("Dangerously_TurnOffThreadSafety doc is %q; want it to say the underscore is deliberate", doc)
	}
}
