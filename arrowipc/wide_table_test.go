package arrowipc_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/arrowipc"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

func TestWideTableConstructionRespectsCancellation(t *testing.T) {
	const width = 65536
	fields := make([]arrow.Field, width)
	for i := range fields {
		fields[i] = arrow.Field{Name: fmt.Sprintf("field_%d", i), Type: arrow.Null, Nullable: true}
	}
	schema := arrow.NewSchema(fields, nil)
	data := writeRecords(t, schema, func(b *array.RecordBuilder) {
		for i := range fields {
			b.Field(i).AppendNull()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	tables, errs := streamFrom(ctx, bytes.NewReader(data), int64(len(data)), arrowipc.ReadOptions{}, 1)
	got, err := drain(t, tables, errs)
	elapsed := time.Since(started)
	if elapsed > 3*time.Second {
		t.Fatalf("wide construction took %v despite a one-second deadline", elapsed)
	}
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err == nil {
		if len(got) != 1 || got[0].GetColNameByNumber(0) != "field_0" || got[0].GetColNameByNumber(width-1) != fmt.Sprintf("field_%d", width-1) || got[0].GetColByNumber(width-1).Get(0) != nil {
			t.Fatal("wide table lost names or nulls")
		}
	}
}
