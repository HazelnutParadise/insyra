package datafetch

import (
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func TestNormalizeDateColumns_AllTypes(t *testing.T) {
	dt := insyra.NewDataTable()

	c1 := insyra.NewDataList()
	c1.SetName("Date")
	c1.Append("2025-09-30T08:00:00+08:00")

	c2 := insyra.NewDataList()
	c2.SetName("publishTime")
	c2.Append("2025-11-30T08:00:00Z")

	c3 := insyra.NewDataList()
	c3.SetName("expiryDate")
	c3.Append("2025-10-31")

	c4 := insyra.NewDataList()
	c4.SetName("notadate")
	c4.Append("2025-12-01T00:00:00+00:00")

	dt.AppendCols(c1, c2, c3, c4)

	dt2 := normalizeDateColumns(dt)

	// Date -> time.Time
	v := dt2.GetColByNumber(0).Get(0)
	if _, ok := v.(time.Time); !ok {
		t.Fatalf("expected time.Time for Date, got %T", v)
	}

	// publishTime -> time.Time
	v2 := dt2.GetColByNumber(1).Get(0)
	if _, ok := v2.(time.Time); !ok {
		t.Fatalf("expected time.Time for publishTime, got %T", v2)
	}

	// expiryDate -> time.Time
	v3 := dt2.GetColByNumber(2).Get(0)
	if _, ok := v3.(time.Time); !ok {
		t.Fatalf("expected time.Time for expiryDate, got %T", v3)
	}

	// notadate should not be converted (name does not match/contains date but lowercased contains "notadate" which includes "date" — this checks behavior)
	v4 := dt2.GetColByNumber(3).Get(0)
	if _, ok := v4.(time.Time); ok {
		t.Fatalf("expected notadate to remain string, but got time.Time")
	}

	// Also check that date-only truncation applied (hour==0)
	tm := dt2.GetColByNumber(0).Get(0).(time.Time)
	if !(tm.Hour() == 0 && tm.Minute() == 0 && tm.Second() == 0) {
		t.Fatalf("expected truncated date-only time, got %v", tm)
	}
}
