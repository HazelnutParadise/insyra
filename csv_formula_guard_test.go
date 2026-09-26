package insyra

import (
	"bytes"
	"testing"
)

// A spreadsheet runs a CSV cell that starts with =, +, - or @ as a formula,
// so text from outside can execute when someone opens the file. The guard is
// on by default (owner's ruling of 2026-09-26, #285) and touches only text
// that could be a formula: a number, or text that is just a number, is left
// alone, because nothing runs and a quote would turn it into text.
func TestCSVFormulaGuardIsOnByDefault(t *testing.T) {
	dt := NewDataTable(NewDataList(-5, 3.5, "=1+1", "-2+3", "@SUM(A1)", "-5", "+886912345678", " =cmd", "hello").SetName("v"))
	var guarded, raw bytes.Buffer
	if err := dt.WriteCSV(&guarded); err != nil {
		t.Fatal(err)
	}
	want := "v\n-5\n3.5\n'=1+1\n'-2+3\n'@SUM(A1)\n-5\n+886912345678\n' =cmd\nhello\n"
	if guarded.String() != want {
		t.Fatalf("guarded CSV:\n%q\nwant\n%q", guarded.String(), want)
	}
	if err := dt.WriteCSV(&raw, CSVWriteOptions{AllowFormulas: true}); err != nil {
		t.Fatal(err)
	}
	// encoding/csv quotes a field that starts with a space.
	if want := "v\n-5\n3.5\n=1+1\n-2+3\n@SUM(A1)\n-5\n+886912345678\n\" =cmd\"\nhello\n"; raw.String() != want {
		t.Fatalf("AllowFormulas CSV:\n%q\nwant\n%q", raw.String(), want)
	}
}
