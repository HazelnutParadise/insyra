package datafetch

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFugleFutOptProductsAndTickersSendTheirSessionInUppercase(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(r.URL.Path, "/futopt/intraday/products"):
			if q.Get("type") != "FUTURE" || q.Get("session") != "AFTERHOURS" || q.Get("contractType") != "I" || q.Get("status") != "N" || q.Get("exchange") != "TAIFEX" {
				t.Errorf("products query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"symbol":"TXF","type":"FUTURE_AH","name":"臺股期貨","contractSize":200,"quoteAcceptable":true,"startDate":""}]}`))
		case strings.HasSuffix(r.URL.Path, "/futopt/intraday/tickers"):
			if q.Get("type") != "OPTION" || q.Get("product") != "TXO" || q.Get("session") != "" || q.Get("isSpread") != "" {
				t.Errorf("tickers query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[
				{"type":"OPTION","symbol":"TXO23000J6","name":"臺指選擇權","limitUpPrice":1000,"priceLimitLevels":[{"level":1,"limitUpPrice":900,"limitDownPrice":10},{"level":2,"limitUpPrice":1000,"limitDownPrice":5}],"limitUpLevel":2},
				{"type":"OPTION","symbol":"TXO23100J6","name":"臺指選擇權"}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	c := fs.client(t, FugleConfig{})
	products, err := c.FutOpt.IntradayProducts(FugleFuture, FugleFutOptProductsOptions{Exchange: "TAIFEX", AfterHours: true, ContractType: "I", Status: "N"})
	if err != nil || products.NumRows() != 1 {
		t.Fatalf("products: %v", err)
	}
	if got := products.GetColByName("contractSize").Data()[0]; got != int64(200) {
		t.Errorf("contractSize = %#v", got)
	}
	tickers, err := c.FutOpt.IntradayTickers(FugleOption, FugleFutOptTickersOptions{Product: "TXO"})
	if err != nil || tickers.NumRows() != 2 {
		t.Fatalf("tickers: %v", err)
	}
	if col := tickers.GetColByName("priceLimitLevels_2_limitUpPrice"); col == nil || col.Data()[0] != int64(1000) || col.Data()[1] != nil {
		t.Errorf("priceLimitLevels_2_limitUpPrice %v", col)
	}
	if _, err := c.FutOpt.IntradayProducts(""); err == nil {
		t.Error("a missing type was accepted")
	}
}

func TestFugleFutOptSingleSymbolEndpointsSendTheirSessionInLowercase(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("session") != "afterhours" {
			t.Errorf("%s: session = %q, want afterhours", r.URL.Path, q.Get("session"))
		}
		switch {
		case strings.Contains(r.URL.Path, "/futopt/intraday/ticker/"):
			_, _ = w.Write([]byte(`{"date":"2026-09-09","type":"FUTURE_AH","symbol":"TXFI6","referencePrice":46955,"priceLimitLevels":[{"level":1,"limitUpPrice":1,"limitDownPrice":2}]}`))
		case strings.Contains(r.URL.Path, "/futopt/intraday/quote/"):
			_, _ = w.Write([]byte(`{"symbol":"TXFI6","lastPrice":46980,"total":{"tradeVolume":900,"totalBidMatch":400,"totalAskMatch":500},"lastTrade":{"price":46980,"size":2},"serial":null,"lastUpdated":1}`))
		case strings.Contains(r.URL.Path, "/futopt/intraday/candles/"):
			if q.Get("timeframe") != "5" {
				t.Errorf("candles query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"date":"2026-09-09T15:00:00.000+08:00","open":1,"high":2,"low":1,"close":2,"volume":10,"average":1.5}]}`))
		case strings.Contains(r.URL.Path, "/futopt/intraday/trades/"):
			if q.Get("limit") != "5" || q.Get("offset") != "1" || q.Get("isTrial") != "true" || q.Get("sort") != "" {
				t.Errorf("trades query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"price":46980,"size":2,"time":1,"serial":1}]}`))
		case strings.Contains(r.URL.Path, "/futopt/intraday/volumes/"):
			_, _ = w.Write([]byte(`{"data":[{"price":46980,"volume":2}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	c := fs.client(t, FugleConfig{})
	if tk, err := c.FutOpt.IntradayTicker("TXFI6", true); err != nil || tk.NumRows() != 1 || tk.GetColByName("priceLimitLevels_1_level") == nil {
		t.Fatalf("ticker: %v", err)
	}
	q, err := c.FutOpt.IntradayQuote("TXFI6", true)
	if err != nil || q.NumRows() != 1 {
		t.Fatalf("quote: %v", err)
	}
	if got := q.GetColByName("total_totalBidMatch").Data()[0]; got != int64(400) {
		t.Errorf("total_totalBidMatch = %#v", got)
	}
	if got := q.GetColByName("serial").Data()[0]; got != nil {
		t.Errorf("serial = %#v, want nil", got)
	}
	bars, err := c.FutOpt.IntradayCandles("TXFI6", FugleFutOptIntradayCandlesOptions{Timeframe: FugleTimeframe5Min, AfterHours: true})
	if err != nil || bars.NumRows() != 1 {
		t.Fatalf("candles: %v", err)
	}
	if _, ok := bars.GetColByName("date").Data()[0].(time.Time); !ok {
		t.Errorf("date is %T", bars.GetColByName("date").Data()[0])
	}
	trades, err := c.FutOpt.IntradayTrades("TXFI6", FugleFutOptIntradayTradesOptions{AfterHours: true, Limit: 5, Offset: 1, Trial: true})
	if err != nil || trades.NumRows() != 1 {
		t.Fatalf("trades: %v", err)
	}
	if col := trades.GetColByName("bid"); col == nil || col.Data()[0] != nil {
		t.Errorf("bid column %v", col)
	}
	if v, err := c.FutOpt.IntradayVolumes("TXFI6", true); err != nil || v.NumRows() != 1 {
		t.Fatalf("volumes: %v", err)
	}
}

func TestFugleFutOptSpreadSymbolsKeepTheirSlash(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"symbol":"TXFJ6/K6"}`)) })
	c := fs.client(t, FugleConfig{})
	if _, err := c.FutOpt.IntradayQuote("TXFJ6/K6"); err != nil {
		t.Fatal(err)
	}
	if got := fs.requests[0].URL.Path; got != "/futopt/intraday/quote/TXFJ6/K6" {
		t.Errorf("path %s", got)
	}
	for _, bad := range []string{"", "/", "A/", "/B", "A/B/C", "A B", "A?B", "A#B"} {
		if _, err := c.FutOpt.IntradayQuote(bad); err == nil {
			t.Errorf("symbol %q was accepted", bad)
		}
	}
	// A stock symbol still cannot carry a slash.
	if _, err := c.Stock.IntradayQuote("2330/1"); err == nil {
		t.Error("a stock symbol with a slash was accepted")
	}
}

func TestFugleFutOptRefusesBadInputBeforeSendingAnything(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) })
	c := fs.client(t, FugleConfig{})
	for name, call := range map[string]func() error{
		"3 minute candles": func() error {
			_, e := c.FutOpt.IntradayCandles("TXFI6", FugleFutOptIntradayCandlesOptions{Timeframe: FugleTimeframe3Min})
			return e
		},
		"day candles": func() error {
			_, e := c.FutOpt.IntradayCandles("TXFI6", FugleFutOptIntradayCandlesOptions{Timeframe: FugleTimeframeDay})
			return e
		},
		"negative limit": func() error {
			_, e := c.FutOpt.IntradayTrades("TXFI6", FugleFutOptIntradayTradesOptions{Limit: -1})
			return e
		},
		"two options": func() error {
			_, e := c.FutOpt.IntradayTickers(FugleFuture, FugleFutOptTickersOptions{}, FugleFutOptTickersOptions{})
			return e
		},
		"two flags":   func() error { _, e := c.FutOpt.IntradayQuote("TXFI6", true, false); return e },
		"zero client": func() error { _, e := (&FugleClient{}).FutOpt.IntradayQuote("TXFI6"); return e },
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(fs.requests) != 0 {
		t.Errorf("%d requests were sent for refused input", len(fs.requests))
	}
}

func TestFugleFutOptHistoricalCandlesSendTheirQueryAndKeepTheContractMonth(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/futopt/historical/candles/TXO" || q.Get("contractMonth") != "202610" || q.Get("strikePrice") != "23000" ||
			q.Get("callPut") != "CALL" || q.Get("timeframe") != "D" || q.Get("session") != "afterhours" ||
			q.Get("sort") != "asc" || q.Get("fields") != "open,close,volume,change,transaction" ||
			q.Get("from") != "2026-10-01" || q.Get("to") != "2026-10-09" {
			t.Errorf("request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"product":"TXO","data":[
			{"date":"2026-10-02","contractMonth":"202610","open":null,"close":null,"volume":0,"change":null},
			{"date":"2026-10-01","contractMonth":"202610","open":120.5,"close":125,"volume":1500,"change":4.5}]}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.FutOpt.HistoricalCandles("TXO", FugleFutOptHistoricalCandlesOptions{
		ContractMonth: "202610", Strike: 23000, CallPut: FugleCall, From: day(2026, 10, 1), To: day(2026, 10, 9),
		AfterHours: true, Fields: []string{"open", "close", "volume", "change", "transaction"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 2 {
		t.Fatalf("%d rows", dt.NumRows())
	}
	// Ascending dates, a day with no trade has nil prices and volume 0, and the contract month stays.
	if d := dt.GetColByName("date").Data(); !d[0].(time.Time).Equal(day(2026, 10, 1)) {
		t.Errorf("dates %v", d)
	}
	if got := dt.GetColByName("open").Data(); got[0] != float64(120.5) || got[1] != nil {
		t.Errorf("open %v", got)
	}
	if got := dt.GetColByName("contractMonth").Data(); got[0] != "202610" {
		t.Errorf("contractMonth %v", got)
	}
}

func TestFugleFutOptHistoricalCandlesCutLongRangesAndKeepRowsOfDifferentContracts(t *testing.T) {
	var n int
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		from := r.URL.Query().Get("from")
		// One date appears for two contract months, as on a roll day of a continuous contract.
		_, _ = w.Write([]byte(`{"data":[{"date":"` + from + `","contractMonth":"202610","close":1},{"date":"` + from + `","contractMonth":"202611","close":2}]}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.FutOpt.HistoricalCandles("TXF", FugleFutOptHistoricalCandlesOptions{From: day(2022, 1, 1), To: day(2024, 6, 30)})
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 || dt.NumRows() != 6 {
		t.Errorf("%d requests and %d rows, want 3 and 6: two contracts on one date are two rows", n, dt.NumRows())
	}
	if q := fs.requests[0].URL.Query(); q.Get("contractMonth") != "" || q.Get("timeframe") != "D" {
		t.Errorf("first request %s", fs.requests[0].URL.RawQuery)
	}
}

func TestFugleFutOptHistoricalDailyIsOneRowPerContract(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/futopt/historical/daily/TXO" || q.Get("date") != "2026-10-02" || q.Get("session") != "afterhours" || q.Get("contractMonth") != "202610" {
			t.Errorf("request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"date":"2026-10-02","product":"TXO","exchange":"TAIFEX","session":"AFTERHOURS","contractMonth":"202610","data":[
			{"contractMonth":"202610","strikePrice":23000,"callPut":"CALL","exchange":"TAIFEX","openPrice":100,"closePrice":110,"change":10,"volume":2500,"openInterest":null,"settlementPrice":null},
			{"contractMonth":"202610","strikePrice":23000,"callPut":"PUT","exchange":"TAIFEX","openPrice":null,"closePrice":null,"change":null,"volume":0}]}`))
	})
	c := fs.client(t, FugleConfig{})
	dt, err := c.FutOpt.HistoricalDaily("TXO", FugleFutOptHistoricalDailyOptions{Date: day(2026, 10, 2), AfterHours: true, ContractMonth: "202610"})
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 2 {
		t.Fatalf("%d rows", dt.NumRows())
	}
	if d := dt.GetColByName("date").Data()[0]; !d.(time.Time).Equal(day(2026, 10, 2)) {
		t.Errorf("date %v", d)
	}
	for name, want := range map[string]any{"callPut": "CALL", "strikePrice": int64(23000), "volume": int64(2500)} {
		if got := dt.GetColByName(name).Data()[0]; got != want {
			t.Errorf("%s = %#v, want %#v", name, got, want)
		}
	}
	// A column the API leaves out of every row is still there; a missing price is nil.
	if col := dt.GetColByName("volumeSpread"); col == nil || col.Data()[0] != nil {
		t.Errorf("volumeSpread %v", col)
	}
	if got := dt.GetColByName("openPrice").Data()[1]; got != nil {
		t.Errorf("openPrice = %#v", got)
	}
}

func TestFugleFutOptHistoricalNoDataIsAnEmptyTable(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, `{}`, http.StatusNotFound) })
	c := fs.client(t, FugleConfig{})
	if dt, err := c.FutOpt.HistoricalDaily("TXO"); err != nil || dt.NumRows() != 0 {
		t.Errorf("daily: rows and error %v", err)
	}
	if dt, err := c.FutOpt.HistoricalCandles("TXF"); err != nil || dt.NumRows() != 0 {
		t.Errorf("candles: rows and error %v", err)
	}
}

func TestFugleFutOptHistoricalRefusesBadInputBeforeSendingAnything(t *testing.T) {
	fs := newFugleServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"data":[]}`)) })
	c := fs.client(t, FugleConfig{})
	opt := func(o FugleFutOptHistoricalCandlesOptions) func() error {
		return func() error { _, e := c.FutOpt.HistoricalCandles("TXO", o); return e }
	}
	for name, call := range map[string]func() error{
		"empty product":       func() error { _, e := c.FutOpt.HistoricalCandles(""); return e },
		"spread as product":   func() error { _, e := c.FutOpt.HistoricalCandles("TXFJ6/K6"); return e },
		"strike without side": opt(FugleFutOptHistoricalCandlesOptions{Strike: 23000}),
		"side without strike": opt(FugleFutOptHistoricalCandlesOptions{CallPut: FugleCall}),
		"bad side":            opt(FugleFutOptHistoricalCandlesOptions{Strike: 1, CallPut: "BOTH"}),
		"negative strike":     opt(FugleFutOptHistoricalCandlesOptions{Strike: -1, CallPut: FugleCall}),
		"3 minute candles":    opt(FugleFutOptHistoricalCandlesOptions{Timeframe: FugleTimeframe3Min}),
		"turnover":            opt(FugleFutOptHistoricalCandlesOptions{Fields: []string{"turnover"}}),
		"month over a year":   opt(FugleFutOptHistoricalCandlesOptions{Timeframe: FugleTimeframeMonth, From: day(2020, 1, 1), To: day(2024, 1, 1)}),
		"to without from":     opt(FugleFutOptHistoricalCandlesOptions{To: day(2024, 1, 1)}),
		"two candle options": func() error {
			_, e := c.FutOpt.HistoricalCandles("TXF", FugleFutOptHistoricalCandlesOptions{}, FugleFutOptHistoricalCandlesOptions{})
			return e
		},
		"daily empty product":  func() error { _, e := c.FutOpt.HistoricalDaily(""); return e },
		"daily spread product": func() error { _, e := c.FutOpt.HistoricalDaily("TXFJ6/K6"); return e },
		"two daily options": func() error {
			_, e := c.FutOpt.HistoricalDaily("TXO", FugleFutOptHistoricalDailyOptions{}, FugleFutOptHistoricalDailyOptions{})
			return e
		},
		"zero client": func() error { _, e := (&FugleClient{}).FutOpt.HistoricalDaily("TXO"); return e },
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(fs.requests) != 0 {
		t.Errorf("%d requests were sent for refused input", len(fs.requests))
	}
}
