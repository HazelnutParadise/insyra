package datafetch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// FugleCallPut is the side of an option.
type FugleCallPut string

const (
	FugleCall FugleCallPut = "CALL"
	FuglePut  FugleCallPut = "PUT"
)

// fugleProduct checks a product code for the historical endpoints. A spread is
// not a product there (the API answers 404), so a slash is refused.
func fugleProduct(product string) (string, error) {
	product = strings.TrimSpace(product)
	if product == "" || strings.ContainsAny(product, "/?# \t\n") {
		return "", errors.New("datafetch: Fugle product is empty or malformed; a spread is asked for with a ContractMonth such as 202610/202611")
	}
	return url.PathEscape(product), nil
}

// FugleFutOptHistoricalCandlesOptions are the optional settings of
// HistoricalCandles.
type FugleFutOptHistoricalCandlesOptions struct {
	// ContractMonth names the contract: yyyyMM, a weekly option yyyyMMWn
	// (Wednesday expiry) or yyyyMMFn (Friday expiry), a futures spread
	// "202610/202611", or for futures a continuous contract "1!", "2!" or "3!".
	// Empty means the API default, the continuous front month "1!", for futures;
	// options need one, with Strike and CallPut.
	ContractMonth string
	// Strike and CallPut pick an option. Give both or neither.
	Strike  float64
	CallPut FugleCallPut
	// From and To are the first and last trading day wanted. Leave both zero for
	// the API default, the last week. To alone is refused; From alone runs to
	// today.
	From, To time.Time
	// Timeframe is the candle length: FugleTimeframe1Min, 5Min, 10Min, 15Min,
	// 30Min, 60Min, Day, Week or Month. Empty means FugleTimeframeDay. Futures
	// and options have no 3-minute candles.
	Timeframe FugleTimeframe
	// AfterHours asks for the night session.
	AfterHours bool
	// Fields names the columns wanted besides date and contractMonth: open, high,
	// low, close, volume, average and transaction (minute candles), change (day,
	// week, month). Empty means the API default, open, high, low, close, volume.
	Fields []string
}

var fugleFutOptHistoricalFields = map[string]bool{"open": true, "high": true, "low": true, "close": true, "volume": true, "average": true, "transaction": true, "change": true}

var fugleFutOptHistoricalColumns = []string{"date", "contractMonth", "open", "high", "low", "close", "volume", "average", "transaction", "change"}

// HistoricalCandles returns the historical candles of a product. It is
// HistoricalCandlesContext with context.Background().
func (f FugleFutOpt) HistoricalCandles(product string, opts ...FugleFutOptHistoricalCandlesOptions) (*insyra.DataTable, error) {
	return f.HistoricalCandlesContext(context.Background(), product, opts...)
}

// HistoricalCandlesContext is HistoricalCandles with a context. product is the
// code without a month, such as TXF, MXF or TXO. As for stocks, a day or minute
// range over 364 days is cut into pieces and joined, a piece with no data adds
// nothing, and a week or month range over 364 days is refused. Rows are in
// ascending date order, and a date that has candles for two contracts, as on the
// roll day of a continuous contract, has two rows; the contractMonth column says
// which contract each row is. A continuous contract is not adjusted at a roll, so
// its candles gap on roll days. A day with no trade has nil prices and volume 0.
// A night session belongs to the next trading day. Daily data is available after
// about 17:30; it goes back to 2010 and minute data to June 2017.
func (f FugleFutOpt) HistoricalCandlesContext(ctx context.Context, product string, opts ...FugleFutOptHistoricalCandlesOptions) (*insyra.DataTable, error) {
	c := f.c
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
	product, err = fugleProduct(product)
	if err != nil {
		return nil, err
	}
	tf := p.Timeframe
	if tf == "" {
		tf = FugleTimeframeDay
	}
	if !tf.valid() || tf == FugleTimeframe3Min {
		return nil, fmt.Errorf("datafetch: Fugle futures and options timeframe %q must be 1, 5, 10, 15, 30, 60, D, W or M", string(tf))
	}
	for _, field := range p.Fields {
		if !fugleFutOptHistoricalFields[field] {
			return nil, fmt.Errorf("datafetch: Fugle futures and options candle field %q is not one of open, high, low, close, volume, average, transaction, change", field)
		}
	}
	if (p.Strike != 0) != (p.CallPut != "") {
		return nil, errors.New("datafetch: Fugle option Strike and CallPut go together; give both or neither")
	}
	if p.CallPut != "" {
		if p.CallPut != FugleCall && p.CallPut != FuglePut {
			return nil, fmt.Errorf("datafetch: Fugle CallPut %q must be CALL or PUT", string(p.CallPut))
		}
		if math.IsNaN(p.Strike) || math.IsInf(p.Strike, 0) || p.Strike <= 0 {
			return nil, errors.New("datafetch: Fugle Strike must be a positive number")
		}
	}
	windows, err := fugleWindows(p.From, p.To, tf)
	if err != nil {
		return nil, err
	}

	base := url.Values{"timeframe": {string(tf)}, "sort": {"asc"}}
	if p.ContractMonth != "" {
		base.Set("contractMonth", p.ContractMonth)
	}
	if p.CallPut != "" {
		base.Set("strikePrice", strconv.FormatFloat(p.Strike, 'f', -1, 64))
		base.Set("callPut", string(p.CallPut))
	}
	if p.AfterHours {
		base.Set("session", "afterhours")
	}
	if len(p.Fields) > 0 {
		base.Set("fields", strings.Join(p.Fields, ","))
	}
	rows, err := c.fugleFetchWindows(ctx, "/futopt/historical/candles/"+product, base, windows)
	if err != nil {
		return nil, err
	}
	key := func(row map[string]any) string {
		return fmt.Sprint(row["date"]) + "|" + fmt.Sprint(row["contractMonth"])
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if di, dj := fmt.Sprint(rows[i]["date"]), fmt.Sprint(rows[j]["date"]); di != dj {
			return di < dj
		}
		return fmt.Sprint(rows[i]["contractMonth"]) < fmt.Sprint(rows[j]["contractMonth"])
	})
	out := rows[:0]
	for i, row := range rows {
		if i > 0 && key(row) == key(rows[i-1]) {
			continue
		}
		out = append(out, row)
	}
	return fugleTable(out, fugleFutOptHistoricalColumns, nil), nil
}

// FugleFutOptHistoricalDailyOptions are the optional settings of HistoricalDaily.
type FugleFutOptHistoricalDailyOptions struct {
	// Date is the trading day. Zero means today.
	Date time.Time
	// AfterHours asks for the night session, which belongs to the next trading
	// day.
	AfterHours bool
	// ContractMonth narrows the answer to one contract, in the formats of
	// FugleFutOptHistoricalCandlesOptions.ContractMonth. Empty means every
	// contract listed that day.
	ContractMonth string
}

var fugleFutOptDailyColumns = []string{"date", "contractMonth", "strikePrice", "callPut", "exchange", "openPrice", "highPrice", "lowPrice", "closePrice", "change", "changePercent", "volume", "volumeSpread", "openInterest", "settlementPrice"}

// HistoricalDaily returns one trading day of a product, a row per contract. It
// is HistoricalDailyContext with context.Background().
func (f FugleFutOpt) HistoricalDaily(product string, opts ...FugleFutOptHistoricalDailyOptions) (*insyra.DataTable, error) {
	return f.HistoricalDailyContext(context.Background(), product, opts...)
}

// HistoricalDailyContext is HistoricalDaily with a context. The columns are date
// (the trading day, a time.Time), contractMonth, strikePrice and callPut (nil for
// futures and spreads), exchange, the day's open, high, low and close prices,
// change, changePercent, volume, volumeSpread, openInterest and settlementPrice;
// a price is nil where there was no trade, and the open interest and settlement
// price are nil for a night session. A day with no data (a non-trading day, or
// data not yet produced, HTTP 404) gives an empty table. Daily data is produced
// after the close, about 17:30. Only one day is asked for at a time; for a
// contract's series over many days use HistoricalCandles.
func (f FugleFutOpt) HistoricalDailyContext(ctx context.Context, product string, opts ...FugleFutOptHistoricalDailyOptions) (*insyra.DataTable, error) {
	c := f.c
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
	product, err = fugleProduct(product)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	if !p.Date.IsZero() {
		values.Set("date", normalizeDate(p.Date).Format("2006-01-02"))
	}
	if p.AfterHours {
		values.Set("session", "afterhours")
	}
	if p.ContractMonth != "" {
		values.Set("contractMonth", p.ContractMonth)
	}
	resp, err := c.fugleGet(ctx, "/futopt/historical/daily/"+product, values)
	if errors.Is(err, errTWStockNoData) {
		return fugleTable(nil, fugleFutOptDailyColumns, fugleFutOptDailyColumns), nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["date"] = resp["date"]
	}
	return fugleTable(rows, fugleFutOptDailyColumns, fugleFutOptDailyColumns), nil
}
