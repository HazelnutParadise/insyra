package parquet

import (
	"context"
	"testing"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
)

// writeInt16File writes a file whose one column, i16, holds int16(i % 300)
// for i in [0, rows).
func writeInt16File(t *testing.T, rows int) string {
	t.Helper()
	fields := []arrow.Field{{Name: "i16", Type: arrow.PrimitiveTypes.Int16, Nullable: true}}
	return writeForeignParquet(t, fields, func(b *array.RecordBuilder) {
		col := b.Field(0).(*array.Int16Builder)
		for i := 0; i < rows; i++ {
			col.Append(int16(i % 300))
		}
	})
}

// TestFilterWithCCLReadsAnInt16Column is the streaming form of the narrow-column
// defect: an int16 cell is not a number to CCL, so a comparison on one answered
// false and an aggregate over one answered 0, neither with an error.
//
// The file's i16 column holds int16(i % 300) for i in [0, 2500), so 9 rows are
// 5. FilterWithCCL computes an aggregate over each batch of 1000 rows rather
// than over the whole file, so no whole-column sum is checked here; SUM over
// int16 cells is pinned in internal/ccl.
func TestFilterWithCCLReadsAnInt16Column(t *testing.T) {
	ctx := context.Background()
	path := writeInt16File(t, 2500)

	for _, expr := range []string{"['i16'] == 5", "['i16'] * 2 == 10"} {
		res, err := FilterWithCCL(ctx, path, expr)
		if err != nil {
			t.Fatalf("FilterWithCCL(%q): %v", expr, err)
		}
		rows, _ := res.Size()
		if rows != 9 {
			t.Fatalf("FilterWithCCL(%q) kept %d rows, want 9", expr, rows)
		}
	}
}
