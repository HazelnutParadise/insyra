package datafetch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// #212 (DF-5): one name for each thing. A replaced name stays for one release,
// marked Deprecated, with its old value and meaning.

// datafetchDocs maps the constants, variables, struct fields (as Type.Field)
// and methods (as Type.Method) of one source file to their doc comments, with
// runs of white space folded to one space.
func datafetchDocs(t *testing.T, file string) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	text := func(g *ast.CommentGroup) string { return strings.Join(strings.Fields(g.Text()), " ") }
	docs := make(map[string]string)
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				continue
			}
			recv := d.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if ident, ok := recv.(*ast.Ident); ok {
				docs[ident.Name+"."+d.Name.Name] = text(d.Doc)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					doc := s.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					for _, n := range s.Names {
						docs[n.Name] = text(doc)
					}
				case *ast.TypeSpec:
					st, ok := s.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						for _, n := range field.Names {
							docs[s.Name.Name+"."+n.Name] = text(field.Doc)
						}
					}
				}
			}
		}
	}
	return docs
}

func TestDeprecatedDatafetchNamesSayWhatReplacedThem(t *testing.T) {
	const removal = "Removed in the release after the one that deprecated it."
	// Later changes add their deprecated names here.
	deprecated := []struct{ file, name, notice string }{
		{"yfinance.go", "YFPeriodYearly", "Deprecated: use YFPeriodAnnual"},
		{"googleMapsCommentCrawler.go", "SortByRelevance", "Deprecated: use GoogleMapsStoreReviewSortByRelevance"},
		{"googleMapsCommentCrawler.go", "SortByNewest", "Deprecated: use GoogleMapsStoreReviewSortByNewest"},
		{"googleMapsCommentCrawler.go", "SortByHighestRating", "Deprecated: use GoogleMapsStoreReviewSortByHighestRating"},
		{"googleMapsCommentCrawler.go", "SortByLowestRating", "Deprecated: use GoogleMapsStoreReviewSortByLowestRating"},
		{"googleMapsCommentCrawler.go", "GoogleMapsStoreReviewsFetchingOptions.MaxWaitingInterval_Milliseconds", "Deprecated: use MaxWaitingInterval"},
		{"geocoding.go", "TWGeocodingClient.ReverseTableByColName", "Deprecated: use ReverseTable(dt, insyra.Name(latColName), insyra.Name(lngColName))"},
	}
	for _, d := range deprecated {
		doc, ok := datafetchDocs(t, d.file)[d.name]
		if !ok {
			t.Errorf("%s not found in %s", d.name, d.file)
			continue
		}
		if !strings.Contains(doc, d.notice) || !strings.Contains(doc, removal) {
			t.Errorf("%s doc is %q; want it to contain %q and %q", d.name, doc, d.notice, removal)
		}
	}
}

func TestDeprecatedDatafetchNamesKeepTheirValues(t *testing.T) {
	if YFPeriodYearly != YFPeriod("yearly") {
		t.Errorf("YFPeriodYearly is %q, want \"yearly\"", YFPeriodYearly)
	}
	pairs := []struct{ old, current, want GoogleMapsStoreReviewSortBy }{
		{SortByRelevance, GoogleMapsStoreReviewSortByRelevance, 1},
		{SortByNewest, GoogleMapsStoreReviewSortByNewest, 2},
		{SortByHighestRating, GoogleMapsStoreReviewSortByHighestRating, 3},
		{SortByLowestRating, GoogleMapsStoreReviewSortByLowestRating, 4},
	}
	for _, p := range pairs {
		if p.old != p.want || p.current != p.want {
			t.Errorf("old name %d and new name %d, want both %d", p.old, p.current, p.want)
		}
	}
}
