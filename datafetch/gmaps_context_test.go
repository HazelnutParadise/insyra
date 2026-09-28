package datafetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// #250: Search and GetReviews have Context forms. A done context stops the
// request and the wait between review pages, and is reported like any other
// failure: nil and a warning.

func TestGoogleMapsContextMethodsStopOnACancelledContext(t *testing.T) {
	logged := captureGmapsWarnings(t)
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := crawlerFor(server).SearchContext(ctx, "鼎泰豐"); got != nil {
		t.Errorf("SearchContext returned %v, want nil", got)
	}
	if got := crawlerFor(server).GetReviewsContext(ctx, "0x1:0x2", 1); got != nil {
		t.Errorf("GetReviewsContext returned %v, want nil", got)
	}
	if n := atomic.LoadInt32(&count); n != 0 {
		t.Errorf("sent %d requests under a cancelled context, want none", n)
	}
	if !strings.Contains(logged.String(), "context canceled") {
		t.Errorf("the warning %q does not say the context was cancelled", logged.String())
	}
}

func TestGoogleMapsRefusesANilContext(t *testing.T) {
	logged := captureGmapsWarnings(t)
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
	}))
	defer server.Close()
	var nilCtx context.Context

	if got := crawlerFor(server).SearchContext(nilCtx, "x"); got != nil {
		t.Errorf("SearchContext returned %v, want nil", got)
	}
	if got := crawlerFor(server).GetReviewsContext(nilCtx, "0x1:0x2", 1); got != nil {
		t.Errorf("GetReviewsContext returned %v, want nil", got)
	}
	if n := atomic.LoadInt32(&count); n != 0 {
		t.Errorf("sent %d requests, want none", n)
	}
	if !strings.Contains(logged.String(), "nil context") {
		t.Errorf("the warning %q does not name the nil context", logged.String())
	}
}

func TestGetReviewsContextStopsDuringTheWaitBetweenPages(t *testing.T) {
	logged := captureGmapsWarnings(t)
	posted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server, requests := reviewServer(t, map[string]string{
		"":       reviewPage(t, "page-2", reviewRecord(5, "", posted, "A", "", "")),
		"page-2": reviewPage(t, "", reviewRecord(4, "", posted, "B", "", "")),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	// Every wait between pages is at least a second, so the context ends
	// during the first one.
	reviews := crawlerFor(server).GetReviewsContext(ctx, "0x1:0x2", 2, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval: 5 * time.Second})

	if reviews != nil {
		t.Errorf("got %d reviews, want nil", len(reviews))
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Errorf("returned after %v; the context ended after 200ms", elapsed)
	}
	if n := len(requests()); n != 1 {
		t.Errorf("sent %d requests, want only the first page", n)
	}
	if !strings.Contains(logged.String(), "deadline exceeded") {
		t.Errorf("the warning %q does not say the deadline passed", logged.String())
	}
}
