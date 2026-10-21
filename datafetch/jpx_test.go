package datafetch

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func jpxFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "jpx", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func newFixtureJPX(t *testing.T) (*JPXClient, *fixtureTransport) {
	t.Helper()
	client, err := JPX(TWStockConfig{})
	if err != nil {
		t.Fatalf("JPX returned error: %v", err)
	}
	transport := newFixtureTransport()
	client.base.client.Transport = transport
	client.baseURL = "https://jpx.test"
	add := func(key, name string) {
		for i := 0; i < 4; i++ { // enough for a few calls on one client
			transport.replies[key] = append(transport.replies[key], fixtureReply{status: http.StatusOK, body: jpxFixture(t, name)})
		}
	}
	const dir = "/markets/statistics-equities/investor-type/"
	add(dir+"index.html", "index.html")
	add(dir+"00-00-archives-00.html", "index.html")
	add(dir+"00-00-archives-03.html", "archives-03.html")
	add(dir+"t13vrt000001yhs9-att/stock_val_1_260902.xls", "stock_val_1_260902.xls")
	add(dir+"t13vrt000002gv8k-att/stock_1_w_20261005_20261009.xlsx", "stock_1_w_20261005_20261009.xlsx")
	add(dir+"aaaa-att/stock_val_1_230101.xls", "stock_val_1_230101.xls")
	add(dir+"aaaa-att/stock_vol_1_230101.xls", "stock_vol_1_230101.xls")
	return client, transport
}

// jpxRows turns the table into one slice per row, in the column order of the
// result: WeekStart, WeekEnd, Market, Investor, Sales, Purchases, Balance, Total.
func jpxRows(t *testing.T, dt *insyra.DataTable) [][]any {
	t.Helper()
	names := []string{"WeekStart", "WeekEnd", "Market", "Investor", "Sales", "Purchases", "Balance", "Total"}
	cols := make([][]any, len(names))
	for i, name := range names {
		col := dt.GetColByName(name)
		if col == nil {
			t.Fatalf("missing column %s", name)
			return nil
		}
		cols[i] = col.Data()
	}
	rows := make([][]any, dt.NumRows())
	for r := range rows {
		rows[r] = make([]any, len(names))
		for c := range names {
			rows[r][c] = cols[c][r]
		}
	}
	return rows
}

func jpxFind(t *testing.T, rows [][]any, start time.Time, market, investor string) []any {
	t.Helper()
	for _, r := range rows {
		if r[0] == start && r[2] == market && r[3] == investor {
			return r
		}
	}
	t.Fatalf("no row for %s %s %s", start.Format("2006-01-02"), market, investor)
	return nil
}

func TestJPXWeeklyWorkbook(t *testing.T) {
	data := jpxFixture(t, "stock_1_w_20261005_20261009.xlsx")
	start, end := dateUTC(2026, time.October, 5), dateUTC(2026, time.October, 9)
	value, err := parseJPXWeekly(data, start, end, JPXUnitValue)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 4*14 {
		t.Fatalf("rows = %d, want 4 markets x 14 investor groups", len(value))
	}
	got := map[string]jpxRow{}
	for _, r := range value {
		got[r.market+"|"+r.investor] = r
	}
	// TSE Prime, proprietary cash: sales 5,403,540,773 and purchases 4,311,871,193 thousand yen.
	r := got["TSE Prime|Proprietary Cash"]
	if r.sales != 5403540773 || r.purchases != 4311871193 || r.balance != -1091669580 || r.total != 9715411966 {
		t.Errorf("TSE Prime proprietary cash: %+v", r)
	}
	if r := got["Tokyo & Nagoya|Individuals Margin"]; r.sales != 8382684653 || r.purchases != 8650601503 || r.total != 17033286156 {
		t.Errorf("Tokyo & Nagoya individuals margin: %+v", r)
	}
	shares, err := parseJPXWeekly(data, start, end, JPXUnitShares)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range shares {
		if r.market == "TSE Prime" && r.investor == "Proprietary Cash" && (r.sales != 2010656 || r.purchases != 1597867) {
			t.Errorf("TSE Prime proprietary cash shares: %+v", r)
		}
	}
	if _, err := parseJPXWeekly([]byte("not a workbook"), start, end, JPXUnitValue); err == nil {
		t.Error("a file that is not a workbook should be an error")
	}
}

func TestJPXLegacyWorkbook(t *testing.T) {
	rows, err := parseJPXLegacy(jpxFixture(t, "stock_val_1_260902.xls"), JPXUnitValue)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]jpxRow{}
	for _, r := range rows {
		got[r.market+"|"+r.investor] = r
	}
	// The report week is the second week in the file: 2026/9 week 2, 9/7 - 9/11.
	r := got["TSE Prime|Individuals Cash"]
	if !r.start.Equal(dateUTC(2026, time.September, 7)) || !r.end.Equal(dateUTC(2026, time.September, 11)) {
		t.Errorf("week = %v to %v", r.start, r.end)
	}
	if r.sales != 2025904508 || r.purchases != 2102427667 || r.balance != 76523159 || r.total != 4128332175 {
		t.Errorf("TSE Prime individuals cash: %+v", r)
	}
	// A figure the source holds as text with separators, in the block that the old reader dropped.
	if r := got["TSE Standard|Business Cos."]; r.sales != 7668920 || r.purchases != 9084993 {
		t.Errorf("TSE Standard business companies: %+v", r)
	}
	// The cash and margin rows of a group add up to the block they split.
	prime := func(inv string) jpxRow { return got["TSE Prime|"+inv] }
	if prime("Individuals Cash").sales+prime("Individuals Margin").sales != 11296015753 {
		t.Error("individuals do not add up to the block")
	}
	if prime("Foreigners Institutions").sales+prime("Foreigners Individuals").sales != 28570404573 {
		t.Error("foreigners do not add up to the block")
	}
	if len(rows) != 14*4 {
		t.Errorf("rows = %d, want 4 markets x 14 groups", len(rows))
	}
	if _, err := parseJPXLegacy([]byte("not a workbook"), JPXUnitValue); err == nil {
		t.Error("a file that is not a workbook should be an error")
	}
}

func TestJPXWeekDates(t *testing.T) {
	for _, c := range []struct {
		text        string
		year, month int
		start, end  time.Time
	}{
		{"09/07～09/11", 2026, 9, dateUTC(2026, 9, 7), dateUTC(2026, 9, 11)},
		{"12/26～12/30", 2023, 1, dateUTC(2022, 12, 26), dateUTC(2022, 12, 30)}, // the previous week of a January file
		{"12/29 - 1/2", 2025, 12, dateUTC(2025, 12, 29), dateUTC(2026, 1, 2)},  // a week across the year end
		{"01/04～01/06", 2023, 1, dateUTC(2023, 1, 4), dateUTC(2023, 1, 6)},
		{"12/30～12/30", 2019, 12, dateUTC(2019, 12, 30), dateUTC(2019, 12, 30)}, // a one-day week
	} {
		start, end, ok := jpxWeekDates(c.text, c.year, c.month)
		if !ok || !start.Equal(c.start) || !end.Equal(c.end) {
			t.Errorf("%q in %d/%d: got %v to %v (%v), want %v to %v", c.text, c.year, c.month, start, end, ok, c.start, c.end)
		}
	}
	if _, _, ok := jpxWeekDates("no dates here", 2026, 9); ok {
		t.Error("text with no dates should not parse")
	}
}

func TestJPXInvestorTypes(t *testing.T) {
	client, _ := newFixtureJPX(t)
	// 2026-09-07 to 2026-10-09: one week from the .xls (report week 9/7-9/11) and one workbook.
	dt, err := client.InvestorTypes(dateUTC(2026, time.September, 7), dateUTC(2026, time.October, 9), JPXUnitValue)
	if err != nil {
		t.Fatalf("InvestorTypes error: %v", err)
	}
	rows := jpxRows(t, dt)
	if len(rows) != 2*4*14 {
		t.Fatalf("rows = %d, want two weeks of 4 markets x 14 groups", len(rows))
	}
	if !rows[0][0].(time.Time).Equal(dateUTC(2026, time.September, 7)) || !rows[len(rows)-1][0].(time.Time).Equal(dateUTC(2026, time.October, 5)) {
		t.Error("rows should run from the earliest week to the latest")
	}
	if r := jpxFind(t, rows, dateUTC(2026, time.October, 5), "TSE Prime", "Proprietary Cash"); r[4] != int64(5403540773) || r[1] != dateUTC(2026, time.October, 9) {
		t.Errorf("workbook row: %v", r)
	}
	if r := jpxFind(t, rows, dateUTC(2026, time.September, 7), "TSE Prime", "Individuals Cash"); r[5] != int64(2102427667) {
		t.Errorf("legacy row: %v", r)
	}
}

func TestJPXInvestorTypesAcrossTheYearEndAndUnits(t *testing.T) {
	client, _ := newFixtureJPX(t)
	// The 2023/1 week 1 file holds 12/26-12/30 and 1/4-1/6; only the report week is used.
	dt, err := client.InvestorTypes(dateUTC(2023, time.January, 2), dateUTC(2023, time.January, 8), JPXUnitShares)
	if err != nil {
		t.Fatalf("InvestorTypes error: %v", err)
	}
	rows := jpxRows(t, dt)
	if len(rows) != 4*14 {
		t.Fatalf("rows = %d, want one week of 4 markets x 14 groups", len(rows))
	}
	for _, r := range rows {
		if r[0] != dateUTC(2023, time.January, 4) || r[1] != dateUTC(2023, time.January, 6) {
			t.Fatalf("week = %v to %v, want 2023-01-04 to 2023-01-06", r[0], r[1])
		}
	}
	// Shares are the vol file, not the val file.
	val, err := client.InvestorTypes(dateUTC(2023, time.January, 2), dateUTC(2023, time.January, 8), JPXUnitValue)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := jpxFind(t, rows, dateUTC(2023, time.January, 4), "TSE Prime", "Individuals Cash")[4], jpxFind(t, jpxRows(t, val), dateUTC(2023, time.January, 4), "TSE Prime", "Individuals Cash")[4]; a == b {
		t.Errorf("shares and value are the same figure %v", a)
	}
}

func TestJPXInvestorTypesRefusesBadCalls(t *testing.T) {
	client, _ := newFixtureJPX(t)
	day := dateUTC(2026, time.October, 5)
	for name, f := range map[string]func() error{
		"unit":  func() error { _, e := client.InvestorTypes(day, day, JPXUnit("Yen")); return e },
		"order": func() error { _, e := client.InvestorTypes(day, day.AddDate(0, 0, -7), JPXUnitValue); return e },
		"zero":  func() error { _, e := client.InvestorTypes(time.Time{}, day, JPXUnitValue); return e },
	} {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	var zero JPXClient
	if _, err := zero.InvestorTypes(day, day, JPXUnitValue); err == nil {
		t.Error("a zero client should be an error")
	}
	// A range with no files gives an empty table.
	none, err := client.InvestorTypes(dateUTC(2026, time.November, 2), dateUTC(2026, time.November, 6), JPXUnitValue)
	if err != nil || none.NumRows() != 0 || none.GetColByName("Investor") == nil {
		t.Errorf("a range with no files: %v, %v", none, err)
	}
}
