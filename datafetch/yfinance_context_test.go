package datafetch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// #250: every YFTicker fetch has a Context form. go-yfinance cannot cancel a
// request it has sent, so a done context abandons the request rather than
// interrupting it. None of these tests reaches the network: each call stops
// before its request is sent, or runs a stand-in for go-yfinance.

func TestRunYFReturnsWhenTheContextEnds(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(50*time.Millisecond, cancel)

	start := time.Now()
	_, err := runYF(ctx, func() (int, error) {
		<-block
		return 1, nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("returned after %v; the context ended after 50ms", elapsed)
	}
}

func TestRunYFTurnsAPanicIntoAnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := runYF(ctx, func() (int, error) { panic("boom") })

	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("got %v, want an error carrying the panic", err)
	}
}

func TestRunYFWithoutACancel(t *testing.T) {
	if v, err := runYF(context.Background(), func() (int, error) { return 7, nil }); v != 7 || err != nil {
		t.Errorf("got %d, %v, want 7, nil", v, err)
	}

	var nilCtx context.Context
	called := false
	_, err := runYF(nilCtx, func() (int, error) {
		called = true
		return 0, nil
	})
	if !errors.Is(err, errNilContext) {
		t.Errorf("a nil context returned %v, want errNilContext", err)
	}
	if called {
		t.Error("the call ran under a nil context")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("under context.Background() the call ran on another goroutine: its panic did not reach the caller")
			}
		}()
		_, _ = runYF(context.Background(), func() (int, error) { panic("on the caller's goroutine") })
	}()
}

func TestYFTickerContextMethodsStopOnACancelledContext(t *testing.T) {
	y, err := YFinance(YFinanceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer y.close()
	tk := y.Ticker("AAPL")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Every YFTicker method that requests data. Earnings, Sustainability, FundsData and TopHoldings send no request and have no Context form.
	calls := []struct {
		name string
		call func(context.Context) error
	}{
		{"HistoryContext", func(ctx context.Context) error {
			_, err := tk.HistoryContext(ctx, YFHistoryParams{Period: "1mo"})
			return err
		}},
		{"QuoteContext", func(ctx context.Context) error { _, err := tk.QuoteContext(ctx); return err }},
		{"InfoContext", func(ctx context.Context) error { _, err := tk.InfoContext(ctx); return err }},
		{"DividendsContext", func(ctx context.Context) error { _, err := tk.DividendsContext(ctx); return err }},
		{"SplitsContext", func(ctx context.Context) error { _, err := tk.SplitsContext(ctx); return err }},
		{"ActionsContext", func(ctx context.Context) error { _, err := tk.ActionsContext(ctx); return err }},
		{"OptionsContext", func(ctx context.Context) error { _, err := tk.OptionsContext(ctx); return err }},
		{"OptionChainContext", func(ctx context.Context) error { _, err := tk.OptionChainContext(ctx, "2026-12-18"); return err }},
		{"NewsContext", func(ctx context.Context) error { _, err := tk.NewsContext(ctx, 5, YFNewsTabNews); return err }},
		{"CalendarContext", func(ctx context.Context) error { _, err := tk.CalendarContext(ctx); return err }},
		{"IncomeStatementContext", func(ctx context.Context) error { _, err := tk.IncomeStatementContext(ctx, YFPeriodAnnual); return err }},
		{"BalanceSheetContext", func(ctx context.Context) error { _, err := tk.BalanceSheetContext(ctx, YFPeriodAnnual); return err }},
		{"CashFlowContext", func(ctx context.Context) error { _, err := tk.CashFlowContext(ctx, YFPeriodAnnual); return err }},
		{"MajorHoldersContext", func(ctx context.Context) error { _, err := tk.MajorHoldersContext(ctx); return err }},
		{"InstitutionalHoldersContext", func(ctx context.Context) error { _, err := tk.InstitutionalHoldersContext(ctx); return err }},
		{"MutualFundHoldersContext", func(ctx context.Context) error { _, err := tk.MutualFundHoldersContext(ctx); return err }},
		{"InsiderTransactionsContext", func(ctx context.Context) error { _, err := tk.InsiderTransactionsContext(ctx); return err }},
		{"FastInfoContext", func(ctx context.Context) error { _, err := tk.FastInfoContext(ctx); return err }},
		{"EarningsEstimateContext", func(ctx context.Context) error { _, err := tk.EarningsEstimateContext(ctx); return err }},
		{"EarningsHistoryContext", func(ctx context.Context) error { _, err := tk.EarningsHistoryContext(ctx); return err }},
		{"EPSTrendContext", func(ctx context.Context) error { _, err := tk.EPSTrendContext(ctx); return err }},
		{"EPSRevisionsContext", func(ctx context.Context) error { _, err := tk.EPSRevisionsContext(ctx); return err }},
		{"RecommendationsContext", func(ctx context.Context) error { _, err := tk.RecommendationsContext(ctx); return err }},
		{"AnalystPriceTargetsContext", func(ctx context.Context) error { _, err := tk.AnalystPriceTargetsContext(ctx); return err }},
		{"RevenueEstimateContext", func(ctx context.Context) error { _, err := tk.RevenueEstimateContext(ctx); return err }},
		{"GrowthEstimatesContext", func(ctx context.Context) error { _, err := tk.GrowthEstimatesContext(ctx); return err }},
	}
	if len(calls) != 26 {
		t.Fatalf("the table covers %d methods, want all 26 that request data", len(calls))
	}
	for _, c := range calls {
		if err := c.call(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("%s returned %v, want context.Canceled", c.name, err)
		}
	}

	var nilCtx context.Context
	if _, err := tk.HistoryContext(nilCtx, YFHistoryParams{}); !errors.Is(err, errNilContext) {
		t.Errorf("a nil context returned %v, want errNilContext", err)
	}
}

func TestYFHistoryContextStopsTheLimiterWait(t *testing.T) {
	y, err := YFinance(YFinanceConfig{Interval: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer y.close()
	// Take the first slot, so the next request has to wait ten seconds for its
	// own. The wait comes before the request, so nothing reaches the network.
	if err := y.limiter.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = y.Ticker("AAPL").HistoryContext(ctx, YFHistoryParams{Period: "1mo"})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %v; the context ended after 50ms", elapsed)
	}
}
