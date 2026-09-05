package insyra

import (
	"testing"
	"time"
)

func TestDataList_ParseDates_ISOStrings(t *testing.T) {
	dl := NewDataList("2026-09-01", "2026-09-02", 3, "bad")
	got := dl.ParseDates().Data()

	if len(got) != 4 {
		t.Fatalf("len = %d want 4", len(got))
	}
	first, ok := got[0].(time.Time)
	if !ok {
		t.Fatalf("element 0 = %T want time.Time", got[0])
	}
	if !first.Equal(time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("element 0 = %v want 2026-09-01 UTC", first)
	}
	if first.Location() != time.UTC {
		t.Errorf("element 0 location = %v want UTC", first.Location())
	}
	second, ok := got[1].(time.Time)
	if !ok {
		t.Fatalf("element 1 = %T want time.Time", got[1])
	}
	if !second.Equal(time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("element 1 = %v want 2026-09-02 UTC", second)
	}
	if got[2] != nil {
		t.Errorf("element 2 = %v want nil (non-string, non-time cell)", got[2])
	}
	if got[3] != nil {
		t.Errorf("element 3 = %v want nil (unparsable string)", got[3])
	}
}

func TestDataList_ParseDates_MutatesInPlaceAndReturnsSelf(t *testing.T) {
	dl := NewDataList("2026-09-01")
	out := dl.ParseDates()
	if out != dl {
		t.Fatalf("ParseDates should return the same DataList")
	}
	if _, ok := dl.Get(0).(time.Time); !ok {
		t.Fatalf("element 0 = %T want time.Time after in-place conversion", dl.Get(0))
	}
}

func TestDataList_ParseDates_CustomLayout(t *testing.T) {
	dl := NewDataList("2026/09/01")
	got := dl.ParseDates("2006/01/02").Data()
	parsed, ok := got[0].(time.Time)
	if !ok {
		t.Fatalf("element 0 = %T want time.Time", got[0])
	}
	if !parsed.Equal(time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("element 0 = %v want 2026-09-01 UTC", parsed)
	}
}

func TestDataList_ParseDates_CustomLayoutReplacesDefaults(t *testing.T) {
	// An explicit layout list is the whole list: an ISO string that only the
	// default layouts would match becomes nil.
	dl := NewDataList("2026-09-01")
	got := dl.ParseDates("2006/01/02").Data()
	if got[0] != nil {
		t.Errorf("element 0 = %v want nil (default layouts not applied)", got[0])
	}
}

func TestDataList_ParseDates_MultipleLayoutsTriedInOrder(t *testing.T) {
	dl := NewDataList("2026/09/01", "01-09-2026")
	got := dl.ParseDates("2006/01/02", "02-01-2006").Data()
	for i, want := range []time.Time{
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
	} {
		parsed, ok := got[i].(time.Time)
		if !ok {
			t.Fatalf("element %d = %T want time.Time", i, got[i])
		}
		if !parsed.Equal(want) {
			t.Errorf("element %d = %v want %v", i, parsed, want)
		}
	}
}

func TestDataList_ParseDates_TimeValuesPassThrough(t *testing.T) {
	original := time.Date(2026, time.September, 1, 8, 30, 0, 0, time.FixedZone("UTC+8", 8*3600))
	dl := NewDataList(original)
	got := dl.ParseDates().Data()
	parsed, ok := got[0].(time.Time)
	if !ok {
		t.Fatalf("element 0 = %T want time.Time", got[0])
	}
	if !parsed.Equal(original) || parsed.Location() != original.Location() {
		t.Errorf("element 0 = %v (%v) want the original value untouched", parsed, parsed.Location())
	}
}

func TestDataList_ParseDates_OffsetStringsBecomeUTC(t *testing.T) {
	dl := NewDataList("2026-09-01T08:30:00+08:00")
	got := dl.ParseDates().Data()
	parsed, ok := got[0].(time.Time)
	if !ok {
		t.Fatalf("element 0 = %T want time.Time", got[0])
	}
	if parsed.Location() != time.UTC {
		t.Errorf("location = %v want UTC", parsed.Location())
	}
	if !parsed.Equal(time.Date(2026, time.September, 1, 0, 30, 0, 0, time.UTC)) {
		t.Errorf("element 0 = %v want 2026-09-01T00:30:00Z", parsed)
	}
}

func TestDataList_ParseDates_EmptyList(t *testing.T) {
	dl := NewDataList()
	if got := dl.ParseDates().Data(); len(got) != 0 {
		t.Fatalf("len = %d want 0", len(got))
	}
}

func parseDatesTestTable() *DataTable {
	date := NewDataList("2024-01-02", "2024-01-15", "2024-02-01")
	date.SetName("Date")
	closeCol := NewDataList(10.0, 11.0, 20.0)
	closeCol.SetName("Close")
	dt := NewDataTable()
	dt.AppendCols(date, closeCol)
	return dt
}

func TestDataTable_ParseDatesCols_InPlaceThenResample(t *testing.T) {
	dt := parseDatesTestTable()
	out := dt.ParseDatesCols([]string{"Date"})
	if out != dt {
		t.Fatalf("ParseDatesCols should return the same DataTable")
	}
	if err := dt.Err(); err != nil {
		t.Fatalf("unexpected error recorded: %v", err)
	}
	if _, ok := dt.GetColByName("Date").Get(0).(time.Time); !ok {
		t.Fatalf("Date[0] = %T want time.Time", dt.GetColByName("Date").Get(0))
	}

	m, err := dt.Resample("Date", ResampleMonthly, ResampleAgg{Col: "Close", Op: OpLast})
	if err != nil {
		t.Fatalf("Resample after ParseDatesCols failed: %v", err)
	}
	if rows := m.NumRows(); rows != 2 {
		t.Errorf("monthly rows = %d want 2", rows)
	}
}

func TestDataTable_ParseDatesCols_AcceptsExcelIndex(t *testing.T) {
	dt := parseDatesTestTable()
	dt.ParseDatesCols([]string{"A"})
	if _, ok := dt.GetColByName("Date").Get(0).(time.Time); !ok {
		t.Fatalf("Date[0] = %T want time.Time", dt.GetColByName("Date").Get(0))
	}
}

func TestDataTable_ParseDatesCols_MissingColumnWarns(t *testing.T) {
	dt := parseDatesTestTable()
	dt.ParseDatesCols([]string{"nope", "Date"})
	if err := dt.Err(); err == nil {
		t.Fatal("expected a warning recorded for the missing column")
	}
	// The named column present in the table is still converted.
	if _, ok := dt.GetColByName("Date").Get(0).(time.Time); !ok {
		t.Fatalf("Date[0] = %T want time.Time", dt.GetColByName("Date").Get(0))
	}
}

func TestDataTable_ParseDatesCols_LeavesOtherColumnsAlone(t *testing.T) {
	dt := parseDatesTestTable()
	dt.ParseDatesCols([]string{"Date"})
	if got := dt.GetColByName("Close").Get(0); got != 10.0 {
		t.Errorf("Close[0] = %v want 10", got)
	}
}

func TestDataTable_ParseDatesCols_CustomLayout(t *testing.T) {
	date := NewDataList("02/01/2024", "15/01/2024")
	date.SetName("Date")
	dt := NewDataTable()
	dt.AppendCols(date)
	dt.ParseDatesCols([]string{"Date"}, "02/01/2006")
	parsed, ok := dt.GetColByName("Date").Get(0).(time.Time)
	if !ok {
		t.Fatalf("Date[0] = %T want time.Time", dt.GetColByName("Date").Get(0))
	}
	if !parsed.Equal(time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Date[0] = %v want 2024-01-02", parsed)
	}
}

// ParseDates / ParseDatesCols belong on the authoritative API surface, so the
// two methods must be reachable through IDataList / IDataTable.
func TestParseDates_OnInterfaces(t *testing.T) {
	var list IDataList = NewDataList("2026-09-01")
	if _, ok := list.ParseDates().Get(0).(time.Time); !ok {
		t.Fatalf("IDataList.ParseDates did not convert the cell")
	}

	var table IDataTable = parseDatesTestTable()
	table.ParseDatesCols([]string{"Date"})
	if _, ok := table.GetColByName("Date").Get(0).(time.Time); !ok {
		t.Fatalf("IDataTable.ParseDatesCols did not convert the column")
	}
}
