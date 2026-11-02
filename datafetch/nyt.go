package datafetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

// ErrNYTAuth matches errors.Is when NYT refuses a key (HTTP 401 or 403).
// Authentication failures are never retried.
var ErrNYTAuth = errors.New("datafetch: NYT refused the API key")

// ErrNYTThrottled matches errors.Is when the local daily budget is exhausted
// or NYT still returns HTTP 429 after the configured retries.
var ErrNYTThrottled = errors.New("datafetch: NYT request quota exhausted")

const defaultNYTInterval = 12100 * time.Millisecond
const nytQuotaWindow = 24 * time.Hour
const nytDailyLimit = 500
const nytRetryWait = 61 * time.Second

// NYTConfig configures the monthly Archive client.
type NYTConfig struct {
	// APIKey is required and sent as the api-key query parameter.
	APIKey string
	// BaseURL defaults to https://api.nytimes.com/svc/archive/v1.
	BaseURL string
	// Timeout limits each HTTP attempt, excluding pacing. Default: 30 seconds.
	Timeout time.Duration
	// Interval spaces request starts. Zero means 12.1 seconds; shorter
	// intervals, including negative values, are rejected.
	Interval time.Duration
	// Retries bounds additional attempts on HTTP 429. Zero means one retry;
	// negative values are rejected. Other HTTP failures are not retried.
	Retries int
}

// NYTClient fetches monthly article metadata. Create one with NYT and reuse it
// across months. Concurrent calls share its pacing and rolling daily budget;
// other clients and processes using the same key are not accounted for.
// A nil or zero client returns an error instead of sending a request.
type NYTClient struct {
	cfg      NYTConfig
	http     *http.Client
	gate     chan struct{}
	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	next     time.Time
	attempts []time.Time
}

// NYT creates a client using the same configuration-struct pattern as Fugle
// and YFinance. It does not read an API key from the environment.
func NYT(cfg NYTConfig) (*NYTClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("datafetch: NYT APIKey is required")
	}
	if cfg.Interval == 0 {
		cfg.Interval = defaultNYTInterval
	}
	if cfg.Interval < defaultNYTInterval || cfg.Retries < 0 || cfg.Timeout < 0 {
		return nil, errors.New("datafetch: NYT requires Interval >= 12.1s and nonnegative Retries and Timeout")
	}
	if cfg.Retries == 0 {
		cfg.Retries = 1
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.nytimes.com/svc/archive/v1"
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("datafetch: NYT BaseURL must be an HTTP(S) root without credentials, query or fragment")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &NYTClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
			// A redirect can echo or forward a credential-bearing URL.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		gate: make(chan struct{}, 1), now: time.Now, wait: sleepContext,
	}, nil
}

// Archive returns one month's article metadata. It is ArchiveContext with
// context.Background(); use ArchiveContext to bound all waits.
func (c *NYTClient) Archive(year, month int) (*insyra.DataTable, error) {
	return c.ArchiveContext(context.Background(), year, month)
}

// ArchiveContext returns headline, abstract, lead_paragraph, section_name,
// pub_date (time.Time), and web_url columns, even for an empty month. Missing
// text is nil. Year must be 1851 through the current year; month is 1 through 12.
//
// Each client spaces request starts by at least 12.1s and counts every attempt
// against a rolling 500-attempt/24-hour budget. Exhausting it returns
// ErrNYTThrottled immediately. A 429 waits at least 61s before a bounded retry,
// then returns ErrNYTThrottled if still refused. Queueing, HTTP and every wait
// stop when ctx ends. NYT's 429 does not distinguish minute and daily limits.
func (c *NYTClient) ArchiveContext(ctx context.Context, year, month int) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.http == nil || c.gate == nil {
		return nil, errors.New("datafetch: NYTClient was not created with NYT")
	}
	if year < 1851 || year > c.now().Year() || month < 1 || month > 12 {
		return nil, errors.New("datafetch: NYT year must be 1851 through the current year and month must be 1 through 12")
	}
	select {
	case c.gate <- struct{}{}:
		defer func() { <-c.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	for attempt := 0; ; attempt++ {
		if err := c.beforeArchive(ctx); err != nil {
			return nil, err
		}
		table, retryAfter, err := c.archiveRequest(ctx, year, month)
		if !errors.Is(err, ErrNYTThrottled) || attempt >= c.cfg.Retries {
			return table, err
		}
		if len(c.attempts) >= nytDailyLimit {
			return nil, ErrNYTThrottled
		}
		if err := c.wait(ctx, retryAfter); err != nil {
			return nil, err
		}
	}
}

// beforeArchive runs with the gate held. Unlike reserving distant time slots,
// this only charges attempts actually about to run, so cancellation cannot
// leave a backlog of abandoned reservations.
func (c *NYTClient) beforeArchive(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := c.now()
	first := 0
	for first < len(c.attempts) && !c.attempts[first].Add(nytQuotaWindow).After(now) {
		first++
	}
	c.attempts = c.attempts[first:]
	if len(c.attempts) >= nytDailyLimit {
		return ErrNYTThrottled
	}
	if delay := c.next.Sub(now); delay > 0 {
		if err := c.wait(ctx, delay); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now = c.now()
	c.attempts = append(c.attempts, now)
	c.next = now.Add(c.cfg.Interval)
	return nil
}

func (c *NYTClient) archiveRequest(ctx context.Context, year, month int) (*insyra.DataTable, time.Duration, error) {
	endpoint := fmt.Sprintf("%s/%d/%d.json?%s", c.cfg.BaseURL, year, month, url.Values{"api-key": {c.cfg.APIKey}}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, errors.New("datafetch: NYT could not create request")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		// net/http errors include the complete URL. Never return or log them.
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, 0, fmt.Errorf("datafetch: NYT request timed out: %w", context.DeadlineExceeded)
		}
		return nil, 0, errors.New("datafetch: NYT HTTP request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, 0, ErrNYTAuth
	case http.StatusTooManyRequests:
		return nil, nytRetryAfter(resp.Header.Get("Retry-After"), c.now()), ErrNYTThrottled
	case http.StatusOK:
	default:
		return nil, 0, fmt.Errorf("datafetch: NYT HTTP status %d", resp.StatusCode)
	}
	table, err := nytArchiveTable(ctx, resp.Body)
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}
	return table, 0, err
}

func nytRetryAfter(header string, now time.Time) time.Duration {
	delay := nytRetryWait
	if seconds, err := strconv.ParseInt(strings.TrimSpace(header), 10, 64); err == nil && seconds > 0 {
		// Saturate instead of overflowing into a negative wait. Context can
		// interrupt even this server-specified duration.
		const maxSeconds = int64((1<<63 - 1) / int64(time.Second))
		if seconds > maxSeconds {
			seconds = maxSeconds
		}
		if d := time.Duration(seconds) * time.Second; d > delay {
			delay = d
		}
	} else if date, err := http.ParseTime(header); err == nil && date.Sub(now) > delay {
		delay = date.Sub(now)
	}
	return delay
}

func nytArchiveTable(ctx context.Context, body io.Reader) (*insyra.DataTable, error) {
	// Keep a missing/null array distinguishable from an empty, valid month.
	var payload struct {
		Response *struct {
			Docs []struct {
				Headline *struct {
					Main *string `json:"main"`
				} `json:"headline"`
				Abstract      *string `json:"abstract"`
				LeadParagraph *string `json:"lead_paragraph"`
				Section       *string `json:"section_name"`
				Published     string  `json:"pub_date"`
				WebURL        *string `json:"web_url"`
			} `json:"docs"`
		} `json:"response"`
	}
	// Read one extra byte so an artificial EOF at the size limit cannot
	// conceal trailing data and turn a truncated response into success.
	reader := &nytArchiveReader{Reader: body}
	limited := &io.LimitedReader{R: reader, N: (128 << 20) + 1}
	dec := json.NewDecoder(limited)
	if err := dec.Decode(&payload); err != nil {
		if reader.err != nil {
			err = reader.err
		}
		return nil, nytArchiveDecodeError(err)
	}
	err := dec.Decode(new(any))
	if reader.err != nil {
		return nil, nytArchiveDecodeError(reader.err)
	}
	if limited.N == 0 {
		return nil, errors.New("datafetch: NYT archive response exceeds 128 MiB")
	}
	if !errors.Is(err, io.EOF) {
		return nil, nytArchiveDecodeError(err)
	}
	if payload.Response == nil || payload.Response.Docs == nil {
		return nil, errors.New("datafetch: NYT response.docs must be an array")
	}
	names := []string{"headline", "abstract", "lead_paragraph", "section_name", "pub_date", "web_url"}
	values := make([][]any, len(names))
	for i := range values {
		values[i] = make([]any, len(payload.Response.Docs))
	}
	for i, article := range payload.Response.Docs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		published, err := time.Parse(time.RFC3339Nano, article.Published)
		if err != nil {
			return nil, fmt.Errorf("datafetch: NYT article %d has an invalid pub_date", i)
		}
		var headline *string
		if article.Headline != nil {
			headline = article.Headline.Main
		}
		for j, text := range []*string{headline, article.Abstract, article.LeadParagraph, article.Section, article.WebURL} {
			column := j
			if j == 4 {
				column = 5
			}
			if text != nil {
				values[column][i] = *text
			}
		}
		values[4][i] = published
	}
	lists := make([]*insyra.DataList, len(names))
	for i, name := range names {
		lists[i] = newNamedCol(name, values[i])
	}
	return insyra.NewDataTable(lists...), nil
}

// Older supported JSON decoders may replace an underlying read error with a
// syntax error. Keep it at the input boundary so timeout classification does
// not depend on the decoder version.
type nytArchiveReader struct {
	io.Reader
	err error
}

func (r *nytArchiveReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.err = err
	}
	return n, err
}

func nytArchiveDecodeError(err error) error {
	// A per-request timeout can occur while reading a body after Do has
	// returned its headers. Preserve the classification without echoing the
	// decoder/transport error, which may contain provider-controlled text.
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("datafetch: NYT response timed out: %w", context.DeadlineExceeded)
	}
	return errors.New("datafetch: NYT invalid archive JSON")
}
