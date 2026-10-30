package datafetch

import (
	"net/http"
	"testing"
	"time"
)

func TestFugleOwnership(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stock/ownership/etf-holdings/0050":
			_, _ = w.Write([]byte(`{"type":"EQUITY","symbol":"0050","data":[
				{"date":"2026-05-20","components":[{"symbol":"2330","name":"台積電","quantity":479641266,"weight":60.79}]},
				{"date":"2026-05-21","components":[{"symbol":"2330","name":"台積電","quantity":481180146,"weight":60.61,"quantityChange":1538880,"weightChange":-0.18},{"symbol":"2454","name":"聯發科","quantity":28975109,"weight":5.4}]}]}`))
		case "/stock/ownership/institutional-trades/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"` + r.URL.Query().Get("from") + `","foreign":{"buy":18369301,"sell":14397070,"net":3972231},"trust":{"buy":2350600,"sell":1614607,"net":735993},"dealer":{"buy":482295,"sell":389114,"net":93181},"total":4801405}]}`))
		case "/stock/ownership/director-holdings/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"2026-07","directors":[{"order":0,"title":"董事長本人","name":"魏哲家","heldShares":7452349,"pledgeRatio":0.2146}]}]}`))
		case "/stock/ownership/tdcc-distribution/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"2026-08-07","distributions":[{"range":"1-999","holders":2523071,"shares":293744991,"proportion":1.13},{"range":"合計","holders":3000000,"shares":25932000000,"proportion":100}]}]}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	})
	c := fs.client(t, FugleConfig{})

	dt, err := c.Stock.OwnershipETFHoldings("0050", FugleOwnershipOptions{From: day(2026, 5, 20), To: day(2026, 5, 21)})
	if err != nil {
		t.Fatal(err)
	}
	q := fs.requests[0].URL.Query()
	if q.Get("from") != "2026-05-20" || q.Get("to") != "2026-05-21" || q.Get("sort") != "asc" {
		t.Errorf("request %s", fs.requests[0].URL.String())
	}
	if dt.NumRows() != 3 || dt.NumCols() != len(fugleETFHoldingColumns) {
		t.Fatalf("%d rows, %d columns", dt.NumRows(), dt.NumCols())
	}
	if d, ok := dt.GetColByName("date").Data()[2].(time.Time); !ok || !d.Equal(day(2026, 5, 21)) {
		t.Errorf("date = %#v", dt.GetColByName("date").Data()[2])
	}
	if v := dt.GetColByName("quantityChange").Data()[0]; v != nil {
		t.Errorf("first day's quantityChange = %#v, want nil", v)
	}
	if v := dt.GetColByName("symbol").Data()[2]; v != "2454" {
		t.Errorf("third holding = %#v", v)
	}

	// A range of two years is cut into three requests, and Descending reverses
	// the days.
	dt, err = c.Stock.OwnershipInstitutionalTrades("2330", FugleOwnershipOptions{From: day(2024, 1, 1), To: day(2025, 12, 31), Descending: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(fs.requests) - 1; got != 3 {
		t.Fatalf("%d requests for two years, want 3", got)
	}
	for _, r := range fs.requests[1:] {
		from, _ := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, _ := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if to.Sub(from) >= 365*24*time.Hour {
			t.Errorf("a piece spans %s", to.Sub(from))
		}
	}
	dates := dt.GetColByName("date").Data()
	if len(dates) != 3 || !dates[0].(time.Time).After(dates[2].(time.Time)) {
		t.Errorf("dates %v are not newest first", dates)
	}
	if v := dt.GetColByName("foreign_net").Data()[0]; v != int64(3972231) {
		t.Errorf("foreign_net = %#v", v)
	}

	before := len(fs.requests)
	dt, err = c.Stock.OwnershipDirectorHoldings("2330", FugleOwnershipOptions{From: day(2025, 3, 15), To: day(2026, 7, 2)})
	if err != nil {
		t.Fatal(err)
	}
	months := fs.requests[before:]
	if len(months) != 2 || months[0].URL.Query().Get("from") != "2025-03" || months[0].URL.Query().Get("to") != "2026-02" ||
		months[1].URL.Query().Get("from") != "2026-03" || months[1].URL.Query().Get("to") != "2026-07" {
		for _, r := range months {
			t.Errorf("month request %s", r.URL.RawQuery)
		}
	}
	if d, ok := dt.GetColByName("date").Data()[0].(time.Time); !ok || !d.Equal(day(2026, 7, 1)) {
		t.Errorf("director date = %#v", dt.GetColByName("date").Data()[0])
	}
	if dt.GetColByName("electedShares").Data()[0] != nil || dt.GetColByName("pledgeRatio").Data()[0] != 0.2146 {
		t.Errorf("director row %v", dt.GetRow(0).Data())
	}

	before = len(fs.requests)
	dt, err = c.Stock.OwnershipTDCCDistribution("2330")
	if err != nil {
		t.Fatal(err)
	}
	if q := fs.requests[before].URL.Query(); q.Has("from") || q.Has("to") {
		t.Errorf("latest report request %s", fs.requests[before].URL.RawQuery)
	}
	if dt.NumRows() != 2 || dt.GetColByName("range").Data()[1] != "合計" {
		t.Errorf("tdcc %d rows", dt.NumRows())
	}

	dt, err = c.Stock.OwnershipETFHoldings("00632R")
	if err != nil || dt.NumRows() != 0 || dt.NumCols() != len(fugleETFHoldingColumns) {
		t.Errorf("an ETF with no stock holdings: %v, %d rows, %d columns", err, dt.NumRows(), dt.NumCols())
	}

	sent := len(fs.requests)
	bad := map[string]func() error{
		"To without From": func() error {
			_, e := c.Stock.OwnershipTDCCDistribution("2330", FugleOwnershipOptions{To: day(2025, 1, 1)})
			return e
		},
		"months reversed": func() error {
			_, e := c.Stock.OwnershipDirectorHoldings("2330", FugleOwnershipOptions{From: day(2025, 3, 1), To: day(2025, 1, 1)})
			return e
		},
		"two options": func() error {
			_, e := c.Stock.OwnershipETFHoldings("0050", FugleOwnershipOptions{}, FugleOwnershipOptions{})
			return e
		},
		"empty symbol": func() error { _, e := c.Stock.OwnershipInstitutionalTrades(""); return e },
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
