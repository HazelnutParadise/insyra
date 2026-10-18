package datafetch

import "fmt"

// String returns the constant name of v, or GoogleMapsStoreReviewSortBy(n) for a value that
// is not one of the constants.
func (g GoogleMapsStoreReviewSortBy) String() string {
	switch g {
	case GoogleMapsStoreReviewSortByRelevance:
		return "GoogleMapsStoreReviewSortByRelevance"
	case GoogleMapsStoreReviewSortByNewest:
		return "GoogleMapsStoreReviewSortByNewest"
	case GoogleMapsStoreReviewSortByHighestRating:
		return "GoogleMapsStoreReviewSortByHighestRating"
	case GoogleMapsStoreReviewSortByLowestRating:
		return "GoogleMapsStoreReviewSortByLowestRating"
	}
	return fmt.Sprintf("GoogleMapsStoreReviewSortBy(%d)", int(g))
}
