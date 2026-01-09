package datafetch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/limiter"
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
}

// YFinance creates a YahooFinance fetcher using a config struct (no WithXxx in public API).
func YFinance(cfg YFinanceConfig) (*yahooFinance, error) {
	normalized, err := cfg.normalize()
	if err != nil {
		return nil, err
	}

	c, err := yfclient.New(
		yfclient.WithTimeout(int(normalized.Timeout)),
		yfclient.WithUserAgent(normalized.UserAgent),
	)
	if err != nil {
		return nil, err
	}

	return &yahooFinance{
		cfg:     normalized,
		client:  c,
		limiter: limiter.NewIntervalLimiter(normalized.Interval),
	}, nil
}

// lifecycle
func (y *yahooFinance) Close() {
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
// The ticker does not manage the lifecycle of the underlying client; call `y.Close()` to release resources when done.
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
			return insyra.ReadJSON(bars)
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
			return insyra.ReadJSON(q)
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
	return insyra.ReadJSON(info)
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
	return insyra.ReadJSON(divs)
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
	return insyra.ReadJSON(splits)
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
	return insyra.ReadJSON(acts)
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
	return insyra.ReadJSON(exps)
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
	return insyra.ReadJSON(chain)
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
	return insyra.ReadJSON(articles)
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
	return insyra.ReadJSON(cal)
}

// Financials: IncomeStatement / BalanceSheet / CashFlow
func (t *ticker) IncomeStatement(freq string) (*insyra.DataTable, error) {
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

	stmt, err := tk.IncomeStatement(freq)
	if err != nil {
		return nil, err
	}
	return insyra.ReadJSON(stmt)
}

func (t *ticker) BalanceSheet(freq string) (*insyra.DataTable, error) {
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

	stmt, err := tk.BalanceSheet(freq)
	if err != nil {
		return nil, err
	}
	return insyra.ReadJSON(stmt)
}

func (t *ticker) CashFlow(freq string) (*insyra.DataTable, error) {
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

	stmt, err := tk.CashFlow(freq)
	if err != nil {
		return nil, err
	}
	return insyra.ReadJSON(stmt)
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
	return insyra.ReadJSON(h)
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
	return insyra.ReadJSON(h)
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
	return insyra.ReadJSON(h)
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
	return insyra.ReadJSON(tx)
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
	return insyra.ReadJSON(fi)
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
	return insyra.ReadJSON(est)
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
	return insyra.ReadJSON(hs)
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
	return insyra.ReadJSON(et)
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
	return insyra.ReadJSON(rev)
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
	return insyra.ReadJSON(recs)
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
	return insyra.ReadJSON(apt)
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
	return insyra.ReadJSON(rev)
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
	return insyra.ReadJSON(g)
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

// Download fetches historical data for a symbol or multiple symbols and
// returns an insyra.DataTable. `symbols` can be a single string or []string.
func (y *yahooFinance) Download(symbols any, params models.HistoryParams) (*insyra.DataTable, error) {
	if y == nil {
		return nil, errors.New("yfinance: yahooFinance is nil")
	}
	switch v := symbols.(type) {
	case string:
		// use underlying ticker client with retry/limiting
		if y.client == nil {
			return nil, errors.New("yfinance: client is nil")
		}
		tk, err := yfticker.New(v, yfticker.WithClient(y.client))
		if err != nil {
			return nil, err
		}
		defer tk.Close()

		var lastErr error
		for attempt := 0; attempt <= y.cfg.Retries; attempt++ {
			if err := y.beforeRequest(); err != nil {
				return nil, err
			}

			bars, err := tk.History(params)
			if err == nil {
				return insyra.ReadJSON(bars)
			}

			lastErr = classifyError(err)
			if !retryable(lastErr) {
				return nil, lastErr
			}
			if attempt < y.cfg.Retries {
				y.sleepBackoff(attempt)
			}
		}
		return nil, fmt.Errorf("yfinance: history failed: %w", lastErr)
	case []string:
		if len(v) == 0 {
			return insyra.ReadJSON(map[string][]models.Bar{})
		}

		conc := y.cfg.Concurrency
		if conc <= 0 {
			conc = 1
		}

		type result struct {
			symbol string
			bars   []models.Bar
			err    error
		}

		jobs := make(chan string)
		results := make(chan result, len(v))

		var wg sync.WaitGroup
		worker := func() {
			defer wg.Done()
			for sym := range jobs {
				// per-job request using underlying ticker and retries
				var lastErr error
				var bars []models.Bar
				if y.client == nil {
					results <- result{symbol: sym, bars: nil, err: errors.New("yfinance: client is nil")}
					continue
				}
				tk, err := yfticker.New(sym, yfticker.WithClient(y.client))
				if err != nil {
					results <- result{symbol: sym, bars: nil, err: err}
					continue
				}
				// close explicitly after use to avoid stacking defers inside loop

				for attempt := 0; attempt <= y.cfg.Retries; attempt++ {
					if err := y.beforeRequest(); err != nil {
						lastErr = err
						break
					}
					bars, err = tk.History(params)
					if err == nil {
						break
					}
					lastErr = classifyError(err)
					if !retryable(lastErr) {
						break
					}
					if attempt < y.cfg.Retries {
						y.sleepBackoff(attempt)
					}
				}
				// close ticker for this job
				tk.Close()
				results <- result{symbol: sym, bars: bars, err: lastErr}
			}
		}

		wg.Add(conc)
		for i := 0; i < conc; i++ {
			go worker()
		}

		for _, sym := range v {
			jobs <- sym
		}
		close(jobs)

		wg.Wait()
		close(results)

		out := make(map[string][]models.Bar, len(v))
		var firstErr error
		for r := range results {
			if r.err != nil && firstErr == nil {
				firstErr = fmt.Errorf("yfinance: multi history failed (first error at %s): %w", r.symbol, r.err)
			}
			if r.err == nil {
				out[r.symbol] = r.bars
			}
		}

		if firstErr != nil && len(out) == 0 {
			return nil, firstErr
		}
		// return available data as DataTable (may be partial)
		dt, derr := insyra.ReadJSON(out)
		if derr != nil {
			return nil, derr
		}
		return dt, firstErr
	default:
		return nil, errors.New("yfinance: Download symbols must be string or []string")
	}
}
