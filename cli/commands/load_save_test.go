package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func TestLoad_CSV_DefaultsHeaderTrueRownamesFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "name,age\nalice,30\nbob,25\n")

	ctx := newTestExecContext(t)
	if err := runLoadCommand(ctx, []string{path, "as", "t"}); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	dt, err := getDataTableVar(ctx, "t")
	if err != nil {
		t.Fatalf("expected table 't': %v", err)
	}
	rows, cols := dt.Size()
	if rows != 2 || cols != 2 {
		t.Fatalf("expected 2x2 (header consumed), got %dx%d", rows, cols)
	}
}

func TestLoad_CSV_HeadersFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "1,2,3\n4,5,6\n7,8,9\n")

	ctx := newTestExecContext(t)
	if err := runLoadCommand(ctx, []string{path, "headers", "false", "as", "t"}); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	dt, _ := getDataTableVar(ctx, "t")
	rows, cols := dt.Size()
	if rows != 3 || cols != 3 {
		t.Fatalf("expected 3x3 (no header consumed), got %dx%d", rows, cols)
	}
}

func TestLoad_CSV_RownamesTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "label,a,b\nrow1,1,2\nrow2,3,4\n")

	ctx := newTestExecContext(t)
	if err := runLoadCommand(ctx, []string{path, "rownames", "true", "as", "t"}); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	dt, _ := getDataTableVar(ctx, "t")
	rows, cols := dt.Size()
	if rows != 2 || cols != 2 {
		t.Fatalf("expected 2x2 (first col consumed as row names), got %dx%d", rows, cols)
	}
	if got, ok := dt.GetRowNameByIndex(0); !ok || got != "row1" {
		t.Fatalf("expected first row name 'row1', got (%q, %v)", got, ok)
	}
}

func TestLoad_CSV_RejectsSheetOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "a,b\n1,2\n")
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{path, "sheet", "Sheet1", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "sheet") {
		t.Fatalf("expected error rejecting 'sheet' for CSV, got %v", err)
	}
}

func TestLoad_CSV_InferFalseKeepsRawStrings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "id,price\n0050,600.855\n00878,\n")

	ctx := newTestExecContext(t)
	if err := runLoadCommand(ctx, []string{path, "infer", "false", "as", "t"}); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	dt, _ := getDataTableVar(ctx, "t")
	if got := dt.GetColByName("id").Data()[0]; got != "0050" {
		t.Fatalf("expected raw string \"0050\", got %v (%T)", got, got)
	}
	if got := dt.GetColByName("price").Data()[1]; got != "" {
		t.Fatalf("expected empty cell to stay \"\", got %v (%T)", got, got)
	}
}

func TestLoad_CSV_RaggedAndTrimSpace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noisy.csv")
	mustWrite(t, path, "id,name\n1, \"Alice\"\ntrailer\n")

	ctx := newTestExecContext(t)
	if err := runLoadCommand(ctx, []string{path, "ragged", "true", "trimspace", "true", "as", "t"}); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	dt, _ := getDataTableVar(ctx, "t")
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Fatalf("expected 2x2 noisy CSV, got %dx%d", rows, cols)
	}
	if got := dt.GetColByName("name").Data()[0]; got != "Alice" {
		t.Fatalf("expected trimmed quoted name Alice, got %v", got)
	}
}

func TestLoad_JSON_RejectsInferOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	mustWrite(t, path, `[{"a":1}]`)
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{path, "infer", "false", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "infer") {
		t.Fatalf("expected error rejecting 'infer' for JSON, got %v", err)
	}
}

func TestLoad_JSON_RejectsRaggedOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	mustWrite(t, path, `[{"a":1}]`)
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{path, "ragged", "true", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "ragged") {
		t.Fatalf("expected error rejecting 'ragged' for JSON, got %v", err)
	}
}

func TestLoad_Excel_RejectsInferOption(t *testing.T) {
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{"in.xlsx", "sheet", "S1", "infer", "false", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "infer") {
		t.Fatalf("expected error rejecting 'infer' for Excel, got %v", err)
	}
}

func TestLoad_RejectsUnknownOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "a,b\n1,2\n")
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{path, "foo", "bar", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown-option error, got %v", err)
	}
}

func TestLoad_RejectsBadBoolValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.csv")
	mustWrite(t, path, "a,b\n1,2\n")
	ctx := newTestExecContext(t)
	err := runLoadCommand(ctx, []string{path, "headers", "maybe", "as", "t"})
	if err == nil || !strings.Contains(err.Error(), "headers") {
		t.Fatalf("expected invalid-bool error, got %v", err)
	}
}

func TestSave_CSV_RoundTripWithRownamesAndNoHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	dl1 := insyra.NewDataList(1, 2, 3).SetName("a")
	dl2 := insyra.NewDataList(4, 5, 6).SetName("b")
	dt := insyra.NewDataTable(dl1, dl2)
	dt.SetRowNames([]string{"r1", "r2", "r3"})

	ctx := newTestExecContext(t)
	ctx.Vars["t"] = dt

	if err := runSaveCommand(ctx, []string{"t", path, "headers", "false", "rownames", "true"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	got := string(body)
	wantPrefix := "r1,1,4\nr2,2,5\nr3,3,6\n"
	if got != wantPrefix {
		t.Fatalf("unexpected CSV body:\n--- want ---\n%s\n--- got ---\n%s", wantPrefix, got)
	}
}

func TestSave_RejectsBadBoolValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1).SetName("a"))
	err := runSaveCommand(ctx, []string{"t", path, "bom", "lol"})
	if err == nil || !strings.Contains(err.Error(), "bom") {
		t.Fatalf("expected invalid-bool error for bom, got %v", err)
	}
}

func TestSave_RejectsUnknownOption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1).SetName("a"))
	err := runSaveCommand(ctx, []string{"t", path, "encoding", "utf-8"})
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown-option error, got %v", err)
	}
}

func TestParseFlexBool(t *testing.T) {
	tests := []struct {
		in   string
		want bool
		ok   bool
	}{
		{"true", true, true},
		{"TRUE", true, true},
		{"yes", true, true},
		{"on", true, true},
		{"1", true, true},
		{"false", false, true},
		{"no", false, true},
		{"off", false, true},
		{"0", false, true},
		{" yes ", true, true},
		{"maybe", false, false},
		{"", false, false},
	}
	for _, tt := range tests {
		got, err := parseFlexBool(tt.in)
		if tt.ok && err != nil {
			t.Errorf("parseFlexBool(%q): unexpected err %v", tt.in, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("parseFlexBool(%q): expected err, got nil", tt.in)
		}
		if tt.ok && got != tt.want {
			t.Errorf("parseFlexBool(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func excelSaveContext(t *testing.T) *ExecContext {
	t.Helper()
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("a"))
	return ctx
}

func TestSave_Excel_WritesSheet1WithHeaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	ctx := excelSaveContext(t)
	if err := runSaveCommand(ctx, []string{"t", path}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	dt, err := insyra.ReadExcelSheet(path, "Sheet1", false, true)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	rows, cols := dt.Size()
	if rows != 2 || cols != 1 || dt.ColNames()[0] != "a" {
		t.Fatalf("read back %dx%d with columns %v", rows, cols, dt.ColNames())
	}
}

// A second save to the same sheet is refused and says how to overwrite it;
// saying so replaces the sheet and keeps the others.
func TestSave_Excel_SecondSaveNeedsIfExistsReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	ctx := excelSaveContext(t)
	for _, sheet := range []string{"2024", "2025"} {
		if err := runSaveCommand(ctx, []string{"t", path, "sheet", sheet}); err != nil {
			t.Fatalf("save to sheet %s failed: %v", sheet, err)
		}
	}
	err := runSaveCommand(ctx, []string{"t", path, "sheet", "2025"})
	if err == nil || !strings.Contains(err.Error(), "if-exists replace") {
		t.Fatalf("expected a refusal naming if-exists replace, got %v", err)
	}
	if err := runSaveCommand(ctx, []string{"t", path, "sheet", "2025", "if-exists", "replace"}); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	if _, err := insyra.ReadExcelSheet(path, "2024", false, true); err != nil {
		t.Fatalf("the other sheet did not survive: %v", err)
	}
}

func TestSave_Excel_RejectsXls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xls")
	err := runSaveCommand(excelSaveContext(t), []string{"t", path})
	if err == nil || !strings.Contains(err.Error(), ".xlsx") {
		t.Fatalf("expected .xls to be refused with a pointer to .xlsx, got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("a refused save left a file behind: %v", statErr)
	}
}

func TestSave_Excel_RejectsBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	err := runSaveCommand(excelSaveContext(t), []string{"t", path, "bom", "true"})
	if err == nil || !strings.Contains(err.Error(), "bom") {
		t.Fatalf("expected bom to be refused for Excel, got %v", err)
	}
}

func TestSave_Excel_RejectsBadIfExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	err := runSaveCommand(excelSaveContext(t), []string{"t", path, "if-exists", "append"})
	if err == nil || !strings.Contains(err.Error(), "fail|replace") {
		t.Fatalf("expected if-exists append to be refused, got %v", err)
	}
}

func TestSave_CSV_RejectsSheetAndIfExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	for _, args := range [][]string{{"sheet", "s"}, {"if-exists", "replace"}} {
		err := runSaveCommand(excelSaveContext(t), append([]string{"t", path}, args...))
		if err == nil || !strings.Contains(err.Error(), args[0]) {
			t.Fatalf("expected %s to be refused for CSV, got %v", args[0], err)
		}
	}
}

// Without a sheet name the caller may have meant a new sheet rather than an
// overwrite, so the refusal offers both.
func TestSave_Excel_UnnamedSheetRefusalOffersBothWays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	ctx := excelSaveContext(t)
	if err := runSaveCommand(ctx, []string{"t", path}); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	err := runSaveCommand(ctx, []string{"t", path})
	if err == nil || !strings.Contains(err.Error(), "sheet <name>") || !strings.Contains(err.Error(), "if-exists replace") {
		t.Fatalf("expected a refusal offering sheet <name> and if-exists replace, got %v", err)
	}
}

// The formula guard is on by default for CSV; allowformulas true writes the
// text exactly, for a file that is read back by a program.
func TestSave_CSV_GuardsFormulasByDefault(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList("=1+1", -5).SetName("v"))

	guarded := filepath.Join(dir, "guarded.csv")
	if err := runSaveCommand(ctx, []string{"t", guarded}); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(guarded); string(body) != "v\n'=1+1\n-5\n" {
		t.Fatalf("default save wrote %q", body)
	}
	raw := filepath.Join(dir, "raw.csv")
	if err := runSaveCommand(ctx, []string{"t", raw, "allowformulas", "true"}); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(raw); string(body) != "v\n=1+1\n-5\n" {
		t.Fatalf("allowformulas true wrote %q", body)
	}
	if err := runSaveCommand(ctx, []string{"t", filepath.Join(dir, "x.json"), "allowformulas", "true"}); err == nil {
		t.Fatal("allowformulas was accepted for JSON")
	}
}
