package parquet

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// Parquet reads from an io.ReaderAt and writes to an io.Writer, and the path
// versions are the same code.

func sampleTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3, 4, 5).SetName("n"),
		insyra.NewDataList("a", "b", "c", "d", "e").SetName("s"),
	)
}

func TestWriteToMatchesWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.parquet")
	if err := Write(sampleTable(), path); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteTo(sampleTable(), &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), onDisk) {
		t.Fatalf("WriteTo wrote %d bytes that differ from the %d Write put on disk", buf.Len(), len(onDisk))
	}
}

// closeRecorder is a writer that remembers whether anyone closed it.
type closeRecorder struct {
	bytes.Buffer
	closed bool
}

func (c *closeRecorder) Close() error { c.closed = true; return nil }

func TestWriteToLeavesTheWriterOpen(t *testing.T) {
	w := &closeRecorder{}
	if err := WriteTo(sampleTable(), w); err != nil {
		t.Fatal(err)
	}
	if w.closed {
		t.Fatal("WriteTo closed a writer it does not own")
	}
}

func TestReadFromMatchesRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.parquet")
	if err := Write(sampleTable(), path); err != nil {
		t.Fatal(err)
	}
	fromFile, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := ReadFrom(context.Background(), bytes.NewReader(content), int64(len(content)), ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromReader.ColNames(), fromFile.ColNames()) || !reflect.DeepEqual(fromReader.To2DSlice(), fromFile.To2DSlice()) {
		t.Fatalf("ReadFrom %v %v, Read %v %v", fromReader.ColNames(), fromReader.To2DSlice(), fromFile.ColNames(), fromFile.To2DSlice())
	}
}

func TestStreamFromYieldsEveryRow(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTo(sampleTable(), &buf); err != nil {
		t.Fatal(err)
	}
	content := buf.Bytes()
	total := 0
	for dt, err := range StreamFrom(context.Background(), bytes.NewReader(content), int64(len(content)), ReadOptions{}, 2) {
		if err != nil {
			t.Fatalf("StreamFrom: %v", err)
		}
		rows, _ := dt.Size()
		total += rows
	}
	if total != 5 {
		t.Fatalf("streamed %d rows, want 5", total)
	}
}
