package datafetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

// ErrFugleAuth matches (with errors.Is) an error returned because Fugle refused
// the API key (HTTP 401). It is not retried. Keep the whole key: it is a base64
// string that usually ends in "==".
var ErrFugleAuth = errors.New("datafetch: Fugle refused the API key")

// ErrFuglePlan matches (with errors.Is) an error returned because the account's
// plan does not include the endpoint (HTTP 403). The free plan has the intraday
// and historical endpoints only. It is not retried.
var ErrFuglePlan = errors.New("datafetch: Fugle plan does not include this endpoint")

// ErrFugleThrottled matches (with errors.Is) an error returned because Fugle
// refused the request as too frequent (HTTP 429) on every attempt. It is retried
// like any failed request; the fix is a larger Interval or more Retries.
var ErrFugleThrottled = errors.New("datafetch: Fugle request limit reached")

var errFugleNotCreated = errors.New("datafetch: FugleClient was not created with Fugle")

const (
	defaultFugleBaseURL  = "https://api.fugle.tw/marketdata/v1.0"
	defaultFugleInterval = 1100 * time.Millisecond // under the 60 requests a minute every plan allows; a 1s spacing hits the limit on the 61st request
	// fugleMaxSpan is the longest range one historical request may cover: the
	// API refuses a range of a year or more, so 364 days is the longest safe one.
	fugleMaxSpan = 364 * 24 * time.Hour
	// fugleMaxRetryAfter caps how long a Retry-After header can make a call wait.
	fugleMaxRetryAfter = 60
)

// FugleTimeframe is the length of a candle.
type FugleTimeframe string

const (
	FugleTimeframe1Min  FugleTimeframe = "1"
	FugleTimeframe3Min  FugleTimeframe = "3"
	FugleTimeframe5Min  FugleTimeframe = "5"
	FugleTimeframe10Min FugleTimeframe = "10"
	FugleTimeframe15Min FugleTimeframe = "15"
	FugleTimeframe30Min FugleTimeframe = "30"
	FugleTimeframe60Min FugleTimeframe = "60"
	FugleTimeframeDay   FugleTimeframe = "D"
	FugleTimeframeWeek  FugleTimeframe = "W"
	FugleTimeframeMonth FugleTimeframe = "M"
)

func (t FugleTimeframe) minute() bool {
	switch t {
	case FugleTimeframe1Min, FugleTimeframe3Min, FugleTimeframe5Min, FugleTimeframe10Min, FugleTimeframe15Min, FugleTimeframe30Min, FugleTimeframe60Min:
		return true
	}
	return false
}

func (t FugleTimeframe) valid() bool {
	return t.minute() || t == FugleTimeframeDay || t == FugleTimeframeWeek || t == FugleTimeframeMonth
}

// FugleTickerType is the kind of security a ticker list is for.
type FugleTickerType string

const (
	FugleTickerEquity  FugleTickerType = "EQUITY"
	FugleTickerIndex   FugleTickerType = "INDEX"
	FugleTickerWarrant FugleTickerType = "WARRANT"
	FugleTickerOddLot  FugleTickerType = "ODDLOT"
)

// FugleConfig configures a Fugle market data client.
type FugleConfig struct {
	// APIKey is the account's API key, sent in the X-API-KEY header. Keep the
	// whole string, which usually ends in "==". Leave it empty to call without
	// one, as an external proxy that adds it would need.
	APIKey string
	// BaseURL is the API root including the version. Default:
	// https://api.fugle.tw/marketdata/v1.0.
	BaseURL string
	// Timeout is the per-request timeout. Default: 15s.
	Timeout time.Duration
	// Interval is the minimum spacing between the scheduled starts of successive
	// requests. Zero means 1.1s, which keeps a client under the 60 requests a
	// minute every plan allows (1s reaches the limit on the 61st request); a
	// negative value turns throttling off.
	Interval time.Duration
	// UserAgent is the HTTP User-Agent header.
	UserAgent string
	// Retries is the number of retry attempts after a failed request. A refused
	// key, a plan without access and a bad request are never retried.
	Retries int
	// RetryBackoff is the base delay before a retry. Default: 300ms.
	RetryBackoff time.Duration
}

// FugleClient reads market data from Fugle (developer.fugle.tw). Create one with
// Fugle. A client that did not come from Fugle returns an error from every
// method.
type FugleClient struct {
	base    *TWStockClient
	baseURL string
}

// Fugle creates a client for the Fugle market data REST API.
func Fugle(cfg FugleConfig) (*FugleClient, error) {
	interval := cfg.Interval
	switch {
	case interval == 0:
		interval = defaultFugleInterval
	case interval < 0:
		interval = 0
	}
	base, err := TWStock(TWStockConfig{
		Timeout: cfg.Timeout, Interval: interval, UserAgent: cfg.UserAgent,
		Retries: cfg.Retries, RetryBackoff: cfg.RetryBackoff,
	})
	if err != nil {
		return nil, err
	}
	if cfg.APIKey != "" {
		base.headers = map[string]string{"X-API-KEY": cfg.APIKey}
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultFugleBaseURL
	}
	return &FugleClient{base: base, baseURL: baseURL}, nil
}

func (c *FugleClient) usable() error {
	if c == nil || c.base == nil {
		return errFugleNotCreated
	}
	return c.base.usable()
}

// fugleGet sends a GET request and returns the decoded JSON object. A 404 is an
// error matching errTWStockNoData, which doGet does not retry.
func (c *FugleClient) fugleGet(ctx context.Context, path string, values url.Values) (map[string]any, error) {
	var out map[string]any
	err := c.base.doGet(ctx, requestURL(c.baseURL, path, values), func(resp *http.Response) error {
		defer func() { _ = resp.Body.Close() }()
		snippet := func() string {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
			return strings.TrimSpace(string(b))
		}
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
		case resp.StatusCode == http.StatusUnauthorized:
			return &finalError{fmt.Errorf("%w: %s", ErrFugleAuth, snippet())}
		case resp.StatusCode == http.StatusForbidden:
			return &finalError{fmt.Errorf("%w: %s", ErrFuglePlan, snippet())}
		case resp.StatusCode == http.StatusNotFound:
			return fmt.Errorf("datafetch: Fugle has no data for this request (HTTP 404): %w", errTWStockNoData)
		case resp.StatusCode == http.StatusTooManyRequests:
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			// Wait as long as the server asks before the retry's own backoff.
			if secs, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); err == nil && secs > 0 {
				if secs > fugleMaxRetryAfter {
					secs = fugleMaxRetryAfter
				}
				if err := sleepContext(ctx, time.Duration(secs)*time.Second); err != nil {
					return &finalError{err}
				}
			}
			return fmt.Errorf("%w; raise Interval or Retries in FugleConfig", ErrFugleThrottled)
		case resp.StatusCode == http.StatusBadRequest:
			return &finalError{fmt.Errorf("datafetch: Fugle rejected the request (HTTP 400): %s", snippet())}
		default:
			return fmt.Errorf("datafetch: Fugle HTTP status %d: %s", resp.StatusCode, snippet())
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
		if err != nil {
			return fmt.Errorf("datafetch: Fugle read response: %w", err)
		}
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		var decoded map[string]any
		if err := dec.Decode(&decoded); err != nil {
			return fmt.Errorf("datafetch: Fugle decode JSON: %w", err)
		}
		out = decoded
		return nil
	})
	return out, err
}

// fugleRows returns the objects of the response's data array.
func fugleRows(resp map[string]any) ([]map[string]any, error) {
	raw, ok := resp["data"]
	if !ok || raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, errors.New("datafetch: Fugle data is not an array")
	}
	rows := make([]map[string]any, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("datafetch: Fugle data row %d is not an object", i)
		}
		rows[i] = m
	}
	return rows, nil
}

// fugleTable builds a table from rows. Columns appear in the order of
// preferred, then any other key sorted; a missing value is nil. A string column
// named date is read as time.Time when every value is a date or a timestamp.
func fugleTable(rows []map[string]any, preferred []string) *insyra.DataTable {
	present := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			present[k] = true
		}
	}
	var names []string
	for _, k := range preferred {
		if present[k] {
			names = append(names, k)
			delete(present, k)
		}
	}
	var rest []string
	for k := range present {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	names = append(names, rest...)

	lists := make([]*insyra.DataList, len(names))
	for c, name := range names {
		values := make([]any, len(rows))
		for r, row := range rows {
			values[r] = finMindValue(row[name])
		}
		if name == "date" {
			values = fugleDates(values)
		}
		lists[c] = newNamedCol(name, values)
	}
	return insyra.NewDataTable(lists...)
}

// fugleDates converts "YYYY-MM-DD" and RFC 3339 strings to times, or returns the
// values unchanged when any one is something else.
func fugleDates(values []any) []any {
	out := make([]any, len(values))
	for i, v := range values {
		s, ok := v.(string)
		if !ok {
			return values
		}
		if d, err := time.Parse("2006-01-02", s); err == nil {
			out[i] = d
		} else if d, err := time.Parse(time.RFC3339Nano, s); err == nil {
			out[i] = d
		} else {
			return values
		}
	}
	return out
}

func fugleSymbol(symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" || strings.ContainsAny(symbol, "/?#") {
		return "", errors.New("datafetch: Fugle symbol is empty or malformed")
	}
	return symbol, nil
}

func oneFlag(flags []bool, name string) (bool, error) {
	switch len(flags) {
	case 0:
		return false, nil
	case 1:
		return flags[0], nil
	}
	return false, fmt.Errorf("datafetch: at most one %s flag may be given, got %d", name, len(flags))
}

// FugleHistoricalCandlesParams are the settings of StockHistoricalCandles.
type FugleHistoricalCandlesParams struct {
	// From and To are the first and last day wanted. Leave both zero for the API's
	// default, the last month. To alone defaults From to nothing and is refused;
	// From alone runs to today.
	From, To time.Time
	// Timeframe is the candle length. Default: FugleTimeframeDay.
	Timeframe FugleTimeframe
	// Adjusted asks for prices adjusted for dividends and splits as of the day of
	// the request. It applies to day, week and month candles only.
	Adjusted bool
	// Fields names the columns wanted besides date: open, high, low, close,
	// volume, average, turnover (day, week, month) and change (day, week, month).
	// Empty means the API default, open, high, low, close, volume and average.
	Fields []string
}

var fugleCandleFields = map[string]bool{"open": true, "high": true, "low": true, "close": true, "volume": true, "average": true, "turnover": true, "change": true}

var fugleCandleColumns = []string{"date", "open", "high", "low", "close", "volume", "average", "turnover", "change"}

// StockHistoricalCandles returns a stock's historical candles. It is StockHistoricalCandlesContext
// with context.Background().
func (c *FugleClient) StockHistoricalCandles(symbol string, p FugleHistoricalCandlesParams) (*insyra.DataTable, error) {
	return c.StockHistoricalCandlesContext(context.Background(), symbol, p)
}

// StockHistoricalCandlesContext is StockHistoricalCandles with a context. The limiter wait, each
// request and each retry backoff stop when ctx is done.
//
// The API refuses a range of a year or more, so a day or minute range longer
// than 364 days is cut into pieces that are fetched one after another and
// joined; a piece with no data (HTTP 404) adds nothing, and a range with no data
// at all gives an empty table. Week and month candles are not cut, because a
// candle would be split between two pieces, so their range must be at most 364
// days. Rows are in ascending date order, a day's date is midnight UTC of that
// day, and a minute's is the exchange's timestamp. Volume is in shares for day,
// week and month candles and in lots (1,000 shares) for minute candles of listed
// stocks. Adjusted prices are adjusted as of the day of the request.
func (c *FugleClient) StockHistoricalCandlesContext(ctx context.Context, symbol string, p FugleHistoricalCandlesParams) (*insyra.DataTable, error) {
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
	tf := p.Timeframe
	if tf == "" {
		tf = FugleTimeframeDay
	}
	if !tf.valid() {
		return nil, fmt.Errorf("datafetch: Fugle timeframe %q must be 1, 3, 5, 10, 15, 30, 60, D, W or M", string(tf))
	}
	if p.Adjusted && tf.minute() {
		return nil, errors.New("datafetch: Fugle adjusted prices apply to day, week and month candles only")
	}
	for _, f := range p.Fields {
		if !fugleCandleFields[f] {
			return nil, fmt.Errorf("datafetch: Fugle candle field %q is not one of open, high, low, close, volume, average, turnover, change", f)
		}
	}
	path := "/stock/historical/candles/" + url.PathEscape(symbol)
	base := url.Values{"timeframe": {string(tf)}, "sort": {"asc"}}
	if p.Adjusted {
		base.Set("adjusted", "true")
	}
	if len(p.Fields) > 0 {
		base.Set("fields", strings.Join(p.Fields, ","))
	}

	type window struct{ from, to time.Time }
	var windows []window
	switch {
	case p.From.IsZero() && p.To.IsZero():
		windows = []window{{}}
	case p.From.IsZero():
		return nil, errors.New("datafetch: Fugle To needs a From")
	default:
		from := normalizeDate(p.From)
		to := time.Now().UTC()
		if !p.To.IsZero() {
			to = p.To
		}
		to = normalizeDate(to)
		if from.After(to) {
			return nil, errors.New("datafetch: Fugle From is after To")
		}
		if to.Sub(from) <= fugleMaxSpan {
			windows = []window{{from, to}}
			break
		}
		if tf == FugleTimeframeWeek || tf == FugleTimeframeMonth {
			return nil, errors.New("datafetch: Fugle week and month candles cannot span more than 364 days in one call; ask for a shorter range or use day candles")
		}
		for start := from; !start.After(to); {
			end := start.Add(fugleMaxSpan - 24*time.Hour)
			if end.After(to) {
				end = to
			}
			windows = append(windows, window{start, end})
			start = end.AddDate(0, 0, 1)
		}
	}

	var rows []map[string]any
	for _, w := range windows {
		values := url.Values{}
		for k, v := range base {
			values[k] = v
		}
		if !w.from.IsZero() {
			values.Set("from", w.from.Format("2006-01-02"))
			values.Set("to", w.to.Format("2006-01-02"))
		}
		resp, err := c.fugleGet(ctx, path, values)
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
		rows = append(rows, part...)
	}
	sort.SliceStable(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["date"]) < fmt.Sprint(rows[j]["date"]) })
	out := rows[:0]
	for i, row := range rows {
		if i > 0 && fmt.Sprint(row["date"]) == fmt.Sprint(rows[i-1]["date"]) {
			continue
		}
		out = append(out, row)
	}
	return fugleTable(out, fugleCandleColumns), nil
}

// FugleIntradayCandlesParams are the settings of StockIntradayCandles.
type FugleIntradayCandlesParams struct {
	// Timeframe is the candle length in minutes, FugleTimeframe1Min to
	// FugleTimeframe60Min. Empty means the API default.
	Timeframe FugleTimeframe
	// OddLot asks for odd-lot trading.
	OddLot bool
	// Descending returns the newest candle first. The default is oldest first.
	Descending bool
}

// StockIntradayCandles returns today's candles of a stock. It is
// StockIntradayCandlesContext with context.Background().
func (c *FugleClient) StockIntradayCandles(symbol string, p FugleIntradayCandlesParams) (*insyra.DataTable, error) {
	return c.StockIntradayCandlesContext(context.Background(), symbol, p)
}

// StockIntradayCandlesContext is StockIntradayCandles with a context. The
// columns are date (a time.Time), open, high, low, close, volume and average.
func (c *FugleClient) StockIntradayCandlesContext(ctx context.Context, symbol string, p FugleIntradayCandlesParams) (*insyra.DataTable, error) {
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
	values := url.Values{}
	if p.Timeframe != "" {
		if !p.Timeframe.minute() {
			return nil, fmt.Errorf("datafetch: Fugle intraday timeframe %q must be 1, 3, 5, 10, 15, 30 or 60 minutes", string(p.Timeframe))
		}
		values.Set("timeframe", string(p.Timeframe))
	}
	if p.OddLot {
		values.Set("type", "oddlot")
	}
	if p.Descending {
		values.Set("sort", "desc")
	}
	resp, err := c.fugleGet(ctx, "/stock/intraday/candles/"+url.PathEscape(symbol), values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, fugleCandleColumns), nil
}

// StockIntradayQuote returns a stock's current quote as a one-row table. It is
// StockIntradayQuoteContext with context.Background().
func (c *FugleClient) StockIntradayQuote(symbol string, oddLot ...bool) (*insyra.DataTable, error) {
	return c.StockIntradayQuoteContext(context.Background(), symbol, oddLot...)
}

var fugleQuoteColumns = []string{
	"date", "type", "exchange", "market", "symbol", "name", "referencePrice", "previousClose",
	"openPrice", "openTime", "highPrice", "highTime", "lowPrice", "lowTime", "closePrice", "closeTime",
	"avgPrice", "change", "changePercent", "amplitude", "lastPrice", "lastSize",
	"bids_1_price", "bids_1_size", "bids_2_price", "bids_2_size", "bids_3_price", "bids_3_size", "bids_4_price", "bids_4_size", "bids_5_price", "bids_5_size",
	"asks_1_price", "asks_1_size", "asks_2_price", "asks_2_size", "asks_3_price", "asks_3_size", "asks_4_price", "asks_4_size", "asks_5_price", "asks_5_size",
	"total_tradeValue", "total_tradeVolume", "total_tradeVolumeAtBid", "total_tradeVolumeAtAsk", "total_transaction", "total_bidOrders", "total_askOrders", "total_bidVolume", "total_askVolume", "total_time",
	"lastTrade_bid", "lastTrade_ask", "lastTrade_price", "lastTrade_size", "lastTrade_time", "lastTrade_serial",
	"lastTrial_bid", "lastTrial_ask", "lastTrial_price", "lastTrial_size", "lastTrial_time", "lastTrial_serial",
	"tradingHalt_isHalted", "tradingHalt_time", "lastUpdated", "serial",
}

// StockIntradayQuoteContext is StockIntradayQuote with a context. oddLot asks for the odd-lot
// quote; more than one flag is an error. The table has one row. A nested object
// becomes columns named after it and its field (total_tradeVolume,
// lastTrade_price), and the best bids and asks become bids_1_price,
// bids_1_size, asks_1_price and so on, best first. A field the API leaves out is
// a missing column, not a zero.
func (c *FugleClient) StockIntradayQuoteContext(ctx context.Context, symbol string, oddLot ...bool) (*insyra.DataTable, error) {
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
	odd, err := oneFlag(oddLot, "oddLot")
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	if odd {
		values.Set("type", "oddlot")
	}
	resp, err := c.fugleGet(ctx, "/stock/intraday/quote/"+url.PathEscape(symbol), values)
	if err != nil {
		return nil, err
	}
	flat := map[string]any{}
	flattenFugle("", resp, flat)
	return fugleTable([]map[string]any{flat}, fugleQuoteColumns), nil
}

// flattenFugle copies m into out, joining nested names with "_" and numbering
// array items from 1.
func flattenFugle(prefix string, v any, out map[string]any) {
	switch x := v.(type) {
	case map[string]any:
		for k, item := range x {
			name := k
			if prefix != "" {
				name = prefix + "_" + k
			}
			flattenFugle(name, item, out)
		}
	case []any:
		for i, item := range x {
			flattenFugle(prefix+"_"+strconv.Itoa(i+1), item, out)
		}
	default:
		out[prefix] = v
	}
}

// FugleIntradayTickersParams are the settings of StockIntradayTickers.
type FugleIntradayTickersParams struct {
	// Type is the kind of security. It is required.
	Type FugleTickerType
	// Exchange is TWSE or TPEx; Market is TSE, OTC, ESB or TIB; Industry is an
	// industry code. Empty means any.
	Exchange, Market, Industry string
	// IsNormal, IsAttention, IsDisposition and IsHalted narrow the list to such
	// stocks. When any is set the API lists equities only and ignores Type.
	IsNormal, IsAttention, IsDisposition, IsHalted bool
}

// StockIntradayTickers returns the list of securities. It is StockIntradayTickersContext with
// context.Background().
func (c *FugleClient) StockIntradayTickers(p FugleIntradayTickersParams) (*insyra.DataTable, error) {
	return c.StockIntradayTickersContext(context.Background(), p)
}

// StockIntradayTickersContext is StockIntradayTickers with a context. The columns are symbol,
// name and industry, which is nil for a ticker without one.
func (c *FugleClient) StockIntradayTickersContext(ctx context.Context, p FugleIntradayTickersParams) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	switch p.Type {
	case FugleTickerEquity, FugleTickerIndex, FugleTickerWarrant, FugleTickerOddLot:
	default:
		return nil, fmt.Errorf("datafetch: Fugle ticker type %q must be EQUITY, INDEX, WARRANT or ODDLOT", string(p.Type))
	}
	values := url.Values{"type": {string(p.Type)}}
	for name, v := range map[string]string{"exchange": p.Exchange, "market": p.Market, "industry": p.Industry} {
		if v != "" {
			values.Set(name, v)
		}
	}
	for name, v := range map[string]bool{"isNormal": p.IsNormal, "isAttention": p.IsAttention, "isDisposition": p.IsDisposition, "isHalted": p.IsHalted} {
		if v {
			values.Set(name, "true")
		}
	}
	resp, err := c.fugleGet(ctx, "/stock/intraday/tickers", values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, []string{"symbol", "name", "industry"}), nil
}

// FugleIntradayTradesParams are the settings of StockIntradayTrades.
type FugleIntradayTradesParams struct {
	// OddLot asks for odd-lot trades.
	OddLot bool
	// Offset skips that many trades and Limit returns at most that many; zero
	// means the API default. Neither may be negative.
	Offset, Limit int
	// Ascending returns the oldest trade first. The default is newest first.
	Ascending bool
	// Trial asks for the trial-match details.
	Trial bool
}

// StockIntradayTrades returns today's trades of a stock. It is StockIntradayTradesContext with
// context.Background().
func (c *FugleClient) StockIntradayTrades(symbol string, p FugleIntradayTradesParams) (*insyra.DataTable, error) {
	return c.StockIntradayTradesContext(context.Background(), symbol, p)
}

// StockIntradayTradesContext is StockIntradayTrades with a context. The columns are bid, ask,
// price, size, volume, time and serial; bid, ask and volume are nil where the
// API leaves them out. time is the API's number, unchanged.
func (c *FugleClient) StockIntradayTradesContext(ctx context.Context, symbol string, p FugleIntradayTradesParams) (*insyra.DataTable, error) {
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
	if p.Offset < 0 || p.Limit < 0 {
		return nil, errors.New("datafetch: Fugle Offset and Limit must not be negative")
	}
	values := url.Values{}
	if p.OddLot {
		values.Set("type", "oddlot")
	}
	if p.Offset > 0 {
		values.Set("offset", strconv.Itoa(p.Offset))
	}
	if p.Limit > 0 {
		values.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Ascending {
		values.Set("sort", "asc")
	}
	if p.Trial {
		values.Set("isTrial", "true")
	}
	resp, err := c.fugleGet(ctx, "/stock/intraday/trades/"+url.PathEscape(symbol), values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, []string{"bid", "ask", "price", "size", "volume", "time", "serial"}), nil
}

// StockIntradayVolumes returns today's volume at each price. It is StockIntradayVolumesContext
// with context.Background().
func (c *FugleClient) StockIntradayVolumes(symbol string, oddLot ...bool) (*insyra.DataTable, error) {
	return c.StockIntradayVolumesContext(context.Background(), symbol, oddLot...)
}

// StockIntradayVolumesContext is StockIntradayVolumes with a context. The columns are price,
// volume, volumeAtBid and volumeAtAsk. The two at-bid and at-ask volumes may not
// add up to volume, because the first trade of the session is not split.
func (c *FugleClient) StockIntradayVolumesContext(ctx context.Context, symbol string, oddLot ...bool) (*insyra.DataTable, error) {
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
	odd, err := oneFlag(oddLot, "oddLot")
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	if odd {
		values.Set("type", "oddlot")
	}
	resp, err := c.fugleGet(ctx, "/stock/intraday/volumes/"+url.PathEscape(symbol), values)
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	return fugleTable(rows, []string{"price", "volume", "volumeAtBid", "volumeAtAsk"}), nil
}
