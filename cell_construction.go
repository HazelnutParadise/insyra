package insyra

// Cell marks v so a constructor stores it as one cell rather than flattening
// it.
//
// NewDataList flattens slices on purpose, so that building a list reads the
// way building a pandas Series does — NewDataList([]int{1, 2, 3}) is three
// cells. Cell opts one argument out of that, inline among ordinary values:
//
//	NewDataList(Cell([]int{1, 2}), 3, "a")   // three cells
//
// The cell holds v at its own type; the mark is removed on the way in and is
// never stored. Every entry point that takes a value from the caller accepts
// it, including the ones that do not flatten, so Append(Cell(x)) and
// Append(x) mean the same thing and writing the first is not a trap.
func Cell(v any) any {
	return cellMarker{v}
}

// cellMarker is unexported so that a value carrying it can only have come
// from Cell, and so the shape can change.
type cellMarker struct{ v any }

// unwrapCell returns what a mark stands for, or its argument unchanged.
func unwrapCell(v any) any {
	if m, ok := v.(cellMarker); ok {
		return m.v
	}
	return v
}

// unwrapCells applies unwrapCell to each value. It allocates only when one of
// them is marked, because it is on Append's path.
func unwrapCells(values []any) []any {
	marked := false
	for _, v := range values {
		if _, ok := v.(cellMarker); ok {
			marked = true
			break
		}
	}
	if !marked {
		return values
	}
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = unwrapCell(v)
	}
	return out
}
