package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// read supplies its own alias before handing the arguments to load, so a
// user-supplied `as` used to come back as `unknown option "as"`.
func TestReadRefusesAsWithGuidance(t *testing.T) {
	ctx := newTestExecContext(t)
	err := Dispatch(ctx, "read", []string{"sales.csv", "as", "x"})
	if err == nil {
		t.Fatal("read accepted an alias")
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown option") {
		t.Errorf("the message still points at the wrong thing: %q", msg)
	}
	if !strings.Contains(msg, "load sales.csv as <var>") {
		t.Errorf("the message %q does not say to use load", msg)
	}
}

// Only an `as` in the alias position is refused. A sheet named "as" is an
// ordinary argument, and `read book.xlsx sheet as` previewed it on v0.3.2.
func TestReadAcceptsASheetNamedAs(t *testing.T) {
	book := filepath.Join(t.TempDir(), "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "as"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("as", "A1", &[]any{"x", "y"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("as", "A2", &[]any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(book); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	ctx := newTestExecContext(t)
	if err := Dispatch(ctx, "read", []string{book, "sheet", "as"}); err != nil {
		t.Fatalf("read refused a sheet named as: %v", err)
	}
}
