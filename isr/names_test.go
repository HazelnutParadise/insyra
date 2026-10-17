package isr

import (
	"reflect"
	"testing"
)

// Of and From are both kept on purpose, after Java's List.of and Rust's
// From::from, and Push is not a second name for Append or AppendCols (#212).

func TestOfAndFromBuildTheSameValue(t *testing.T) {
	ofList, fromList := DL.Of(1, "a", 2.5), DL.From(1, "a", 2.5)
	if reflect.TypeOf(ofList) != reflect.TypeOf(fromList) || !reflect.DeepEqual(ofList.Data(), fromList.Data()) {
		t.Fatalf("DL.Of gave %T %v, DL.From gave %T %v", ofList, ofList.Data(), fromList, fromList.Data())
	}
	row := Row{Name("x"): 1, Name("y"): "b"}
	ofTable, fromTable := DT.Of(row), DT.From(row)
	if reflect.TypeOf(ofTable) != reflect.TypeOf(fromTable) || !reflect.DeepEqual(ofTable.Data(), fromTable.Data()) {
		t.Fatalf("DT.Of gave %T %v, DT.From gave %T %v", ofTable, ofTable.Data(), fromTable, fromTable.Data())
	}
}

func TestPushKeepsTheIsrChainAndTakesRows(t *testing.T) {
	if got := DL.From(1, 2, 3).Push(4).At(3); got != 4 {
		t.Fatalf("DL.From(1, 2, 3).Push(4).At(3) = %v, want 4", got)
	}
	table := DT.From(Row{Name("x"): 1})
	table.Push(Row{Name("x"): 2})
	if rows, cols := table.Size(); rows != 2 || cols != 1 {
		t.Fatalf("after Push(Row) the table is %dx%d, want 2x1", rows, cols)
	}
}
