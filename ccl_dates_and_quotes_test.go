package insyra

import (
	"testing"
)

// End-to-end form of a CCL fix on a table: a doubled quote is one quote inside
// a literal and inside a bracketed column name (#364).
func TestCCLDatesAndQuotesOnATable(t *testing.T) {
	q := NewDataTable(NewDataList("O'Brien", "Lee").SetName("O'Brien"))
	q.AddColUsingCCL("is_ob", "IF(['O''Brien'] == 'O''Brien', 'it''s him', 'no')")
	if err := q.Err(); err != nil {
		t.Fatal(err)
	}
	if got := q.GetColByName("is_ob").Data(); got[0] != "it's him" || got[1] != "no" {
		t.Errorf("doubled quotes: got %v, want [it's him no]", got)
	}
}
