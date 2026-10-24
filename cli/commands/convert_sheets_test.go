package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// #270 (C-11): convert can name each sheet together with the CSV file it is
// written to, never by position, and sheets lists what a workbook holds.

func writeSheetsBook(t *testing.T, path string, sheets ...string) {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for i, s := range sheets {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", s); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(s); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellStr(s, "A1", s); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}

func filesIn(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".csv" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		got[e.Name()] = string(b)
	}
	return got
}

func TestConvertPairsEachSheetWithItsFile(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.xlsx")
	writeSheetsBook(t, book, "Q1", "Q2", "Q3")
	ctx := newTestExecContext(t)
	args := []string{book, "sheet", "Q1", filepath.Join(dir, "q1.csv"), "sheet", "Q3", filepath.Join(dir, "q3.csv")}
	if err := runConvertCommand(ctx, args); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"q1.csv": "Q1\n", "q3.csv": "Q3\n"}
	if got := filesIn(t, dir); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
}

func TestConvertPairsAcrossDirectories(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.xlsx")
	writeSheetsBook(t, book, "Q1", "Q2")
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	ctx := newTestExecContext(t)
	if err := runConvertCommand(ctx, []string{book, "sheet", "Q1", filepath.Join(a, "x.csv"), "sheet", "Q2", filepath.Join(b, "x.csv"), "allowformulas", "true"}); err != nil {
		t.Fatal(err)
	}
	if got := filesIn(t, a); !reflect.DeepEqual(got, map[string]string{"x.csv": "Q1\n"}) {
		t.Errorf("a = %v", got)
	}
	if got := filesIn(t, b); !reflect.DeepEqual(got, map[string]string{"x.csv": "Q2\n"}) {
		t.Errorf("b = %v", got)
	}
}

// On the 0.3.x line the plain xlsx->csv form keeps its released behaviour: a
// workbook of several sheets writes the first to the named file and the others
// as <sheet>.csv beside it, with no formula guard. 0.4 refuses it.
func TestConvertPlainFormIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.xlsx")
	writeSheetsBook(t, book, "Q1", "Q2", "Q3")
	ctx := newTestExecContext(t)
	if err := runConvertCommand(ctx, []string{book, filepath.Join(dir, "out.csv")}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"out.csv": "Q1\n", "Q2.csv": "Q2\n", "Q3.csv": "Q3\n"}
	if got := filesIn(t, dir); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}

	csvDir := t.TempDir()
	in := filepath.Join(csvDir, "in.csv")
	if err := os.WriteFile(in, []byte("=1+1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	xlsx := filepath.Join(csvDir, "in.xlsx")
	if err := runConvertCommand(ctx, []string{in, xlsx}); err != nil {
		t.Fatal(err)
	}
	back := filepath.Join(t.TempDir(), "back.csv")
	if err := runConvertCommand(ctx, []string{xlsx, back}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(back); string(b) != "=1+1\n" {
		t.Errorf("plain form output = %q, want it unguarded", b)
	}
}

func TestConvertPairsGuardFormulasByDefault(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	if err := os.WriteFile(in, []byte("=1+1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	xlsx := filepath.Join(dir, "in.xlsx")
	ctx := newTestExecContext(t)
	if err := runConvertCommand(ctx, []string{in, xlsx}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.csv")
	if err := runConvertCommand(ctx, []string{xlsx, "sheet", "in", out}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(out); string(b) != "'=1+1\n" {
		t.Errorf("guarded output = %q", b)
	}
	if err := runConvertCommand(ctx, []string{xlsx, "sheet", "in", out, "allowformulas", "true"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(out); string(b) != "=1+1\n" {
		t.Errorf("allowformulas output = %q", b)
	}
}

func TestConvertRefusesBadPairsBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.xlsx")
	writeSheetsBook(t, book, "Q1", "Q2", "Q3")
	out := func(name string) string { return filepath.Join(dir, name) }
	cases := map[string]struct {
		args []string
		want string
	}{
		"an unknown sheet":        {[]string{book, "sheet", "Q1", out("q1.csv"), "sheet", "Q9", out("q9.csv")}, "Q9"},
		"one file for two sheets": {[]string{book, "sheet", "Q1", out("same.csv"), "sheet", "Q2", out("SAME.csv")}, "same.csv"},
		"one sheet given twice":   {[]string{book, "sheet", "Q1", out("a.csv"), "sheet", "Q1", out("b.csv")}, "Q1"},
		"a pair missing its file": {[]string{book, "sheet", "Q1"}, "sheet"},
		"a file that is not CSV":  {[]string{book, "sheet", "Q1", out("q1.xlsx")}, "q1.xlsx"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := runConvertCommand(newTestExecContext(t), c.args)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want an error naming %q", err, c.want)
			}
			if got := filesIn(t, dir); len(got) != 0 {
				t.Errorf("nothing should be written, got %v", got)
			}
		})
	}
}

func TestSheetsListsAWorkbooksSheets(t *testing.T) {
	book := filepath.Join(t.TempDir(), "book.xlsx")
	writeSheetsBook(t, book, "Q3", "2025 Q1", "Q2")
	ctx := newTestExecContext(t)
	if err := runSheetsCommand(ctx, []string{book}); err != nil {
		t.Fatal(err)
	}
	if got := ctx.Output.(*bytes.Buffer).String(); got != "Q3\n2025 Q1\nQ2\n" {
		t.Errorf("sheets printed %q", got)
	}
	if err := runSheetsCommand(ctx, []string{filepath.Join(t.TempDir(), "missing.xlsx")}); err == nil {
		t.Error("a missing workbook was listed")
	}
}
