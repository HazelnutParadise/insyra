package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// FugleTechnicalOptions are the optional settings of the technical indicator
// methods. From and To bound the dates, both or neither; a minute timeframe
// takes neither, because Fugle always answers it with the last 30 days.
type FugleTechnicalOptions struct {
	From, To time.Time
}

// technical fetches one technical indicator endpoint. periods are the
// indicator's required whole-number settings, in the order of their query
// names.
func (s FugleStock) technical(ctx context.Context, indicator, symbol string, timeframe FugleTimeframe, names []string, periods []int, columns []string, opts []FugleTechnicalOptions) (*insyra.DataTable, error) {
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
	if !timeframe.valid() {
		return nil, fmt.Errorf("datafetch: Fugle timeframe %q is not one of 1, 3, 5, 10, 15, 30, 60, D, W, M", string(timeframe))
	}
	values := url.Values{"timeframe": {string(timeframe)}}
	for i, n := range names {
		if periods[i] < 1 {
			return nil, fmt.Errorf("datafetch: Fugle %s %s must be at least 1, got %d", indicator, n, periods[i])
		}
		values.Set(n, strconv.Itoa(periods[i]))
	}
	switch {
	case p.From.IsZero() && p.To.IsZero():
	case p.From.IsZero() || p.To.IsZero():
		return nil, errors.New("datafetch: Fugle technical indicators take From and To together")
	case timeframe.minute():
		return nil, errors.New("datafetch: Fugle technical indicators of minute candles cover the last 30 days and take no From or To")
	case normalizeDate(p.To).Before(normalizeDate(p.From)):
		return nil, errors.New("datafetch: Fugle To is before From")
	default:
		values.Set("from", p.From.Format("2006-01-02"))
		values.Set("to", p.To.Format("2006-01-02"))
	}
	resp, err := c.fugleGet(ctx, "/stock/technical/"+indicator+"/"+url.PathEscape(symbol), values)
	if errors.Is(err, errTWStockNoData) {
		return fugleTable(nil, columns, columns), nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, columns, columns), nil
}

// TechnicalSMA returns a stock's simple moving average over period candles of
// timeframe, as columns date and sma. It is TechnicalSMAContext with
// context.Background().
func (s FugleStock) TechnicalSMA(symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.TechnicalSMAContext(context.Background(), symbol, timeframe, period, opts...)
}

// TechnicalSMAContext is TechnicalSMA with a context.
func (s FugleStock) TechnicalSMAContext(ctx context.Context, symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.technical(ctx, "sma", symbol, timeframe, []string{"period"}, []int{period}, []string{"date", "sma"}, opts)
}

// TechnicalRSI returns a stock's relative strength index over period candles of
// timeframe, as columns date and rsi. It is TechnicalRSIContext with
// context.Background().
func (s FugleStock) TechnicalRSI(symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.TechnicalRSIContext(context.Background(), symbol, timeframe, period, opts...)
}

// TechnicalRSIContext is TechnicalRSI with a context.
func (s FugleStock) TechnicalRSIContext(ctx context.Context, symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.technical(ctx, "rsi", symbol, timeframe, []string{"period"}, []int{period}, []string{"date", "rsi"}, opts)
}

// TechnicalKDJ returns a stock's stochastic oscillator, as columns date, k, d
// and j. rPeriod is the look-back of the raw stochastic value, kPeriod and
// dPeriod the smoothing of %K and %D (9, 3, 3 is the common choice). It is
// TechnicalKDJContext with context.Background().
func (s FugleStock) TechnicalKDJ(symbol string, timeframe FugleTimeframe, rPeriod, kPeriod, dPeriod int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.TechnicalKDJContext(context.Background(), symbol, timeframe, rPeriod, kPeriod, dPeriod, opts...)
}

// TechnicalKDJContext is TechnicalKDJ with a context.
func (s FugleStock) TechnicalKDJContext(ctx context.Context, symbol string, timeframe FugleTimeframe, rPeriod, kPeriod, dPeriod int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.technical(ctx, "kdj", symbol, timeframe, []string{"rPeriod", "kPeriod", "dPeriod"}, []int{rPeriod, kPeriod, dPeriod}, []string{"date", "k", "d", "j"}, opts)
}

// TechnicalMACD returns a stock's MACD, as columns date, macdLine and
// signalLine. fast and slow are the periods of the two moving averages and
// signal the period of the signal line (12, 26, 9 is the common choice); fast
// must be shorter than slow. It is TechnicalMACDContext with
// context.Background().
func (s FugleStock) TechnicalMACD(symbol string, timeframe FugleTimeframe, fast, slow, signal int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.TechnicalMACDContext(context.Background(), symbol, timeframe, fast, slow, signal, opts...)
}

// TechnicalMACDContext is TechnicalMACD with a context.
func (s FugleStock) TechnicalMACDContext(ctx context.Context, symbol string, timeframe FugleTimeframe, fast, slow, signal int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	if fast >= slow && fast >= 1 {
		return nil, fmt.Errorf("datafetch: Fugle MACD fast period %d must be shorter than slow period %d", fast, slow)
	}
	return s.technical(ctx, "macd", symbol, timeframe, []string{"fast", "slow", "signal"}, []int{fast, slow, signal}, []string{"date", "macdLine", "signalLine"}, opts)
}

// TechnicalBB returns a stock's Bollinger bands over period candles, as columns
// date, upper, middle and lower. Fugle sets the band width; it cannot be
// chosen. It is TechnicalBBContext with context.Background().
func (s FugleStock) TechnicalBB(symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.TechnicalBBContext(context.Background(), symbol, timeframe, period, opts...)
}

// TechnicalBBContext is TechnicalBB with a context.
func (s FugleStock) TechnicalBBContext(ctx context.Context, symbol string, timeframe FugleTimeframe, period int, opts ...FugleTechnicalOptions) (*insyra.DataTable, error) {
	return s.technical(ctx, "bb", symbol, timeframe, []string{"period"}, []int{period}, []string{"date", "upper", "middle", "lower"}, opts)
}
