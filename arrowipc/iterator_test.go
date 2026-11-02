package arrowipc_test

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
)

type iteratorSource struct {
	*bytes.Reader
	calls  atomic.Int64
	active atomic.Int64
}

func (r *iteratorSource) ReadAt(p []byte, off int64) (int, error) {
	r.calls.Add(1)
	r.active.Add(1)
	defer r.active.Add(-1)
	return r.Reader.ReadAt(p, off)
}

func TestIteratorIsLazyAndStopsOnBreak(t *testing.T) {
	data := scalarFixture(t)
	src := &iteratorSource{Reader: bytes.NewReader(data)}
	seq := arrowipc.StreamFrom(context.Background(), src, int64(len(data)), arrowipc.ReadOptions{}, 1)
	if src.calls.Load() != 0 {
		t.Fatal("iterator read eagerly")
	}
	count := 0
	for dt, err := range seq {
		if err != nil || dt == nil {
			t.Fatalf("first table: %v", err)
		}
		count++
		break
	}
	if count != 1 || src.active.Load() != 0 {
		t.Fatalf("break did not finish reader: count=%d active=%d", count, src.active.Load())
	}
	// Reiterating opens a fresh reader over the same caller-owned source.
	rows := 0
	for dt, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		n, _ := dt.Size()
		rows += n
	}
	if rows != 6 {
		t.Fatalf("second iteration rows=%d", rows)
	}
}
func TestIteratorYieldsOneTerminalError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	count := 0
	for dt, err := range arrowipc.StreamFrom(ctx, bytes.NewReader(nil), 0, arrowipc.ReadOptions{}, 1) {
		count++
		if dt != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("error tuple: %v %v", dt, err)
		}
	}
	if count != 1 {
		t.Fatalf("error yielded %d times", count)
	}
}
func TestIteratorPreservesCallerPanicAndCleansUp(t *testing.T) {
	data := scalarFixture(t)
	src := &iteratorSource{Reader: bytes.NewReader(data)}
	token := &struct{ label string }{label: "caller loop panic"}
	func() {
		defer func() {
			if got := recover(); got != token {
				t.Fatalf("caller panic changed: %v", got)
			}
		}()
		for _, err := range arrowipc.StreamFrom(context.Background(), src, int64(len(data)), arrowipc.ReadOptions{}, 1) {
			if err != nil {
				t.Fatal(err)
			}
			panic(token)
		}
	}()
	if src.active.Load() != 0 {
		t.Fatal("reader remains active after caller panic")
	}
}
