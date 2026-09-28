package stats_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestResultTypesHaveStringAndShow parses the stats package's non-test files
// and requires every exported struct type whose name ends in "Result" to have
// both a String and a Show method with a pointer receiver.
func TestResultTypesHaveStringAndShow(t *testing.T) {
	fset := token.NewFileSet()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob stats directory: %v", err)
	}

	var resultTypes []string
	methods := map[string]map[string]bool{}

	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, ok := ts.Type.(*ast.StructType); !ok {
						continue
					}
					name := ts.Name.Name
					if ast.IsExported(name) && strings.HasSuffix(name, "Result") {
						resultTypes = append(resultTypes, name)
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || len(d.Recv.List) != 1 {
					continue
				}
				if d.Name.Name != "String" && d.Name.Name != "Show" {
					continue
				}
				star, ok := d.Recv.List[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				id, ok := star.X.(*ast.Ident)
				if !ok {
					continue
				}
				if methods[id.Name] == nil {
					methods[id.Name] = map[string]bool{}
				}
				methods[id.Name][d.Name.Name] = true
			}
		}
	}

	sort.Strings(resultTypes)

	var missing []string
	for _, name := range resultTypes {
		for _, m := range []string{"String", "Show"} {
			if !methods[name][m] {
				missing = append(missing, name+" ("+m+")")
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("result types missing String or Show:\n  %s", strings.Join(missing, "\n  "))
	}
}

// resultTextCase is one family's result: its title, the expected prefix of
// the second line of String(), and lines the text must contain.
type resultTextCase struct {
	name       string
	title      string
	secondLine string
	contains   []string
	text       string
}

func TestResultStringFirstLines(t *testing.T) {
	tRes, err := stats.SingleSampleTTest(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0), 0)
	if err != nil {
		t.Fatalf("SingleSampleTTest: %v", err)
	}
	zRes, err := stats.TwoSampleZTest(insyra.NewDataList(1.0, 2.0, 3.0), insyra.NewDataList(2.0, 3.0, 4.0), 1, 1)
	if err != nil {
		t.Fatalf("TwoSampleZTest: %v", err)
	}
	fRes, err := stats.FTestForVarianceEquality(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0))
	if err != nil {
		t.Fatalf("FTestForVarianceEquality: %v", err)
	}
	chiRes, err := stats.ChiSquareIndependenceTest(
		insyra.NewDataList("a", "a", "b", "b"),
		insyra.NewDataList("x", "y", "x", "y"))
	if err != nil {
		t.Fatalf("ChiSquareIndependenceTest: %v", err)
	}
	mwuRes, err := stats.MannWhitneyU(
		insyra.NewDataList(1.0, 2.0, 3.0),
		insyra.NewDataList(4.0, 5.0, 6.0))
	if err != nil {
		t.Fatalf("MannWhitneyU: %v", err)
	}
	kwRes, err := stats.KruskalWallis([]insyra.IDataList{
		insyra.NewDataList(1.0, 2.0),
		insyra.NewDataList(3.0, 4.0),
	})
	if err != nil {
		t.Fatalf("KruskalWallis: %v", err)
	}
	friRes, err := stats.FriedmanTest([]insyra.IDataList{
		insyra.NewDataList(1.0, 2.0),
		insyra.NewDataList(2.0, 1.0),
		insyra.NewDataList(1.5, 2.5),
	})
	if err != nil {
		t.Fatalf("FriedmanTest: %v", err)
	}
	oneRes, err := stats.OneWayANOVA([]insyra.IDataList{
		insyra.NewDataList(1.0, 2.0, 3.0),
		insyra.NewDataList(4.0, 5.0, 6.0),
	})
	if err != nil {
		t.Fatalf("OneWayANOVA: %v", err)
	}
	twoRes, err := stats.TwoWayANOVA(2, 2, []insyra.IDataList{
		insyra.NewDataList(5, 6, 5),
		insyra.NewDataList(7, 8, 9),
		insyra.NewDataList(4, 3, 4),
		insyra.NewDataList(10, 11, 9),
	})
	if err != nil {
		t.Fatalf("TwoWayANOVA: %v", err)
	}
	rmRes, err := stats.RepeatedMeasuresANOVA([]insyra.IDataList{
		insyra.NewDataList(1.0, 2.0),
		insyra.NewDataList(2.0, 3.0),
		insyra.NewDataList(1.5, 2.5),
	})
	if err != nil {
		t.Fatalf("RepeatedMeasuresANOVA: %v", err)
	}
	pcaRes, err := stats.PCA(insyra.NewDataTable(
		insyra.NewDataList(1.0, 1.2, 0.8, 4.9, 5.2).SetName("a"),
		insyra.NewDataList(1.1, 1.0, 0.9, 5.1, 4.8).SetName("b"),
		insyra.NewDataList(5.0, 4.8, 5.3, 1.1, 0.9).SetName("c"),
	), 2)
	if err != nil {
		t.Fatalf("PCA: %v", err)
	}
	seed := int64(7)
	kmRes, err := stats.KMeans(insyra.NewDataTable(
		insyra.NewDataList(0.0, 0.0, 1.0, 10.0, 10.0, 11.0).SetName("x"),
		insyra.NewDataList(0.0, 1.0, 0.0, 10.0, 11.0, 10.0).SetName("y"),
	), 2, stats.KMeansOptions{NStart: 3, IterMax: 20, Seed: &seed})
	if err != nil {
		t.Fatalf("KMeans: %v", err)
	}
	lrRes, err := stats.LinearRegression(
		insyra.NewDataList(3.0, 5.0, 7.0, 9.0),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0))
	if err != nil {
		t.Fatalf("LinearRegression: %v", err)
	}
	logRes, err := stats.LogisticRegression(
		insyra.NewDataList(0, 0, 0, 0, 1, 1, 1, 1),
		insyra.NewDataList(-4.0, -3.0, -2.0, -1.0, 1.0, 2.0, 3.0, 4.0))
	if err != nil {
		t.Fatalf("LogisticRegression: %v", err)
	}
	faRes, err := stats.FactorAnalysis(factorAnalysisTestTable(), stats.FactorAnalysisOptions{
		Count: stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
	})
	if err != nil {
		t.Fatalf("FactorAnalysis: %v", err)
	}

	cases := []resultTextCase{
		{name: "t-test", title: "t-test", secondLine: "  Statistic: ", text: tRes.String()},
		{name: "z-test", title: "z-test", secondLine: "  Statistic: ", text: zRes.String()},
		{name: "F-test", title: "F-test", secondLine: "  Statistic: ", text: fRes.String()},
		{name: "Chi-square test", title: "Chi-square test", text: chiRes.String(),
			contains: []string{"  Observed:", "  Expected:"}},
		{name: "Mann-Whitney U test", title: "Mann-Whitney U test", secondLine: "  Statistic: ", text: mwuRes.String()},
		{name: "Kruskal-Wallis test", title: "Kruskal-Wallis test", secondLine: "  Statistic: ", text: kwRes.String()},
		{name: "Friedman test", title: "Friedman test", secondLine: "  Statistic: ", text: friRes.String()},
		{name: "One-way ANOVA", title: "One-way ANOVA", secondLine: "  Factor: ", text: oneRes.String()},
		{name: "Two-way ANOVA", title: "Two-way ANOVA", secondLine: "  FactorA: ", text: twoRes.String()},
		{name: "Repeated-measures ANOVA", title: "Repeated-measures ANOVA", secondLine: "  Factor: ", text: rmRes.String()},
		{name: "PCA", title: "PCA", secondLine: "  Components:", text: pcaRes.String()},
		{name: "K-means clustering", title: "K-means clustering", secondLine: "  Cluster: ", text: kmRes.String()},
		{name: "Linear regression", title: "Linear regression", secondLine: "  Slope: ", text: lrRes.String()},
		{name: "Logistic regression", title: "Logistic regression", secondLine: "  Link: ", text: logRes.String()},
		{name: "Factor analysis", title: "Factor analysis", text: faRes.String(),
			contains: []string{"  Loadings:"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := strings.Split(tc.text, "\n")
			if len(lines) < 2 {
				t.Fatalf("String() has %d lines, want at least 2:\n%s", len(lines), tc.text)
			}
			if lines[0] != tc.title {
				t.Errorf("first line = %q, want %q", lines[0], tc.title)
			}
			if tc.secondLine != "" && !strings.HasPrefix(lines[1], tc.secondLine) {
				t.Errorf("second line = %q, want it to start with %q", lines[1], tc.secondLine)
			}
			for _, want := range tc.contains {
				if !strings.Contains(tc.text, "\n"+want) && !strings.HasPrefix(tc.text, want) {
					t.Errorf("String() is missing the line %q:\n%s", want, tc.text)
				}
			}
		})
	}

	t.Run("one-sample t-test omits nil fields", func(t *testing.T) {
		for _, line := range strings.Split(tRes.String(), "\n") {
			for _, prefix := range []string{"  Mean2:", "  MeanDiff:", "  N2:"} {
				if strings.HasPrefix(line, prefix) {
					t.Errorf("one-sample t-test String() has a line %q, want it omitted (nil field)", line)
				}
			}
		}
	})
}

func TestResultShowPrintsString(t *testing.T) {
	tRes, err := stats.SingleSampleTTest(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0), 0)
	if err != nil {
		t.Fatalf("SingleSampleTTest: %v", err)
	}
	chiRes, err := stats.ChiSquareIndependenceTest(
		insyra.NewDataList("a", "a", "b", "b"),
		insyra.NewDataList("x", "y", "x", "y"))
	if err != nil {
		t.Fatalf("ChiSquareIndependenceTest: %v", err)
	}
	faRes, err := stats.FactorAnalysis(factorAnalysisTestTable(), stats.FactorAnalysisOptions{
		Count: stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
	})
	if err != nil {
		t.Fatalf("FactorAnalysis: %v", err)
	}

	for _, tc := range []struct {
		name string
		show func()
		text string
	}{
		{"t-test", tRes.Show, tRes.String()},
		{"chi-square", chiRes.Show, chiRes.String()},
		{"factor analysis", func() { faRes.Show() }, faRes.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := captureStdout(t, tc.show)
			if got != tc.text+"\n" {
				t.Errorf("Show() printed %q, want %q", got, tc.text+"\n")
			}
		})
	}

	if got := fmt.Sprint(tRes); got != tRes.String() {
		t.Errorf("fmt.Sprint(res) = %q, want %q", got, tRes.String())
	}
	if got := (*stats.TTestResult)(nil).String(); got != "<nil>" {
		t.Errorf("(*TTestResult)(nil).String() = %q, want %q", got, "<nil>")
	}
}

func TestResultStringLongResiduals(t *testing.T) {
	x := insyra.NewDataList()
	y := insyra.NewDataList()
	for i := 1; i <= 100; i++ {
		x.Append(float64(i))
		y.Append(2*float64(i) + float64(i%7) - 3)
	}
	res, err := stats.LinearRegression(y, x)
	if err != nil {
		t.Fatalf("LinearRegression: %v", err)
	}

	var line string
	for _, l := range strings.Split(res.String(), "\n") {
		if strings.HasPrefix(l, "  Residuals: ") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("no Residuals line in String():\n%s", res.String())
	}
	if !strings.HasSuffix(line, "(100 values)") {
		t.Errorf("Residuals line = %q, want it to end with %q", line, "(100 values)")
	}
}

// factorAnalysisTestTable is the same 10x6 input factor_analysis_test.go
// uses, whose correlation matrix is not singular.
func factorAnalysisTestTable() *insyra.DataTable {
	rows := [][]float64{
		{1.0, 1.1, 0.9, 5.0, 5.2, 4.8},
		{1.2, 1.0, 1.1, 4.8, 5.0, 4.9},
		{0.8, 0.9, 1.0, 5.3, 5.1, 5.2},
		{4.9, 5.1, 5.0, 1.1, 1.0, 1.2},
		{5.2, 4.8, 5.1, 0.9, 1.2, 1.0},
		{5.0, 5.2, 4.9, 1.0, 0.8, 1.1},
		{2.6, 2.7, 2.5, 3.2, 3.1, 3.3},
		{3.1, 3.0, 3.2, 2.6, 2.5, 2.7},
		{1.5, 1.4, 1.6, 4.6, 4.4, 4.5},
		{4.5, 4.6, 4.4, 1.5, 1.6, 1.4},
	}
	dt := insyra.NewDataTable()
	for c := 0; c < 6; c++ {
		col := insyra.NewDataList().SetName(fmt.Sprintf("v%d", c+1))
		for r := range rows {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	return dt
}

func TestResultNilReceiversDoNotPanic(t *testing.T) {
	t.Helper()
	var nilFAModel *stats.FactorModel
	var nilFARes *stats.FactorAnalysisResult

	// FactorModel.String()
	assertNoPanic(t, "(*FactorModel)(nil).String()", func() string {
		return nilFAModel.String()
	}, "<nil>")

	// FactorAnalysisResult.String()
	assertNoPanic(t, "(*FactorAnalysisResult)(nil).String()", func() string {
		return nilFARes.String()
	}, "<nil>")

	// FactorModel.Show()
	assertNoPanicPrints(t, "(*FactorModel)(nil).Show()", func() {
		nilFAModel.Show()
	}, "<nil>\n")

	// FactorModel.Show(5)
	assertNoPanicPrints(t, "(*FactorModel)(nil).Show(5)", func() {
		nilFAModel.Show(5)
	}, "<nil>\n")

	// FactorAnalysisResult.Show(5)
	assertNoPanicPrints(t, "(*FactorAnalysisResult)(nil).Show(5)", func() {
		nilFARes.Show(5)
	}, "<nil>\n")
}

func assertNoPanic(t *testing.T, name string, f func() string, want string) {
	t.Helper()
	var panicked bool
	var got string
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		got = f()
	}()
	if panicked {
		t.Errorf("%s panicked", name)
	}
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

func assertNoPanicPrints(t *testing.T, name string, f func(), want string) {
	t.Helper()
	var panicked bool
	var got string
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		got = captureStdout(t, f)
	}()
	if panicked {
		t.Errorf("%s panicked", name)
	}
	if got != want {
		t.Errorf("%s printed %q, want %q", name, got, want)
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = pw
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(pr)
		done <- string(b)
	}()
	f()
	if err := pw.Close(); err != nil {
		os.Stdout = old
		t.Fatalf("close pipe: %v", err)
	}
	os.Stdout = old
	return <-done
}
