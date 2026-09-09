package insyra

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// A chainable method returns its receiver's type and no error, so the only way
// a caller learns about a failure is Err() on what comes back. Returning nil
// there leaves them with nothing at all — a nil *DataList panics on every
// method it has, Err() included — and it breaks the chain the shape exists for.
//
// The rule is written in AGENTS.md, in Docs/DataList.md and Docs/DataTable.md,
// and in the chainable-never-nil spec. This test is what checks it. Ten
// DataList transforms, the isr overrides, Diff and PctChange were each found by
// a person reading code; a method added tomorrow would not be.
//
// The check is static and looks for a literal `return nil` in the chainable
// result position. Every instance of this defect so far has had that shape, and
// resolving a nil-valued variable would need a full type-checking pass for a
// case that has not occurred.
func TestChainableMethodsNeverReturnNil(t *testing.T) {
	// The root package's test runs with the module root as its directory.
	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("expected to run from the module root: %v", err)
	}

	fset := token.NewFileSet()
	var (
		filesParsed  int
		methodsFound int
		violations   []string
	)

	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "openspec", "Docs", "skills", "node_modules", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			// A file the module builds must parse; anything else is a real
			// problem, not something to walk past.
			t.Errorf("%s: %v", path, perr)
			return nil
		}
		filesParsed++

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || fn.Type.Results == nil {
				continue
			}
			recv := exprTypeName(fn.Recv.List[0].Type)
			pos, hasErr := chainableResult(fn.Type.Results, recv)
			if pos < 0 || hasErr {
				continue
			}
			methodsFound++

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				// A nested function literal has its own signature; its
				// `return nil` says nothing about this method. Descending
				// into one reports a window reducer that yields "no value"
				// as a broken chain.
				if _, isLit := n.(*ast.FuncLit); isLit {
					return false
				}
				ret, ok := n.(*ast.ReturnStmt)
				if !ok || pos >= len(ret.Results) {
					return true
				}
				if id, ok := ret.Results[pos].(*ast.Ident); ok && id.Name == "nil" {
					p := fset.Position(ret.Pos())
					violations = append(violations, fmt.Sprintf("%s:%d  (%s).%s",
						p.Filename, p.Line, recv, fn.Name.Name))
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	t.Logf("parsed %d files, found %d chainable methods", filesParsed, methodsFound)

	// Without these the test would report success on an empty set the moment
	// the walk or the matcher broke, and a floor set far below reality does not
	// catch a walk that silently skips half the module — the root package alone
	// carries 165 of the methods. The floors sit just under the counts at the
	// time of writing (449 files, 182 methods); raise them when the module
	// grows, and treat a drop as something to explain rather than to lower.
	if filesParsed < 400 {
		t.Fatalf("only %d files parsed, expected at least 400; the walk is skipping the module", filesParsed)
	}
	if methodsFound < 170 {
		t.Fatalf("only %d chainable methods found, expected at least 170; the matcher is broken", methodsFound)
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("a chainable method must return a usable receiver, never nil "+
			"(record the failure with fail/setErr and return the receiver or an "+
			"empty value carrying it):\n  %s", strings.Join(violations, "\n  "))
	}
}

// chainableResult returns the index of the first result whose type matches the
// receiver, and whether the signature also carries an error. A method with an
// error result is the ordinary (T, error) shape, where nil comes with a reason.
func chainableResult(results *ast.FieldList, recv string) (pos int, hasErr bool) {
	pos = -1
	i := 0
	for _, field := range results.List {
		n := len(field.Names)
		if n == 0 {
			n = 1
		}
		name := exprTypeName(field.Type)
		if name == "error" {
			hasErr = true
		}
		for k := 0; k < n; k++ {
			if name == recv && pos < 0 {
				pos = i
			}
			i++
		}
	}
	return pos, hasErr
}

// exprTypeName renders a type expression the way the source writes it, which is
// all this comparison needs: two spellings of the same type inside one file are
// the same string.
func exprTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + exprTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprTypeName(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr: // a generic type: Foo[T]
		return exprTypeName(t.X)
	}
	return ""
}
