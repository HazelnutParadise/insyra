package datafetch

import (
	"fmt"
	"testing"
)

// TestEnumsPrintTheirNames pins enum-conventions: every numeric enum
// prints a name, and a value outside its constants prints TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{GoogleMapsStoreReviewSortByRelevance, "GoogleMapsStoreReviewSortByRelevance"},
		{GoogleMapsStoreReviewSortByNewest, "GoogleMapsStoreReviewSortByNewest"},
		{GoogleMapsStoreReviewSortByHighestRating, "GoogleMapsStoreReviewSortByHighestRating"},
		{GoogleMapsStoreReviewSortByLowestRating, "GoogleMapsStoreReviewSortByLowestRating"},
		{GoogleMapsStoreReviewSortBy(99), "GoogleMapsStoreReviewSortBy(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}
