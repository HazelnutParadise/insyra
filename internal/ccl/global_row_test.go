package ccl

import (
	"reflect"
	"testing"
)

// batchContext is one batch of a larger table for tests: a MapContext holding
// the batch's rows, and the position of its first row in the whole table.
type batchContext struct {
	*MapContext
	offset int
}

func (b *batchContext) GlobalRowIndex() int { return b.offset + b.GetRowIndex() }

// splitIntoBatches cuts every column of data into batches of at most size
// rows, in order, and builds a batchContext for each. The offset of a batch is
// the position of its first row in the whole table.
func splitIntoBatches(t *testing.T, data map[string][]any, size int) []*batchContext {
	t.Helper()
	if size < 1 {
		t.Fatalf("batch size %d is not positive", size)
	}
	rows := -1
	for name, col := range data {
		if rows >= 0 && len(col) != rows {
			t.Fatalf("column %q has %d rows, expected %d", name, len(col), rows)
		}
		rows = len(col)
	}

	var batches []*batchContext
	for start := 0; start < rows; start += size {
		end := min(start+size, rows)
		part := make(map[string][]any, len(data))
		for name, col := range data {
			part[name] = col[start:end]
		}
		ctx, err := NewMapContext(part)
		if err != nil {
			t.Fatalf("batch starting at row %d: %v", start, err)
		}
		batches = append(batches, &batchContext{MapContext: ctx, offset: start})
	}
	return batches
}

func TestRowIndexIsTheGlobalRowInABatch(t *testing.T) {
	data := map[string][]any{"A": {10.0, 20.0, 30.0, 40.0, 50.0}}
	batches := splitIntoBatches(t, data, 2)
	if len(batches) != 3 {
		t.Fatalf("5 rows in batches of 2 gave %d batches, want 3", len(batches))
	}

	rowIndex, err := CompileExpression("#")
	if err != nil {
		t.Fatal(err)
	}
	cellAtRow, err := CompileExpression("A.#")
	if err != nil {
		t.Fatal(err)
	}

	for _, batch := range batches {
		for local := 0; local < batch.GetRowCount(); local++ {
			global := batch.offset + local
			if err := batch.SetRowIndex(local); err != nil {
				t.Fatal(err)
			}

			got, err := Evaluate(rowIndex, batch)
			if err != nil {
				t.Fatalf("# at global row %d: %v", global, err)
			}
			if got != float64(global) {
				t.Errorf("# at global row %d (batch offset %d, local row %d) = %v (%T), want %v",
					global, batch.offset, local, got, got, float64(global))
			}

			got, err = Evaluate(cellAtRow, batch)
			if err != nil {
				t.Fatalf("A.# at global row %d: %v", global, err)
			}
			if want := data["A"][global]; got != want {
				t.Errorf("A.# at global row %d (batch offset %d, local row %d) = %v, want %v",
					global, batch.offset, local, got, want)
			}
		}
	}
}

func TestRowIndexOfAPlainContextIsUnchanged(t *testing.T) {
	ctx, err := NewMapContext(map[string][]any{"A": {10.0, 20.0, 30.0, 40.0, 50.0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := any(ctx).(GlobalRowContext); ok {
		t.Fatal("MapContext must stay a plain Context")
	}

	rowIndex, err := CompileExpression("#")
	if err != nil {
		t.Fatal(err)
	}
	for row := 0; row < ctx.GetRowCount(); row++ {
		if err := ctx.SetRowIndex(row); err != nil {
			t.Fatal(err)
		}
		got, err := Evaluate(rowIndex, ctx)
		if err != nil {
			t.Fatalf("# at row %d: %v", row, err)
		}
		if want := float64(ctx.GetRowIndex()); got != want {
			t.Errorf("# at row %d = %v (%T), want %v", row, got, got, want)
		}
	}
}

func TestRowShapedPartsReadTheCurrentRowInABatch(t *testing.T) {
	data := map[string][]any{
		"A": {10.0, 20.0, 30.0, 40.0, 50.0},
		"B": {1.0, 2.0, 3.0, 4.0, 5.0},
	}
	whole, err := NewMapContext(data)
	if err != nil {
		t.Fatal(err)
	}
	batches := splitIntoBatches(t, data, 2)

	for _, expr := range []string{"A:B", "@"} {
		compiled, err := CompileExpression(expr)
		if err != nil {
			t.Fatal(err)
		}
		for _, batch := range batches {
			for local := 0; local < batch.GetRowCount(); local++ {
				global := batch.offset + local
				if err := batch.SetRowIndex(local); err != nil {
					t.Fatal(err)
				}
				if err := whole.SetRowIndex(global); err != nil {
					t.Fatal(err)
				}

				want, err := Evaluate(compiled, whole)
				if err != nil {
					t.Fatalf("%s on the whole table at row %d: %v", expr, global, err)
				}
				got, err := Evaluate(compiled, batch)
				if err != nil {
					t.Fatalf("%s at global row %d (batch offset %d, local row %d): %v",
						expr, global, batch.offset, local, err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s at global row %d (batch offset %d, local row %d) = %v, want %v",
						expr, global, batch.offset, local, got, want)
				}
			}
		}
	}
}
