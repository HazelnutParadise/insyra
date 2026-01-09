package datafetch

import (
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

func TestTryParseTime(t *testing.T) {
	cases := []string{
		"2025-09-30T08:00:00+08:00",
		"2025-09-30T08:00:00Z",
		"2025-09-30",
	}
	for _, c := range cases {
		if _, ok := utils.TryParseTime(c); !ok {
			t.Fatalf("tryParseTime failed to parse %s", c)
		}
	}
}

func TestMapDateTruncate(t *testing.T) {
	// Build a simple DataTable with a column named dateReported
	dt := insyra.NewDataTable()
	col := insyra.NewDataList()
	col.SetName("dateReported")
	col.Append("2025-09-30T08:00:00+08:00")
	col.Append("2025-11-30T08:00:00+08:00")
	dt.AppendCols(col)

	// apply the same mapping used in MutualFundHolders
	dt2 := dt.Map(func(rowIndex int, colIndex string, element any) any {
		if dt.GetColNameByIndex(colIndex) == "dateReported" {
			if str, ok := element.(string); ok {
				if parsed, ok := utils.TryParseTime(str); ok {
					dateOnly := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
					return dateOnly
				}
			}
		}
		return element
	})

	col0 := dt2.GetColByNumber(0)
	v0 := col0.Get(0)
	t0, ok := v0.(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", v0)
	}
	// Use De Morgan's law instead of negating the conjunction
	if t0.Year() != 2025 || t0.Month() != time.September || t0.Day() != 30 {
		t.Fatalf("unexpected date: %v", t0)
	}
}
