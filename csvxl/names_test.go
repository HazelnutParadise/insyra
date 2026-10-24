package csvxl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The exported names in this package follow Go's initialism convention (CSV,
// not Csv), and a name that is replaced by that spelling stays for one release,
// marked Deprecated, with its old meaning.

// csvxlSourceFiles parses every non-test Go file in the package directory and
// returns them keyed by file name.
func csvxlSourceFiles(t *testing.T) map[string]*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	fset := token.NewFileSet()
	files := make(map[string]*ast.File)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		files[name] = file
	}
	if len(files) == 0 {
		t.Fatal("found no source files to check")
	}
	return files
}

// csvxlDocNames collects every exported identifier of the package together
// with the doc comment a reader sees on it. A doc attached to a declaration
// with a single spec is that spec's doc, so a grouped const or type block that
// documents each entry passes and one that documents the block as a whole does
// not.
func csvxlDocNames(t *testing.T) map[string]string {
	t.Helper()
	docs := make(map[string]string)
	for _, file := range csvxlSourceFiles(t) {
		oneLine := func(g *ast.CommentGroup) string {
			if g == nil {
				return ""
			}
			return strings.Join(strings.Fields(g.Text()), " ")
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil || !d.Name.IsExported() {
					continue
				}
				docs[d.Name.Name] = oneLine(d.Doc)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						doc := s.Doc
						if doc == nil && len(d.Specs) == 1 {
							doc = d.Doc
						}
						docs[s.Name.Name] = oneLine(doc)
					case *ast.ValueSpec:
						doc := s.Doc
						if doc == nil && len(d.Specs) == 1 {
							doc = d.Doc
						}
						for _, n := range s.Names {
							if n.IsExported() {
								docs[n.Name] = oneLine(doc)
							}
						}
					}
				}
			}
		}
	}
	return docs
}

func TestExportedCSVXLNamesHaveGoDocComments(t *testing.T) {
	docs := csvxlDocNames(t)
	for _, name := range []string{"CSVToExcel", "AppendCSVToExcel", "CSVDirToExcel", "ReadCSVToString", "ExcelToCSV", "ExcelToCSVOptions", "ExcelDirToCSV"} {
		doc := docs[name]
		if doc == "" || !strings.HasPrefix(doc, name+" ") {
			t.Errorf("%s doc is %q; want a comment starting with %q", name, doc, name+" ")
		}
	}
}

func TestDeprecatedCSVXLNamesSayWhatReplacedThem(t *testing.T) {
	const removal = "Removed in 0.4."
	// Later changes add their deprecated names here.
	deprecated := map[string]string{
		"CsvToExcel":        "CSVToExcel",
		"AppendCsvToExcel":  "AppendCSVToExcel",
		"ExcelToCsv":        "ExcelToCSV",
		"EachCsvToOneExcel": "CSVDirToExcel",
		"EachExcelToCsv":    "ExcelDirToCSV",
		"ReadCsvToString":   "ReadCSVToString",
	}
	docs := csvxlDocNames(t)
	for old, current := range deprecated {
		doc, ok := docs[old]
		if !ok {
			t.Errorf("%s is not an exported name of the package", old)
			continue
		}
		notice := "Deprecated: use " + current
		if !strings.Contains(doc, notice) || !strings.Contains(doc, removal) {
			t.Errorf("%s doc is %q; want it to contain %q and %q", old, doc, notice, removal)
		}
	}
}
