package insyra

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingWriter struct{ n int }

func (w *failingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return 0, errors.New("disk full")
}

// SEC-2: a write failure surfaces as an error even when it only happens at
// the final flush.
func TestToCSVReportsWriteFailure(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3))
	if err := dt.WriteCSV(&failingWriter{}, CSVWriteOptions{}); err == nil {
		t.Fatal("writeCSV to a failing writer returned nil")
	}
}

// closeFailer accepts every write and fails to close, the way a file on a full
// or network filesystem reports a write it could not complete.
type closeFailer struct{ bytes.Buffer }

func (*closeFailer) Close() error { return errors.New("close: no space left on device") }

// ToCSV and ToCSVWithOptions return the error from closing the file when the
// write itself succeeded; it used to be discarded, reporting a lost write as
// success.
func TestToCSVReportsCloseFailure(t *testing.T) {
	dt := NewDataTable(NewDataList(1, 2, 3))
	w := &closeFailer{}
	err := dt.writeCSVAndClose(w, CSVWriteOptions{})
	if err == nil || !strings.Contains(err.Error(), "no space left") {
		t.Fatalf("writeCSVAndClose with a failing Close returned %v", err)
	}
	if w.Len() == 0 {
		t.Fatal("nothing was written before the close")
	}
	// A write error still wins over the close error.
	if err := dt.writeCSVAndClose(struct {
		io.Writer
		io.Closer
	}{&failingWriter{}, &closeFailer{}}, CSVWriteOptions{}); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("a write error should be returned before a close error, got %v", err)
	}
}

// SEC-3: a cell that a spreadsheet would execute as a formula can be escaped
// on request. The default is unchanged so a round trip keeps the exact value.
func TestToCSVFormulaSanitization(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList("=cmd|' /C calc'!A0", "+1+1", "@SUM(A1)", "-2+3", "safe").SetName("v"))
	dir := t.TempDir()

	plain := filepath.Join(dir, "plain.csv")
	if err := dt.ToCSV(plain, false, true, false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(plain)
	if !strings.Contains(string(raw), "=cmd") {
		t.Fatalf("ToCSV changed the value by default:\n%s", raw)
	}

	// The zero options write exactly what ToCSV writes.
	zero := filepath.Join(dir, "zero.csv")
	if err := dt.ToCSVWithOptions(zero, CSVWriteOptions{SetColNamesToFirstRow: true}); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, zero); string(got) != string(raw) {
		t.Fatalf("ToCSVWithOptions without SanitizeFormulas differs from ToCSV:\n%s\nvs\n%s", got, raw)
	}

	safe := filepath.Join(dir, "safe.csv")
	if err := dt.ToCSVWithOptions(safe, CSVWriteOptions{
		SetColNamesToFirstRow: true,
		SanitizeFormulas:      true,
	}); err != nil {
		t.Fatal(err)
	}
	out := string(mustRead(t, safe))
	for _, dangerous := range []string{"\n=cmd", "\n+1+1", "\n@SUM", "\n-2+3"} {
		if strings.Contains(out, dangerous) {
			t.Fatalf("cell starting with a formula character was not escaped: %q in\n%s", dangerous, out)
		}
	}
	if !strings.Contains(out, "safe") {
		t.Fatalf("ordinary cell was lost:\n%s", out)
	}
	if strings.Contains(out, "'safe") {
		t.Fatalf("ordinary cell was escaped:\n%s", out)
	}
}
