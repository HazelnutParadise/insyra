package insyra

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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

const readerJSON = `[{"name":"apple","qty":3},{"name":"banana","qty":12}]`

func TestReadJSONReadsFromAReader(t *testing.T) {
	fromBytes, err := ReadJSON([]byte(readerJSON))
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadJSON(strings.NewReader(readerJSON))
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "ReadJSON", fromReader, fromBytes)
}

func TestWriteJSONMatchesTheFileWriter(t *testing.T) {
	dt, err := ReadJSON([]byte(readerJSON))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	if err := dt.ToJSON(path, true); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := dt.WriteJSON(&buf, true); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), onDisk) {
		t.Fatalf("WriteJSON wrote %q, the file holds %q", buf.String(), onDisk)
	}
}

func TestReadExcelMatchesTheFileReader(t *testing.T) {
	book := excelize.NewFile()
	for r, row := range [][]any{{"name", "qty"}, {"apple", 3}, {"banana", 12}} {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := book.SetCellValue("Sheet1", cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "book.xlsx")
	if err := book.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = book.Close()

	fromFile, err := ReadExcelSheet(path, "Sheet1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadExcel(bytes.NewReader(content), "Sheet1", false, true)
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "ReadExcel", fromReader, fromFile)
}

// A value that can both marshal itself and read as a stream is encoded by its
// MarshalJSON, the way it was before ReadJSON accepted a reader. The reader
// path must not take over from a type that chose how to marshal itself.
type marshalingReader struct {
	viaMarshal []byte
	viaRead    io.Reader
}

func (m *marshalingReader) MarshalJSON() ([]byte, error) { return m.viaMarshal, nil }
func (m *marshalingReader) Read(p []byte) (int, error)   { return m.viaRead.Read(p) }

func TestReadJSONPrefersMarshalOverRead(t *testing.T) {
	m := &marshalingReader{
		viaMarshal: []byte(`[{"name":"via-marshal"}]`),
		viaRead:    strings.NewReader(`[{"name":"via-reader"}]`),
	}
	got, err := ReadJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ReadJSON([]byte(`[{"name":"via-marshal"}]`))
	if err != nil {
		t.Fatal(err)
	}
	tablesEqual(t, "MarshalJSON wins over Read", got, want)
}

// The file reader must report an encoding-detection failure the way it always
// did, the detection wrap around a file read, so callers that match the message
// keep working. Reading a directory is a read failure after a successful open.
func TestReadCSVFileReportsTheDetectionReadFailure(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadCSV_FileWithOptions(dir, CSVReadOptions{})
	if err == nil {
		t.Fatal("reading a directory as a CSV file succeeded")
	}
	want := fmt.Sprintf("failed to auto-detect encoding for %s: failed to read file %s: ", dir, dir)
	if !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("error %q does not start with %q", err.Error(), want)
	}
}
