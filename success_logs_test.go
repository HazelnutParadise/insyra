package insyra_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/csvxl"
	"github.com/HazelnutParadise/insyra/gplot"
	"github.com/HazelnutParadise/insyra/plot"
)

// successRecord is what successLogHandler keeps of one record.
type successRecord struct {
	level   slog.Level
	message string
	pkg     string
	fn      string
}

type successLogHandler struct {
	mu      sync.Mutex
	records []successRecord
}

func (h *successLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *successLogHandler) Handle(_ context.Context, r slog.Record) error {
	rec := successRecord{level: r.Level, message: r.Message}
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "package":
			rec.pkg = a.Value.String()
		case "func":
			rec.fn = a.Value.String()
		}
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, rec)
	h.mu.Unlock()
	return nil
}

func (h *successLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *successLogHandler) WithGroup(string) slog.Handler      { return h }

func (h *successLogHandler) take() []successRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.records
	h.records = nil
	return out
}

// recordAtInfo sets a recording logger at Info level and restores the previous
// logger and level when the test ends.
func recordAtInfo(t *testing.T) *successLogHandler {
	t.Helper()
	previousLogger := insyra.Config.GetLogger()
	previousLevel := insyra.Config.GetLogLevel()
	h := &successLogHandler{}
	insyra.Config.SetLogger(slog.New(h))
	insyra.Config.SetLogLevel(insyra.LogLevelInfo)
	t.Cleanup(func() {
		insyra.Config.SetLogger(previousLogger)
		insyra.Config.SetLogLevel(previousLevel)
	})
	return h
}

func TestSavingAChartLogsNothing(t *testing.T) {
	h := recordAtInfo(t)
	dir := t.TempDir()

	bar, err := plot.CreateBarChart(plot.BarChartConfig{Title: "t", XAxis: []string{"a"}}, insyra.NewDataList(1))
	if err != nil {
		t.Fatal(err)
	}
	h.take()
	if err := plot.SaveHTML(bar, filepath.Join(dir, "c.html")); err != nil {
		t.Fatal(err)
	}
	if got := h.take(); len(got) != 0 {
		t.Errorf("plot.SaveHTML logged %+v", got)
	}

	line, err := gplot.CreateLineChart(gplot.LineChartConfig{Title: "t"}, insyra.NewDataList(1.0, 2.0))
	if err != nil {
		t.Fatal(err)
	}
	h.take()
	if err := gplot.SaveChart(line, filepath.Join(dir, "c.png")); err != nil {
		t.Fatal(err)
	}
	if got := h.take(); len(got) != 0 {
		t.Errorf("gplot.SaveChart logged %+v", got)
	}
}

// onlyEncodingLines fails the test for any record other than csvxl's
// encoding detection.
func onlyEncodingLines(t *testing.T, call string, records []successRecord) {
	t.Helper()
	for _, r := range records {
		if r.fn != "readCsvRecords" || !strings.Contains(r.message, "Auto-detected encoding") {
			t.Errorf("%s logged %+v", call, r)
		}
	}
}

func TestConvertingFilesLogsOnlyTheEncodingGuess(t *testing.T) {
	h := recordAtInfo(t)
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "u.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	xlsx := filepath.Join(dir, "u.xlsx")

	if err := csvxl.CSVToExcel([]string{csvPath}, []string{"first"}, xlsx); err != nil {
		t.Fatal(err)
	}
	onlyEncodingLines(t, "CSVToExcel", h.take())

	if err := csvxl.AppendCSVToExcel([]string{csvPath}, []string{"second"}, xlsx); err != nil {
		t.Fatal(err)
	}
	onlyEncodingLines(t, "AppendCSVToExcel", h.take())

	if err := csvxl.ExcelToCSV(xlsx, filepath.Join(dir, "out"), nil); err != nil {
		t.Fatal(err)
	}
	onlyEncodingLines(t, "ExcelToCSV", h.take())

	excelDir := filepath.Join(dir, "books")
	if err := os.Mkdir(excelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(xlsx, filepath.Join(excelDir, "u.xlsx")); err != nil {
		t.Fatal(err)
	}
	if err := csvxl.ExcelDirToCSV(excelDir, filepath.Join(dir, "out2")); err != nil {
		t.Fatal(err)
	}
	onlyEncodingLines(t, "ExcelDirToCSV", h.take())
}

func TestAGuessedEncodingIsStillLogged(t *testing.T) {
	h := recordAtInfo(t)
	path := filepath.Join(t.TempDir(), "u.csv")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := insyra.ReadCSVFile(path); err != nil {
		t.Fatal(err)
	}

	var guesses int
	for _, r := range h.take() {
		if r.level == slog.LevelInfo && strings.Contains(r.message, "Auto-detected encoding utf-8") {
			guesses++
		}
	}
	if guesses != 1 {
		t.Fatalf("got %d encoding-detection records, want 1", guesses)
	}
}

// TestUnrunnableSuccessPathsDoNotLogInfo covers the functions this test
// cannot run on every machine (SavePNG needs Chrome, the py functions a Python
// environment) by reading their source: none of them may call LogInfo.
func TestUnrunnableSuccessPathsDoNotLogInfo(t *testing.T) {
	cases := map[string][]string{
		filepath.Join("plot", "save_chart.go"): {"SavePNG"},
		filepath.Join("py", "py.go"):           {"PipInstallContext", "PipUninstallContext", "PipList"},
	}
	for file, funcs := range cases {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]bool{}
		for _, f := range funcs {
			want[f] = false
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if _, tracked := want[fn.Name.Name]; !tracked {
				continue
			}
			want[fn.Name.Name] = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "LogInfo" {
					t.Errorf("%s: %s logs at Info (%s)", file, fn.Name.Name, fset.Position(call.Pos()))
				}
				return true
			})
		}
		for name, found := range want {
			if !found {
				t.Errorf("%s: function %s not found; update this test", file, name)
			}
		}
	}
}
