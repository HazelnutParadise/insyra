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
