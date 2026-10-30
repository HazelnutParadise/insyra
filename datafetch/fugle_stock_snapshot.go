package datafetch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"

	"github.com/HazelnutParadise/insyra"
)

// FugleMarket is a Taiwan stock market.
type FugleMarket string

const (
	// FugleMarketTSE is the listed market, FugleMarketOTC the over-the-counter
	// market, FugleMarketESB the emerging stock board and FugleMarketTIB the
	// Taiwan Innovation Board.
	FugleMarketTSE FugleMarket = "TSE"
	FugleMarketOTC FugleMarket = "OTC"
	FugleMarketESB FugleMarket = "ESB"
	FugleMarketTIB FugleMarket = "TIB"
)

func (m FugleMarket) valid() bool {
	switch m {
	case FugleMarketTSE, FugleMarketOTC, FugleMarketESB, FugleMarketTIB:
		return true
	}
	return false
}

// FugleMoverDirection says whether SnapshotMovers lists gainers or losers.
type FugleMoverDirection string

const (
	FugleMoverUp   FugleMoverDirection = "up"
	FugleMoverDown FugleMoverDirection = "down"
)

// FugleChangeBasis says whether SnapshotMovers ranks by the change in percent or
// by the price change.
type FugleChangeBasis string

const (
	FugleChangePercent FugleChangeBasis = "percent"
	FugleChangeValue   FugleChangeBasis = "value"
)

// FugleTradeBasis says whether SnapshotActives ranks by volume or by value.
type FugleTradeBasis string

const (
	FugleTradeVolume FugleTradeBasis = "volume"
	FugleTradeValue  FugleTradeBasis = "value"
)

// FugleThresholdOp is the comparison of a FugleThreshold.
type FugleThresholdOp string

const (
	FugleGreaterThan    FugleThresholdOp = "gt"
	FugleGreaterOrEqual FugleThresholdOp = "gte"
	FugleLessThan       FugleThresholdOp = "lt"
	FugleLessOrEqual    FugleThresholdOp = "lte"
	FugleEqual          FugleThresholdOp = "eq"
)

// FugleThreshold keeps the stocks whose change (in the basis SnapshotMovers was
// asked for) compares to Value as Op says. The zero value keeps every stock.
type FugleThreshold struct {
	Op    FugleThresholdOp
	Value float64
}

var fugleStatsColumns = []string{"date", "type", "exchange", "market", "symbol", "name", "openPrice", "highPrice", "lowPrice", "closePrice", "change", "changePercent", "tradeVolume", "tradeValue", "previousClose", "week52High", "week52Low"}

// HistoricalStats returns a stock's price statistics as a one-row table. It is
// HistoricalStatsContext with context.Background().
func (s FugleStock) HistoricalStats(symbol string) (*insyra.DataTable, error) {
	return s.HistoricalStatsContext(context.Background(), symbol)
}

// HistoricalStatsContext is HistoricalStats with a context. The columns are
// date, type, exchange, market, symbol, name, the day's open, high, low and
// close prices, change, changePercent, tradeVolume, tradeValue, previousClose,
// week52High and week52Low. Emerging stocks and warrants are not supported by
// Fugle and give an error (HTTP 404).
func (s FugleStock) HistoricalStatsContext(ctx context.Context, symbol string) (*insyra.DataTable, error) {
	c := s.c
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	symbol, err := fugleSymbol(symbol)
	if err != nil {
		return nil, err
	}
	resp, err := c.fugleGet(ctx, "/stock/historical/stats/"+url.PathEscape(symbol), nil)
	if err != nil {
		return nil, err
	}
	return fugleTable([]map[string]any{resp}, fugleStatsColumns, nil), nil
}

var fugleSnapshotColumns = []string{"date", "time", "market", "type", "symbol", "name", "openPrice", "highPrice", "lowPrice", "closePrice", "change", "changePercent", "tradeVolume", "tradeValue", "lastPrice", "lastUpdated"}

// snapshot fetches a snapshot endpoint and returns its rows with the snapshot's
// date, time and market on every row.
func (s FugleStock) snapshot(ctx context.Context, path string, values url.Values) (*insyra.DataTable, error) {
	c := s.c
	resp, err := c.fugleGet(ctx, path, values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		for _, k := range []string{"date", "time", "market"} {
			row[k] = resp[k]
		}
	}
	return fugleTable(rows, fugleSnapshotColumns, fugleSnapshotColumns), nil
}

func (s FugleStock) snapshotPrelude(ctx context.Context, market FugleMarket) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := s.c.usable(); err != nil {
		return err
	}
	if !market.valid() {
		return fmt.Errorf("datafetch: Fugle market %q must be TSE, OTC, ESB or TIB", string(market))
	}
	return nil
}

// FugleSnapshotQuotesOptions are the optional settings of SnapshotQuotes.
type FugleSnapshotQuotesOptions struct {
	// OnlyCommonStock lists common stocks only; the default also has preferred
	// stocks and ETFs.
	OnlyCommonStock bool
}

// SnapshotQuotes returns every stock of a market as of now. It is
// SnapshotQuotesContext with context.Background().
func (s FugleStock) SnapshotQuotes(market FugleMarket, opts ...FugleSnapshotQuotesOptions) (*insyra.DataTable, error) {
	return s.SnapshotQuotesContext(context.Background(), market, opts...)
}

// SnapshotQuotesContext is SnapshotQuotes with a context. The snapshot refreshes
// every five seconds and needs the Developer or Advanced plan (ErrFuglePlan
// otherwise). The columns are date, time, market (the snapshot's, on every row),
// type, symbol, name, the day's open, high, low and close prices, change,
// changePercent, tradeVolume, tradeValue, lastPrice (nil for an index or before
// the day's first trade) and lastUpdated.
func (s FugleStock) SnapshotQuotesContext(ctx context.Context, market FugleMarket, opts ...FugleSnapshotQuotesOptions) (*insyra.DataTable, error) {
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := s.snapshotPrelude(ctx, market); err != nil {
		return nil, err
	}
	values := url.Values{}
	if p.OnlyCommonStock {
		values.Set("type", "COMMONSTOCK")
	}
	return s.snapshot(ctx, "/stock/snapshot/quotes/"+string(market), values)
}

// FugleSnapshotMoversOptions are the optional settings of SnapshotMovers.
type FugleSnapshotMoversOptions struct {
	// OnlyCommonStock lists common stocks only.
	OnlyCommonStock bool
	// Threshold keeps the stocks whose change compares to Value as Op says. The
	// zero value keeps every stock.
	Threshold FugleThreshold
}

// SnapshotMovers returns the biggest gainers or losers of a market as of now. It
// is SnapshotMoversContext with context.Background().
func (s FugleStock) SnapshotMovers(market FugleMarket, direction FugleMoverDirection, basis FugleChangeBasis, opts ...FugleSnapshotMoversOptions) (*insyra.DataTable, error) {
	return s.SnapshotMoversContext(context.Background(), market, direction, basis, opts...)
}

// SnapshotMoversContext is SnapshotMovers with a context. Its table has the
// columns of SnapshotQuotes, ranked by the change in the basis asked for.
func (s FugleStock) SnapshotMoversContext(ctx context.Context, market FugleMarket, direction FugleMoverDirection, basis FugleChangeBasis, opts ...FugleSnapshotMoversOptions) (*insyra.DataTable, error) {
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := s.snapshotPrelude(ctx, market); err != nil {
		return nil, err
	}
	if direction != FugleMoverUp && direction != FugleMoverDown {
		return nil, fmt.Errorf("datafetch: Fugle direction %q must be up or down", string(direction))
	}
	if basis != FugleChangePercent && basis != FugleChangeValue {
		return nil, fmt.Errorf("datafetch: Fugle change basis %q must be percent or value", string(basis))
	}
	values := url.Values{"direction": {string(direction)}, "change": {string(basis)}}
	if p.OnlyCommonStock {
		values.Set("type", "COMMONSTOCK")
	}
	if p.Threshold != (FugleThreshold{}) {
		switch p.Threshold.Op {
		case FugleGreaterThan, FugleGreaterOrEqual, FugleLessThan, FugleLessOrEqual, FugleEqual:
		default:
			return nil, fmt.Errorf("datafetch: Fugle threshold operator %q must be gt, gte, lt, lte or eq", string(p.Threshold.Op))
		}
		if math.IsNaN(p.Threshold.Value) || math.IsInf(p.Threshold.Value, 0) {
			return nil, errors.New("datafetch: Fugle threshold value must be a finite number")
		}
		values.Set(string(p.Threshold.Op), strconv.FormatFloat(p.Threshold.Value, 'f', -1, 64))
	}
	return s.snapshot(ctx, "/stock/snapshot/movers/"+string(market), values)
}

// FugleSnapshotActivesOptions are the optional settings of SnapshotActives.
type FugleSnapshotActivesOptions struct {
	// OnlyCommonStock lists common stocks only.
	OnlyCommonStock bool
}

// SnapshotActives returns the most actively traded stocks of a market as of now.
// It is SnapshotActivesContext with context.Background().
func (s FugleStock) SnapshotActives(market FugleMarket, basis FugleTradeBasis, opts ...FugleSnapshotActivesOptions) (*insyra.DataTable, error) {
	return s.SnapshotActivesContext(context.Background(), market, basis, opts...)
}

// SnapshotActivesContext is SnapshotActives with a context. Its table has the
// columns of SnapshotQuotes, ranked by volume or by value.
func (s FugleStock) SnapshotActivesContext(ctx context.Context, market FugleMarket, basis FugleTradeBasis, opts ...FugleSnapshotActivesOptions) (*insyra.DataTable, error) {
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := s.snapshotPrelude(ctx, market); err != nil {
		return nil, err
	}
	if basis != FugleTradeVolume && basis != FugleTradeValue {
		return nil, fmt.Errorf("datafetch: Fugle trade basis %q must be volume or value", string(basis))
	}
	values := url.Values{"trade": {string(basis)}}
	if p.OnlyCommonStock {
		values.Set("type", "COMMONSTOCK")
	}
	return s.snapshot(ctx, "/stock/snapshot/actives/"+string(market), values)
}
