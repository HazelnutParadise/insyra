package parquet

import (
	"context"
	"testing"
)

// TestFilterWithCCLReadsAnInt16Column is the streaming form of the narrow-column
// defect: an int16 cell is not a number to CCL, so a comparison on one answered
// false and an aggregate over one answered 0, neither with an error.
//
// The file's i16 column holds int16(i % 300) for i in [0, 2500), so 9 rows are
// 5 and the column sums to 363750.
func TestFilterWithCCLReadsAnInt16Column(t *testing.T) {
	ctx := context.Background()
	path := writeManyTypesFile(t, 2500)

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

	res, err := FilterWithCCL(ctx, path, "SUM(['i16']) == 363750")
	if err != nil {
		t.Fatalf("FilterWithCCL over the whole column: %v", err)
	}
	rows, _ := res.Size()
	if rows != 2500 {
		t.Fatalf("FilterWithCCL over the whole column kept %d rows, want 2500", rows)
	}
}
