package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// FugleOwnershipOptions are the optional settings of the ownership methods.
// With neither From nor To, Fugle picks the period: the last 30 days for ETF
// holdings and institutional trades, the latest report for director holdings
// and the shareholding distribution. From alone runs to today, and To needs a
// From. A range of a year or more is cut into pieces and joined, as for
// historical candles.
type FugleOwnershipOptions struct {
	From, To time.Time
	// Descending lists the newest first; the default is oldest first.
	Descending bool
}

var (
	fugleETFHoldingColumns         = []string{"date", "symbol", "name", "quantity", "weight", "quantityChange", "weightChange"}
	fugleInstitutionalColumns      = []string{"date", "foreign_buy", "foreign_sell", "foreign_net", "trust_buy", "trust_sell", "trust_net", "dealer_buy", "dealer_sell", "dealer_net", "total"}
	fugleDirectorHoldingColumns    = []string{"date", "order", "title", "name", "electedShares", "heldShares", "pledgedShares", "pledgeRatio", "relatedHeldShares", "relatedPledgedShares", "relatedPledgeRatio"}
	fugleTDCCDistributionColumns   = []string{"date", "range", "holders", "shares", "proportion"}
	errFugleOwnershipToWithoutFrom = errors.New("datafetch: Fugle To needs a From")
)

// fugleMonthWindows cuts a month range into pieces of at most twelve months,
// the most one request may span. Days are dropped: a month is asked for by
// its first day.
func fugleMonthWindows(fromIn, toIn time.Time) ([]fugleWindow, error) {
	switch {
	case fromIn.IsZero() && toIn.IsZero():
		return []fugleWindow{{}}, nil
	case fromIn.IsZero():
		return nil, errFugleOwnershipToWithoutFrom
	}
	month := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC) }
	from := month(fromIn)
	to := month(time.Now().UTC())
	if !toIn.IsZero() {
		to = month(toIn)
	}
	if from.After(to) {
		return nil, errors.New("datafetch: Fugle From is after To")
	}
	var windows []fugleWindow
	for start := from; !start.After(to); start = start.AddDate(0, 12, 0) {
		end := start.AddDate(0, 11, 0)
		if end.After(to) {
			end = to
		}
		windows = append(windows, fugleWindow{start, end})
	}
	return windows, nil
}

// ownership fetches one ownership endpoint over the windows and returns the
// data entries in date order, oldest first unless descending.
func (s FugleStock) ownership(ctx context.Context, path, symbol string, monthly bool, opts []FugleOwnershipOptions) ([]map[string]any, error) {
	c := s.c
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	symbol, err = fugleSymbol(symbol)
	if err != nil {
		return nil, err
	}
	var windows []fugleWindow
	if monthly {
		windows, err = fugleMonthWindows(p.From, p.To)
	} else {
		windows, err = fugleWindows(p.From, p.To, FugleTimeframeDay)
	}
	if err != nil {
		return nil, err
	}
	layout := "2006-01-02"
	if monthly {
		layout = "2006-01"
	}
	var entries []map[string]any
	for _, w := range windows {
		values := url.Values{"sort": {"asc"}}
		if !w.from.IsZero() {
			values.Set("from", w.from.Format(layout))
			values.Set("to", w.to.Format(layout))
		}
		resp, err := c.fugleGet(ctx, "/stock/ownership/"+path+"/"+url.PathEscape(symbol), values)
		if errors.Is(err, errTWStockNoData) {
			continue
		}
		if err != nil {
			return nil, err
		}
		part, err := fugleRows(resp)
		if err != nil {
			return nil, err
		}
		entries = append(entries, part...)
	}
	if p.Descending {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}
	return entries, nil
}

// fugleLongRows turns each entry's list under key into rows that carry the
// entry's date.
func fugleLongRows(entries []map[string]any, key string) ([]map[string]any, error) {
	var rows []map[string]any
	for i, e := range entries {
		raw, ok := e[key]
		if !ok || raw == nil {
			continue
		}
		list, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("datafetch: Fugle data row %d has %s that is not an array", i, key)
		}
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("datafetch: Fugle data row %d has a %s entry that is not an object", i, key)
			}
			row := map[string]any{"date": e["date"]}
			for k, v := range m {
				row[k] = v
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// OwnershipETFHoldings returns an ETF's daily stock holdings, one row per day
// and holding (Developer or Advanced plan). It is
// OwnershipETFHoldingsContext with context.Background().
func (s FugleStock) OwnershipETFHoldings(symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	return s.OwnershipETFHoldingsContext(context.Background(), symbol, opts...)
}

// OwnershipETFHoldingsContext is OwnershipETFHoldings with a context. The
// columns are date, symbol and name of the holding, quantity in shares, weight
// in percent, and quantityChange and weightChange since the ETF's previous
// day with data (nil on its first such day). Only the stock part of an ETF is
// listed, so the weights need not add up to 100.
func (s FugleStock) OwnershipETFHoldingsContext(ctx context.Context, symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	entries, err := s.ownership(ctx, "etf-holdings", symbol, false, opts)
	if err != nil {
		return nil, err
	}
	rows, err := fugleLongRows(entries, "components")
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, fugleETFHoldingColumns, fugleETFHoldingColumns), nil
}

// OwnershipInstitutionalTrades returns the daily buying and selling of a stock
// by the three groups of institutional investors, one row per day they traded
// (Developer or Advanced plan). It is OwnershipInstitutionalTradesContext with
// context.Background().
func (s FugleStock) OwnershipInstitutionalTrades(symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	return s.OwnershipInstitutionalTradesContext(context.Background(), symbol, opts...)
}

// OwnershipInstitutionalTradesContext is OwnershipInstitutionalTrades with a
// context. The columns are date, then buy, sell and net in shares for foreign
// investors (foreign_buy...), investment trusts (trust_...) and dealers
// (dealer_...), and total, the three nets together.
func (s FugleStock) OwnershipInstitutionalTradesContext(ctx context.Context, symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	entries, err := s.ownership(ctx, "institutional-trades", symbol, false, opts)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, len(entries))
	for i, e := range entries {
		m := map[string]any{}
		flattenFugle("", e, m)
		rows[i] = m
	}
	return fugleTable(rows, fugleInstitutionalColumns, fugleInstitutionalColumns), nil
}

// OwnershipDirectorHoldings returns the monthly holdings of a company's
// directors, supervisors and managers, one row per month and person (Developer
// or Advanced plan). From and To are read as months. It is
// OwnershipDirectorHoldingsContext with context.Background().
func (s FugleStock) OwnershipDirectorHoldings(symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	return s.OwnershipDirectorHoldingsContext(context.Background(), symbol, opts...)
}

// OwnershipDirectorHoldingsContext is OwnershipDirectorHoldings with a context.
// date is the first day of the month. The share counts are in shares and the
// pledge ratios are fractions (0.2146 is 21.46%); the related columns are the
// holdings of the person's spouse, minor children and nominees.
func (s FugleStock) OwnershipDirectorHoldingsContext(ctx context.Context, symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	entries, err := s.ownership(ctx, "director-holdings", symbol, true, opts)
	if err != nil {
		return nil, err
	}
	rows, err := fugleLongRows(entries, "directors")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if str, ok := row["date"].(string); ok {
			if d, err := time.Parse("2006-01", str); err == nil {
				row["date"] = d
			}
		}
	}
	return fugleTable(rows, fugleDirectorHoldingColumns, fugleDirectorHoldingColumns), nil
}

// OwnershipTDCCDistribution returns a stock's shareholding distribution from
// the Taiwan Depository and Clearing Corporation, one row per report date and
// holding range (Developer or Advanced plan). It is
// OwnershipTDCCDistributionContext with context.Background().
func (s FugleStock) OwnershipTDCCDistribution(symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	return s.OwnershipTDCCDistributionContext(context.Background(), symbol, opts...)
}

// OwnershipTDCCDistributionContext is OwnershipTDCCDistribution with a context.
// range is the holding range in shares as Fugle writes it, with a total row
// and, in some reports, an adjustment row; holders is the number of holders,
// shares their shares and proportion their percentage of the shares held at
// the depository. Reports are weekly, monthly before May 2015.
func (s FugleStock) OwnershipTDCCDistributionContext(ctx context.Context, symbol string, opts ...FugleOwnershipOptions) (*insyra.DataTable, error) {
	entries, err := s.ownership(ctx, "tdcc-distribution", symbol, false, opts)
	if err != nil {
		return nil, err
	}
	rows, err := fugleLongRows(entries, "distributions")
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, fugleTDCCDistributionColumns, fugleTDCCDistributionColumns), nil
}
