package datafetch

import (
	"errors"
	"fmt"
	"time"
)

// Stable sentinel errors for the TWGeocoding fetcher. Callers should match these
// with errors.Is rather than comparing error strings.
var (
	// ErrGeocodeNotFound means the coordinate does not fall inside any Taiwan
	// village polygon (e.g. a point at sea or outside Taiwan). It is a definitive,
	// non-retryable outcome.
	ErrGeocodeNotFound = errors.New("datafetch: geocoding point not found")
	// ErrGeocodeTimeout means the request exceeded the configured timeout.
	ErrGeocodeTimeout = errors.New("datafetch: geocoding request timeout")
	// ErrGeocodeRateLimited means the free-tier quota (15 requests/hour) is
	// exhausted. A *RateLimitError unwraps to this sentinel.
	ErrGeocodeRateLimited = errors.New("datafetch: geocoding rate limited")
)

// RateLimitError carries the rate-limit state parsed from the API's
// X-Ratelimit-* response headers. It unwraps to ErrGeocodeRateLimited so callers
// can either branch with errors.Is(err, ErrGeocodeRateLimited) or read ResetAt
// for a precise back-off.
type RateLimitError struct {
	Limit     int       // X-Ratelimit-Limit (requests per window)
	Remaining int       // X-Ratelimit-Remaining
	ResetAt   time.Time // X-Ratelimit-Reset, zero if the header was absent/unparseable
}

func (e *RateLimitError) Error() string {
	if !e.ResetAt.IsZero() {
		return fmt.Sprintf("datafetch: geocoding rate limited (limit %d, remaining %d, resets at %s)",
			e.Limit, e.Remaining, e.ResetAt.Format(time.RFC3339))
	}
	return fmt.Sprintf("datafetch: geocoding rate limited (limit %d, remaining %d)", e.Limit, e.Remaining)
}

// Unwrap lets errors.Is(err, ErrGeocodeRateLimited) succeed for a *RateLimitError.
func (e *RateLimitError) Unwrap() error { return ErrGeocodeRateLimited }
