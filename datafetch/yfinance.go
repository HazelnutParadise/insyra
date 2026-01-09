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
// Caller should call Close() when done to release resources.
func (y *yahooFinance) Ticker(symbol string) (*ticker, error) {
	if y == nil {
		return nil, errors.New("yfinance: yahooFinance is nil")
	}
	return &ticker{yf: y, symbol: symbol}, nil
}

// Close closes underlying resources used by the ticker.
func (t *ticker) Close() {
	if t == nil || t.yf == nil {
		return
	}
	t.yf.Close()
	t.yf = nil
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

// Download fetches historical data for a symbol or multiple symbols and
// returns an insyra.DataTable. `symbols` can be a single string or []string.
func Download(symbols any, params models.HistoryParams) (*insyra.DataTable, error) {
	switch v := symbols.(type) {
	case string:
		yf, err := YFinance(YFinanceConfig{})
		if err != nil {
			return nil, err
		}
		defer yf.Close()

		// use underlying ticker client with retry/limiting
		if yf.client == nil {
			return nil, errors.New("yfinance: client is nil")
		}
		tk, err := yfticker.New(v, yfticker.WithClient(yf.client))
		if err != nil {
			return nil, err
		}
		defer tk.Close()

		var lastErr error
		for attempt := 0; attempt <= yf.cfg.Retries; attempt++ {
			if err := yf.beforeRequest(); err != nil {
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
			if attempt < yf.cfg.Retries {
				yf.sleepBackoff(attempt)
			}
		}
		return nil, fmt.Errorf("yfinance: history failed: %w", lastErr)
	case []string:
		yf, err := YFinance(YFinanceConfig{})
		if err != nil {
			return nil, err
		}
		defer yf.Close()

		if len(v) == 0 {
			return insyra.ReadJSON(map[string][]models.Bar{})
		}

		conc := yf.cfg.Concurrency
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
				if yf.client == nil {
					results <- result{symbol: sym, bars: nil, err: errors.New("yfinance: client is nil")}
					continue
				}
				tk, err := yfticker.New(sym, yfticker.WithClient(yf.client))
				if err != nil {
					results <- result{symbol: sym, bars: nil, err: err}
					continue
				}
				// close explicitly after use to avoid stacking defers inside loop

				for attempt := 0; attempt <= yf.cfg.Retries; attempt++ {
					if err := yf.beforeRequest(); err != nil {
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
					if attempt < yf.cfg.Retries {
						yf.sleepBackoff(attempt)
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
