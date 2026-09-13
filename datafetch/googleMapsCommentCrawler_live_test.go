//go:build gmaps_live

// Live smoke tests against Google.
//
// Excluded from the default build so `go test ./...` never depends on Google
// or sends it traffic. Run explicitly with:
//
//	go test -tags gmaps_live -run 'TestGoogleMaps.*Live' ./datafetch/
package datafetch

import (
	"strings"
	"testing"
)

func TestGoogleMapsSearchLive(t *testing.T) {
	stores := GoogleMapsStores().Search("鼎泰豐")
	if len(stores) == 0 {
		t.Fatal("Search returned no stores; Google may have changed the response")
	}
	for _, s := range stores {
		if !gmapsFeatureIDRe.MatchString(s.ID) {
			t.Errorf("store %q has an ID that is not a feature ID: %q", s.Name, s.ID)
		}
	}
	if !strings.Contains(stores[0].Name, "鼎泰豐") {
		t.Errorf("the first result is %q, which does not look like a match", stores[0].Name)
	}
	t.Logf("live result: %d stores, first %s (%s)", len(stores), stores[0].Name, stores[0].ID)
}

// Two pages of the newest reviews for 鼎泰豐 板橋店, which has thousands.
func TestGoogleMapsReviewsLive(t *testing.T) {
	reviews := GoogleMapsStores().GetReviews("0x3442a819486a12c5:0x999160f21e520bde", 2,
		GoogleMapsStoreReviewsFetchingOptions{SortBy: SortByNewest, MaxWaitingInterval_Milliseconds: 2000})
	if len(reviews) != 20 {
		t.Fatalf("got %d reviews over two pages, want 20; Google may have changed the response", len(reviews))
	}
	for i, r := range reviews {
		if r.Reviewer == "" || r.ReviewerID == "" || r.Rating < 1 || r.Rating > 5 || len(r.ReviewDate) != len("2006-01-02") {
			t.Errorf("review %d is missing fields: %+v", i, r)
		}
		if i > 0 && r.ReviewDate > reviews[i-1].ReviewDate {
			t.Errorf("review %d (%s) is newer than review %d (%s); SortByNewest was not applied", i, r.ReviewDate, i-1, reviews[i-1].ReviewDate)
		}
	}
	t.Logf("live result: %d reviews, newest %s, oldest %s", len(reviews), reviews[0].ReviewDate, reviews[len(reviews)-1].ReviewDate)
}
