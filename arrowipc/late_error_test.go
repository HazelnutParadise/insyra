package arrowipc_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	flatbuffers "github.com/google/flatbuffers/go"
)

var errBrokenRecord = errors.New("second record unavailable")

type brokenSecondRecord struct {
	data       []byte
	start, end int64
}

func (r brokenSecondRecord) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.start && off < r.end {
		return 0, errBrokenRecord
	}
	return bytes.NewReader(r.data).ReadAt(p, off)
}
func TestLateReadErrorIsNotLostAfterDeliveredBatches(t *testing.T) {
	data := scalarFixture(t)
	footerAt := len(data) - 10 - int(binary.LittleEndian.Uint32(data[len(data)-10:]))
	footer := data[footerAt : len(data)-10]
	tab := flatbuffers.Table{Bytes: footer, Pos: flatbuffers.GetUOffsetT(footer)}
	vector := tab.Vector(flatbuffers.UOffsetT(tab.Offset(10)))
	second := int64(binary.LittleEndian.Uint64(footer[int(vector)+24:]))
	src := brokenSecondRecord{data: data, start: second, end: int64(footerAt)}
	tables, errs := streamFrom(context.Background(), src, int64(len(data)), arrowipc.ReadOptions{Columns: namedColumns([]string{"id"})}, 3)
	got, err := drain(t, tables, errs)
	if !errors.Is(err, errBrokenRecord) {
		t.Fatalf("late error: %v", err)
	}
	rows := 0
	for _, dt := range got {
		n, _ := dt.Size()
		rows += n
	}
	if rows != 4 {
		t.Fatalf("delivered %d rows before second-record error, want 4", rows)
	}
}

type fullBodyError struct {
	data   []byte
	offset int64
}

func (r fullBodyError) ReadAt(p []byte, off int64) (int, error) {
	n, err := bytes.NewReader(r.data).ReadAt(p, off)
	if off == r.offset {
		return n, errBrokenRecord
	}
	return n, err
}
func TestFullBodyReadDoesNotHideSourceFailure(t *testing.T) {
	data := scalarFixture(t)
	footer := footerTable(data)
	vector := footer.Vector(flatbuffers.UOffsetT(footer.Offset(10)))
	offset := int64(binary.LittleEndian.Uint64(footer.Bytes[vector : vector+8]))
	meta := int64(binary.LittleEndian.Uint32(footer.Bytes[vector+8 : vector+12]))
	tables, errs := streamFrom(context.Background(), fullBodyError{data: data, offset: offset + meta}, int64(len(data)), arrowipc.ReadOptions{}, 2)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || !errors.Is(err, errBrokenRecord) {
		t.Fatalf("full body source error lost: %d tables, %v", len(got), err)
	}
}
