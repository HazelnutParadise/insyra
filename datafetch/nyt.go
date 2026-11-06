package datafetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/limiter"
	json "github.com/goccy/go-json"
)

// ErrNYTAuth matches errors.Is when NYT refuses a key (HTTP 401 or 403).
// Authentication failures are never retried.
var ErrNYTAuth = errors.New("datafetch: NYT refused the API key")

// ErrNYTThrottled matches errors.Is when NYT still returns HTTP 429 after the
// configured retries, and when the daily quota is exhausted, which is a
// *NYTQuotaError.
var ErrNYTThrottled = errors.New("datafetch: NYT request quota exhausted")

// NYTQuotaError reports that the daily quota is exhausted: the client's own
// count of 500 requests in 24 hours is used up, or NYT answered HTTP 429 with a
// Retry-After longer than a per-minute limit asks for. RetryAt is when the
// quota is expected to allow a request again. It matches ErrNYTThrottled.
type NYTQuotaError struct {
	RetryAt time.Time
}

func (e *NYTQuotaError) Error() string {
	return "datafetch: NYT daily request quota exhausted until " + e.RetryAt.UTC().Format(time.RFC3339)
}

func (e *NYTQuotaError) Unwrap() error { return ErrNYTThrottled }

const defaultNYTInterval = 12100 * time.Millisecond
const nytQuotaWindow = 24 * time.Hour
const nytDailyLimit = 500
const nytRetryWait = 61 * time.Second

// nytMaxRetryAfter is the longest Retry-After treated as the per-minute limit.
// A longer one means the daily quota, reported as a *NYTQuotaError.
const nytMaxRetryAfter = 2 * time.Minute

// NYTConfig configures the monthly Archive client.
type NYTConfig struct {
	// APIKey is required and sent as the api-key query parameter.
	APIKey string
	// BaseURL defaults to https://api.nytimes.com/svc/archive/v1. HTTPS is
	// required except for literal loopback IPs used by local test servers.
	BaseURL string
	// Timeout limits each HTTP attempt, excluding pacing. Default: 30 seconds.
	Timeout time.Duration
	// Interval spaces request starts. Zero means 12.1 seconds; shorter
	// intervals, including negative values, are rejected.
	Interval time.Duration
	// Retries is the number of retry attempts after HTTP 429 from the
	// per-minute limit. Zero means none; negative values are rejected. Other
	// HTTP failures are not retried.
	Retries int
}

// NYTClient fetches monthly article metadata. Create one with NYT and reuse it
// across months. Concurrent calls share its pacing and rolling daily budget;
// other clients and processes using the same key are not accounted for.
// A nil or zero client returns an error instead of sending a request.
type NYTClient struct {
	cfg      NYTConfig
	http     *http.Client
	limiter  *limiter.IntervalLimiter
	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	mu       sync.Mutex
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
	if u.Scheme == "http" && !net.ParseIP(u.Hostname()).IsLoopback() {
		return nil, errors.New("datafetch: NYT BaseURL requires HTTPS outside a literal loopback IP")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	c := &NYTClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
			// A redirect can echo or forward a credential-bearing URL.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		now: time.Now, wait: sleepContext,
	}
	c.limiter = limiter.NewIntervalLimiterWithClock(cfg.Interval,
		func() time.Time { return c.now() },
		func(ctx context.Context, d time.Duration) error { return c.wait(ctx, d) })
	return c, nil
}

// Archive returns one month's article metadata. It is ArchiveContext with
// context.Background(), so an exhausted daily quota returns a *NYTQuotaError
// at once; use ArchiveContext to bound all waits.
func (c *NYTClient) Archive(year, month int) (*insyra.DataTable, error) {
	return c.ArchiveContext(context.Background(), year, month)
}

// ArchiveContext returns _id, headline, abstract, lead_paragraph, section_name,
// news_desk, type_of_material, word_count (int64), pub_date (time.Time) and
// web_url columns, even for an empty month. A missing value is nil. Year is
// 1851 or later and the month is not after the current one.
//
// Each client spaces request starts by at least 12.1s and counts every attempt
// against a rolling 500-attempt/24-hour budget. A 429 waits at least 61s, or
// as long as Retry-After asks up to two minutes, before each of the configured
// retries, then returns ErrNYTThrottled if still refused. An exhausted daily
// quota is a *NYTQuotaError: when ctx has a deadline at or after its RetryAt
// the call waits until then and tries again, otherwise it returns the error at
// once. Every wait and the HTTP request stop when ctx ends.
func (c *NYTClient) ArchiveContext(ctx context.Context, year, month int) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.http == nil || c.limiter == nil {
		return nil, errors.New("datafetch: NYTClient was not created with NYT")
	}
	now := c.now()
	if year < 1851 || month < 1 || month > 12 || year > now.Year() || (year == now.Year() && month > int(now.Month())) {
		return nil, errors.New("datafetch: NYT year must be 1851 or later, month 1 through 12, and the month not in the future")
	}
	for retries := 0; ; {
		table, retryAfter, err := c.attempt(ctx, year, month)
		var quota *NYTQuotaError
		switch {
		case errors.As(err, &quota):
			deadline, ok := ctx.Deadline()
			if !ok || deadline.Before(quota.RetryAt) {
				return nil, err
			}
			if err := c.wait(ctx, quota.RetryAt.Sub(c.now())); err != nil {
				return nil, err
			}
		case errors.Is(err, ErrNYTThrottled) && retries < c.cfg.Retries:
			retries++
			if err := c.wait(ctx, retryAfter); err != nil {
				return nil, err
			}
		default:
			return table, err
		}
	}
}

// attempt waits for the next request slot, charges the attempt to the daily
// budget and sends it. Only attempts about to run are charged, so a cancelled
// wait leaves the budget alone.
func (c *NYTClient) attempt(ctx context.Context, year, month int) (*insyra.DataTable, time.Duration, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, 0, err
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	c.mu.Lock()
	now := c.now()
	first := 0
	for first < len(c.attempts) && !c.attempts[first].Add(nytQuotaWindow).After(now) {
		first++
	}
	c.attempts = c.attempts[first:]
	if len(c.attempts) >= nytDailyLimit {
		retryAt := c.attempts[0].Add(nytQuotaWindow)
		c.mu.Unlock()
		return nil, 0, &NYTQuotaError{RetryAt: retryAt}
	}
	c.attempts = append(c.attempts, now)
	c.mu.Unlock()
	return c.archiveRequest(ctx, year, month)
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
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, 0, fmt.Errorf("datafetch: NYT request timed out: %w", context.DeadlineExceeded)
		}
		// A *url.Error's text includes the complete URL, key and all. Keep
		// only the cause, with the key removed in case it echoes it.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, 0, fmt.Errorf("datafetch: NYT HTTP request failed: %w", &nytRedactedError{err: err, key: c.cfg.APIKey})
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, 0, fmt.Errorf("%w (HTTP %d): %s", ErrNYTAuth, resp.StatusCode, nytBodySnippet(resp.Body, c.cfg.APIKey))
	case http.StatusTooManyRequests:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		delay := nytRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if delay > nytMaxRetryAfter {
			return nil, 0, &NYTQuotaError{RetryAt: c.now().Add(delay)}
		}
		return nil, delay, ErrNYTThrottled
	case http.StatusOK:
	default:
		return nil, 0, fmt.Errorf("datafetch: NYT HTTP status %d: %s", resp.StatusCode, nytBodySnippet(resp.Body, c.cfg.APIKey))
	}
	table, err := nytArchiveTable(ctx, resp.Body)
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}
	return table, 0, err
}

// nytRetryAfter is the wait a 429 asks for: Retry-After in seconds or as an
// HTTP date, and at least 61 seconds.
func nytRetryAfter(header string, now time.Time) time.Duration {
	delay := nytRetryWait
	if seconds, err := strconv.ParseInt(strings.TrimSpace(header), 10, 64); err == nil && seconds > 0 {
		// Saturate instead of overflowing into a negative wait.
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

// nytBodySnippet is the start of an error response as one line of printable
// text, with the API key removed, so an error can say why NYT refused.
func nytBodySnippet(body io.Reader, key string) string {
	raw, _ := io.ReadAll(io.LimitReader(body, 300))
	text := nytRedact(string(raw), key)
	text = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return ' '
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "no response body"
	}
	return text
}

// nytRedact removes the API key, as sent and as URL-encoded, from text.
func nytRedact(text, key string) string {
	for _, form := range []string{key, url.QueryEscape(key), url.PathEscape(key)} {
		if form != "" {
			text = strings.ReplaceAll(text, form, "[redacted]")
		}
	}
	return text
}

// nytRedactedError keeps a transport error for errors.Is and errors.As while
// its text never carries the API key.
type nytRedactedError struct {
	err error
	key string
}

func (e *nytRedactedError) Error() string { return nytRedact(e.err.Error(), e.key) }

func (e *nytRedactedError) Unwrap() error { return e.err }

func nytArchiveTable(ctx context.Context, body io.Reader) (*insyra.DataTable, error) {
	// Keep a missing/null array distinguishable from an empty, valid month.
	var payload struct {
		Response *struct {
			Docs []*struct {
				ID       *string `json:"_id"`
				Headline *struct {
					Main *string `json:"main"`
				} `json:"headline"`
				Abstract       *string         `json:"abstract"`
				LeadParagraph  *string         `json:"lead_paragraph"`
				Section        *string         `json:"section_name"`
				NewsDesk       *string         `json:"news_desk"`
				TypeOfMaterial *string         `json:"type_of_material"`
				WordCount      json.RawMessage `json:"word_count"`
				Published      *string         `json:"pub_date"`
				WebURL         *string         `json:"web_url"`
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
	names := []string{"_id", "headline", "abstract", "lead_paragraph", "section_name", "news_desk", "type_of_material", "word_count", "pub_date", "web_url"}
	values := make([][]any, len(names))
	for i := range values {
		values[i] = make([]any, len(payload.Response.Docs))
	}
	for i, article := range payload.Response.Docs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if article == nil {
			return nil, fmt.Errorf("datafetch: NYT article %d is null", i)
		}
		var headline *string
		if article.Headline != nil {
			headline = article.Headline.Main
		}
		for column, text := range []*string{article.ID, headline, article.Abstract, article.LeadParagraph, article.Section, article.NewsDesk, article.TypeOfMaterial} {
			if text != nil {
				values[column][i] = *text
			}
		}
		words, err := nytWordCount(article.WordCount)
		if err != nil {
			return nil, fmt.Errorf("datafetch: NYT article %d has an invalid word_count", i)
		}
		values[7][i] = words
		if article.Published != nil && *article.Published != "" {
			published, err := time.Parse(time.RFC3339Nano, *article.Published)
			if err != nil {
				// NYT also publishes ISO 8601 timestamps with colonless offsets.
				published, err = time.Parse("2006-01-02T15:04:05-0700", *article.Published)
			}
			if err != nil {
				return nil, fmt.Errorf("datafetch: NYT article %d has an invalid pub_date", i)
			}
			values[8][i] = published
		}
		if article.WebURL != nil {
			values[9][i] = *article.WebURL
		}
	}
	lists := make([]*insyra.DataList, len(names))
	for i, name := range names {
		lists[i] = newNamedCol(name, values[i])
	}
	return insyra.NewDataTable(lists...), nil
}

// nytWordCount reads word_count, which the OpenAPI documents as an integer in
// one version and as a string in another: nil when it is missing or null.
func nytWordCount(raw json.RawMessage) (any, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return nil, nil
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		text = unquoted
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return nil, err
	}
	return n, nil
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
