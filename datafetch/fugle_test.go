package datafetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fugleServer answers every request with handler and records them.
type fugleServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []*http.Request
}

func newFugleServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *fugleServer {
	t.Helper()
	fs := &fugleServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.mu.Lock()
		fs.requests = append(fs.requests, r)
		fs.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *fugleServer) client(t *testing.T, cfg FugleConfig) *FugleClient {
	t.Helper()
	cfg.BaseURL = fs.URL
	if cfg.Interval == 0 {
		cfg.Interval = -1 // no throttling in tests
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = time.Millisecond
	}
	c, err := Fugle(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestFugleSendsTheAPIKeyAndReadsDailyCandles(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"2330","timeframe":"D","data":[
			{"date":"2024-01-03","open":590,"high":594,"low":588,"close":593,"volume":23000000,"change":3},
			{"date":"2024-01-02","open":585,"high":589,"low":583,"close":590,"volume":21000000,"change":5}],"sort":"asc"}`))
	})
	c := fs.client(t, FugleConfig{APIKey: "key=="})
	dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2024, 1, 1), To: day(2024, 1, 31), Adjusted: true, Fields: []string{"open", "close", "volume", "change"}})
	if err != nil {
		t.Fatal(err)
	}
	r := fs.requests[0]
	if r.Header.Get("X-API-KEY") != "key==" {
		t.Errorf("X-API-KEY = %q", r.Header.Get("X-API-KEY"))
	}
	q := r.URL.Query()
	if r.URL.Path != "/stock/historical/candles/2330" || q.Get("from") != "2024-01-01" || q.Get("to") != "2024-01-31" ||
		q.Get("timeframe") != "D" || q.Get("adjusted") != "true" || q.Get("sort") != "asc" || q.Get("fields") != "open,close,volume,change" {
		t.Errorf("request %s", r.URL.String())
	}
	if dt.NumRows() != 2 {
		t.Fatalf("%d rows", dt.NumRows())
	}
	// Rows come back in ascending date order whatever the server sent.
	dates := dt.GetColByName("date").Data()
	if !dates[0].(time.Time).Equal(day(2024, 1, 2)) || !dates[1].(time.Time).Equal(day(2024, 1, 3)) {
		t.Errorf("dates %v", dates)
	}
	if v := dt.GetColByName("volume").Data()[0]; v != int64(21000000) {
		t.Errorf("volume = %#v", v)
	}
	if v := dt.GetColByName("close").Data()[0]; v != float64(590) && v != int64(590) {
		t.Errorf("close = %#v", v)
	}
}

func TestFugleCandlesCutARangeOverAYearIntoPieces(t *testing.T) {
	var calls atomic.Int32
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		from, _ := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, _ := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if !to.Before(from.AddDate(1, 0, 0)) {
			http.Error(w, `{"message":"range"}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"date":"` + from.Format("2006-01-02") + `","open":1,"high":1,"low":1,"close":1,"volume":1}]}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2022, 1, 1), To: day(2024, 6, 30)})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Errorf("%d requests, want 3 pieces of at most 364 days", calls.Load())
	}
	dates := dt.GetColByName("date").Data()
	for i := 1; i < len(dates); i++ {
		if !dates[i].(time.Time).After(dates[i-1].(time.Time)) {
			t.Errorf("dates not ascending: %v", dates)
		}
	}
	// Each piece starts the day after the previous one ended.
	if fs.requests[1].URL.Query().Get("from") != "2022-12-31" {
		t.Errorf("second piece starts %s", fs.requests[1].URL.Query().Get("from"))
	}
}

func TestFugleCandlesTreatA404AsNoDataInThatPiece(t *testing.T) {
	var n atomic.Int32
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			http.Error(w, `{"message":"Resource Not Found"}`, http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"date":"` + r.URL.Query().Get("from") + `","open":1,"high":1,"low":1,"close":1,"volume":1}]}`))
	})
	c := fs.client(t, FugleConfig{Retries: 3})
	dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2022, 1, 1), To: day(2024, 6, 30)})
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 2 {
		t.Errorf("%d rows, want the two pieces that had data", dt.NumRows())
	}
	if got := int(n.Load()); got != 3 {
		t.Errorf("%d requests: a 404 must not be retried", got)
	}
}

func TestFugleCandlesWithNoDataAnywhereAreAnEmptyTable(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, `{}`, http.StatusNotFound) })
	c := fs.client(t, FugleConfig{})
	dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2024, 1, 1), To: day(2024, 1, 5)})
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 0 {
		t.Errorf("%d rows", dt.NumRows())
	}
}

func TestFugleWeeklyAndMonthlyCandlesAreNotCut(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) })
	c := fs.client(t, FugleConfig{})
	if _, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2020, 1, 1), To: day(2024, 1, 1), Timeframe: FugleTimeframeMonth}); err == nil {
		t.Error("a monthly range over a year was accepted")
	}
	if len(fs.requests) != 0 {
		t.Errorf("%d requests were sent for a refused range", len(fs.requests))
	}
	if _, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2023, 6, 1), To: day(2024, 1, 1), Timeframe: FugleTimeframeWeek}); err != nil {
		t.Errorf("a weekly range under a year: %v", err)
	}
	if len(fs.requests) != 1 {
		t.Errorf("%d requests, want one", len(fs.requests))
	}
}

func TestFugleMinuteCandlesParseTheirTimestamps(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"date":"2024-01-02T09:05:00.000+08:00","open":590,"high":591,"low":589,"close":590.5,"volume":1200,"average":590.2},
			{"date":"2024-01-02T09:00:00.000+08:00","open":589,"high":591,"low":588,"close":590,"volume":3000,"average":589.8}]}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2024, 1, 2), To: day(2024, 1, 2), Timeframe: FugleTimeframe5Min})
	if err != nil {
		t.Fatal(err)
	}
	first := dt.GetColByName("date").Data()[0].(time.Time)
	if first.UTC().Hour() != 1 || first.UTC().Minute() != 0 {
		t.Errorf("first candle at %v, want 09:00+08:00", first)
	}
	if dt.GetColByName("average") == nil {
		t.Error("no average column")
	}
}

func TestFugleErrorsAreTypedAndOnlyThrottlingIsRetried(t *testing.T) {
	for _, tc := range []struct {
		status    int
		want      error
		wantCalls int32
	}{
		{http.StatusUnauthorized, ErrFugleAuth, 1},
		{http.StatusForbidden, ErrFuglePlan, 1},
		{http.StatusTooManyRequests, ErrFugleThrottled, 3},
	} {
		var calls atomic.Int32
		fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Error(w, `{"statusCode":`+http.StatusText(tc.status)+`,"message":"nope"}`, tc.status)
		})
		c := fs.client(t, FugleConfig{Retries: 2})
		_, err := c.Stock.IntradayQuote("2330")
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: error %v, want %v", tc.status, err, tc.want)
		}
		if calls.Load() != tc.wantCalls {
			t.Errorf("status %d: %d requests, want %d", tc.status, calls.Load(), tc.wantCalls)
		}
	}
}

func TestFugleQuoteIsOneRowWithNestedFieldsFlattened(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "oddlot" {
			t.Errorf("type = %q", r.URL.Query().Get("type"))
		}
		_, _ = w.Write([]byte(`{"date":"2024-01-02","symbol":"2330","name":"台積電","lastPrice":590,
			"total":{"tradeVolume":23000,"tradeValue":13000000000},
			"bids":[{"price":589,"size":10},{"price":588,"size":20}],
			"asks":[{"price":590,"size":5}],"isClose":false}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.Stock.IntradayQuote("2330", true)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 1 {
		t.Fatalf("%d rows", dt.NumRows())
	}
	for name, want := range map[string]any{"symbol": "2330", "total_tradeVolume": int64(23000), "bids_1_price": int64(589), "bids_2_size": int64(20), "asks_1_price": int64(590), "isClose": false} {
		col := dt.GetColByName(name)
		if col == nil {
			t.Errorf("no column %q", name)
			continue
		}
		if got := col.Data()[0]; got != want {
			t.Errorf("%s = %#v, want %#v", name, got, want)
		}
	}
	if _, err := c.Stock.IntradayQuote("2330", true, false); err == nil {
		t.Error("two odd-lot flags were accepted")
	}
}

func TestFugleTickersTradesVolumesAndIntradayCandles(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/intraday/tickers"):
			q := r.URL.Query()
			if q.Get("type") != "EQUITY" || q.Get("exchange") != "TWSE" || q.Get("isNormal") != "true" {
				t.Errorf("tickers query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"symbol":"0050","name":"元大台灣50"},{"symbol":"2330","name":"台積電","industry":"24"}]}`))
		case strings.Contains(r.URL.Path, "/intraday/trades/"):
			q := r.URL.Query()
			if q.Get("limit") != "5" || q.Get("offset") != "2" || q.Get("sort") != "asc" || q.Get("isTrial") != "true" {
				t.Errorf("trades query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"price":590,"size":3,"time":1704157800000000,"serial":1},{"bid":589,"ask":590,"price":590,"size":1,"volume":4,"time":1704157801000000,"serial":2}]}`))
		case strings.Contains(r.URL.Path, "/intraday/volumes/"):
			_, _ = w.Write([]byte(`{"data":[{"price":590,"volume":4000,"volumeAtBid":1500,"volumeAtAsk":2400}]}`))
		case strings.Contains(r.URL.Path, "/intraday/candles/"):
			if r.URL.Query().Get("timeframe") != "5" {
				t.Errorf("candles query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"date":"2024-01-02T09:00:00.000+08:00","open":589,"high":591,"low":588,"close":590,"volume":3000,"average":589.8}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	c := fs.client(t, FugleConfig{})

	tickers, err := c.Stock.IntradayTickers(FugleIntradayTickersParams{Type: FugleTickerEquity, Exchange: "TWSE", IsNormal: true})
	if err != nil || tickers.NumRows() != 2 {
		t.Fatalf("tickers: %v rows=%d", err, tickers.NumRows())
	}
	if got := tickers.GetColByName("industry").Data(); got[0] != nil || got[1] != "24" {
		t.Errorf("industry column %v: a ticker without one must be nil", got)
	}
	if _, err := c.Stock.IntradayTickers(FugleIntradayTickersParams{}); err == nil {
		t.Error("tickers without a type were accepted")
	}

	trades, err := c.Stock.IntradayTrades("2330", FugleIntradayTradesParams{Limit: 5, Offset: 2, Ascending: true, Trial: true})
	if err != nil || trades.NumRows() != 2 {
		t.Fatalf("trades: %v", err)
	}
	if got := trades.GetColByName("bid").Data(); got[0] != nil || got[1] != int64(589) {
		t.Errorf("bid column %v", got)
	}

	volumes, err := c.Stock.IntradayVolumes("2330")
	if err != nil || volumes.NumRows() != 1 || volumes.GetColByName("volumeAtBid").Data()[0] != int64(1500) {
		t.Fatalf("volumes: %v", err)
	}

	bars, err := c.Stock.IntradayCandles("2330", FugleIntradayCandlesParams{Timeframe: FugleTimeframe5Min})
	if err != nil || bars.NumRows() != 1 {
		t.Fatalf("intraday candles: %v", err)
	}
	if _, ok := bars.GetColByName("date").Data()[0].(time.Time); !ok {
		t.Errorf("date is %T", bars.GetColByName("date").Data()[0])
	}
}

func TestFugleRefusesBadInputBeforeSendingAnything(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) })
	c := fs.client(t, FugleConfig{})
	for name, call := range map[string]func() error{
		"empty symbol": func() error { _, e := c.Stock.IntradayQuote(""); return e },
		"bad timeframe": func() error {
			_, e := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{Timeframe: "2"})
			return e
		},
		"bad field": func() error {
			_, e := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{Fields: []string{"nope"}})
			return e
		},
		"from after to": func() error {
			_, e := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{From: day(2024, 2, 1), To: day(2024, 1, 1)})
			return e
		},
		"to without from": func() error {
			_, e := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{To: day(2024, 1, 1)})
			return e
		},
		"adjusted minute": func() error {
			_, e := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesParams{Timeframe: FugleTimeframe5Min, Adjusted: true})
			return e
		},
		"daily as intraday": func() error {
			_, e := c.Stock.IntradayCandles("2330", FugleIntradayCandlesParams{Timeframe: FugleTimeframeDay})
			return e
		},
		"negative limit": func() error { _, e := c.Stock.IntradayTrades("2330", FugleIntradayTradesParams{Limit: -1}); return e },
		"zero client":    func() error { _, e := (&FugleClient{}).Stock.IntradayQuote("2330"); return e },
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(fs.requests) != 0 {
		t.Errorf("%d requests were sent for refused input", len(fs.requests))
	}
	if _, err := Fugle(FugleConfig{Retries: -1}); err == nil {
		t.Error("negative Retries accepted")
	}
}

func TestFugleContextCancellationStopsTheCall(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	c := fs.client(t, FugleConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Stock.IntradayQuoteContext(ctx, "2330"); !errors.Is(err, context.Canceled) {
		t.Errorf("error %v, want context.Canceled", err)
	}
	if len(fs.requests) != 0 {
		t.Errorf("%d requests after cancel", len(fs.requests))
	}
}

func TestFugleThrottlingWaitsAsLongAsRetryAfterAsks(t *testing.T) {
	var calls atomic.Int32
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{}`, http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"symbol":"2330"}`))
	})
	c := fs.client(t, FugleConfig{Retries: 1})
	start := time.Now()
	if _, err := c.Stock.IntradayQuote("2330"); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("retried after %v, want at least the 1s Retry-After asked", elapsed)
	}
	if calls.Load() != 2 {
		t.Errorf("%d requests, want 2", calls.Load())
	}
}

func TestFugleDefaultIntervalStaysUnderTheLimit(t *testing.T) {
	if defaultFugleInterval < 1020*time.Millisecond {
		t.Errorf("default interval %v lets 61 requests through in a minute", defaultFugleInterval)
	}
}
