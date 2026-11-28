package csv

import (
	"math"
	"strconv"
)

// InferColumn types one column of raw text cells, the rule CSV and Excel
// reading share, and returns the typed cells (data itself when it stays text):
//   - every cell a valid int64 and no cell empty  → int64
//   - otherwise every non-empty cell numeric       → float64 (empty → NaN)
//   - otherwise                                     → cells left as they are
//
// Integer columns become int64 rather than float64 so large-integer columns
// (e.g. 19-digit IDs, nanosecond timestamps) keep full precision instead of
// being rounded to the nearest float64 above 2^53. This matches how
// pandas.read_csv types a column. A nil cell is one the file left out, which
// Excel does past a row's last value; CSV never has one.
func InferColumn(data []any) []any {
	if len(data) == 0 {
		return data
	}
	allInt, allFloat, hasEmpty := true, true, false
	for _, v := range data {
		if v == nil {
			hasEmpty = true
			continue
		}
		s, ok := v.(string)
		if !ok {
			return data
		}
		if s == "" {
			hasEmpty = true
			continue
		}
		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			allInt = false
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			allFloat = false
		}
	}
	switch {
	case allInt && !hasEmpty:
		for i, v := range data {
			n, _ := strconv.ParseInt(v.(string), 10, 64)
			data[i] = n
		}
	case allFloat:
		for i, v := range data {
			s, _ := v.(string)
			if s == "" {
				data[i] = math.NaN()
				continue
			}
			f, _ := strconv.ParseFloat(s, 64)
			data[i] = f
		}
	}
	return data
}
