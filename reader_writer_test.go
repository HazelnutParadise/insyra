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

// countingReader counts how much of its input has been read.
type countingReader struct {
	r    *strings.Reader
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}

func csvWithRows(n int) string {
	var b strings.Builder
	b.WriteString("id,value\n")
	for i := 0; i < n; i++ {
		b.WriteString("r")
		b.WriteString(strings.Repeat("x", 3))
		b.WriteString(",")
		b.WriteString("42\n")
	}
	return b.String()
}

func TestStreamCSVYieldsEveryRowInBatches(t *testing.T) {
	var sizes []int
	for dt, err := range StreamCSV(strings.NewReader(csvWithRows(25)), CSVReadOptions{FirstRowToColNames: true}, 10) {
		if err != nil {
			t.Fatalf("StreamCSV: %v", err)
		}
		rows, _ := dt.Size()
		sizes = append(sizes, rows)
		if !reflect.DeepEqual(dt.ColNames(), []string{"id", "value"}) {
			t.Fatalf("batch %d has columns %v, want the header's", len(sizes), dt.ColNames())
		}
	}
	if !reflect.DeepEqual(sizes, []int{10, 10, 5}) {
		t.Fatalf("batch sizes %v, want [10 10 5]", sizes)
	}
}

func TestStreamCSVStopsReadingWhenTheLoopBreaks(t *testing.T) {
	input := csvWithRows(200000)
	counter := &countingReader{r: strings.NewReader(input)}
	for _, err := range StreamCSV(counter, CSVReadOptions{FirstRowToColNames: true}, 100) {
		if err != nil {
			t.Fatalf("StreamCSV: %v", err)
		}
		break
	}
	if counter.read > len(input)/10 {
		t.Fatalf("read %d of %d bytes after one batch of 100 rows", counter.read, len(input))
	}
}

func TestStreamCSVRefusesANonPositiveBatchSize(t *testing.T) {
	yields := 0
	for dt, err := range StreamCSV(strings.NewReader(readerCSV), CSVReadOptions{}, 0) {
		yields++
		if dt != nil || err == nil {
			t.Fatalf("got table %v and error %v, want only an error", dt, err)
		}
	}
	if yields != 1 {
		t.Fatalf("yielded %d times, want one failure", yields)
	}
}
