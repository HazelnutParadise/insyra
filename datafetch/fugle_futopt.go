package datafetch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// FugleFutOptType is the kind of derivative a product or ticker list is for.
type FugleFutOptType string

const (
	FugleFuture FugleFutOptType = "FUTURE"
	FugleOption FugleFutOptType = "OPTION"
)

// FugleFutOpt is the futures and options endpoints of a FugleClient, reached as
// client.FutOpt. The Intraday methods are about the current trading day, which
// for the night session is the next trading day's. A FugleFutOpt that is not the
// FutOpt field of a client from Fugle returns an error from every method.
type FugleFutOpt struct {
	c *FugleClient
}

// fugleFutOptSymbol checks a futures or options symbol. A spread symbol such as
// TXFJ6/K6 has two parts around a slash, which the API takes unencoded in the
// path.
func fugleFutOptSymbol(symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	parts := strings.Split(symbol, "/")
	if len(parts) > 2 {
		return "", errors.New("datafetch: Fugle futures and options symbol has more than one slash")
	}
	for i, p := range parts {
		if p == "" || strings.ContainsAny(p, "?# \t\n") {
			return "", errors.New("datafetch: Fugle futures and options symbol is empty or malformed")
		}
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/"), nil
}

func fugleFutOptKind(kind FugleFutOptType) error {
	if kind != FugleFuture && kind != FugleOption {
		return fmt.Errorf("datafetch: Fugle type %q must be FUTURE or OPTION", string(kind))
	}
	return nil
}

// fugleFlatRows flattens each row, so an array in a row becomes numbered columns.
func fugleFlatRows(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		flat := map[string]any{}
		flattenFugle("", row, flat)
		out[i] = flat
	}
	return out
}

// FugleFutOptProductsOptions are the optional settings of IntradayProducts.
type FugleFutOptProductsOptions struct {
	// Exchange is TAIFEX. Empty means any.
	Exchange string
	// AfterHours asks for the night session's products instead of the day session's.
	AfterHours bool
	// ContractType is I (index), R (interest rate), B (bond), C (commodity), S
	// (stock) or E (currency). Empty means any.
	ContractType string
	// Status is N (normal), P (trading suspended) or U (upcoming listing). Empty
	// means any.
	Status string
}

var fugleProductColumns = []string{"symbol", "type", "name", "underlyingSymbol", "contractType", "contractSize", "statusCode", "tradingCurrency", "quoteAcceptable", "startDate", "canBlockTrade", "expiryType", "underlyingType", "marketCloseGroup", "endSession"}

// IntradayProducts returns the list of products of a kind. It is
// IntradayProductsContext with context.Background().
func (f FugleFutOpt) IntradayProducts(kind FugleFutOptType, opts ...FugleFutOptProductsOptions) (*insyra.DataTable, error) {
	return f.IntradayProductsContext(context.Background(), kind, opts...)
}

// IntradayProductsContext is IntradayProducts with a context. The columns are
// those of the API's product list, such as symbol, type, name, contractSize and
// expiryType; a startDate the API leaves empty stays an empty string.
func (f FugleFutOpt) IntradayProductsContext(ctx context.Context, kind FugleFutOptType, opts ...FugleFutOptProductsOptions) (*insyra.DataTable, error) {
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
	if err := fugleFutOptKind(kind); err != nil {
		return nil, err
	}
	values := url.Values{"type": {string(kind)}}
	if p.Exchange != "" {
		values.Set("exchange", p.Exchange)
	}
	if p.AfterHours {
		values.Set("session", "AFTERHOURS")
	}
	if p.ContractType != "" {
		values.Set("contractType", p.ContractType)
	}
	if p.Status != "" {
		values.Set("status", p.Status)
	}
	resp, err := c.fugleGet(ctx, "/futopt/intraday/products", values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(fugleFlatRows(rows), fugleProductColumns, []string{"symbol", "type", "name"}), nil
}

// FugleFutOptTickersOptions are the optional settings of IntradayTickers.
type FugleFutOptTickersOptions struct {
	// Exchange is TAIFEX. Empty means any.
	Exchange string
	// AfterHours asks for the night session's contracts.
	AfterHours bool
	// Product is a product code such as TXF. Empty means every product.
	Product string
	// ContractType is I, R, B, C, S or E as for IntradayProducts.
	ContractType string
	// IsSpread asks for the calendar spread contracts of futures instead of the
	// plain contracts.
	IsSpread bool
}

var fugleFutOptTickerColumns = []string{"type", "symbol", "name", "referencePrice", "limitUpPrice", "limitDownPrice", "limitUpLevel", "limitDownLevel", "contractType", "startDate", "endDate", "openDatetime", "closeDatetime", "flowGroup", "settlementDate", "isDynamicBanding", "isSpread"}

// IntradayTickers returns the contracts of a kind. It is IntradayTickersContext
// with context.Background().
func (f FugleFutOpt) IntradayTickers(kind FugleFutOptType, opts ...FugleFutOptTickersOptions) (*insyra.DataTable, error) {
	return f.IntradayTickersContext(context.Background(), kind, opts...)
}

// IntradayTickersContext is IntradayTickers with a context. The columns are
// those of the API's ticker list; a stepped price limit becomes numbered columns
// (priceLimitLevels_1_level, priceLimitLevels_1_limitUpPrice and so on), and a
// field a contract does not have is nil.
func (f FugleFutOpt) IntradayTickersContext(ctx context.Context, kind FugleFutOptType, opts ...FugleFutOptTickersOptions) (*insyra.DataTable, error) {
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
	if err := fugleFutOptKind(kind); err != nil {
		return nil, err
	}
	values := url.Values{"type": {string(kind)}}
	if p.Exchange != "" {
		values.Set("exchange", p.Exchange)
	}
	if p.AfterHours {
		values.Set("session", "AFTERHOURS")
	}
	if p.Product != "" {
		values.Set("product", p.Product)
	}
	if p.ContractType != "" {
		values.Set("contractType", p.ContractType)
	}
	if p.IsSpread {
		values.Set("isSpread", "true")
	}
	resp, err := c.fugleGet(ctx, "/futopt/intraday/tickers", values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(fugleFlatRows(rows), fugleFutOptTickerColumns, []string{"symbol", "type", "name"}), nil
}

// symbolPath builds the path and query of an endpoint that takes one symbol.
// These endpoints take the night session as the lowercase afterhours.
func (f FugleFutOpt) symbolPath(symbol, group string, afterHours []bool) (string, url.Values, error) {
	symbol, err := fugleFutOptSymbol(symbol)
	if err != nil {
		return "", nil, err
	}
	after, err := oneFlag(afterHours, "afterHours")
	if err != nil {
		return "", nil, err
	}
	values := url.Values{}
	if after {
		values.Set("session", "afterhours")
	}
	return "/futopt/intraday/" + group + "/" + symbol, values, nil
}

var fugleFutOptTickerSymbolColumns = []string{"date", "type", "exchange", "symbol", "name", "referencePrice", "limitUpPrice", "limitDownPrice", "limitUpLevel", "limitDownLevel", "settlementDate", "startDate", "endDate", "openDatetime", "closeDatetime", "settleDate", "listType", "contractDate"}

// IntradayTicker returns one contract's details. It is IntradayTickerContext
// with context.Background().
func (f FugleFutOpt) IntradayTicker(symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	return f.IntradayTickerContext(context.Background(), symbol, afterHours...)
}

// IntradayTickerContext is IntradayTicker with a context. The table has one row;
// a stepped price limit becomes numbered columns as in IntradayTickers. A spread
// symbol such as TXFJ6/K6 is allowed. afterHours asks for the night session;
// more than one flag is an error.
func (f FugleFutOpt) IntradayTickerContext(ctx context.Context, symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	return f.symbolObject(ctx, symbol, "ticker", afterHours, fugleFutOptTickerSymbolColumns)
}

var fugleFutOptQuoteColumns = []string{
	"date", "type", "exchange", "symbol", "name", "previousClose",
	"openPrice", "openTime", "highPrice", "highTime", "lowPrice", "lowTime", "closePrice", "closeTime",
	"avgPrice", "change", "changePercent", "amplitude", "lastPrice", "lastSize",
	"total_tradeVolume", "total_totalBidMatch", "total_totalAskMatch", "total_time",
	"lastTrade_bid", "lastTrade_ask", "lastTrade_price", "lastTrade_size", "lastTrade_time", "lastTrade_serial",
	"lastTrial_bid", "lastTrial_ask", "lastTrial_price", "lastTrial_size", "lastTrial_time", "lastTrial_serial",
	"isTrial", "serial", "lastUpdated",
}

// IntradayQuote returns one contract's quote. It is IntradayQuoteContext with
// context.Background().
func (f FugleFutOpt) IntradayQuote(symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	return f.IntradayQuoteContext(context.Background(), symbol, afterHours...)
}

// IntradayQuoteContext is IntradayQuote with a context. The table has one row;
// nested objects become columns named after the object and the field
// (total_tradeVolume, lastTrade_price). isTrial is present only while the
// exchange is in its trial-match phase, and serial is nil before the day's first
// trade. The symbol column holds the contract the API answered for, which is
// not the symbol asked for when the request used a continuous code such as TXF1!.
func (f FugleFutOpt) IntradayQuoteContext(ctx context.Context, symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	return f.symbolObject(ctx, symbol, "quote", afterHours, fugleFutOptQuoteColumns)
}

func (f FugleFutOpt) symbolObject(ctx context.Context, symbol, group string, afterHours []bool, columns []string) (*insyra.DataTable, error) {
	c := f.c
	path, values, err := f.symbolPath(symbol, group, afterHours)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	resp, err := c.fugleGet(ctx, path, values)
	if err != nil {
		return nil, err
	}
	flat := map[string]any{}
	flattenFugle("", resp, flat)
	return fugleTable([]map[string]any{flat}, columns, nil), nil
}

// FugleFutOptIntradayCandlesOptions are the optional settings of IntradayCandles.
type FugleFutOptIntradayCandlesOptions struct {
	// Timeframe is the candle length in minutes: FugleTimeframe1Min, 5Min, 10Min,
	// 15Min, 30Min or 60Min. Empty means the API default, 1 minute. Futures and
	// options have no 3-minute candles.
	Timeframe FugleTimeframe
	// AfterHours asks for the night session.
	AfterHours bool
}

// IntradayCandles returns today's candles of a contract. It is
// IntradayCandlesContext with context.Background().
func (f FugleFutOpt) IntradayCandles(symbol string, opts ...FugleFutOptIntradayCandlesOptions) (*insyra.DataTable, error) {
	return f.IntradayCandlesContext(context.Background(), symbol, opts...)
}

// IntradayCandlesContext is IntradayCandles with a context. The columns are date
// (a time.Time), open, high, low, close, volume and average.
func (f FugleFutOpt) IntradayCandlesContext(ctx context.Context, symbol string, opts ...FugleFutOptIntradayCandlesOptions) (*insyra.DataTable, error) {
	c := f.c
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	switch p.Timeframe {
	case "", FugleTimeframe1Min, FugleTimeframe5Min, FugleTimeframe10Min, FugleTimeframe15Min, FugleTimeframe30Min, FugleTimeframe60Min:
	default:
		return nil, fmt.Errorf("datafetch: Fugle futures and options candle timeframe %q must be 1, 5, 10, 15, 30 or 60 minutes", string(p.Timeframe))
	}
	path, values, err := f.symbolPath(symbol, "candles", []bool{p.AfterHours})
	if err != nil {
		return nil, err
	}
	if p.Timeframe != "" {
		values.Set("timeframe", string(p.Timeframe))
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	resp, err := c.fugleGet(ctx, path, values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, fugleCandleColumns, nil), nil
}

// FugleFutOptIntradayTradesOptions are the optional settings of IntradayTrades.
type FugleFutOptIntradayTradesOptions struct {
	// AfterHours asks for the night session.
	AfterHours bool
	// Offset skips that many trades and Limit returns at most that many; zero
	// means the API default. Neither may be negative.
	Offset, Limit int
	// Trial asks for the trial-match details instead of the official trades.
	Trial bool
}

// IntradayTrades returns today's trades of a contract. It is
// IntradayTradesContext with context.Background().
func (f FugleFutOpt) IntradayTrades(symbol string, opts ...FugleFutOptIntradayTradesOptions) (*insyra.DataTable, error) {
	return f.IntradayTradesContext(context.Background(), symbol, opts...)
}

// IntradayTradesContext is IntradayTrades with a context. The columns are bid,
// ask, price, size, time and serial; bid and ask are nil where the API leaves
// them out. time is the API's number, unchanged.
func (f FugleFutOpt) IntradayTradesContext(ctx context.Context, symbol string, opts ...FugleFutOptIntradayTradesOptions) (*insyra.DataTable, error) {
	c := f.c
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if p.Offset < 0 || p.Limit < 0 {
		return nil, errors.New("datafetch: Fugle Offset and Limit must not be negative")
	}
	path, values, err := f.symbolPath(symbol, "trades", []bool{p.AfterHours})
	if err != nil {
		return nil, err
	}
	if p.Offset > 0 {
		values.Set("offset", strconv.Itoa(p.Offset))
	}
	if p.Limit > 0 {
		values.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Trial {
		values.Set("isTrial", "true")
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	resp, err := c.fugleGet(ctx, path, values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	cols := []string{"bid", "ask", "price", "size", "time", "serial"}
	return fugleTable(rows, cols, cols), nil
}

// IntradayVolumes returns today's volume at each price. It is
// IntradayVolumesContext with context.Background().
func (f FugleFutOpt) IntradayVolumes(symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	return f.IntradayVolumesContext(context.Background(), symbol, afterHours...)
}

// IntradayVolumesContext is IntradayVolumes with a context. The columns are
// price and volume, the volume traded at that price so far.
func (f FugleFutOpt) IntradayVolumesContext(ctx context.Context, symbol string, afterHours ...bool) (*insyra.DataTable, error) {
	c := f.c
	path, values, err := f.symbolPath(symbol, "volumes", afterHours)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	resp, err := c.fugleGet(ctx, path, values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, []string{"price", "volume"}, []string{"price", "volume"}), nil
}
