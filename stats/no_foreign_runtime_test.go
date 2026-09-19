package stats_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every computation in stats is pure Go. R and Python are reference answers
// the tests consult, never something the library shells out to, so a method
// that needs another language has to be ported first. This walks every
// non-test file under stats/ and fails on an import that would let a
// computation cheat.
func TestStatsNeverCallsAnotherLanguage(t *testing.T) {
	forbidden := map[string]string{
		"os/exec":                               "spawning Rscript, python or uv",
		"github.com/HazelnutParadise/insyra/py": "handing the computation to Python",
		"github.com/HazelnutParadise/insyra/pd": "handing the computation to pandas",
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if why, bad := forbidden[strings.Trim(imp.Path.Value, `"`)]; bad {
				t.Errorf("%s imports %s: %s is not allowed in a stats computation", path, imp.Path.Value, why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
