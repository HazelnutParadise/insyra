package datafetch

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/limiter"
	"github.com/HazelnutParadise/insyra/internal/utils"
	yfclient "github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
	yfticker "github.com/wnjoon/go-yfinance/pkg/ticker"
)

// defaults and config
const (
	defaultYFTimeout     = 15 * time.Second
	defaultYFUserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"
	defaultYFBackoff     = 300 * time.Millisecond
	defaultYFConcurrency = 6
)

type YFinanceConfig struct {
	// Timeout: 單次請求最多等待多久（避免卡死）
	Timeout time.Duration

	// Interval: 相鄰兩次請求排定的開始時間最少要隔多久（節流），請求不會早於排定時間開始
	// 0 表示不節流
	Interval time.Duration

	// UserAgent: HTTP User-Agent
	UserAgent string

	// Retries: 失敗時重試次數（0 表示不重試）
	Retries int

	// RetryBackoff: 每次重試前等待多久（0 表示用預設）
	RetryBackoff time.Duration

	// Concurrency: 多 ticker 並行抓取時的最大並行數（0 表示用預設）
	Concurrency int
}

// YFHistoryParams selects the bars History returns.
type YFHistoryParams struct {
	// Period is the range to fetch: 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y,
	// ytd or max.
	Period string `json:"period,omitempty"`

	// Interval is the bar size: 1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d,
	// 1wk, 1mo or 3mo.
	Interval string `json:"interval,omitempty"`

	// Start and End bound the range by date instead of Period.
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`

	// PrePost includes pre- and post-market bars.
	PrePost bool `json:"prepost,omitempty"`

	// AutoAdjust adjusts open, high, low and close for splits and dividends.
	AutoAdjust bool `json:"autoAdjust,omitempty"`

	// Actions includes dividend and split events.
	Actions bool `json:"actions,omitempty"`

	// Repair repairs bad data, such as prices off by 100x and missing values.
	Repair bool `json:"repair,omitempty"`

	// RepairOptions chooses which repairs Repair applies. Nil applies all of
	// them.
	RepairOptions *YFRepairOptions `json:"repairOptions,omitempty"`

	// KeepNA keeps rows whose values are missing.
	KeepNA bool `json:"keepna,omitempty"`
}

// YFRepairOptions chooses which repairs YFHistoryParams.Repair applies.
type YFRepairOptions struct {
	// FixUnitMixups repairs prices off by 100x from a currency unit mix-up,
	// such as dollars and cents or pounds and pence.
	FixUnitMixups bool `json:"fixUnitMixups,omitempty"`

	// FixZeroes repairs missing or zero prices.
	FixZeroes bool `json:"fixZeroes,omitempty"`

	// FixSplits repairs bad stock split adjustments.
	FixSplits bool `json:"fixSplits,omitempty"`

	// FixDividends repairs bad dividend adjustments.
	FixDividends bool `json:"fixDividends,omitempty"`

	// FixCapitalGains repairs capital gains counted twice, for ETFs and mutual
	// funds.
	FixCapitalGains bool `json:"fixCapitalGains,omitempty"`
}

// toModel converts p to go-yfinance's parameters, field by field, so that a
// field go-yfinance renames fails to compile here. The result shares no
// memory with p.
func (p YFHistoryParams) toModel() models.HistoryParams {
	out := models.HistoryParams{
		Period:     p.Period,
		Interval:   p.Interval,
		PrePost:    p.PrePost,
		AutoAdjust: p.AutoAdjust,
		Actions:    p.Actions,
		Repair:     p.Repair,
		KeepNA:     p.KeepNA,
	}
	// The dates are copied, not shared: HistoryContext may abandon a request
	// that is still reading them.
	if p.Start != nil {
		start := *p.Start
		out.Start = &start
	}
	if p.End != nil {
		end := *p.End
		out.End = &end
	}
	if o := p.RepairOptions; o != nil {
		out.RepairOptions = &models.RepairOptions{
			FixUnitMixups:   o.FixUnitMixups,
			FixZeroes:       o.FixZeroes,
			FixSplits:       o.FixSplits,
			FixDividends:    o.FixDividends,
			FixCapitalGains: o.FixCapitalGains,
		}
	}
	return out
}

// YFPeriod selects the frequency of a financial statement: YFPeriodAnnual or
// YFPeriodQuarterly. The empty value means YFPeriodAnnual, and a value
// go-yfinance does not accept comes back as an error.
type YFPeriod string

const (
	YFPeriodAnnual YFPeriod = "annual"

	// Deprecated: use YFPeriodAnnual, which fetches the same statements. This
	// spelling labels the tables it returns "yearly" rather than "annual".
	// Removed in the release after the one that deprecated it.
	YFPeriodYearly YFPeriod = "yearly"

	YFPeriodQuarterly YFPeriod = "quarterly"
)

// YFNewsTab selects which articles News returns.
type YFNewsTab string

const (
	YFNewsTabNews          YFNewsTab = "news"           // news articles; the empty tab means this too
	YFNewsTabAll           YFNewsTab = "all"            // news articles and press releases
	YFNewsTabPressReleases YFNewsTab = "press releases" // press releases only
)

// toModel converts tab to go-yfinance's tab. A tab News does not know is an
// error rather than news, which is what go-yfinance would fetch for it.
func (tab YFNewsTab) toModel() (models.NewsTab, error) {
	switch tab {
	case "", YFNewsTabNews:
		return models.NewsTabNews, nil
	case YFNewsTabAll:
		return models.NewsTabAll, nil
	case YFNewsTabPressReleases:
		return models.NewsTabPressReleases, nil
	default:
		return "", fmt.Errorf("yfinance: unknown news tab %q; use YFNewsTabNews, YFNewsTabAll or YFNewsTabPressReleases", string(tab))
	}
}

func (cfg YFinanceConfig) normalize() (YFinanceConfig, error) {
	out := cfg

	if out.Timeout <= 0 {
		out.Timeout = defaultYFTimeout
	}

	if out.Interval < 0 {
		return YFinanceConfig{}, errors.New("yfinance: Interval must be >= 0")
	}

	if out.UserAgent == "" {
		out.UserAgent = defaultYFUserAgent
	}

	if out.Retries < 0 {
		return YFinanceConfig{}, errors.New("yfinance: Retries must be >= 0")
	}
	if out.RetryBackoff < 0 {
		return YFinanceConfig{}, errors.New("yfinance: RetryBackoff must be >= 0")
	}
	if out.RetryBackoff == 0 {
		out.RetryBackoff = defaultYFBackoff
	}
	if out.Concurrency < 0 {
		return YFinanceConfig{}, errors.New("yfinance: Concurrency must be >= 0")
	}
	if out.Concurrency == 0 {
		out.Concurrency = defaultYFConcurrency
	}

	return out, nil
}

// YFinanceClient fetches Yahoo Finance data. Each client has its own request
// limiter. Create one with YFinance. A client that did not come from YFinance,
// such as a zero value, returns an error from every method.
type YFinanceClient struct {
	cfg     YFinanceConfig
	client  *yfclient.Client
	limiter *limiter.IntervalLimiter
	// timeoutSeconds holds the timeout value passed to the underlying client in seconds.
	// This allows tests to verify the unit conversion from time.Duration.
	timeoutSeconds int
}

// YFinance creates a YahooFinance fetcher using a config struct (no WithXxx in public API).
func YFinance(cfg YFinanceConfig) (*YFinanceClient, error) {
	normalized, err := cfg.normalize()
	if err != nil {
		return nil, err
	}

	secs := int(normalized.Timeout / time.Second)
	c, err := yfclient.New(
		// WithTimeout expects an integer timeout value in seconds.
		yfclient.WithTimeout(secs),
		yfclient.WithUserAgent(normalized.UserAgent),
	)
	if err != nil {
		return nil, err
	}

	yf := &YFinanceClient{
		cfg:            normalized,
		client:         c,
		limiter:        limiter.NewIntervalLimiter(normalized.Interval),
		timeoutSeconds: secs,
	}

	// ensure resources are cleaned up automatically when yf is garbage-collected
	runtime.SetFinalizer(yf, func(y *YFinanceClient) { y.close() })

	return yf, nil
}

// lifecycle (internal)
// close closes underlying resources. It is unexported because resources are
// managed automatically; callers do not need to call this directly.
func (y *YFinanceClient) close() {
	if y == nil || y.client == nil {
		return
	}
	y.client.Close()
}

// helpers

func (y *YFinanceClient) beforeRequest(ctx context.Context) error {
	return y.limiter.Wait(ctx)
}

func (y *YFinanceClient) sleepBackoff(ctx context.Context, attempt int) error {
	// attempt: 0,1,2...
	if y.cfg.RetryBackoff <= 0 {
		return nil
	}
	return sleepContext(ctx, y.cfg.RetryBackoff*time.Duration(attempt+1))
}

// runYF runs call, one go-yfinance call, under ctx. go-yfinance cannot stop a
// call once it has started, and a call can send several requests, so when ctx
// can be cancelled the call runs on its own goroutine and runYF stops waiting
// for it once ctx is done: it returns ctx.Err() at once, and the call runs to
// its end in the background, where it may still send its remaining requests,
// with its result discarded. A panic in the call comes back as an error, since
// the caller could not recover it on that goroutine. With a context that is
// never done, such as context.Background(), the call runs on the caller's
// goroutine.
func runYF[T any](ctx context.Context, call func() (T, error)) (T, error) {
	var zero T
	if err := contextErr(ctx); err != nil {
		return zero, err
	}
	if ctx.Done() == nil {
		return call()
	}
	type outcome struct {
		value T
		err   error
	}
	// Room for one value, so the goroutine never blocks after runYF has
	// stopped waiting for it.
	done := make(chan outcome, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- outcome{err: fmt.Errorf("yfinance: request panicked: %v", r)}
			}
		}()
		value, err := call()
		done <- outcome{value: value, err: err}
	}()
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case o := <-done:
		return o.value, o.err
	}
}

// tickerValue runs one go-yfinance request for t's symbol under ctx.
func tickerValue[T any](ctx context.Context, t *YFTicker, call func(*yfticker.Ticker) (T, error)) (T, error) {
	var zero T
	if err := t.checkError(); err != nil {
		return zero, err
	}
	if err := contextErr(ctx); err != nil {
		return zero, err
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return zero, err
	}
	// A ticker built on a shared client leaves the client open when closed, so
	// closing tk while an abandoned request still uses it is safe.
	defer tk.Close()
	return runYF(ctx, func() (T, error) {
		// Keep the client reachable until the call ends. Without it an
		// abandoned call could outlive every other reference, and the
		// finalizer's Close would block on the request still holding the
		// client, stalling every finalizer in the program.
		defer runtime.KeepAlive(t.yf)
		return call(tk)
	})
}

// tickerTable runs one go-yfinance request for t's symbol under ctx and turns
// its result into a table named SYMBOL.name.
func tickerTable[T any](ctx context.Context, t *YFTicker, name string, convertDates bool, call func(*yfticker.Ticker) (T, error)) (*insyra.DataTable, error) {
	value, err := tickerValue(ctx, t, call)
	if err != nil {
		return nil, err
	}
	return namedTable(value, t.symbol, name, convertDates)
}

// namedTable turns a go-yfinance result into a DataTable named SYMBOL.name,
// converting date-like columns to time.Time when convertDates is set.
func namedTable(value any, symbol, name string, convertDates bool) (*insyra.DataTable, error) {
	dt, err := insyra.ReadJSON(value)
	if err != nil {
		return nil, err
	}
	if convertDates {
		dt = normalizeDateColumns(dt)
	}
	dt.SetName(fmt.Sprintf("%s.%s", strings.ToUpper(symbol), name))
	return dt, nil
}

// withRetries runs call the way History and Quote do: each attempt waits for
// the limiter, and a rate limit or a timeout is retried after a backoff, up
// to Retries times. A done ctx stops the wait, the request and the backoff,
// and is returned as ctx.Err() without another attempt.
func withRetries[T any](ctx context.Context, t *YFTicker, what string, call func(*yfticker.Ticker) (T, error)) (T, error) {
	var zero T
	if err := t.checkError(); err != nil {
		return zero, err
	}
	if err := contextErr(ctx); err != nil {
		return zero, err
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return zero, err
	}
	defer tk.Close()

	var lastErr error
	for attempt := 0; attempt <= t.yf.cfg.Retries; attempt++ {
		if err := t.yf.beforeRequest(ctx); err != nil {
			return zero, err
		}
		value, err := runYF(ctx, func() (T, error) {
			defer runtime.KeepAlive(t.yf) // see tickerValue
			return call(tk)
		})
		if err == nil {
			return value, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		lastErr = classifyError(err)
		if !retryable(lastErr) {
			return zero, lastErr
		}
		if attempt < t.yf.cfg.Retries {
			if err := t.yf.sleepBackoff(ctx, attempt); err != nil {
				return zero, err
			}
		}
	}
	return zero, fmt.Errorf("yfinance: %s failed: %w", what, lastErr)
}

// normalizeDateColumns converts any string columns whose name suggests a date/time into time.Time
// with time truncated to date-only (midnight in original location).
func normalizeDateColumns(dt *insyra.DataTable) *insyra.DataTable {
	return dt.Map(func(rowIndex int, colIndex string, element any) any {
		name := dt.GetColNameByIndex(colIndex)
		// Heuristics: conservatively detect date/time columns by extracting the last semantic
		// token from the column name (handles snake_case and camelCase) and matching it
		// against a small whitelist. This avoids accidental matches like "notadate".
		convert := false

		getLastToken := func(s string) string {
			if s == "" {
				return ""
			}
			// snake_case: prefer splitting on '_'
			if strings.Contains(s, "_") {
				parts := strings.Split(s, "_")
				return strings.ToLower(parts[len(parts)-1])
			}
			// camelCase / PascalCase: split on uppercase transitions
			var parts []string
			var cur []rune
			for i, r := range s {
				if i > 0 && unicode.IsUpper(r) {
					parts = append(parts, strings.ToLower(string(cur)))
					cur = []rune{r}
				} else {
					cur = append(cur, r)
				}
			}
			if len(cur) > 0 {
				parts = append(parts, strings.ToLower(string(cur)))
			}
			if len(parts) == 0 {
				return strings.ToLower(s)
			}
			return parts[len(parts)-1]
		}

		last := getLastToken(name)
		if last == "date" || last == "time" || last == "expiry" || last == "expire" || strings.EqualFold(name, "date") || strings.EqualFold(name, "time") {
			convert = true
		}
		if convert {
			switch v := element.(type) {
			case string:
				if parsed, ok := utils.TryParseTime(v); ok {
					return parsed
				}
			case []any:
				out := make([]any, len(v))
				for i, item := range v {
					if str, ok := item.(string); ok {
						if parsed, ok := utils.TryParseTime(str); ok {
							out[i] = parsed
							continue
						}
					}
					out[i] = item
				}
				return out
			}
		}
		return element
	})
}

// High-level Python-like API

// YFTicker is a handle on one symbol, bound to the YFinanceClient that made it,
// with methods similar to Python yfinance's Ticker. Get one from
// (*YFinanceClient).Ticker. A ticker that did not come from a YFinanceClient
// made by YFinance, such as a zero value, returns an error from every method.
//
// Each method that requests data has a Context form. A method whose context is
// already done sends nothing. When the context ends while a call is running,
// the method returns ctx.Err() at once and abandons the call, which
// go-yfinance cannot stop: it runs to its end in the background and may still
// send its remaining requests, and a later call on the same client may have to
// wait for it.
type YFTicker struct {
	yf     *YFinanceClient
	symbol string
	err    error
}

// Ticker returns a YFTicker bound to this YFinanceClient instance.
// The ticker does not manage the lifecycle of the underlying client; resources
// are automatically cleaned up when the fetcher is no longer referenced.
func (y *YFinanceClient) Ticker(symbol string) *YFTicker {
	if y == nil {
		return &YFTicker{err: errors.New("yfinance: YFinanceClient is nil; create one with YFinance")}
	}
	return &YFTicker{yf: y, symbol: symbol}
}

func (t *YFTicker) checkError() error {
	if t == nil {
		return errors.New("yfinance: YFTicker is nil; get one from a YFinanceClient made by YFinance")
	}
	if t.err != nil {
		return t.err
	}
	if t.yf == nil {
		return errors.New("yfinance: YFTicker was not created by (*YFinanceClient).Ticker; create a client with YFinance")
	}
	if t.yf.client == nil {
		return errors.New("yfinance: YFinanceClient was not created with YFinance")
	}
	return nil
}

// History returns historical OHLCV bars as an insyra.DataTable. It is
// HistoryContext with context.Background().
func (t *YFTicker) History(params YFHistoryParams) (*insyra.DataTable, error) {
	return t.HistoryContext(context.Background(), params)
}

// HistoryContext is History with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) HistoryContext(ctx context.Context, params YFHistoryParams) (*insyra.DataTable, error) {
	// Convert before the request starts, so the call reads only its own copy.
	model := params.toModel()
	bars, err := withRetries(ctx, t, "history", func(tk *yfticker.Ticker) ([]models.Bar, error) {
		return tk.History(model)
	})
	if err != nil {
		return nil, err
	}
	return namedTable(bars, t.symbol, "History", true)
}

// Quote returns quote information for the ticker as an insyra.DataTable. It is
// QuoteContext with context.Background().
func (t *YFTicker) Quote() (*insyra.DataTable, error) {
	return t.QuoteContext(context.Background())
}

// QuoteContext is Quote with a context; see YFTicker for what a done context
// does.
func (t *YFTicker) QuoteContext(ctx context.Context) (*insyra.DataTable, error) {
	q, err := withRetries(ctx, t, "quote", (*yfticker.Ticker).Quote)
	if err != nil {
		return nil, err
	}
	return namedTable(q, t.symbol, "Quote", true)
}

// Info returns metadata for the ticker as a DataTable. It is InfoContext with
// context.Background().
func (t *YFTicker) Info() (*insyra.DataTable, error) {
	return t.InfoContext(context.Background())
}

// InfoContext is Info with a context; see YFTicker for what a done context
// does.
func (t *YFTicker) InfoContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Info", true, (*yfticker.Ticker).Info)
}

// Dividends returns dividends history for the ticker as a DataTable. It is
// DividendsContext with context.Background().
func (t *YFTicker) Dividends() (*insyra.DataTable, error) {
	return t.DividendsContext(context.Background())
}

// DividendsContext is Dividends with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) DividendsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Dividends", true, (*yfticker.Ticker).Dividends)
}

// Splits returns stock splits history for the ticker as a DataTable. It is
// SplitsContext with context.Background().
func (t *YFTicker) Splits() (*insyra.DataTable, error) {
	return t.SplitsContext(context.Background())
}

// SplitsContext is Splits with a context; see YFTicker for what a done context
// does.
func (t *YFTicker) SplitsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Splits", true, (*yfticker.Ticker).Splits)
}

// Actions returns corporate actions (dividends + splits) as a DataTable. It is
// ActionsContext with context.Background().
func (t *YFTicker) Actions() (*insyra.DataTable, error) {
	return t.ActionsContext(context.Background())
}

// ActionsContext is Actions with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) ActionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Actions", true, (*yfticker.Ticker).Actions)
}

// Options returns the list of option expiration dates (like `Ticker.options`).
// It is OptionsContext with context.Background().
func (t *YFTicker) Options() (*insyra.DataTable, error) {
	return t.OptionsContext(context.Background())
}

// OptionsContext is Options with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) OptionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Options", true, (*yfticker.Ticker).Options)
}

// OptionChain returns option chain data split into calls/puts/underlying tables.
// It is OptionChainContext with context.Background().
func (t *YFTicker) OptionChain(date string) (*YFOptionChainTables, error) {
	return t.OptionChainContext(context.Background(), date)
}

// OptionChainContext is OptionChain with a context; see YFTicker for what a
// done context does.
func (t *YFTicker) OptionChainContext(ctx context.Context, date string) (*YFOptionChainTables, error) {
	chain, err := tickerValue(ctx, t, func(tk *yfticker.Ticker) (*models.OptionChain, error) {
		return tk.OptionChain(date)
	})
	if err != nil {
		return nil, err
	}
	return buildOptionChainTables(t.symbol, date, chain)
}

// News fetches up to count news articles for this ticker; count <= 0 means 10.
// An empty tab means YFNewsTabNews, and a tab outside the three YFNewsTab
// values is an error, returned before any request. It is NewsContext with
// context.Background().
func (t *YFTicker) News(count int, tab YFNewsTab) (*insyra.DataTable, error) {
	return t.NewsContext(context.Background(), count, tab)
}

// NewsContext is News with a context; see YFTicker for what a done context
// does.
func (t *YFTicker) NewsContext(ctx context.Context, count int, tab YFNewsTab) (*insyra.DataTable, error) {
	if err := t.checkError(); err != nil {
		return nil, err
	}
	modelTab, err := tab.toModel()
	if err != nil {
		return nil, err
	}
	return tickerTable(ctx, t, "News", true, func(tk *yfticker.Ticker) ([]models.NewsArticle, error) {
		return tk.News(count, modelTab)
	})
}

// Calendar returns upcoming calendar events (earnings, dividends) for the
// ticker. It is CalendarContext with context.Background().
func (t *YFTicker) Calendar() (*insyra.DataTable, error) {
	return t.CalendarContext(context.Background())
}

// CalendarContext is Calendar with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) CalendarContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Calendar", true, (*yfticker.Ticker).Calendar)
}

// Financials: IncomeStatement / BalanceSheet / CashFlow
// IncomeStatement returns multi-table statements (values/items/meta). It is
// IncomeStatementContext with context.Background().
func (t *YFTicker) IncomeStatement(freq YFPeriod) (*YFFinancialStatementTables, error) {
	return t.IncomeStatementContext(context.Background(), freq)
}

// IncomeStatementContext is IncomeStatement with a context; see YFTicker for
// what a done context does.
func (t *YFTicker) IncomeStatementContext(ctx context.Context, freq YFPeriod) (*YFFinancialStatementTables, error) {
	stmt, err := tickerValue(ctx, t, func(tk *yfticker.Ticker) (*models.FinancialStatement, error) {
		return tk.IncomeStatement(string(freq))
	})
	if err != nil {
		return nil, err
	}
	return buildFinancialStatementTables(t.symbol, "IncomeStatement", freq, stmt)
}

// BalanceSheet returns multi-table statements (values/items/meta). It is
// BalanceSheetContext with context.Background().
func (t *YFTicker) BalanceSheet(freq YFPeriod) (*YFFinancialStatementTables, error) {
	return t.BalanceSheetContext(context.Background(), freq)
}

// BalanceSheetContext is BalanceSheet with a context; see YFTicker for what a
// done context does.
func (t *YFTicker) BalanceSheetContext(ctx context.Context, freq YFPeriod) (*YFFinancialStatementTables, error) {
	stmt, err := tickerValue(ctx, t, func(tk *yfticker.Ticker) (*models.FinancialStatement, error) {
		return tk.BalanceSheet(string(freq))
	})
	if err != nil {
		return nil, err
	}
	return buildFinancialStatementTables(t.symbol, "BalanceSheet", freq, stmt)
}

// CashFlow returns multi-table statements (values/items/meta). It is
// CashFlowContext with context.Background().
func (t *YFTicker) CashFlow(freq YFPeriod) (*YFFinancialStatementTables, error) {
	return t.CashFlowContext(context.Background(), freq)
}

// CashFlowContext is CashFlow with a context; see YFTicker for what a done
// context does.
func (t *YFTicker) CashFlowContext(ctx context.Context, freq YFPeriod) (*YFFinancialStatementTables, error) {
	stmt, err := tickerValue(ctx, t, func(tk *yfticker.Ticker) (*models.FinancialStatement, error) {
		return tk.CashFlow(string(freq))
	})
	if err != nil {
		return nil, err
	}
	return buildFinancialStatementTables(t.symbol, "CashFlow", freq, stmt)
}

// Holders

// MajorHolders returns major holders information as a DataTable. It is
// MajorHoldersContext with context.Background().
func (t *YFTicker) MajorHolders() (*insyra.DataTable, error) {
	return t.MajorHoldersContext(context.Background())
}

// MajorHoldersContext is MajorHolders with a context; see YFTicker for what a
// done context does.
func (t *YFTicker) MajorHoldersContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "MajorHolders", true, (*yfticker.Ticker).MajorHolders)
}

func (t *YFTicker) InstitutionalHolders() (*insyra.DataTable, error) {
	return t.InstitutionalHoldersContext(context.Background())
}

// InstitutionalHoldersContext is InstitutionalHolders with a context; see YFTicker for what a done context does.
func (t *YFTicker) InstitutionalHoldersContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "InstitutionalHolders", true, (*yfticker.Ticker).InstitutionalHolders)
}

func (t *YFTicker) MutualFundHolders() (*insyra.DataTable, error) {
	return t.MutualFundHoldersContext(context.Background())
}

// MutualFundHoldersContext is MutualFundHolders with a context; see YFTicker for what a done context does.
func (t *YFTicker) MutualFundHoldersContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "MutualFundHolders", true, (*yfticker.Ticker).MutualFundHolders)
}

func (t *YFTicker) InsiderTransactions() (*insyra.DataTable, error) {
	return t.InsiderTransactionsContext(context.Background())
}

// InsiderTransactionsContext is InsiderTransactions with a context; see YFTicker for what a done context does.
func (t *YFTicker) InsiderTransactionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "InsiderTransactions", true, (*yfticker.Ticker).InsiderTransactions)
}

// FastInfo returns a quick summary as DataTable. It is FastInfoContext with context.Background().
func (t *YFTicker) FastInfo() (*insyra.DataTable, error) {
	return t.FastInfoContext(context.Background())
}

// FastInfoContext is FastInfo with a context; see YFTicker for what a done context does.
func (t *YFTicker) FastInfoContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "FastInfo", true, (*yfticker.Ticker).FastInfo)
}

// Earnings returns earnings report data for the ticker.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *YFTicker) Earnings() (*insyra.DataTable, error) {
	if err := t.checkError(); err != nil {
		return nil, err
	}
	return nil, errors.New("yfinance: Earnings not supported by the go-yfinance backend")
}

// EarningsEstimate returns earnings estimates as a DataTable. It is EarningsEstimateContext with context.Background().
func (t *YFTicker) EarningsEstimate() (*insyra.DataTable, error) {
	return t.EarningsEstimateContext(context.Background())
}

// EarningsEstimateContext is EarningsEstimate with a context; see YFTicker for what a done context does.
func (t *YFTicker) EarningsEstimateContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "EarningsEstimate", true, (*yfticker.Ticker).EarningsEstimate)
}

// EarningsHistory returns historical earnings data as a DataTable. It is EarningsHistoryContext with context.Background().
func (t *YFTicker) EarningsHistory() (*insyra.DataTable, error) {
	return t.EarningsHistoryContext(context.Background())
}

// EarningsHistoryContext is EarningsHistory with a context; see YFTicker for what a done context does.
func (t *YFTicker) EarningsHistoryContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "EarningsHistory", true, (*yfticker.Ticker).EarningsHistory)
}

// EPSTrend returns EPS trend data as a DataTable. It is EPSTrendContext with context.Background().
func (t *YFTicker) EPSTrend() (*insyra.DataTable, error) {
	return t.EPSTrendContext(context.Background())
}

// EPSTrendContext is EPSTrend with a context; see YFTicker for what a done context does.
func (t *YFTicker) EPSTrendContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "EPSTrend", true, (*yfticker.Ticker).EPSTrend)
}

// EPSRevisions returns EPS revisions data as a DataTable. It is EPSRevisionsContext with context.Background().
func (t *YFTicker) EPSRevisions() (*insyra.DataTable, error) {
	return t.EPSRevisionsContext(context.Background())
}

// EPSRevisionsContext is EPSRevisions with a context; see YFTicker for what a done context does.
func (t *YFTicker) EPSRevisionsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "EPSRevisions", false, (*yfticker.Ticker).EPSRevisions)
}

// Recommendations returns analyst recommendations as a DataTable. It is RecommendationsContext with context.Background().
func (t *YFTicker) Recommendations() (*insyra.DataTable, error) {
	return t.RecommendationsContext(context.Background())
}

// RecommendationsContext is Recommendations with a context; see YFTicker for what a done context does.
func (t *YFTicker) RecommendationsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "Recommendations", false, (*yfticker.Ticker).Recommendations)
}

// AnalystPriceTargets returns analyst price targets as a DataTable. It is AnalystPriceTargetsContext with context.Background().
func (t *YFTicker) AnalystPriceTargets() (*insyra.DataTable, error) {
	return t.AnalystPriceTargetsContext(context.Background())
}

// AnalystPriceTargetsContext is AnalystPriceTargets with a context; see YFTicker for what a done context does.
func (t *YFTicker) AnalystPriceTargetsContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "AnalystPriceTargets", false, (*yfticker.Ticker).AnalystPriceTargets)
}

// RevenueEstimate returns revenue estimates as a DataTable. It is RevenueEstimateContext with context.Background().
func (t *YFTicker) RevenueEstimate() (*insyra.DataTable, error) {
	return t.RevenueEstimateContext(context.Background())
}

// RevenueEstimateContext is RevenueEstimate with a context; see YFTicker for what a done context does.
func (t *YFTicker) RevenueEstimateContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "RevenueEstimate", false, (*yfticker.Ticker).RevenueEstimate)
}

// Sustainability returns sustainability data as a DataTable.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *YFTicker) Sustainability() (*insyra.DataTable, error) {
	if err := t.checkError(); err != nil {
		return nil, err
	}
	return nil, errors.New("yfinance: Sustainability not supported by the installed go-yfinance backend")
}

// GrowthEstimates returns growth estimates as a DataTable. It is GrowthEstimatesContext with context.Background().
func (t *YFTicker) GrowthEstimates() (*insyra.DataTable, error) {
	return t.GrowthEstimatesContext(context.Background())
}

// GrowthEstimatesContext is GrowthEstimates with a context; see YFTicker for what a done context does.
func (t *YFTicker) GrowthEstimatesContext(ctx context.Context) (*insyra.DataTable, error) {
	return tickerTable(ctx, t, "GrowthEstimates", false, (*yfticker.Ticker).GrowthEstimates)
}

// FundsData returns fund-related data for ETFs/mutual funds.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *YFTicker) FundsData() (*insyra.DataTable, error) {
	if err := t.checkError(); err != nil {
		return nil, err
	}
	return nil, errors.New("yfinance: FundsData not supported by the installed go-yfinance backend")
}

// TopHoldings returns top holdings for a fund as a DataTable.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *YFTicker) TopHoldings() (*insyra.DataTable, error) {
	if err := t.checkError(); err != nil {
		return nil, err
	}
	return nil, errors.New("yfinance: TopHoldings not supported by the installed go-yfinance backend")
}
