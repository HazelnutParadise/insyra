package arrowipc_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
)

func TestSuccessfulStreamClosesErrorChannelWithoutValue(t *testing.T) {
	data := scalarFixture(t)
	tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 3)
	for range tables {
	}
	if err, ok := <-errs; ok {
		t.Fatalf("successful stream sent a value: %v", err)
	}
}

func TestZeroColumnRowsCannotBeSilentlyLost(t *testing.T) {
	schema := arrow.NewSchema(nil, nil)
	var buf bytes.Buffer
	w, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	rec := array.NewRecordBatch(schema, nil, 3)
	err = w.Write(rec)
	rec.Release()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(buf.Bytes()), int64(buf.Len()), arrowipc.ReadOptions{}, 1)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "zero-column") {
		t.Fatalf("nonempty zero-column record: %d tables, %v", len(got), err)
	}
}

func TestSchemaOnlyFileEmitsNoTables(t *testing.T) {
	data := schemaOnlyFixture(t, arrow.NewSchema(nil, nil))
	info, err := arrowipc.InspectFrom(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil || info.NumBatches != 0 {
		t.Fatalf("empty inspect: %#v %v", info, err)
	}
	tables, errs := arrowipc.StreamFrom(context.Background(), bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 1)
	got, err := drain(t, tables, errs)
	if len(got) != 0 || err != nil {
		t.Fatalf("schema-only stream: %d tables %v", len(got), err)
	}
}
