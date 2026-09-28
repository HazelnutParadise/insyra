package datafetch

import (
	"strings"
	"testing"
	"time"
)

// #212 (DF-5): MaxWaitingInterval takes the wait between review pages as a
// time.Duration, with the rules the deprecated millisecond field has.

func TestGetReviewsWaitsWithADurationLimit(t *testing.T) {
	logged := captureGmapsWarnings(t)
	posted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server, requests := reviewServer(t, map[string]string{
		"":       reviewPage(t, "page-2", reviewRecord(5, "", posted, "A", "", "")),
		"page-2": reviewPage(t, "page-3", reviewRecord(4, "", posted, "B", "", "")),
	})

	start := time.Now()
	reviews := crawlerFor(server).GetReviews("0x1:0x2", 2, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval: time.Second})

	if len(reviews) != 2 {
		t.Errorf("got %d reviews over two pages, want 2", len(reviews))
	}
	if n := len(requests()); n != 2 {
		t.Errorf("sent %d requests, want 2", n)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("waited %v between pages, want at least 1s", elapsed)
	}
	if logged.Len() != 0 {
		t.Errorf("a valid limit logged %q", logged.String())
	}
}

func TestGetReviewsRefusesBothWaitingFields(t *testing.T) {
	logged := captureGmapsWarnings(t)
	server, requests := reviewServer(t, map[string]string{"": reviewPage(t, "")})

	reviews := crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{
		MaxWaitingInterval:              2 * time.Second,
		MaxWaitingInterval_Milliseconds: 2000,
	})

	if reviews != nil {
		t.Errorf("got %d reviews, want nil", len(reviews))
	}
	if n := len(requests()); n != 0 {
		t.Errorf("sent %d requests, want none", n)
	}
	if !strings.Contains(logged.String(), "MaxWaitingInterval_Milliseconds") {
		t.Errorf("the warning %q does not name the deprecated field", logged.String())
	}
}

func TestGetReviewsReplacesAWaitUnderOneSecond(t *testing.T) {
	logged := captureGmapsWarnings(t)
	posted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server, _ := reviewServer(t, map[string]string{"": reviewPage(t, "", reviewRecord(5, "", posted, "A", "", ""))})

	reviews := crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval: 500 * time.Millisecond})

	if len(reviews) != 1 {
		t.Errorf("got %d reviews, want 1", len(reviews))
	}
	if !strings.Contains(logged.String(), "too small") {
		t.Errorf("the warning %q does not say the limit is too small", logged.String())
	}
}
