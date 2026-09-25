package insyra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// An exported function must be declared with func. Exported as a variable
// (`var ToFloat64Safe = utils.ToFloat64Safe`), it can be reassigned by any
// caller, and the library's own calls go through the same variable: one
// assignment changed every t-test and decision tree in the process. Go's
// documentation also files it under Variables, where nobody looks for a
// function. Four were found by reading the code (K-12 of #211); this test
// finds the next one.
//
// The check is static and needs no type information. It flags an exported
// package-level variable whose value is a function literal, a function
// declared in the same package, or a function declared in another package of
// this module. A variable holding a function from outside the module would
// slip past, and none exists.
func TestExportedFunctionsAreNotVariables(t *testing.T) {
	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("expected to run from the module root: %v", err)
	}
	const modulePath = "github.com/HazelnutParadise/insyra"

	type parsedFile struct {
		dir  string
		file *ast.File
	}
	fset := token.NewFileSet()
	var files []parsedFile
	funcsByDir := map[string]map[string]bool{}

	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The same directories TestChainableMethodsNeverReturnNil skips,
			// for the same reasons: what the go command ignores, another
			// branch's worktree under .claude, and non-code trees.
			if name := d.Name(); p != "." && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			switch d.Name() {
			case "openspec", "Docs", "skills", "node_modules", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			t.Errorf("%s: %v", p, perr)
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(p))
		files = append(files, parsedFile{dir: dir, file: file})
		if funcsByDir[dir] == nil {
			funcsByDir[dir] = map[string]bool{}
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				funcsByDir[dir][fn.Name.Name] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 100 {
		t.Fatalf("parsed only %d files; the walk is not seeing the module", len(files))
	}

	var violations []string
	for _, pf := range files {
		// Module-internal imports, by the name the file refers to them with.
		imports := map[string]string{}
		for _, imp := range pf.file.Imports {
			ip, _ := strconv.Unquote(imp.Path.Value)
			if ip != modulePath && !strings.HasPrefix(ip, modulePath+"/") {
				continue
			}
			dir := strings.TrimPrefix(strings.TrimPrefix(ip, modulePath), "/")
			if dir == "" {
				dir = "."
			}
			name := path.Base(ip)
			if ip == modulePath {
				name = "insyra"
			}
			if imp.Name != nil {
				name = imp.Name.Name
			}
			imports[name] = dir
		}

		isFunction := func(e ast.Expr) bool {
			switch v := e.(type) {
			case *ast.FuncLit:
				return true
			case *ast.Ident:
				return funcsByDir[pf.dir][v.Name]
			case *ast.SelectorExpr:
				pkg, ok := v.X.(*ast.Ident)
				if !ok {
					return false
				}
				dir, ok := imports[pkg.Name]
				return ok && funcsByDir[dir][v.Sel.Name]
			}
			return false
		}

		for _, decl := range pf.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if !name.IsExported() || i >= len(vs.Values) {
						continue
					}
					if isFunction(vs.Values[i]) {
						pkg := pf.dir
						if pkg == "." {
							pkg = "insyra"
						}
						violations = append(violations, pkg+"."+name.Name+" ("+fset.Position(name.Pos()).String()+")")
					}
				}
			}
		}
	}

	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("exported variable holds a function; declare it with func: %s", v)
	}
}
