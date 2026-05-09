package ccl

import (
	"fmt"
	"math"
	"sort"
)

// collectFloats walks the aggregate input the same way forEachValue does and
// collects all values that can be coerced to float64.
func collectFloats(args [][]any) []float64 {
	var out []float64
	forEachValue(args, func(val any) {
		if f, ok := toFloat64(val); ok {
			out = append(out, f)
		}
	})
	return out
}

// registerAggregateStatFunctions registers MEDIAN / STDEV / STDEVP / VAR / VARP.
// Sample variants (STDEV, VAR) divide by n-1; population variants (STDEVP,
// VARP) divide by n. All ignore nil values, matching SUM/AVG/COUNT semantics.
func registerAggregateStatFunctions() {
	registerAggregateFunction("MEDIAN", func(args ...[]any) (any, error) {
		vals := collectFloats(args)
		if len(vals) == 0 {
			return nil, nil
		}
		sort.Float64s(vals)
		n := len(vals)
		if n%2 == 1 {
			return vals[n/2], nil
		}
		return (vals[n/2-1] + vals[n/2]) / 2.0, nil
	})

	variance := func(vals []float64, sample bool) (float64, bool) {
		n := len(vals)
		if n < 1 {
			return 0, false
		}
		if sample && n < 2 {
			return 0, false
		}
		var sum float64
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(n)
		var ss float64
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		denom := float64(n)
		if sample {
			denom = float64(n - 1)
		}
		return ss / denom, true
	}

	registerAggregateFunction("VAR", func(args ...[]any) (any, error) {
		vals := collectFloats(args)
		v, ok := variance(vals, true)
		if !ok {
			if len(vals) < 2 {
				return nil, fmt.Errorf("VAR requires at least 2 numeric values")
			}
			return nil, nil
		}
		return v, nil
	})

	registerAggregateFunction("VARP", func(args ...[]any) (any, error) {
		vals := collectFloats(args)
		v, ok := variance(vals, false)
		if !ok {
			return nil, fmt.Errorf("VARP requires at least 1 numeric value")
		}
		return v, nil
	})

	registerAggregateFunction("STDEV", func(args ...[]any) (any, error) {
		vals := collectFloats(args)
		v, ok := variance(vals, true)
		if !ok {
			if len(vals) < 2 {
				return nil, fmt.Errorf("STDEV requires at least 2 numeric values")
			}
			return nil, nil
		}
		return math.Sqrt(v), nil
	})

	registerAggregateFunction("STDEVP", func(args ...[]any) (any, error) {
		vals := collectFloats(args)
		v, ok := variance(vals, false)
		if !ok {
			return nil, fmt.Errorf("STDEVP requires at least 1 numeric value")
		}
		return math.Sqrt(v), nil
	})
}
