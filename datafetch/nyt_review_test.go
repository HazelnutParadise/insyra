package datafetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A Retry-After far longer than the per-minute limit means the daily quota.
// Without a deadline the call returns at once instead of holding the client.
func TestNYTHugeRetryAfterReturnsQuotaErrorAtOnce(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Retry-After", "99999999999")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.cfg.Retries = 3
	c.wait = func(context.Context, time.Duration) error {
		t.Error("waited on a daily-quota Retry-After without a deadline")
		return nil
	}
	start := time.Now()
	_, err := c.Archive(2026, 10)
	var quota *NYTQuotaError
	if !errors.As(err, &quota) || !errors.Is(err, ErrNYTThrottled) {
		t.Fatalf("error = %v, want a *NYTQuotaError matching ErrNYTThrottled", err)
	}
	if quota.RetryAt.Before(start.Add(100 * 365 * 24 * time.Hour)) {
		t.Fatalf("RetryAt = %v, want the far future the header names", quota.RetryAt)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestNYTZeroRetriesDoesNotRetry(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.wait = func(context.Context, time.Duration) error { return nil }
	_, err := c.Archive(2026, 10)
	var quota *NYTQuotaError
	if !errors.Is(err, ErrNYTThrottled) || errors.As(err, &quota) {
		t.Fatalf("error = %v, want the per-minute ErrNYTThrottled", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 with Retries 0", calls)
	}
}

// fillNYTBudget charges the whole daily budget, the oldest attempt at first.
func fillNYTBudget(c *NYTClient, first time.Time) {
	c.attempts = nil
	for i := range nytDailyLimit {
		c.attempts = append(c.attempts, first.Add(time.Duration(i)*time.Second))
	}
}

func TestNYTExhaustedQuotaWaitsOnlyWithinTheDeadline(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(nytAcceptanceEmpty))
	})
	now := time.Now()
	c.now = func() time.Time { return now }
	c.wait = func(ctx context.Context, d time.Duration) error { now = now.Add(d); return ctx.Err() }
	first := now.Add(-23 * time.Hour)
	retryAt := first.Add(24 * time.Hour)

	fillNYTBudget(c, first)
	short, cancel := context.WithDeadline(context.Background(), retryAt.Add(-time.Minute))
	defer cancel()
	_, err := c.ArchiveContext(short, 2026, 10)
	var quota *NYTQuotaError
	if !errors.As(err, &quota) || !quota.RetryAt.Equal(retryAt) {
		t.Fatalf("short deadline: error = %v, want a *NYTQuotaError with RetryAt %v", err, retryAt)
	}
	if calls != 0 || now.Sub(first) > 23*time.Hour+defaultNYTInterval {
		t.Fatalf("short deadline sent %d requests or waited until %v", calls, now)
	}

	fillNYTBudget(c, first)
	long, cancelLong := context.WithDeadline(context.Background(), retryAt.Add(time.Hour))
	defer cancelLong()
	if _, err := c.ArchiveContext(long, 2026, 10); err != nil {
		t.Fatalf("long deadline: %v", err)
	}
	if calls != 1 || now.Before(retryAt) {
		t.Fatalf("long deadline: calls = %d at %v, want 1 at or after %v", calls, now, retryAt)
	}
}

func TestNYTRefusesAFutureMonthWithoutSending(t *testing.T) {
	c := nytAcceptanceClient(t, func(http.ResponseWriter, *http.Request) { t.Error("future month sent a request") })
	c.now = func() time.Time { return time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC) }
	if _, err := c.Archive(2026, 12); err == nil {
		t.Fatal("accepted December 2026 in November 2026")
	}
}

func TestNYTMissingPublicationDateIsNil(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"docs":[{"headline":{"main":"no date"}},{"pub_date":null},{"pub_date":"2026-10-01T00:00:00Z"}]}}`))
	})
	dt, err := c.Archive(2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	dates := dt.GetColByName("pub_date").Data()
	if dates[0] != nil || dates[1] != nil {
		t.Fatalf("missing pub_date = %#v, want nil", dates[:2])
	}
	if _, ok := dates[2].(time.Time); !ok {
		t.Fatalf("pub_date = %#v", dates[2])
	}
}

func TestNYTWordCountAcceptsNumberAndText(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"docs":[{"word_count":812},{"word_count":"95"},{}]}}`))
	})
	dt, err := c.Archive(2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	got := dt.GetColByName("word_count").Data()
	if got[0] != int64(812) || got[1] != int64(95) || got[2] != nil {
		t.Fatalf("word_count = %#v", got)
	}
}

func TestNYTStatusErrorCarriesFilteredReason(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("{\"fault\":\"backend down\",\n\"key\":\"" + r.URL.Query().Get("api-key") + "\"}\x00"))
	})
	_, err := c.Archive(2026, 10)
	if err == nil || !strings.Contains(err.Error(), "backend down") || !strings.Contains(err.Error(), "500") {
		t.Fatalf("error = %v, want the status and the reason", err)
	}
	if strings.Contains(err.Error(), c.cfg.APIKey) || strings.ContainsAny(err.Error(), "\n\x00") {
		t.Fatalf("error carries the key or control characters: %q", err)
	}
}

func TestNYTConnectionErrorKeepsTheCauseWithoutTheURL(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	base := server.URL
	server.Close()
	c, err := NYT(NYTConfig{APIKey: nytTestAPIKey, BaseURL: base})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Archive(2026, 10)
	var opErr *net.OpError
	if !errors.As(err, &opErr) || opErr.Op != "dial" {
		t.Fatalf("error = %v, want the dial error as its cause", err)
	}
	if strings.Contains(err.Error(), nytTestAPIKey) || strings.Contains(err.Error(), "api-key") {
		t.Fatalf("error carries the URL or key: %v", err)
	}
}

// The key is removed before the reason is cut to 300 bytes, so a key that
// crosses the cut leaves no part of itself behind.
func TestNYTStatusReasonCutDoesNotLeaveHalfAKey(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 290) + r.URL.Query().Get("api-key") + strings.Repeat("y", 50)))
	})
	_, err := c.Archive(2026, 10)
	if err == nil {
		t.Fatal("error status accepted")
	}
	for i := 4; i <= len(c.cfg.APIKey); i++ {
		if strings.Contains(err.Error(), c.cfg.APIKey[:i]) {
			t.Fatalf("error carries %q of the key: %v", c.cfg.APIKey[:i], err)
		}
	}
}

// RetryAt is when a request can actually start: a budget that frees up before
// the next request slot reports the slot, so a deadline between the two
// returns the quota error at once instead of waiting into the deadline.
func TestNYTQuotaRetryAtIncludesTheNextRequestSlot(t *testing.T) {
	calls := 0
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(nytAcceptanceEmpty))
	})
	now := time.Now()
	c.now = func() time.Time { return now }
	c.wait = func(ctx context.Context, d time.Duration) error { now = now.Add(d); return ctx.Err() }
	if _, err := c.Archive(2026, 10); err != nil {
		t.Fatal(err)
	}
	fillNYTBudget(c, now.Add(-24*time.Hour+2*time.Second))
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(5*time.Second))
	defer cancel()
	start := now
	_, err := c.ArchiveContext(ctx, 2026, 10)
	var quota *NYTQuotaError
	if !errors.As(err, &quota) {
		t.Fatalf("error = %v, want a *NYTQuotaError", err)
	}
	if !quota.RetryAt.Equal(start.Add(defaultNYTInterval)) {
		t.Fatalf("RetryAt = %v, want the next slot %v", quota.RetryAt, start.Add(defaultNYTInterval))
	}
	if calls != 1 || !now.Equal(start) {
		t.Fatalf("sent %d requests and waited %v", calls, now.Sub(start))
	}
}

// A per-minute retry the deadline cannot reach is reported as throttling.
func TestNYTRetryBeyondTheDeadlineReturnsThrottled(t *testing.T) {
	c := nytAcceptanceClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.cfg.Retries = 2
	c.wait = func(context.Context, time.Duration) error {
		t.Error("waited for a retry the deadline cannot reach")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.ArchiveContext(ctx, 2026, 10); !errors.Is(err, ErrNYTThrottled) {
		t.Fatalf("error = %v, want ErrNYTThrottled", err)
	}
}
