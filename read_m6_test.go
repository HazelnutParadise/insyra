package insyra_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// M6: CSV columns get pandas-style, column-level type inference. An all-integer
// column parses as int64 so large integers keep full precision; a column with
// any decimal parses as float64; empty cells in a numeric column become NaN;
// non-numeric columns stay strings.
func TestReadCSV_M6ColumnTypeInference(t *testing.T) {
	dt, err := insyra.ReadCSV_String(
		"id,val,mix,note\n9007199254740993,1,1,a\n9007199254740994,2,2.5,b\n", false, true)
	if err != nil {
		t.Fatalf("ReadCSV_String: %v", err)
	}

	// Large integers (> 2^53) must survive exactly as int64. As float64 both rows
	// would collapse to 9007199254740992.
	id0 := dt.GetColByName("id").Data()[0]
	if got, ok := id0.(int64); !ok || got != 9007199254740993 {
		t.Fatalf("id[0] = %v (%T), want int64(9007199254740993)", id0, id0)
	}

	// All-integer column -> int64.
	if _, ok := dt.GetColByName("val").Data()[0].(int64); !ok {
		t.Fatalf("val column not int64: %T", dt.GetColByName("val").Data()[0])
	}
	// Column containing 2.5 -> whole column float64.
	if _, ok := dt.GetColByName("mix").Data()[0].(float64); !ok {
		t.Fatalf("mix column not float64: %T", dt.GetColByName("mix").Data()[0])
	}
	// Non-numeric column -> string.
	if _, ok := dt.GetColByName("note").Data()[0].(string); !ok {
		t.Fatalf("note column not string: %T", dt.GetColByName("note").Data()[0])
	}

	// Empty cell in an otherwise-numeric column -> NaN (float64), homogeneous.
	// A two-column CSV is used because encoding/csv skips fully blank lines, so a
	// lone empty line would not produce an empty cell.
	dt2, err := insyra.ReadCSV_String("y,z\n1,10\n,20\n3,30\n", false, true)
	if err != nil {
		t.Fatalf("ReadCSV_String: %v", err)
	}
	yc := dt2.GetColByName("y").Data()
	if _, ok := yc[0].(float64); !ok {
		t.Fatalf("y[0] not float64: %T", yc[0])
	}
	if f, ok := yc[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("y[1] (empty) should be NaN float64, got %v (%T)", yc[1], yc[1])
	}

	// int64 CSV data must flow through numeric operations without panicking.
	if mean := dt.GetColByName("val").Mean(); mean != 1.5 {
		t.Fatalf("Mean over int64 column = %v, want 1.5", mean)
	}
}
