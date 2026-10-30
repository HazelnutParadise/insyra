package datafetch

import (
	"net/http"
	"testing"
	"time"
)

func TestFugleCorporateActions(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stock/corporate-actions/dividends":
			_, _ = w.Write([]byte(`{"start_date":"2025-08-26","end_date":"2026-01-08","sort":"asc","data":[
				{"date":"2026-01-06","exchange":"TWSE","symbol":"00946","name":"群益科技高息成長","previousClose":9.6,"referencePrice":9.54,"dividend":0.058,"dividendType":"息","cashDividend":0.058,"stockDividendShares":0},
				{"date":"2026-01-08","exchange":"TWSE","symbol":"2247","name":"汎德永業","previousClose":null,"referencePrice":null,"dividend":6.5,"dividendType":"息","cashDividend":6.5,"stockDividendShares":null}]}`))
		case "/stock/corporate-actions/capital-changes":
			_, _ = w.Write([]byte(`{"data":[{"symbol":"4530","name":"宏易","actionType":"capital_reduction","resumeDate":"2025-12-29","haltDate":"2025-12-18","exchange":"TPEx",
				"raw":{"reason":"彌補虧損","referencePrice":31.32,"sharesPerThousand":392.72795}}]}`))
		case "/stock/corporate-actions/listing-applicants":
			_, _ = w.Write([]byte(`{"data":[{"symbol":"7762","name":"吉晟生","exchange":"TWSE","applicationDate":"2025-09-24","listedDate":null,"capitalAtApplication":572328,"remarks":"創新板"}]}`))
		default:
			http.NotFound(w, r)
		}
	})
	c := fs.client(t, FugleConfig{})

	dt, err := c.Stock.CorporateActionsDividends(day(2025, 8, 26), FugleCorporateActionsOptions{To: day(2026, 1, 8)})
	if err != nil {
		t.Fatal(err)
	}
	q := fs.requests[0].URL.Query()
	if q.Get("start_date") != "2025-08-26" || q.Get("end_date") != "2026-01-08" || q.Get("sort") != "asc" {
		t.Errorf("request %s", fs.requests[0].URL.String())
	}
	if dt.NumRows() != 2 || dt.NumCols() != len(fugleDividendColumns) {
		t.Fatalf("%d rows, %d columns", dt.NumRows(), dt.NumCols())
	}
	if d, ok := dt.GetColByName("date").Data()[1].(time.Time); !ok || !d.Equal(day(2026, 1, 8)) {
		t.Errorf("date = %#v", dt.GetColByName("date").Data()[1])
	}
	if v := dt.GetColByName("previousClose").Data()[1]; v != nil {
		t.Errorf("an announced event's previousClose = %#v, want nil", v)
	}
	if v := dt.GetColByName("limitUpPrice"); v == nil {
		t.Error("limitUpPrice, which no row has, is missing")
	}

	dt, err = c.Stock.CorporateActionsCapitalChanges(day(2025, 1, 1), FugleCorporateActionsOptions{Descending: true})
	if err != nil {
		t.Fatal(err)
	}
	q = fs.requests[1].URL.Query()
	if q.Get("sort") != "desc" || q.Has("end_date") {
		t.Errorf("request %s", fs.requests[1].URL.String())
	}
	for name, want := range map[string]any{"raw_reason": "彌補虧損", "raw_referencePrice": 31.32, "raw_splitRatio": nil, "haltDate": day(2025, 12, 18)} {
		col := dt.GetColByName(name)
		if col == nil {
			t.Errorf("no column %q", name)
			continue
		}
		got := col.Data()[0]
		if tm, ok := want.(time.Time); ok {
			if g, ok := got.(time.Time); !ok || !g.Equal(tm) {
				t.Errorf("%s = %#v", name, got)
			}
		} else if got != want {
			t.Errorf("%s = %#v, want %#v", name, got, want)
		}
	}

	dt, err = c.Stock.CorporateActionsListingApplicants(day(2025, 1, 7))
	if err != nil {
		t.Fatal(err)
	}
	if v := dt.GetColByName("listedDate").Data()[0]; v != nil {
		t.Errorf("listedDate = %#v, want nil", v)
	}
	if _, ok := dt.GetColByName("applicationDate").Data()[0].(time.Time); !ok {
		t.Errorf("applicationDate = %#v, want a time", dt.GetColByName("applicationDate").Data()[0])
	}

	sent := len(fs.requests)
	bad := map[string]func() error{
		"zero from": func() error { _, e := c.Stock.CorporateActionsDividends(time.Time{}); return e },
		"To too early": func() error {
			_, e := c.Stock.CorporateActionsDividends(day(2025, 2, 1), FugleCorporateActionsOptions{To: day(2025, 1, 1)})
			return e
		},
		"two options": func() error {
			_, e := c.Stock.CorporateActionsListingApplicants(day(2025, 1, 1), FugleCorporateActionsOptions{}, FugleCorporateActionsOptions{})
			return e
		},
	}
	for name, f := range bad {
		if f() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if len(fs.requests) != sent {
		t.Errorf("refused calls sent %d requests", len(fs.requests)-sent)
	}
}
