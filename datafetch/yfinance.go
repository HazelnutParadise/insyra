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

	// Interval: 每次請求之間最少要隔多久（節流）
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

type YFHistoryParams = models.HistoryParams

// YFPeriod represents frequency values used for financial statements.
// Accepted values: YFPeriodAnnual, YFPeriodYearly, YFPeriodQuarterly.
// When empty or unrecognized, it defaults to YFPeriodAnnual.
type YFPeriod string

const (
	YFPeriodAnnual    YFPeriod = "annual"
	YFPeriodYearly    YFPeriod = "yearly"
	YFPeriodQuarterly YFPeriod = "quarterly"
)

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

// yahooFinance is a stateful fetcher.
// Each instance has its own interval limiter.
type yahooFinance struct {
	cfg     YFinanceConfig
	client  *yfclient.Client
	limiter *limiter.IntervalLimiter
	// timeoutSeconds holds the timeout value passed to the underlying client in seconds.
	// This allows tests to verify the unit conversion from time.Duration.
	timeoutSeconds int
}

// YFinance creates a YahooFinance fetcher using a config struct (no WithXxx in public API).
func YFinance(cfg YFinanceConfig) (*yahooFinance, error) {
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

	yf := &yahooFinance{
		cfg:            normalized,
		client:         c,
		limiter:        limiter.NewIntervalLimiter(normalized.Interval),
		timeoutSeconds: secs,
	}

	// ensure resources are cleaned up automatically when yf is garbage-collected
	runtime.SetFinalizer(yf, func(y *yahooFinance) { y.close() })

	return yf, nil
}

// lifecycle (internal)
// close closes underlying resources. It is unexported because resources are
// managed automatically; callers do not need to call this directly.
func (y *yahooFinance) close() {
	if y == nil || y.client == nil {
		return
	}
	y.client.Close()
}

// helpers
// NOTE: previously there was a helper newTicker; calls are inlined below
// to avoid an extra indirection and keep client checks local.

func (y *yahooFinance) beforeRequest() error {
	return y.limiter.Wait(context.Background())
}

func (y *yahooFinance) sleepBackoff(attempt int) {
	// attempt: 0,1,2...
	if y.cfg.RetryBackoff <= 0 {
		return
	}
	time.Sleep(y.cfg.RetryBackoff * time.Duration(attempt+1))
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
			if str, ok := element.(string); ok {
				if parsed, ok := utils.TryParseTime(str); ok {
					return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
				}
			}
		}
		return element
	})
}

// public fetch methods
// QuoteRaw fetches quote data for a symbol and returns the library's native quote struct.
// (Use this as your stable base; you can convert it to DataTable later.)
// Quote fetches quote data for a symbol and returns the library's native quote struct.
// Uses instance timeout and retries; callers don't need to pass a context.
// Note: low-level convenience methods like Quote/History/MultiHistory
// were removed from `yahooFinance` to keep a smaller surface API.
// Use `y.Ticker(symbol)` and the returned `ticker` methods instead.

// High-level Python-like API

// ticker wraps a symbol and provides methods similar to python yfinance's Ticker.
type ticker struct {
	yf     *yahooFinance
	symbol string
}

// Ticker returns a ticker bound to this yahooFinance instance.
// The ticker does not manage the lifecycle of the underlying client; resources
// are automatically cleaned up when the fetcher is no longer referenced.
func (y *yahooFinance) Ticker(symbol string) (*ticker, error) {
	if y == nil {
		return nil, errors.New("yfinance: yahooFinance is nil")
	}
	return &ticker{yf: y, symbol: symbol}, nil
}

// History returns historical OHLCV bars as an insyra.DataTable.
func (t *ticker) History(params YFHistoryParams) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	var lastErr error
	for attempt := 0; attempt <= t.yf.cfg.Retries; attempt++ {
		if err := t.yf.beforeRequest(); err != nil {
			return nil, err
		}

		bars, err := tk.History(models.HistoryParams(params))
		if err == nil {
			dt, err := insyra.ReadJSON(bars)
			if err != nil {
				return nil, err
			}
			dt = normalizeDateColumns(dt)
			dt.SetName(fmt.Sprintf("%s.History", strings.ToUpper(t.symbol)))
			return dt, nil
		}
		lastErr = classifyError(err)
		if !retryable(lastErr) {
			return nil, lastErr
		}
		if attempt < t.yf.cfg.Retries {
			t.yf.sleepBackoff(attempt)
		}
	}

	return nil, fmt.Errorf("yfinance: history failed: %w", lastErr)
}

// Quote returns quote information for the ticker as an insyra.DataTable.
func (t *ticker) Quote() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	var lastErr error
	for attempt := 0; attempt <= t.yf.cfg.Retries; attempt++ {
		if err := t.yf.beforeRequest(); err != nil {
			return nil, err
		}

		q, err := tk.Quote()
		if err == nil {
			dt, err := insyra.ReadJSON(q)
			if err != nil {
				return nil, err
			}
			dt = normalizeDateColumns(dt)
			dt.SetName(fmt.Sprintf("%s.Quote", strings.ToUpper(t.symbol)))
			return dt, nil
		}
		lastErr = classifyError(err)
		if !retryable(lastErr) {
			return nil, lastErr
		}
		if attempt < t.yf.cfg.Retries {
			t.yf.sleepBackoff(attempt)
		}
	}

	return nil, fmt.Errorf("yfinance: quote failed: %w", lastErr)
}

// Info returns metadata for the ticker as a DataTable.
func (t *ticker) Info() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	info, err := tk.Info()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(info)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Info", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Dividends returns dividends history for the ticker as a DataTable.
func (t *ticker) Dividends() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	divs, err := tk.Dividends()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(divs)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Dividends", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Splits returns stock splits history for the ticker as a DataTable.
func (t *ticker) Splits() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	splits, err := tk.Splits()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(splits)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Splits", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Actions returns corporate actions (dividends + splits) as a DataTable.
func (t *ticker) Actions() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	acts, err := tk.Actions()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(acts)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Actions", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Options returns the list of option expiration dates (like `Ticker.options`).
func (t *ticker) Options() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	exps, err := tk.Options()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(exps)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Options", strings.ToUpper(t.symbol)))
	return dt, nil
}

// OptionChain returns option chain data for a given expiration date.
func (t *ticker) OptionChain(date string) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	chain, err := tk.OptionChain(date)
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(chain)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.OptionChain(%s)", strings.ToUpper(t.symbol), date))
	return dt, nil
}

// News fetches news articles for this ticker.
func (t *ticker) News(count int, tab models.NewsTab) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	articles, err := tk.News(count, tab)
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(articles)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.News", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Calendar returns upcoming calendar events (earnings, dividends) for the ticker.
func (t *ticker) Calendar() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	cal, err := tk.Calendar()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(cal)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.Calendar", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Financials: IncomeStatement / BalanceSheet / CashFlow
// FIXME: the return needs new structure
func (t *ticker) IncomeStatement(freq YFPeriod) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	stmt, err := tk.IncomeStatement(string(freq))
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(stmt)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.IncomeStatement(%s)", strings.ToUpper(t.symbol), string(freq)))
	return dt, nil
}

// FIXME: the return needs new structure
func (t *ticker) BalanceSheet(freq YFPeriod) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	stmt, err := tk.BalanceSheet(string(freq))
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(stmt)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.BalanceSheet(%s)", strings.ToUpper(t.symbol), string(freq)))
	return dt, nil
}

// FIXME: the return needs new structure
func (t *ticker) CashFlow(freq YFPeriod) (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	stmt, err := tk.CashFlow(string(freq))
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(stmt)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.CashFlow(%s)", strings.ToUpper(t.symbol), string(freq)))
	return dt, nil
}

// Holders
func (t *ticker) MajorHolders() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	h, err := tk.MajorHolders()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(h)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.MajorHolders", strings.ToUpper(t.symbol)))
	return dt, nil
}

func (t *ticker) InstitutionalHolders() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	h, err := tk.InstitutionalHolders()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(h)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.InstitutionalHolders", strings.ToUpper(t.symbol)))
	return dt, nil
}

func (t *ticker) MutualFundHolders() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	h, err := tk.MutualFundHolders()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(h)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.MutualFundHolders", strings.ToUpper(t.symbol)))
	return dt, nil
}

func (t *ticker) InsiderTransactions() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	tx, err := tk.InsiderTransactions()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(tx)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.InsiderTransactions", strings.ToUpper(t.symbol)))
	return dt, nil
}

// FastInfo returns a quick summary as DataTable.
func (t *ticker) FastInfo() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	fi, err := tk.FastInfo()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(fi)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.FastInfo", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Earnings returns earnings report data for the ticker.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *ticker) Earnings() (*insyra.DataTable, error) {
	return nil, errors.New("yfinance: Earnings not supported by the go-yfinance backend")
}

// EarningsEstimate returns earnings estimates as a DataTable.
func (t *ticker) EarningsEstimate() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	est, err := tk.EarningsEstimate()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(est)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.EarningsEstimate", strings.ToUpper(t.symbol)))
	return dt, nil
}

// EarningsHistory returns historical earnings data as a DataTable.
func (t *ticker) EarningsHistory() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	hs, err := tk.EarningsHistory()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(hs)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.EarningsHistory", strings.ToUpper(t.symbol)))
	return dt, nil
}

// EPSTrend returns EPS trend data as a DataTable.
func (t *ticker) EPSTrend() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	et, err := tk.EPSTrend()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(et)
	if err != nil {
		return nil, err
	}
	dt = normalizeDateColumns(dt)
	dt.SetName(fmt.Sprintf("%s.EPSTrend", strings.ToUpper(t.symbol)))
	return dt, nil
}

// EPSRevisions returns EPS revisions data as a DataTable.
func (t *ticker) EPSRevisions() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	rev, err := tk.EPSRevisions()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(rev)
	if err != nil {
		return nil, err
	}
	dt.SetName(fmt.Sprintf("%s.EPSRevisions", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Recommendations returns analyst recommendations as a DataTable.
func (t *ticker) Recommendations() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	recs, err := tk.Recommendations()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(recs)
	if err != nil {
		return nil, err
	}
	dt.SetName(fmt.Sprintf("%s.Recommendations", strings.ToUpper(t.symbol)))
	return dt, nil
}

// AnalystPriceTargets returns analyst price targets as a DataTable.
func (t *ticker) AnalystPriceTargets() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	apt, err := tk.AnalystPriceTargets()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(apt)
	if err != nil {
		return nil, err
	}
	dt.SetName(fmt.Sprintf("%s.AnalystPriceTargets", strings.ToUpper(t.symbol)))
	return dt, nil
}

// RevenueEstimate returns revenue estimates as a DataTable.
func (t *ticker) RevenueEstimate() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	rev, err := tk.RevenueEstimate()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(rev)
	if err != nil {
		return nil, err
	}
	dt.SetName(fmt.Sprintf("%s.RevenueEstimate", strings.ToUpper(t.symbol)))
	return dt, nil
}

// Sustainability returns sustainability data as a DataTable.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *ticker) Sustainability() (*insyra.DataTable, error) {
	return nil, errors.New("yfinance: Sustainability not supported by the installed go-yfinance backend")
}

// GrowthEstimates returns growth estimates as a DataTable.
func (t *ticker) GrowthEstimates() (*insyra.DataTable, error) {
	if t == nil || t.yf == nil {
		return nil, errors.New("yfinance: ticker is nil")
	}
	if t.yf.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	tk, err := yfticker.New(t.symbol, yfticker.WithClient(t.yf.client))
	if err != nil {
		return nil, err
	}
	defer tk.Close()

	g, err := tk.GrowthEstimates()
	if err != nil {
		return nil, err
	}
	dt, err := insyra.ReadJSON(g)
	if err != nil {
		return nil, err
	}
	dt.SetName(fmt.Sprintf("%s.GrowthEstimates", strings.ToUpper(t.symbol)))
	return dt, nil
}

// FundsData returns fund-related data for ETFs/mutual funds.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *ticker) FundsData() (*insyra.DataTable, error) {
	return nil, errors.New("yfinance: FundsData not supported by the installed go-yfinance backend")
}

// TopHoldings returns top holdings for a fund as a DataTable.
// Note: not implemented because underlying go-yfinance version does not expose this method.
func (t *ticker) TopHoldings() (*insyra.DataTable, error) {
	return nil, errors.New("yfinance: TopHoldings not supported by the installed go-yfinance backend")
}
