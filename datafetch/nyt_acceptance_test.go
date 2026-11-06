package datafetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

const nytAcceptanceEmpty = `{"response":{"docs":[]}}`

func nytAcceptanceClient(t *testing.T, handler http.HandlerFunc) *NYTClient {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	c, err := NYT(NYTConfig{APIKey: "local-fake+/=key", BaseURL: s.URL})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNYTAcceptanceMonth(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/2026/10.json" || r.URL.Query().Get("api-key") != "local-fake+/=key" {
			t.Error("incorrect month/auth request")
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[{"_id":"nyt://article/1","headline":{"main":"A headline"},"abstract":"Summary","lead_paragraph":"Lead","section_name":"Business","news_desk":"Business","type_of_material":"News","word_count":812,"pub_date":"2026-10-18T14:05:06-04:00","web_url":"https://www.nytimes.com/example"},{"headline":null,"abstract":null,"word_count":null,"pub_date":"2026-10-01T00:00:00Z"}]}}`))
	})
	dt, err := c.Archive(2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || dt.NumRows() != 2 || dt.NumCols() != 10 {
		t.Fatalf("calls/rows/columns = %d/%d/%d", calls, dt.NumRows(), dt.NumCols())
	}
	for name, want := range map[string]any{"_id": "nyt://article/1", "headline": "A headline", "abstract": "Summary", "lead_paragraph": "Lead", "section_name": "Business", "news_desk": "Business", "type_of_material": "News", "word_count": int64(812), "web_url": "https://www.nytimes.com/example"} {
		got := dt.GetColByName(name).Data()
		if got[0] != want || got[1] != nil {
			t.Errorf("%s values = %#v", name, got)
		}
	}
	date, ok := dt.GetColByName("pub_date").Data()[0].(time.Time)
	if !ok || date.Format(time.RFC3339) != "2026-10-18T14:05:06-04:00" {
		t.Errorf("publication timestamp = %v", date)
	}
}

func TestNYTAcceptanceEmptyMonth(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(nytAcceptanceEmpty)) })
	dt, err := c.Archive(1851, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 0 || dt.NumCols() != 10 {
		t.Fatalf("empty shape = %d/%d", dt.NumRows(), dt.NumCols())
	}
	for _, name := range []string{"_id", "headline", "abstract", "lead_paragraph", "section_name", "news_desk", "type_of_material", "word_count", "pub_date", "web_url"} {
		if dt.GetColByName(name) == nil {
			t.Errorf("missing column %s", name)
		}
	}
}

func TestNYTAcceptancePublicationOffsets(t *testing.T) {
	for _, value := range []string{
		"1980-11-23T05:00:00+0000",
		"2026-10-18T14:05:06+0530",
		"2026-10-18T14:05:06.123456789-0400",
	} {
		t.Run(value, func(t *testing.T) {
			c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"response":{"docs":[{"pub_date":"` + value + `"}]}}`))
			})
			dt, err := c.Archive(2026, 10)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := dt.GetColByName("pub_date").Data()[0].(time.Time)
			if !ok || got.Format("2006-01-02T15:04:05.999999999-0700") != value {
				t.Fatalf("publication timestamp lost value or offset: %v", got)
			}
		})
	}
}

func TestNYTAcceptancePacingAndQuota(t *testing.T) {
	calls := 0
	var starts []time.Time
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 2 {
			w.WriteHeader(429)
			return
		}
		_, _ = w.Write([]byte(nytAcceptanceEmpty))
	})
	c.cfg.Retries = 1
	c.now = func() time.Time { return now }
	var waits []time.Duration
	c.wait = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		waits = append(waits, d)
		now = now.Add(d)
		return nil
	}
	for range 2 {
		starts = append(starts, now)
		if _, err := c.Archive(2026, 10); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 including retry", calls)
	}
	if now.Sub(starts[0]) < defaultNYTInterval+nytRetryWait {
		t.Fatalf("insufficient pacing: %v", waits)
	}
	// Every attempt, including the 429, consumes the same daily budget.
	for i := 3; i < nytDailyLimit; i++ {
		if _, err := c.Archive(2026, 10); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	before := now
	_, err := c.Archive(2026, 10)
	var quota *NYTQuotaError
	if !errors.As(err, &quota) || !errors.Is(err, ErrNYTThrottled) {
		t.Fatalf("daily quota error = %v", err)
	}
	if !quota.RetryAt.Equal(starts[0].Add(24 * time.Hour)) {
		t.Fatalf("RetryAt = %v, want the first attempt plus 24 hours (%v)", quota.RetryAt, starts[0].Add(24*time.Hour))
	}
	if calls != 500 || now.Sub(before) > defaultNYTInterval {
		t.Fatalf("daily exhaustion sent/waited: calls=%d elapsed=%v", calls, now.Sub(before))
	}
	now = now.Add(24 * time.Hour)
	if _, err := c.Archive(2026, 10); err != nil {
		t.Fatalf("next rolling window: %v", err)
	}
}

func TestNYTAcceptanceRefusedAndExhausted(t *testing.T) {
	for _, status := range []int{401, 403, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				http.Error(w, r.URL.Query().Get("api-key"), status)
			})
			c.wait = func(context.Context, time.Duration) error { return nil }
			_, err := c.Archive(2026, 10)
			want, attempts := ErrNYTAuth, 1
			if status == 429 {
				want = ErrNYTThrottled
			}
			if !errors.Is(err, want) || calls != attempts {
				t.Fatalf("error/calls = %v/%d", err, calls)
			}
			if strings.Contains(err.Error(), c.cfg.APIKey) {
				t.Fatal("error exposed credential")
			}
			if status != 429 && !strings.Contains(err.Error(), "[redacted]") {
				t.Fatalf("error does not carry the filtered response body: %v", err)
			}
		})
	}
}

func TestNYTAcceptanceCancelledWait(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) { calls++; _, _ = w.Write([]byte(nytAcceptanceEmpty)) })
	if _, err := c.Archive(2026, 10); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	c.wait = func(ctx context.Context, d time.Duration) error { close(entered); return sleepContext(ctx, d) }
	done := make(chan error, 1)
	go func() { _, err := c.ArchiveContext(ctx, 2026, 10); done <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("did not enter pacing")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not stop wait")
	}
	if calls != 1 {
		t.Fatalf("canceled request sent; calls=%d", calls)
	}
}

func TestNYTAcceptanceInvalidInputs(t *testing.T) {
	c := nytAcceptanceClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid input sent a request") })
	for _, date := range [][2]int{{0, 1}, {1850, 12}, {time.Now().Year() + 1, 1}, {2026, 0}, {2026, 13}, {-1, -1}} {
		if _, err := c.Archive(date[0], date[1]); err == nil {
			t.Errorf("accepted %v", date)
		}
	}
	for _, key := range []string{"", "  "} {
		if _, err := NYT(NYTConfig{APIKey: key}); err == nil {
			t.Error("accepted empty key")
		}
	}
	for _, c := range []*NYTClient{nil, {}} {
		if _, err := c.Archive(2026, 10); err == nil {
			t.Error("accepted uncreated client")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.ArchiveContext(ctx, 2026, 10); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestNYTAcceptanceEndpointTransport(t *testing.T) {
	for _, endpoint := range []string{"http://example.test", "http://192.0.2.1", "http://[2001:db8::1]", "http://localhost"} {
		if _, err := NYT(NYTConfig{APIKey: "local-fake-key", BaseURL: endpoint}); err == nil {
			t.Errorf("accepted plaintext endpoint %s", endpoint)
		}
	}
	for _, endpoint := range []string{"https://example.test", "http://127.0.0.1:12345", "http://[::1]:12345"} {
		if _, err := NYT(NYTConfig{APIKey: "local-fake-key", BaseURL: endpoint}); err != nil {
			t.Errorf("refused HTTPS or loopback endpoint: %v", err)
		}
	}
}

func TestNYTAcceptanceSchemaErrors(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"response":{}}`, `{"response":{"docs":null}}`, `{"response":{"docs":[{"pub_date":"bad"}]}}`, `{"response":{"docs":[null]}}`} {
		t.Run(body, func(t *testing.T) {
			c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
			if _, err := c.Archive(2026, 10); err == nil {
				t.Fatal("malformed archive accepted")
			}
		})
	}
}

// Keep this independent expectation explicit rather than using production column names.
func TestNYTAcceptanceColumnOrder(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(nytAcceptanceEmpty)) })
	dt, err := c.Archive(2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, index := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"} {
		got = append(got, dt.GetColNameByIndex(index))
	}
	if !reflect.DeepEqual(got, []string{"_id", "headline", "abstract", "lead_paragraph", "section_name", "news_desk", "type_of_material", "word_count", "pub_date", "web_url"}) {
		t.Fatalf("columns = %v", got)
	}
}

func TestNYTAcceptanceLastBudgetAttemptThrottledDoesNotWait(t *testing.T) {
	var calls int
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 500 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(nytAcceptanceEmpty))
	})
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	c.wait = func(_ context.Context, d time.Duration) error { now = now.Add(d); return nil }
	for range 499 {
		if _, err := c.Archive(2026, 10); err != nil {
			t.Fatal(err)
		}
	}
	before := now
	if _, err := c.Archive(2026, 10); !errors.Is(err, ErrNYTThrottled) {
		t.Fatal(err)
	}
	if elapsed := now.Sub(before); elapsed > defaultNYTInterval {
		t.Fatalf("exhausted daily budget waited %v, want only scheduled request spacing", elapsed)
	}
}

func TestNYTAcceptanceBodyTimeout(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	c.http.Timeout = 100 * time.Millisecond
	if _, err := c.Archive(2026, 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("body timeout = %v, want DeadlineExceeded", err)
	}
}

func TestNYTAcceptanceBodyTimeoutAfterJSON(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(nytAcceptanceEmpty))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	c.http.Timeout = 100 * time.Millisecond
	if _, err := c.Archive(2026, 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("body timeout after valid JSON = %v, want DeadlineExceeded", err)
	}
}

type nytAcceptanceSpaces struct{}

func (nytAcceptanceSpaces) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

func TestNYTAcceptanceOversizedResponse(t *testing.T) {
	// A fake stream avoids allocating a 128 MiB fixture. The invalid suffix
	// must not disappear behind the reader's size limit and become success.
	body := io.MultiReader(strings.NewReader(nytAcceptanceEmpty),
		io.LimitReader(nytAcceptanceSpaces{}, (128<<20)-int64(len(nytAcceptanceEmpty))),
		strings.NewReader("invalid suffix"))
	if _, err := nytArchiveTable(context.Background(), body); err == nil {
		t.Fatal("oversized response was accepted as a complete empty month")
	}
}
