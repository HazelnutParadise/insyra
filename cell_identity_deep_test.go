package insyra

import "testing"

// nestedAny wraps leaf in n levels of []any.
func nestedAny(n int, leaf any) any {
	v := leaf
	for i := 0; i < n; i++ {
		v = []any{v}
	}
	return v
}

func TestDeeplyNestedValuesWithDifferentLeavesStayApart(t *testing.T) {
	one, two := nestedAny(40, 1), nestedAny(40, 2)
	dl := NewDataList()
	dl.Append(one)
	dl.Append(two)
	dl.Append(nestedAny(40, 2))
	if got := dl.Count(one); got != 1 {
		t.Errorf("Count(leaf 1) = %d, want 1", got)
	}
	// Past the depth limit a slice is identified by its address, so a separate
	// copy of the same shape is another value; the value itself is found once.
	if got := dl.Count(two); got != 1 {
		t.Errorf("Count(leaf 2) = %d, want 1", got)
	}
}
