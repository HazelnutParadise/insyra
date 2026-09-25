package datafetch

import "testing"

// GetReviews used to warn and fall back to the default options when given
// more than one. It refuses, before any request is made.
func TestGetReviewsRefusesMoreThanOneOptions(t *testing.T) {
	crawler := &googleMapsStoreCrawler{}
	opts := GoogleMapsStoreReviewsFetchingOptions{}
	if reviews := crawler.GetReviews("store", 1, opts, opts); reviews != nil {
		t.Fatal("GetReviews ran with two option sets")
	}
}
