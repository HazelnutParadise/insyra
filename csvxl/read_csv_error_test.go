package csvxl

import (
	"errors"
	"io/fs"
	"testing"
)

// When auto-detection fails, the error from DetectEncoding is wrapped, so a
// caller can reach the cause with errors.As, as the other csvxl errors allow.
func TestReadCsvToStringWrapsTheDetectionError(t *testing.T) {
	// A directory opens but cannot be read, so DetectEncoding fails with a
	// *fs.PathError.
	_, err := ReadCsvToString(t.TempDir())
	if err == nil {
		t.Fatal("ReadCsvToString on a directory returned no error")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("errors.As(err, *fs.PathError) is false for %q", err)
	}
}
