package parquet

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// cancelOnWrite cancels ctx on the after-th write, so the writer's own output
// is what ends the write rather than the caller.
type cancelOnWrite struct {
	buf    bytes.Buffer
	cancel context.CancelFunc
	after  int
	calls  int
}

func (c *cancelOnWrite) Write(p []byte) (int, error) {
	c.calls++
	n, err := c.buf.Write(p)
	if c.calls == c.after {
		c.cancel()
	}
	return n, err
}

func TestWriteContextCancelledLeavesTheFileAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cancel.parquet")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := WriteContext(ctx, mixedTable(10), path); !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteContext returned %v, want context.Canceled", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("the file at path is %q, want it left as %q", got, "old")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the directory holds %d file(s), want only the one written before: %v", len(entries), entries)
	}
}

func TestWriteToContextStopsBetweenRowGroups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := &cancelOnWrite{cancel: cancel, after: 3}
	if err := WriteToContext(ctx, mixedTable(50), w, WriteOptions{RowGroupSize: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteToContext returned %v, want context.Canceled", err)
	}
	// The Parquet writer writes the file's magic as it is built, so a context
	// cancelled on the third write is done only after the writer has already
	// put a row group's worth of data out. Without this the rest could pass on
	// a sink nothing ever wrote to.
	if w.calls < 3 {
		t.Fatalf("the sink took %d write(s), want the context cancelled on the third", w.calls)
	}

	var full bytes.Buffer
	if err := WriteTo(mixedTable(50), &full, WriteOptions{RowGroupSize: 1}); err != nil {
		t.Fatal(err)
	}
	if w.buf.Len() >= full.Len() {
		t.Fatalf("the cancelled write produced %d bytes, the whole file is %d", w.buf.Len(), full.Len())
	}

	// Without the footer the bytes are not a file, so reading them has to fail
	// rather than hand back a short table.
	if _, err := ReadFrom(context.Background(), bytes.NewReader(w.buf.Bytes()), int64(w.buf.Len()), ReadOptions{}); err == nil {
		t.Fatal("a write stopped by its context read back as a whole Parquet file")
	}
}

func TestWriteToContextCancelledBeforeConverting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	if err := WriteToContext(ctx, mixedTable(5), &buf); !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteToContext returned %v, want context.Canceled", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("WriteToContext wrote %d bytes to a context already done", buf.Len())
	}
}

func TestWriteIsWriteContextWithBackground(t *testing.T) {
	dt := mixedTable(20)

	var a bytes.Buffer
	if err := WriteTo(dt, &a); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := WriteToContext(context.Background(), dt, &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatalf("WriteToContext wrote %d bytes, WriteTo wrote %d", b.Len(), a.Len())
	}
}

func TestWriteContextRefusesANilContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nil.parquet")
	// A nil context is what an unchecked caller passes; the library returns an
	// error rather than panicking the way the standard library's own methods
	// would.
	var nilCtx context.Context
	var buf bytes.Buffer

	if err := WriteContext(nilCtx, mixedTable(5), path); !errors.Is(err, errNilContext) {
		t.Fatalf("WriteContext returned %v, want errNilContext", err)
	}
	if err := WriteToContext(nilCtx, mixedTable(5), &buf); !errors.Is(err, errNilContext) {
		t.Fatalf("WriteToContext returned %v, want errNilContext", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the directory holds %d file(s) after two refusals, want none: %v", len(entries), entries)
	}
	if buf.Len() != 0 {
		t.Fatalf("WriteToContext wrote %d bytes for a nil context", buf.Len())
	}
}

func TestWriteLeavesAFileNamedLikeItsTempAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mine.parquet")
	// A caller who keeps something next to the file it asks Insyra to write
	// must keep it: the name a temporary file happens to take is not Insyra's
	// to destroy.
	userTemp := path + ".tmp"
	if err := os.WriteFile(userTemp, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Write(mixedTable(10), path); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(userTemp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "mine" {
		t.Fatalf("%s holds %q, want the caller's %q", userTemp, got, "mine")
	}

	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if rows := dt.NumRows(); rows != 10 {
		t.Fatalf("the file written holds %d rows, want 10", rows)
	}
}

// Two writers race for one path. Windows can refuse one of two renames onto
// the same target ("Access is denied."), so this is deliberately not a test
// that every write succeeds: what Insyra promises is that the two never mix
// their bytes, that the file left behind is one writer's table whole, and that
// neither a refused nor a completed write leaves a temporary file behind. Each
// side is therefore asked only to land at least one file.
func TestConcurrentWritesToOnePathDoNotMix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.parquet")
	small := mixedTable(1000)
	large := mixedTable(3000)

	type outcome struct {
		succeeded int
		failures  []error
	}
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	repeat := func(dt *insyra.DataTable) {
		defer wg.Done()
		got := outcome{}
		for range 20 {
			if err := Write(dt, path); err != nil {
				got.failures = append(got.failures, err)
				continue
			}
			got.succeeded++
		}
		results <- got
	}
	wg.Add(2)
	go repeat(small)
	go repeat(large)
	wg.Wait()
	close(results)

	writers := 0
	for got := range results {
		writers++
		if got.succeeded == 0 {
			t.Errorf("all 20 of one writer's writes to a path another write was using returned an error, want at least one to succeed: %v", got.failures)
		}
		if len(got.failures) > 0 {
			t.Logf("one writer's %d of 20 writes returned an error, which Windows allows when two writers rename onto one target: %v", len(got.failures), got.failures)
		}
	}
	if writers != 2 {
		t.Fatalf("%d writer(s) reported an outcome, want 2", writers)
	}

	// Whoever wrote last owns the file, but the file is one writer's whole
	// table, never a splice of the two, and it is readable.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reading the file the two writers left behind panicked: %v", r)
		}
	}()
	dt, err := Read(context.Background(), path, ReadOptions{})
	if err != nil {
		t.Fatalf("reading the file the two writers left behind: %v", err)
	}
	rows := dt.NumRows()
	if rows != 1000 && rows != 3000 {
		t.Fatalf("the file holds %d rows, want 1000 or 3000 — one writer's table whole", rows)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("the directory holds %v, want only the file both wrote to", entries)
	}
}
