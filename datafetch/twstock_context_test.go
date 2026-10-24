package datafetch

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

// #250: every TWStockClient fetch has a Context form, and a done context stops
// the limiter wait, the retry backoff and any further request.

func TestTWStockContextMethodsStopOnACancelledContext(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	day := dateUTC(2026, time.September, 3)

	calls := map[string]func() error{
		"DailyPricesContext": func() error {
			_, err := stock.DailyPricesContext(ctx, "2330", day, day, TWMarketTWSE)
			return err
		},
		"DailyPricesAdjustedContext": func() error {
			_, err := stock.DailyPricesAdjustedContext(ctx, "2330", day, day, TWMarketTWSE)
			return err
		},
		"ExRightsContext": func() error {
			_, err := stock.ExRightsContext(ctx, day, day, TWMarketTWSE)
			return err
		},
		"InstitutionalTradesContext": func() error {
			_, err := stock.InstitutionalTradesContext(ctx, day, TWMarketTWSE)
			return err
		},
		"MarginBalanceContext": func() error {
			_, err := stock.MarginBalanceContext(ctx, day, TWMarketTWSE)
			return err
		},
		"AllDailyQuotesContext": func() error {
			_, err := stock.AllDailyQuotesContext(ctx, TWMarketTWSE)
			return err
		},
	}
	for name, call := range calls {
		if err := call(); !errors.Is(err, context.Canceled) {
			t.Errorf("%s returned %v, want context.Canceled", name, err)
		}
	}
	if n := transport.requestCount(); n != 0 {
		t.Errorf("sent %d requests under a cancelled context, want none", n)
	}
}

func TestTWStockRefusesANilContext(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	var nilCtx context.Context
	day := dateUTC(2026, time.September, 3)

	if _, err := stock.DailyPricesContext(nilCtx, "2330", day, day, TWMarketTWSE); !errors.Is(err, errNilContext) {
		t.Errorf("a nil context returned %v, want errNilContext", err)
	}
	if n := transport.requestCount(); n != 0 {
		t.Errorf("sent %d requests, want none", n)
	}
}

func TestTWStockContextStopsTheLimiterWait(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{Interval: 10 * time.Second})
	key := fixtureKey("/v1/exchangeReport/STOCK_DAY_ALL", nil)
	transport.addFixture(t, key, "twse_stock_day_all.json")
	transport.addFixture(t, key, "twse_stock_day_all.json")
	if _, err := stock.AllDailyQuotesContext(context.Background(), TWMarketTWSE); err != nil {
		t.Fatalf("first request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := stock.AllDailyQuotesContext(ctx, TWMarketTWSE)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %v; the context ended after 50ms", elapsed)
	}
	if n := transport.requestCount(); n != 1 {
		t.Errorf("sent %d requests, want only the first", n)
	}
}

func TestTWStockContextStopsTheRetryBackoff(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{Retries: 3, RetryBackoff: 10 * time.Second})
	key := fixtureKey("/v1/exchangeReport/STOCK_DAY_ALL", nil)
	for range 4 {
		transport.addStatus(key, http.StatusInternalServerError)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := stock.AllDailyQuotesContext(ctx, TWMarketTWSE)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %v; the context ended after 50ms", elapsed)
	}
	if n := transport.requestCount(); n != 1 {
		t.Errorf("sent %d requests, want only the first", n)
	}
}

// cancelAfterFirst cancels its context once the first request has been
// answered, the way a caller gives up part-way through a month range.
type cancelAfterFirst struct {
	next   http.RoundTripper
	cancel context.CancelFunc
	once   sync.Once
}

func (c *cancelAfterFirst) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.next.RoundTrip(req)
	c.once.Do(c.cancel)
	return resp, err
}

func TestTWStockContextStopsBetweenMonths(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260801", "response", "json", "stockNo", "2330")), "twse_stock_day_2330_202608.json")
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260901", "response", "json", "stockNo", "2330")), "twse_stock_day_2330_202609.json")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stock.client.Transport = &cancelAfterFirst{next: transport, cancel: cancel}

	_, err := stock.DailyPricesContext(ctx, "2330", dateUTC(2026, time.August, 15), dateUTC(2026, time.September, 3), TWMarketTWSE)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if n := transport.requestCount(); n != 1 {
		t.Errorf("sent %d requests, want only August's", n)
	}
}
