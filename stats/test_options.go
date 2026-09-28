package stats

import "fmt"

// oneOptions returns the value a trailing `opts ...T` parameter was given,
// or T's zero value when it was given none. The parameter stands for one
// optional value, so more than one is an error.
func oneOptions[T any](opts []T) (T, error) {
	var zero T
	switch len(opts) {
	case 0:
		return zero, nil
	case 1:
		return opts[0], nil
	default:
		return zero, fmt.Errorf("at most one %T may be given, got %d", zero, len(opts))
	}
}

// resolveTestSettings reads the alternative hypothesis and confidence level
// a hypothesis test was given: an empty alternative is TwoSided and a zero
// level is 0.95. Any other alternative, and any other level outside (0, 1),
// NaN included, is an error.
func resolveTestSettings(alt AlternativeHypothesis, cl float64) (AlternativeHypothesis, float64, error) {
	switch alt {
	case "":
		alt = TwoSided
	case TwoSided, Greater, Less:
	default:
		return alt, 0, fmt.Errorf("alternative must be two-sided, greater or less, got %q", string(alt))
	}

	switch {
	case cl == 0:
		cl = defaultConfidenceLevel
	case cl > 0 && cl < 1:
	default:
		return alt, 0, fmt.Errorf("confidence level must be strictly between 0 and 1, got %v", cl)
	}

	return alt, cl, nil
}
