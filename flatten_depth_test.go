package insyra

import "testing"

// A slice that contains itself must not take the process down.
func TestNewDataListSelfReferentialSliceDoesNotOverflow(t *testing.T) {
	cyclic := []any{1}
	cyclic[0] = cyclic
	dl := NewDataList(cyclic)
	if dl.Len() != 1 {
		t.Fatalf("Len = %d, want the innermost slice kept as one cell", dl.Len())
	}
}

func TestNewDataListFlattensOrdinaryNesting(t *testing.T) {
	dl := NewDataList([]any{1, []any{2, []any{3, 4}}}, 5)
	if dl.Len() != 5 {
		t.Fatalf("Len = %d, want 5", dl.Len())
	}
}

// Shift moves cells; it must not flatten a slice a cell holds.
func TestShiftKeepsASliceCellWhole(t *testing.T) {
	dl := NewDataList()
	dl.Append([]byte{1, 2}, 3)
	got := dl.Shift(1)
	if got.Len() != 2 {
		t.Fatalf("Len = %d, want 2", got.Len())
	}
	if got.Get(0) != nil {
		t.Fatalf("first cell = %v, want nil", got.Get(0))
	}
	if b, ok := got.Get(1).([]byte); !ok || len(b) != 2 {
		t.Fatalf("second cell = %v, want the []byte{1, 2}", got.Get(1))
	}
	if dl.Shift(0).Len() != 2 || dl.Shift(-1).Len() != 2 {
		t.Fatal("Shift(0) and Shift(-1) changed the length")
	}
}
