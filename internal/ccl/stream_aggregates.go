package ccl

import (
	"fmt"
	"math"
	"strings"
)

// StreamingAggregate computes a built-in aggregate from values fed in order, a
// batch at a time, without holding them. Fed the same values in the same order
// as the aggregate function would see them all at once, it returns what that
// function returns, bit for bit, because it runs the same arithmetic in the
// same order.
type StreamingAggregate interface {
	// Passes is how many times every value has to be fed: 2 for VAR, VARP,
	// STDEV and STDEVP, which need the mean before the squared deviations,
	// and 1 for the rest.
	Passes() int
	// BeginPass starts pass p, counting from 0. Pass 0 starts on its own.
	BeginPass(p int)
	// Add feeds the next values. A value that is itself a []any is walked the
	// way the aggregate function walks its arguments.
	Add(values []any)
	// Result returns the aggregate's value once every pass has been fed.
	Result() (any, error)
}

// streamingAggregate is the one implementation behind every name. Which fields
// mean anything depends on the name, the way the fields a given aggregate
// function reads depend on which function it is.
type streamingAggregate struct {
	kind string

	sum   float64
	count int // AVG's numeric values; n is the variance family's
	n     int

	maxVal float64
	minVal float64
	found  bool

	ss   float64
	mean float64

	pass int
}

// NewStreamingAggregate returns the streaming form of the built-in aggregate
// name, in any letter case: SUM, AVG, COUNT, MIN, MAX, VAR, VARP, STDEV or
// STDEVP. It returns false for any other name, MEDIAN included, which needs
// every value at once, and for one of these names after a caller re-registered
// it with RegisterAggregateFunction, whose arithmetic is then the caller's.
func NewStreamingAggregate(name string) (StreamingAggregate, bool) {
	if isUserAggregate(name) {
		return nil, false
	}
	kind := strings.ToUpper(name)
	switch kind {
	case "SUM", "AVG", "COUNT", "MAX", "MIN", "VAR", "VARP", "STDEV", "STDEVP":
	default:
		return nil, false
	}
	s := &streamingAggregate{kind: kind}
	switch kind {
	case "MAX":
		s.maxVal = -math.MaxFloat64
	case "MIN":
		s.minVal = math.MaxFloat64
	}
	return s, true
}

// isUserAggregate reports whether a caller has registered name through
// RegisterAggregateFunction, in which case its arithmetic is the caller's
// rather than a built-in this file can reproduce.
func isUserAggregate(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return userAggregates[strings.ToUpper(name)]
}

func (s *streamingAggregate) Passes() int {
	switch s.kind {
	case "VAR", "VARP", "STDEV", "STDEVP":
		return 2
	}
	return 1
}

func (s *streamingAggregate) BeginPass(p int) {
	s.pass = p
	if p == 1 {
		s.mean = s.sum / float64(s.n)
	}
}

func (s *streamingAggregate) Add(values []any) {
	switch s.kind {
	case "SUM":
		forEachValue([][]any{values}, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				s.sum += f
			}
		})
	case "AVG":
		forEachValue([][]any{values}, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				s.sum += f
				s.count++
			}
		})
	case "COUNT":
		// forEachValue hands over everything but nil, which is all COUNT counts.
		forEachValue([][]any{values}, func(val any) {
			s.count++
		})
	case "MAX":
		forEachValue([][]any{values}, func(val any) {
			if f, ok := toFloat64(val); ok {
				if f > s.maxVal {
					s.maxVal = f
					s.found = true
				}
			}
		})
	case "MIN":
		forEachValue([][]any{values}, func(val any) {
			if f, ok := toFloat64(val); ok {
				if f < s.minVal {
					s.minVal = f
					s.found = true
				}
			}
		})
	case "VAR", "VARP", "STDEV", "STDEVP":
		if s.pass == 0 {
			forEachValue([][]any{values}, func(val any) {
				if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
					s.sum += f
					s.n++
				}
			})
			return
		}
		forEachValue([][]any{values}, func(val any) {
			if f, ok := toFloat64(val); ok && !math.IsNaN(f) {
				d := f - s.mean
				s.ss += d * d
			}
		})
	}
}

func (s *streamingAggregate) Result() (any, error) {
	switch s.kind {
	case "SUM":
		return s.sum, nil
	case "AVG":
		if s.count == 0 {
			return 0.0, nil
		}
		return s.sum / float64(s.count), nil
	case "COUNT":
		return float64(s.count), nil
	case "MAX":
		if !s.found {
			return nil, nil
		}
		return s.maxVal, nil
	case "MIN":
		if !s.found {
			return nil, nil
		}
		return s.minVal, nil
	case "VAR":
		v, ok := s.variance(true)
		if !ok {
			return nil, fmt.Errorf("VAR requires at least 2 numeric values")
		}
		return v, nil
	case "VARP":
		v, ok := s.variance(false)
		if !ok {
			return nil, fmt.Errorf("VARP requires at least 1 numeric value")
		}
		return v, nil
	case "STDEV":
		v, ok := s.variance(true)
		if !ok {
			return nil, fmt.Errorf("STDEV requires at least 2 numeric values")
		}
		return math.Sqrt(v), nil
	case "STDEVP":
		v, ok := s.variance(false)
		if !ok {
			return nil, fmt.Errorf("STDEVP requires at least 1 numeric value")
		}
		return math.Sqrt(v), nil
	}
	return nil, fmt.Errorf("no streaming form for aggregate %s", s.kind)
}

// variance closes out the four variance aggregates from what two passes
// accumulated: the mean is settled, so only the squared deviations and the
// denominator are left.
func (s *streamingAggregate) variance(sample bool) (float64, bool) {
	n := s.n
	if n < 1 {
		return 0, false
	}
	if sample && n < 2 {
		return 0, false
	}
	denom := float64(n)
	if sample {
		denom = float64(n - 1)
	}
	return s.ss / denom, true
}
