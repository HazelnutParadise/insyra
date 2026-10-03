package insyra

import (
	"strings"
	"testing"
	"time"
)

// End-to-end forms of three CCL fixes on a table: a date reaching DAY is an
// error that names DAYOFMONTH (#359), a doubled quote is one quote inside a
// literal and inside a bracketed column name (#364), and DATEADD by a month
// stops at the end of a short month (#367).
func TestCCLDatesAndQuotesOnATable(t *testing.T) {
	dates := func() *DataTable {
		return NewDataTable(NewDataList("2024-01-31", "2024-03-15").SetName("d"))
	}

	dt := dates()
	dt.AddColUsingCCL("x", "DAY(A)")
	if err := dt.PopErr(); err == nil || !strings.Contains(err.Error(), "DAYOFMONTH") {
		t.Errorf("DAY on a date column: error %v, want one naming DAYOFMONTH", err)
	}

	dt = dates()
	dt.AddColUsingCCL("dom", "DAYOFMONTH(A)")
	dt.AddColUsingCCL("next", "DATEADD(A, 1, 'month')")
	if err := dt.Err(); err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("dom").Get(0); got != 31.0 {
		t.Errorf("DAYOFMONTH row 0 = %v, want 31", got)
	}
	if got := dt.GetColByName("next").Get(0); !got.(time.Time).Equal(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("DATEADD(2024-01-31, 1, 'month') = %v, want 2024-02-29", got)
	}

	q := NewDataTable(NewDataList("O'Brien", "Lee").SetName("O'Brien"))
	q.AddColUsingCCL("is_ob", "IF(['O''Brien'] == 'O''Brien', 'it''s him', 'no')")
	if err := q.Err(); err != nil {
		t.Fatal(err)
	}
	if got := q.GetColByName("is_ob").Data(); got[0] != "it's him" || got[1] != "no" {
		t.Errorf("doubled quotes: got %v, want [it's him no]", got)
	}
}
