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
