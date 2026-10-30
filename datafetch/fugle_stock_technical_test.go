package datafetch

import (
	"net/http"
	"testing"
	"time"
)

func TestFugleTechnicalIndicators(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stock/technical/sma/2330":
			_, _ = w.Write([]byte(`{"symbol":"2330","timeframe":"D","period":5,"data":[{"date":"2024-08-01","sma":940.4},{"date":"2024-08-02","sma":936.2}]}`))
		case "/stock/technical/kdj/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"2024-08-01","k":32.9,"d":27.7,"j":43.3}]}`))
		case "/stock/technical/macd/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"2024-08-01","macdLine":-8.8,"signalLine":1.1}]}`))
		case "/stock/technical/bb/2330":
			_, _ = w.Write([]byte(`{"data":[{"date":"2024-08-01","upper":1089.4,"middle":997.45,"lower":905.4}]}`))
		default:
			http.NotFound(w, r)
		}
	})
	c := fs.client(t, FugleConfig{})

	dt, err := c.Stock.TechnicalSMA("2330", FugleTimeframeDay, 5, FugleTechnicalOptions{From: day(2024, 8, 1), To: day(2024, 8, 10)})
	if err != nil {
		t.Fatal(err)
	}
	q := fs.requests[0].URL.Query()
	if q.Get("timeframe") != "D" || q.Get("period") != "5" || q.Get("from") != "2024-08-01" || q.Get("to") != "2024-08-10" {
		t.Errorf("request %s", fs.requests[0].URL.String())
	}
	if dt.NumRows() != 2 || dt.GetColByName("sma").Data()[1] != 936.2 {
		t.Errorf("sma table %v", dt.GetColByName("sma").Data())
	}
	if d, ok := dt.GetColByName("date").Data()[0].(time.Time); !ok || !d.Equal(day(2024, 8, 1)) {
		t.Errorf("date = %#v", dt.GetColByName("date").Data()[0])
	}

	if _, err := c.Stock.TechnicalKDJ("2330", FugleTimeframeWeek, 9, 3, 3); err != nil {
		t.Fatal(err)
	}
	q = fs.requests[1].URL.Query()
	if q.Get("rPeriod") != "9" || q.Get("kPeriod") != "3" || q.Get("dPeriod") != "3" || q.Has("from") {
		t.Errorf("request %s", fs.requests[1].URL.String())
	}
	dt, err = c.Stock.TechnicalMACD("2330", FugleTimeframe5Min, 12, 26, 9)
	if err != nil {
		t.Fatal(err)
	}
	q = fs.requests[2].URL.Query()
	if q.Get("timeframe") != "5" || q.Get("fast") != "12" || q.Get("slow") != "26" || q.Get("signal") != "9" {
		t.Errorf("request %s", fs.requests[2].URL.String())
	}
	if names := dt.ColNames(); len(names) != 3 || names[1] != "macdLine" || names[2] != "signalLine" {
		t.Errorf("macd columns %v", names)
	}
	dt, err = c.Stock.TechnicalBB("2330", FugleTimeframeMonth, 20)
	if err != nil || dt.GetColByName("lower") == nil {
		t.Fatalf("bb %v", err)
	}

	// An RSI with no data is an empty table with its columns.
	dt, err = c.Stock.TechnicalRSI("2330", FugleTimeframeDay, 6)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 0 || dt.GetColByName("rsi") == nil || dt.GetColByName("date") == nil {
		t.Errorf("empty rsi table %v %d", dt.ColNames(), dt.NumRows())
	}

	sent := len(fs.requests)
	bad := map[string]func() error{
		"zero period":      func() error { _, e := c.Stock.TechnicalSMA("2330", FugleTimeframeDay, 0); return e },
		"zero kPeriod":     func() error { _, e := c.Stock.TechnicalKDJ("2330", FugleTimeframeDay, 9, 0, 3); return e },
		"fast not shorter": func() error { _, e := c.Stock.TechnicalMACD("2330", FugleTimeframeDay, 26, 12, 9); return e },
		"bad timeframe":    func() error { _, e := c.Stock.TechnicalRSI("2330", "2", 6); return e },
		"From without To": func() error {
			_, e := c.Stock.TechnicalRSI("2330", FugleTimeframeDay, 6, FugleTechnicalOptions{From: day(2024, 1, 1)})
			return e
		},
		"To before From": func() error {
			_, e := c.Stock.TechnicalRSI("2330", FugleTimeframeDay, 6, FugleTechnicalOptions{From: day(2024, 2, 1), To: day(2024, 1, 1)})
			return e
		},
		"minute with range": func() error {
			_, e := c.Stock.TechnicalBB("2330", FugleTimeframe1Min, 20, FugleTechnicalOptions{From: day(2024, 1, 1), To: day(2024, 1, 5)})
			return e
		},
		"two options": func() error {
			_, e := c.Stock.TechnicalBB("2330", FugleTimeframeDay, 20, FugleTechnicalOptions{}, FugleTechnicalOptions{})
			return e
		},
		"empty symbol": func() error { _, e := c.Stock.TechnicalSMA("", FugleTimeframeDay, 5); return e },
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
