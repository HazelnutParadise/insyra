package datafetch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
)

// #252: nothing a caller can name in datafetch may be a type from another
// module, or an upgrade of that module changes insyra's API without insyra
// saying so.

// thirdParty reports whether an import path belongs to a module other than
// the standard library and insyra.
func thirdParty(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(first, ".") && !strings.HasPrefix(importPath, "github.com/HazelnutParadise/insyra")
}

// receiverExported reports whether a method's receiver type is exported.
func receiverExported(expr ast.Expr) bool {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e.IsExported()
		default:
			return false
		}
	}
}

// anyExported reports whether one of names is exported.
func anyExported(names []*ast.Ident) bool {
	for _, n := range names {
		if n.IsExported() {
			return true
		}
	}
	return false
}

// visibleTypeExprs returns the type expressions of one file that a caller
// outside the package can see, keyed by the declaration they belong to:
// exported functions and methods of exported types, the exported and
// embedded fields of exported structs, the other exported types whole, and
// the declared types of exported variables and constants.
func visibleTypeExprs(file *ast.File) map[string][]ast.Expr {
	exprs := make(map[string][]ast.Expr)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() || (d.Recv != nil && !receiverExported(d.Recv.List[0].Type)) {
				continue
			}
			exprs[d.Name.Name] = append(exprs[d.Name.Name], d.Type)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if !s.Name.IsExported() {
						continue
					}
					st, ok := s.Type.(*ast.StructType)
					if !ok {
						exprs[s.Name.Name] = append(exprs[s.Name.Name], s.Type)
						continue
					}
					for _, f := range st.Fields.List {
						if len(f.Names) == 0 || anyExported(f.Names) {
							exprs[s.Name.Name] = append(exprs[s.Name.Name], f.Type)
						}
					}
				case *ast.ValueSpec:
					if s.Type != nil && anyExported(s.Names) {
						exprs[s.Names[0].Name] = append(exprs[s.Names[0].Name], s.Type)
					}
				}
			}
		}
	}
	return exprs
}

func TestExportedAPINamesNoThirdPartyType(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	parsed := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		parsed++

		imports := make(map[string]string)
		for _, imp := range file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			local := path.Base(p)
			if imp.Name != nil {
				local = imp.Name.Name
			}
			imports[local] = p
		}

		for decl, exprs := range visibleTypeExprs(file) {
			for _, expr := range exprs {
				ast.Inspect(expr, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if pkg, ok := sel.X.(*ast.Ident); ok && thirdParty(imports[pkg.Name]) {
						t.Errorf("%s: %s names %s.%s from %s", fset.Position(sel.Pos()), decl, pkg.Name, sel.Sel.Name, imports[pkg.Name])
					}
					return true
				})
			}
		}
	}
	if parsed == 0 {
		t.Fatal("found no source files to check")
	}
}
