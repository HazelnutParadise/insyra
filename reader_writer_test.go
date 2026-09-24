package insyra

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/encoding/traditionalchinese"
)

// Every format reads from a reader and writes to a writer, and the path
// versions are the same code: a path and a reader over the same bytes must
// give the same table, and a writer must receive the bytes the file would.

func tablesEqual(t *testing.T, label string, got, want *DataTable) {
	t.Helper()
	if !reflect.DeepEqual(got.ColNames(), want.ColNames()) {
		t.Fatalf("%s: column names %v, want %v", label, got.ColNames(), want.ColNames())
	}
	if !reflect.DeepEqual(got.To2DSlice(), want.To2DSlice()) {
		t.Fatalf("%s: cells %v, want %v", label, got.To2DSlice(), want.To2DSlice())
	}
}

const readerCSV = "name,qty,price\napple,3,1.5\nbanana,12,0.25\ncherry,7,4\n"

func TestReadCSVMatchesTheFileReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(path, []byte(readerCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := CSVReadOptions{FirstRowToColNames: true}
	fromFile, err := ReadCSV_FileWithOptions(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadCSV(strings.NewReader(readerCSV), opts)
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "ReadCSV", fromReader, fromFile)
}

func TestReadCSVDetectsEncodingOnAReader(t *testing.T) {
	var text strings.Builder
	text.WriteString("品名,數量\n")
	for i := 0; i < 60; i++ {
		text.WriteString("臺灣高山烏龍茶葉,十二\n")
	}
	big5, err := traditionalchinese.Big5.NewEncoder().String(text.String())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "big5.csv")
	if err := os.WriteFile(path, []byte(big5), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := CSVReadOptions{FirstRowToColNames: true}
	fromFile, err := ReadCSV_FileWithOptions(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadCSV(bytes.NewReader([]byte(big5)), opts)
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "Big5 reader", fromReader, fromFile)
	if got := fromReader.ColNames()[0]; got != "品名" {
		t.Fatalf("header decoded as %q, want 品名", got)
	}
}

func TestWriteCSVMatchesTheFileWriter(t *testing.T) {
	dt, err := ReadCSV_StringWithOptions(readerCSV, CSVReadOptions{FirstRowToColNames: true})
	if err != nil {
		t.Fatal(err)
	}
	opts := CSVWriteOptions{SetColNamesToFirstRow: true}
	path := filepath.Join(t.TempDir(), "out.csv")
	if err := dt.ToCSVWithOptions(path, opts); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := dt.WriteCSV(&buf, opts); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), onDisk) {
		t.Fatalf("WriteCSV wrote %q, the file holds %q", buf.String(), onDisk)
	}
}
