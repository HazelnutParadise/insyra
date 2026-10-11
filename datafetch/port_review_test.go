package datafetch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Port review, issue 1: a *Context method checks the context at its entry, so a
// done context is reported even when the call would otherwise do no work — an
// empty batch, or one whose every row is unusable.

func TestReverseColsContextStopsOnACancelledEmptyBatch(t *testing.T) {
	g := newTestGeocoder(t, "http://example.invalid", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dt, err := g.ReverseColsContext(ctx, insyra.NewDataList(), insyra.NewDataList())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if dt != nil {
		t.Errorf("got a table, want nil")
	}
}

func TestReverseColsContextStopsOnACancelledAllInvalidBatch(t *testing.T) {
	g := newTestGeocoder(t, "http://example.invalid", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	lat := insyra.NewDataList("not-a-number")
	lng := insyra.NewDataList("also-not-a-number")
	dt, err := g.ReverseColsContext(ctx, lat, lng)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if dt != nil {
		t.Errorf("got a table, want nil")
	}
}

// A done context beats a client that never went through its constructor: the
// check is at the entry, so every *Context method reports the context first.
func TestContextMethodsCheckTheContextBeforeTheClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	day := dateUTC(2026, time.September, 3)

	var g *TWGeocodingClient
	if _, err := g.ReverseContext(ctx, 24.98, 121.45); !errors.Is(err, context.Canceled) {
		t.Errorf("geocoding ReverseContext: got %v, want context.Canceled", err)
	}

	var s *TWStockClient
	if _, err := s.AllDailyQuotesContext(ctx, TWMarketTWSE); !errors.Is(err, context.Canceled) {
		t.Errorf("TWStock AllDailyQuotesContext: got %v, want context.Canceled", err)
	}
	// The context check comes before even the input checks dev ran first.
	if _, err := s.ExRightsContext(ctx, day, day, TWMarketTPEx); !errors.Is(err, context.Canceled) {
		t.Errorf("TWStock ExRightsContext: got %v, want context.Canceled", err)
	}

	bad := (*YFinanceClient)(nil).Ticker("AAPL")
	if _, err := bad.InfoContext(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("yfinance broken-client InfoContext: got %v, want context.Canceled", err)
	}
	var tk YFTicker
	if _, err := tk.InfoContext(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("yfinance zero-ticker InfoContext: got %v, want context.Canceled", err)
	}

	logged := captureGmapsWarnings(t)
	var c *GoogleMapsStoresClient
	if got := c.SearchContext(ctx, "x"); got != nil {
		t.Errorf("Google Maps SearchContext returned %v, want nil", got)
	}
	if !strings.Contains(logged.String(), "context canceled") {
		t.Errorf("Google Maps warning %q does not report the context first", logged.String())
	}
}

// Port review, issue 2: on a path dev did not panic on, the port reproduces
// dev's value and error text exactly. Only the paths dev panicked on gain our
// usable() error.

func TestNilGeocoderKeepsDevInputChecks(t *testing.T) {
	var g *TWGeocodingClient

	// dev returned an empty table for two empty lists, never touching the client.
	dt, err := g.ReverseCols(insyra.NewDataList(), insyra.NewDataList())
	if err != nil {
		t.Errorf("empty lists: got %v, want nil", err)
	} else if dt == nil || dt.NumRows() != 0 {
		t.Errorf("empty lists: got %v, want an empty table", dt)
	}

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"ReverseCols nil list", func() error { _, err := g.ReverseCols(nil, insyra.NewDataList(1.0)); return err },
			"datafetch: ReverseCols requires non-nil lat and lng lists"},
		{"ReverseCols length mismatch", func() error { _, err := g.ReverseCols(insyra.NewDataList(1.0), insyra.NewDataList()); return err },
			"datafetch: ReverseCols lat/lng length mismatch (1 vs 0)"},
		{"ReverseTable nil table", func() error { _, err := g.ReverseTable(nil, "A", "B"); return err },
			"datafetch: ReverseTable requires a non-nil DataTable"},
		{"ReverseTableByColName nil table", func() error { _, err := g.ReverseTableByColName(nil, "lat", "lng"); return err },
			"datafetch: ReverseTableByColName requires a non-nil DataTable"},
	}
	for _, c := range cases {
		if err := c.call(); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}

	// A well-formed, non-empty batch would have panicked in dev: our usable() error.
	if _, err := g.ReverseCols(insyra.NewDataList(24.98), insyra.NewDataList(121.45)); !errors.Is(err, errTWGeocodingNotCreated) {
		t.Errorf("valid batch: got %v, want errTWGeocodingNotCreated", err)
	}
}

func TestNilTWStockKeepsDevInputChecks(t *testing.T) {
	var s *TWStockClient
	day := dateUTC(2026, time.September, 3)
	next := day.AddDate(0, 0, 1)

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"DailyPrices empty code", func() error { _, err := s.DailyPrices("", day, day, TWMarketTWSE); return err },
			"datafetch: DailyPrices requires a non-empty code"},
		{"DailyPrices reversed range", func() error { _, err := s.DailyPrices("2330", next, day, TWMarketTWSE); return err },
			"datafetch: DailyPrices from must not be after to"},
		{"DailyPrices bad market", func() error { _, err := s.DailyPrices("2330", day, day, "BAD"); return err },
			`datafetch: unsupported market "BAD"`},
		{"DailyPricesAdjusted TPEx", func() error { _, err := s.DailyPricesAdjusted("2330", day, day, TWMarketTPEx); return err },
			errTPExExRightsUnsupported.Error()},
		{"DailyPricesAdjusted empty code", func() error { _, err := s.DailyPricesAdjusted("", day, day, TWMarketTWSE); return err },
			"datafetch: DailyPricesAdjusted requires a non-empty code"},
		{"ExRights bad market", func() error { _, err := s.ExRights(day, day, "BAD"); return err },
			`datafetch: unsupported market "BAD"`},
		{"ExRights reversed range", func() error { _, err := s.ExRights(next, day, TWMarketTWSE); return err },
			"datafetch: ExRights from must not be after to"},
		{"ExRights TPEx", func() error { _, err := s.ExRights(day, day, TWMarketTPEx); return err },
			errTPExExRightsUnsupported.Error()},
		{"InstitutionalTrades bad market", func() error { _, err := s.InstitutionalTrades(day, "BAD"); return err },
			`datafetch: unsupported market "BAD"`},
		{"MarginBalance bad market", func() error { _, err := s.MarginBalance(day, "BAD"); return err },
			`datafetch: unsupported market "BAD"`},
		{"AllDailyQuotes bad market", func() error { _, err := s.AllDailyQuotes("BAD"); return err },
			`datafetch: unsupported market "BAD"`},
	}
	for _, c := range cases {
		if err := c.call(); err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}

	// Well-formed calls would have panicked in dev: our usable() error.
	if _, err := s.DailyPrices("2330", day, day, TWMarketTWSE); !errors.Is(err, errTWStockNotCreated) {
		t.Errorf("DailyPrices valid args: got %v, want errTWStockNotCreated", err)
	}
	if _, err := s.ExRights(day, day, TWMarketTWSE); !errors.Is(err, errTWStockNotCreated) {
		t.Errorf("ExRights valid args: got %v, want errTWStockNotCreated", err)
	}
	if _, err := s.AllDailyQuotes(TWMarketTWSE); !errors.Is(err, errTWStockNotCreated) {
		t.Errorf("AllDailyQuotes valid market: got %v, want errTWStockNotCreated", err)
	}
}

func TestNilYFinanceKeepsDevErrorTexts(t *testing.T) {
	// dev: Ticker on a nil client hands back a ticker carrying this error.
	if _, err := (*YFinanceClient)(nil).Ticker("AAPL").Info(); err == nil ||
		err.Error() != "yfinance: yahooFinance is nil" {
		t.Errorf("nil client: got %v, want \"yfinance: yahooFinance is nil\"", err)
	}
	// dev: a zero-value ticker names the missing client.
	var tk YFTicker
	if _, err := tk.Info(); err == nil || err.Error() != "yfinance: yahooFinance is nil" {
		t.Errorf("zero ticker: got %v, want \"yfinance: yahooFinance is nil\"", err)
	}
	// dev: a zero-value client reaches checkError and names the missing HTTP client.
	var y YFinanceClient
	if _, err := y.Ticker("AAPL").Info(); err == nil || err.Error() != "yfinance: client is nil" {
		t.Errorf("zero client: got %v, want \"yfinance: client is nil\"", err)
	}
}
