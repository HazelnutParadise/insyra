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
func (y *yahooFinance) newTicker(symbol string) (*yfticker.Ticker, error) {
	if y == nil || y.client == nil {
		return nil, errors.New("yfinance: client is nil")
	}
	return yfticker.New(symbol, yfticker.WithClient(y.client))
}

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
func (y *yahooFinance) Quote(symbol string) (*insyra.DataTable, error) {
	t, err := y.newTicker(symbol)
	if err != nil {
		return nil, err
	}
	defer t.Close()

	var lastErr error
	for attempt := 0; attempt <= y.cfg.Retries; attempt++ {
		if err := y.beforeRequest(); err != nil {
			return nil, err
		}

		q, err := t.Quote()
		if err == nil {
			dt, err := insyra.ReadJSON(q)
			if err != nil {
				return nil, err
			}
			return dt, nil
		}

		lastErr = classifyError(err)
		if !retryable(lastErr) {
			return nil, lastErr
		}
		if attempt < y.cfg.Retries {
			y.sleepBackoff(attempt)
		}
	}

	return nil, fmt.Errorf("yfinance: quote failed: %w", lastErr)
}

// HistoryBars fetches historical OHLCV bars for a symbol (native models.Bar slice).
// History fetches historical OHLCV bars for a symbol (native models.Bar slice).
func (y *yahooFinance) History(symbol string, params models.HistoryParams) ([]models.Bar, error) {
	t, err := y.newTicker(symbol)
	if err != nil {
		return nil, err
	}
	defer t.Close()

	var lastErr error
	for attempt := 0; attempt <= y.cfg.Retries; attempt++ {
		if err := y.beforeRequest(); err != nil {
			return nil, err
		}

		bars, err := t.History(params)
		if err == nil {
			return bars, nil
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
}

// MultiHistoryBars fetches histories for multiple symbols.
// IMPORTANT: interval limiting is per instance, so even concurrent workers will still be spaced.
// MultiHistory fetches histories for multiple symbols concurrently.
func (y *yahooFinance) MultiHistory(symbols []string, params models.HistoryParams) (map[string][]models.Bar, error) {
	if len(symbols) == 0 {
		return map[string][]models.Bar{}, nil
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
	results := make(chan result, len(symbols))

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for sym := range jobs {
			bars, err := y.History(sym, params)
			results <- result{symbol: sym, bars: bars, err: err}
		}
	}

	wg.Add(conc)
	for i := 0; i < conc; i++ {
		go worker()
	}

	for _, sym := range symbols {
		jobs <- sym
	}
	close(jobs)

	wg.Wait()
	close(results)

	out := make(map[string][]models.Bar, len(symbols))
	var firstErr error

	for r := range results {
		if r.err != nil && firstErr == nil {
			firstErr = fmt.Errorf("yfinance: multi history failed (first error at %s): %w", r.symbol, r.err)
		}
		if r.err == nil {
			out[r.symbol] = r.bars
		}
	}

	return out, firstErr
}
