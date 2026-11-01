package insyra

import (
	"testing"
	"time"
)

func clockTestTable(n int) *DataTable {
	col := NewDataList()
	for i := range n {
		col.Append(i)
	}
	return NewDataTable(col)
}

// One call reads one moment, whichever way the rows are walked.
func TestClockIsTheSameOnEveryRow(t *testing.T) {
	check := func(name string, col []any) {
		t.Helper()
		first, ok := col[0].(time.Time)
		if !ok {
			t.Fatalf("%s: first cell is %#v", name, col[0])
		}
		for i, v := range col {
			if v != any(first) {
				t.Fatalf("%s: row %d is %v, row 0 is %v", name, i, v, first)
			}
		}
	}
	expression := clockTestTable(5000)
	expression.AddColUsingCCL("now", "NOW()")
	if err := expression.Err(); err != nil {
		t.Fatal(err)
	}
	check("AddColUsingCCL", expression.GetColByName("now").Data())

	statements := clockTestTable(5000)
	statements.ExecuteCCL("NEW('a') = NOW(); NEW('b') = NOW()")
	if err := statements.Err(); err != nil {
		t.Fatal(err)
	}
	check("ExecuteCCL", statements.GetColByName("a").Data())
	if statements.GetColByName("a").Data()[0] != statements.GetColByName("b").Data()[0] {
		t.Error("two statements of one ExecuteCCL read different moments")
	}
}
