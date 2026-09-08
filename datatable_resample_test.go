package insyra

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDataTable_Resample_MonthlyOHLCV(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	dates := []time.Time{
		time.Date(2024, 1, 2, 0, 0, 0, 0, loc),
		time.Date(2024, 1, 31, 0, 0, 0, 0, loc),
		time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		time.Date(2024, 2, 29, 0, 0, 0, 0, loc),
	}
	dt := NewDataTable(
		NewDataList(anyTimes(dates)...).SetName("Date"),
		NewDataList(10.0, 11.0, 12.0, 13.0).SetName("Open"),
		NewDataList(12.0, 15.0, 14.0, 18.0).SetName("High"),
		NewDataList(9.0, 8.0, 10.0, 11.0).SetName("Low"),
		NewDataList(11.0, 14.0, 13.0, 17.0).SetName("Close"),
		NewDataList(100.0, 200.0, 300.0, 400.0).SetName("Volume"),
	)
	out, err := dt.Resample("Date", ResampleMonthly,
		ResampleAgg{Col: "Open", Op: OpFirst},
		ResampleAgg{Col: "High", Op: OpMax},
		ResampleAgg{Col: "Low", Op: OpMin},
		ResampleAgg{Col: "Close", Op: OpLast},
		ResampleAgg{Col: "Volume", Op: OpSum},
	)
	if err != nil {
		t.Fatalf("Resample returned error: %v", err)
	}
	if out.NumRows() != 2 {
		t.Fatalf("got %d rows, want 2", out.NumRows())
	}
	wantDates := []any{time.Date(2024, 1, 31, 0, 0, 0, 0, loc), time.Date(2024, 2, 29, 0, 0, 0, 0, loc)}
	if got := out.GetColByName("Date").Data(); !reflect.DeepEqual(got, wantDates) {
		t.Errorf("dates = %v, want %v", got, wantDates)
	}
	for name, want := range map[string][]any{
		"Open":   {10.0, 12.0},
		"High":   {15.0, 18.0},
		"Low":    {8.0, 10.0},
		"Close":  {14.0, 17.0},
		"Volume": {300.0, 700.0},
	} {
		sliceEqualApprox(t, out.GetColByName(name).Data(), want, 1e-12)
	}
}

func TestDataTable_Resample_WeeklyMondaySunday(t *testing.T) {
	loc := time.UTC
	dt := NewDataTable(
		NewDataList(time.Date(2024, 1, 5, 0, 0, 0, 0, loc), time.Date(2024, 1, 8, 0, 0, 0, 0, loc)).SetName("Date"),
		NewDataList(5.0, 8.0).SetName("Value"),
	)
	out, err := dt.Resample("Date", ResampleWeekly, ResampleAgg{Col: "Value", Op: OpSum})
	if err != nil {
		t.Fatalf("Resample returned error: %v", err)
	}
	want := []any{time.Date(2024, 1, 7, 0, 0, 0, 0, loc), time.Date(2024, 1, 14, 0, 0, 0, 0, loc)}
	if got := out.GetColByName("Date").Data(); !reflect.DeepEqual(got, want) {
		t.Errorf("weekly labels = %v, want %v", got, want)
	}
}

func TestDataTable_Resample_SkipsEmptyPeriodsAndNamesAs(t *testing.T) {
	loc := time.UTC
	dt := NewDataTable(
		NewDataList(time.Date(2024, 1, 1, 0, 0, 0, 0, loc), time.Date(2024, 4, 1, 0, 0, 0, 0, loc)).SetName("Date"),
		NewDataList(10.0, 40.0).SetName("Close"),
	)
	out, err := dt.Resample("Date", ResampleMonthly, ResampleAgg{Col: "Close", Op: OpLast, As: "MonthClose"})
	if err != nil {
		t.Fatalf("Resample returned error: %v", err)
	}
	if out.NumRows() != 2 || out.GetColByName("MonthClose") == nil {
		t.Fatalf("got %d rows and columns %v, want two rows and MonthClose", out.NumRows(), out.ColNames())
	}
	if out.GetColByName("Date").Data()[1].(time.Time).Month() != time.April {
		t.Fatalf("missing month was fabricated: dates=%v", out.GetColByName("Date").Data())
	}
}

func TestDataTable_Resample_RejectsBadTimeColumnWithRow(t *testing.T) {
	dt := NewDataTable(
		NewDataList(time.Now(), "not a time").SetName("Date"),
		NewDataList(1.0, 2.0).SetName("Value"),
	)
	_, err := dt.Resample("Date", ResampleMonthly, ResampleAgg{Col: "Value", Op: OpSum})
	if err == nil || !strings.Contains(err.Error(), "row 2") {
		t.Fatalf("error = %v, want row 2", err)
	}
}

func TestDataTable_Resample_IsInputOrderIndependent(t *testing.T) {
	loc := time.UTC
	dates := []time.Time{time.Date(2024, 2, 29, 0, 0, 0, 0, loc), time.Date(2024, 1, 31, 0, 0, 0, 0, loc), time.Date(2024, 1, 2, 0, 0, 0, 0, loc)}
	values := []any{29.0, 31.0, 2.0}
	unsorted := NewDataTable(NewDataList(anyTimes(dates)...).SetName("Date"), NewDataList(values...).SetName("Value"))
	sorted := NewDataTable(NewDataList(anyTimes([]time.Time{dates[2], dates[1], dates[0]})...).SetName("Date"), NewDataList(2.0, 31.0, 29.0).SetName("Value"))
	configs := []ResampleAgg{{Col: "Value", Op: OpFirst}, {Col: "Value", Op: OpLast}}
	a, errA := unsorted.Resample("Date", ResampleMonthly, configs...)
	b, errB := sorted.Resample("Date", ResampleMonthly, configs...)
	if errA != nil || errB != nil {
		t.Fatalf("Resample errors: %v, %v", errA, errB)
	}
	if !reflect.DeepEqual(a.To2DSlice(), b.To2DSlice()) {
		t.Fatalf("order changed result: unsorted=%v sorted=%v", a.To2DSlice(), b.To2DSlice())
	}
}

func TestDataTable_Resample_MixedTimeZonesKeepLocalPeriods(t *testing.T) {
	plus8 := time.FixedZone("UTC+8", 8*60*60)
	minus5 := time.FixedZone("UTC-5", -5*60*60)
	dt := NewDataTable(
		NewDataList(
			time.Date(2024, 1, 31, 0, 0, 0, 0, plus8),
			time.Date(2024, 1, 31, 0, 0, 0, 0, minus5),
		).SetName("Date"),
		NewDataList(1.0, 2.0).SetName("Value"),
	)
	out, err := dt.Resample("Date", ResampleMonthly, ResampleAgg{Col: "Value", Op: OpSum})
	if err != nil {
		t.Fatalf("Resample returned error: %v", err)
	}
	if out.NumRows() != 2 {
		t.Fatalf("got %d rows, want one period per timezone", out.NumRows())
	}
	for _, value := range out.GetColByName("Date").Data() {
		if value.(time.Time).Day() != 31 {
			t.Errorf("label = %v, want local month end", value)
		}
	}
}

func anyTimes(values []time.Time) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func TestDataTable_Resample_ErrorCases(t *testing.T) {
	dt := NewDataTable(NewDataList(time.Now()).SetName("Date"), NewDataList(1.0).SetName("Value"))
	cases := []struct {
		name string
		freq ResampleFreq
		aggs []ResampleAgg
		want string
	}{
		{"missing column", ResampleMonthly, []ResampleAgg{{Col: "missing", Op: OpSum}}, "column"},
		{"empty aggs", ResampleMonthly, nil, "aggregate"},
		{"unknown frequency", ResampleFreq(99), []ResampleAgg{{Col: "Value", Op: OpSum}}, "frequency"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dt.Resample("Date", tc.freq, tc.aggs...)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Errorf("error = %v, want mention %q", err, tc.want)
			}
		})
	}
}
