package commands

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/spf13/cobra"
)

// help prints a command's Usage, so a Usage that leaves out an option the
// command parses hides it, and one that marks a required argument optional
// sends people straight into an error. pca and regression stored their result
// under `as <var>` without saying so, and count called its required value
// optional.
func TestUsageNamesWhatTheCommandParses(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for _, name := range []string{"pca", "regression"} {
		if u := Registry[name].Usage; !strings.HasSuffix(u, "[as <var>]") {
			t.Errorf("%s parses `as <var>` but its Usage is %q", name, u)
		}
	}
	if u := Registry["count"].Usage; u != "count <var> <value>" {
		t.Errorf("count requires a value, but its Usage is %q", u)
	}

	ctx := newTestExecContext(t)
	if err := runCountCommand(ctx, nil); err == nil || strings.Contains(err.Error(), "[value]") {
		t.Errorf("count with no arguments: got %v, want a usage error that does not call the value optional", err)
	}
	if err := runAccelCommand(ctx, nil); err == nil || err.Error() != "usage: "+Registry["accel"].Usage {
		t.Errorf("accel with no arguments: got %v, want its Usage, %q", err, Registry["accel"].Usage)
	}
	// save … sql accepts `rownames true|false` as well as the bare flag; its
	// usage error still showed only the bare flag.
	if err := runSaveSQL(ctx, "t", nil, nil); err == nil || !strings.Contains(err.Error(), "[rownames [true|false]]") {
		t.Errorf("save sql with no arguments: got %v, want the usage to show [rownames [true|false]]", err)
	}
}

// A flag a command parses has to be named where `help` shows the command, in
// its Usage or its Forms, and, unless the command turns Cobra's flag parsing
// off, Cobra has to know it too, or the shell rejects as an unknown flag what a
// script accepts. accel parsed --precision with neither for as long as it had
// it. The flags are read from each command's source, so a new one fails here
// without anyone listing it.
func TestParsedFlagsAreDocumentedAndReachTheShell(t *testing.T) {
	shell := map[string]*cobra.Command{}
	for _, c := range BuildCobraCommands(newTestExecContext(t)) {
		shell[c.Name()] = c
	}

	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	found := map[string][]string{} // command name → the flags its file parses
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		var names, flags []string
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CompositeLit:
				if id, ok := n.Type.(*ast.Ident); ok && id.Name == "CommandHandler" {
					for _, elt := range n.Elts {
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Name" {
								if s, ok := stringLiteral(kv.Value); ok {
									names = append(names, s)
								}
							}
						}
					}
				}
			case *ast.CaseClause:
				for _, e := range n.List {
					if s, ok := stringLiteral(e); ok && flagLiteral.MatchString(s) {
						flags = append(flags, s)
					}
				}
			}
			return true
		})
		if len(flags) == 0 {
			continue
		}
		if len(names) != 1 {
			t.Errorf("%s parses %v but registers %d commands, so this test cannot tell whose flags they are", path, flags, len(names))
			continue
		}
		found[names[0]] = append(found[names[0]], flags...)
	}
	// Without this a scan that stopped finding anything would pass.
	if !slices.Contains(found["accel"], "--mode") {
		t.Fatalf("the scan found %v; it no longer finds accel's flags", found)
	}

	registryMu.RLock()
	defer registryMu.RUnlock()
	for name, flags := range found {
		h := Registry[name]
		named := strings.FieldsFunc(h.Usage+" "+strings.Join(h.Forms, " "), func(r rune) bool {
			return r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		for _, flag := range flags {
			if !slices.Contains(named, flag) {
				t.Errorf("%s parses %s, but neither its Usage nor its Forms name it", name, flag)
			}
			if h.DisableFlagParsing {
				continue
			}
			if c := shell[name]; c == nil || c.Flags().Lookup(strings.TrimPrefix(flag, "--")) == nil {
				t.Errorf("%s parses %s, but Cobra does not know it, so the shell rejects it as an unknown flag", name, flag)
			}
		}
	}
}

var flagLiteral = regexp.MustCompile(`^--[a-z][a-z0-9-]*$`)

func stringLiteral(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
