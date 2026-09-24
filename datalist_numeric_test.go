package insyra

import (
	"math"
	"reflect"
	"testing"
)

func TestReplaceLastLeavesTrailingNaN(t *testing.T) {
	dl := NewDataList(5, math.NaN())
	dl.ReplaceLast(5, 0)
	got := dl.Data()
	if got[0] != 0 {
		t.Fatalf("expected first cell replaced, got %v", got)
	}
	if f, ok := got[1].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("expected trailing NaN untouched, got %v", got)
	}
}

func TestReplaceLastReplacesLastMatch(t *testing.T) {
	dl := NewDataList(5, 1, 5)
	dl.ReplaceLast(5, 0)
	if !reflect.DeepEqual(dl.Data(), []any{5, 1, 0}) {
		t.Fatalf("got %v", dl.Data())
	}
}

func TestReplaceLastNaNTargetsLastNaN(t *testing.T) {
	dl := NewDataList(math.NaN(), 1, math.NaN())
	dl.ReplaceLast(math.NaN(), 0)
	got := dl.Data()
	if f, ok := got[0].(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("first NaN should stay, got %v", got)
	}
	if got[2] != 0 {
		t.Fatalf("last NaN should be replaced, got %v", got)
	}
}
