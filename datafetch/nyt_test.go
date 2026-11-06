package datafetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const nytTestAPIKey = "test-key-never-print"

func newNYTTestClient(t *testing.T, handler http.HandlerFunc) (*NYTClient, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NYT(NYTConfig{APIKey: nytTestAPIKey, BaseURL: server.URL})
	if err != nil {
		t.Fatal("NYT rejected a valid test configuration")
	}
	return client, server
}

func TestNYTConfigDefaultsAndValidation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	t.Cleanup(server.Close)

	client, err := NYT(NYTConfig{APIKey: nytTestAPIKey, BaseURL: server.URL})
	if err != nil {
		t.Fatal("NYT rejected a valid test configuration")
	}
	if client.cfg.Interval != 12100*time.Millisecond {
		t.Error("zero Interval did not default to 12.1 seconds")
	}
	if client.cfg.Retries != 0 {
		t.Error("zero Retries did not stay zero")
	}

	configs := []NYTConfig{
		{APIKey: nytTestAPIKey, BaseURL: server.URL, Interval: defaultNYTInterval - time.Nanosecond},
		{APIKey: nytTestAPIKey, BaseURL: server.URL, Interval: -time.Second},
		{APIKey: nytTestAPIKey, BaseURL: server.URL, Retries: -1},
		{APIKey: nytTestAPIKey, BaseURL: server.URL, Timeout: -time.Second},
		{APIKey: nytTestAPIKey, BaseURL: "://invalid"},
		{APIKey: nytTestAPIKey, BaseURL: "ftp://example.test"},
		{APIKey: nytTestAPIKey, BaseURL: "https://user:pass@example.test"},
		{APIKey: nytTestAPIKey, BaseURL: server.URL + "?unexpected=query"},
		{APIKey: nytTestAPIKey, BaseURL: "https://example.test/#fragment"},
	}
	for i, cfg := range configs {
		if _, err := NYT(cfg); err == nil {
			t.Errorf("configuration %d was accepted", i)
		}
	}
	if _, err := NYT(NYTConfig{BaseURL: server.URL}); err == nil {
		t.Error("missing API key was accepted")
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("constructing or rejecting configuration sent %d requests", got)
	}
}

func TestNYTCancelsActiveHTTPRequest(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	handlerCanceled := make(chan struct{})
	client, _ := newNYTTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		close(started)
		<-r.Context().Done()
		close(handlerCanceled)
	})

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.ArchiveContext(ctx, time.Now().Year(), 1)
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("request did not reach the local server")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Error("active request did not preserve context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active request did not stop after cancellation")
	}
	select {
	case <-handlerCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not observe request cancellation")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("active cancellation sent %d requests", got)
	}
}

type nytObservedContext struct {
	context.Context
	once     sync.Once
	observed chan struct{}
}

func (c *nytObservedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func TestNYTCancelsWhileQueued(t *testing.T) {
	var calls atomic.Int32
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	client, _ := newNYTTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[]}}`))
	})

	firstResult := make(chan error, 1)
	go func() {
		_, err := client.Archive(time.Now().Year(), 1)
		firstResult <- err
	}()
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		close(releaseFirst)
		t.Fatal("first request did not reach the local server")
	}

	ctx, cancel := context.WithCancel(context.Background())
	queuedCtx := &nytObservedContext{Context: ctx, observed: make(chan struct{})}
	queuedResult := make(chan error, 1)
	go func() {
		_, err := client.ArchiveContext(queuedCtx, time.Now().Year(), 2)
		queuedResult <- err
	}()
	select {
	case <-queuedCtx.observed:
	case <-time.After(2 * time.Second):
		cancel()
		close(releaseFirst)
		t.Fatal("second call did not enter the client queue")
	}
	cancel()
	select {
	case err := <-queuedResult:
		if !errors.Is(err, context.Canceled) {
			t.Error("queued call did not preserve context cancellation")
		}
	case <-time.After(2 * time.Second):
		close(releaseFirst)
		t.Fatal("queued call did not stop after cancellation")
	}
	close(releaseFirst)
	select {
	case err := <-firstResult:
		if err != nil {
			t.Error("first request failed after being released")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not finish")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("queued cancellation sent %d requests", got)
	}
}

func TestNYTConcurrentCallsSharePacing(t *testing.T) {
	var calls atomic.Int32
	var clockMu sync.Mutex
	now := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	var starts []time.Time
	client, _ := newNYTTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		clockMu.Lock()
		starts = append(starts, now)
		clockMu.Unlock()
		_, _ = w.Write([]byte(`{"response":{"docs":[]}}`))
	})

	client.now = func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	var waits []time.Duration
	client.wait = func(ctx context.Context, d time.Duration) error {
		clockMu.Lock()
		now = now.Add(d)
		waits = append(waits, d)
		clockMu.Unlock()
		return ctx.Err()
	}

	results := make(chan error, 2)
	for month := 1; month <= 2; month++ {
		go func(month int) {
			_, err := client.ArchiveContext(context.Background(), time.Now().Year()-1, month)
			results <- err
		}(month)
	}
	for range 2 {
		select {
		case err := <-results:
			if err != nil {
				t.Error("concurrent archive call failed")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("concurrent archive calls did not finish")
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("concurrent calls made %d requests, want 2", got)
	}
	clockMu.Lock()
	startsCopy := append([]time.Time(nil), starts...)
	waitsCopy := append([]time.Duration(nil), waits...)
	clockMu.Unlock()
	if len(startsCopy) != 2 {
		t.Fatalf("recorded %d request starts, want 2", len(startsCopy))
	}
	// The fake clock is shared, so the starts the handler records cannot show
	// the spacing; the schedule can: the second call waits one whole interval.
	if len(waitsCopy) != 1 || waitsCopy[0] != defaultNYTInterval {
		t.Errorf("pacing waits = %v, want one wait of %v", waitsCopy, defaultNYTInterval)
	}
}

func TestNYTCancelsLongRetryAfter(t *testing.T) {
	var calls atomic.Int32
	client, _ := newNYTTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	fixed := time.Now()
	client.now = func() time.Time { return fixed }
	enteredWait := make(chan struct{})
	var requestedWait time.Duration
	client.wait = func(ctx context.Context, d time.Duration) error {
		requestedWait = d
		close(enteredWait)
		return sleepContext(ctx, d)
	}

	// A Retry-After this long is the daily quota, waited for only because the
	// deadline is after it.
	ctx, cancel := context.WithDeadline(context.Background(), fixed.Add(2*time.Hour))
	result := make(chan error, 1)
	go func() {
		_, err := client.ArchiveContext(ctx, time.Now().Year(), 1)
		result <- err
	}()
	select {
	case <-enteredWait:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("429 response did not enter its retry wait")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Error("retry wait did not preserve context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("long retry wait did not stop after cancellation")
	}
	if requestedWait < time.Hour {
		t.Error("client did not honor the longer Retry-After value")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("canceled retry wait sent %d requests", got)
	}
}

func TestNYTHTTPTimeoutAndCredentialSafeErrors(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		started := make(chan struct{})
		client, _ := newNYTTestClient(t, func(_ http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done()
		})
		client.http.Timeout = 40 * time.Millisecond
		_, err := client.Archive(time.Now().Year(), 1)
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed-out request did not reach the local server")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Error("HTTP timeout did not preserve deadline exceeded")
		}
		if err != nil && strings.Contains(err.Error(), nytTestAPIKey) {
			t.Error("HTTP timeout error exposed the API key")
		}
	})

	t.Run("malformed response echo", func(t *testing.T) {
		client, _ := newNYTTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(nytTestAPIKey + " is not JSON"))
		})
		_, err := client.Archive(time.Now().Year(), 1)
		if err == nil {
			t.Fatal("malformed response was accepted")
		}
		if strings.Contains(err.Error(), nytTestAPIKey) {
			t.Error("malformed response error exposed the API key")
		}
	})

	t.Run("transport error echo", func(t *testing.T) {
		client, _ := newNYTTestClient(t, func(http.ResponseWriter, *http.Request) {
			t.Error("custom transport unexpectedly reached the server")
		})
		client.http.Transport = nytRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("upstream echoed " + nytTestAPIKey)
		})
		_, err := client.Archive(time.Now().Year(), 1)
		if err == nil {
			t.Fatal("transport failure was ignored")
		}
		if strings.Contains(err.Error(), nytTestAPIKey) {
			t.Error("transport error exposed the API key")
		}
	})

	t.Run("redirect refused", func(t *testing.T) {
		var destinationCalls atomic.Int32
		destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			destinationCalls.Add(1)
		}))
		t.Cleanup(destination.Close)
		client, _ := newNYTTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, destination.URL+"/redirected", http.StatusFound)
		})
		_, err := client.Archive(time.Now().Year(), 1)
		if err == nil {
			t.Fatal("redirect response was accepted")
		}
		if strings.Contains(err.Error(), nytTestAPIKey) {
			t.Error("redirect error exposed the API key")
		}
		if got := destinationCalls.Load(); got != 0 {
			t.Errorf("redirect destination received %d requests", got)
		}
	})
}

type nytRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f nytRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
