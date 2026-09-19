package insyra

import (
	"reflect"
	"testing"
)

// CCL-2: `@` as a value hands each row its own slice.
func TestCCLCurrentRowIsCopied(t *testing.T) {
	dt := NewDataTable(NewDataList(10, 20, 30), NewDataList(1, 2, 3))
	dt.AddColUsingCCL("r", "@")
	got := dt.GetColByName("r").Data()
	want := []any{[]any{10, 1}, []any{20, 2}, []any{30, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("@ column = %v, want %v", got, want)
	}
	dt2 := NewDataTable(NewDataList(10, 20, 30), NewDataList(1, 2, 3))
	dt2.ExecuteCCL("NEW('r') = @")
	if got := dt2.GetColByName("r").Data(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ExecuteCCL @ column = %v, want %v", got, want)
	}
}
