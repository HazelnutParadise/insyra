package stats

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// Layer 0 — numeric input conversion.
//
// Every routine in this package that reads numbers out of a DataList or a
// DataTable goes through here. It exists because `DataList.ToF64Slice` has no
// failure channel: it routes each value through `insyra.ToFloat64`, which
// yields 0 for anything it cannot parse. A slice of the right length comes
// back whatever was in the column, so the caller has no way to tell a real
// zero from a value that was never read — and the answer computed from it is
// wrong in a way that looks entirely plausible.
//
// The families that deliberately do something other than refuse do not come
// through here: the decision tree learns a direction for missing values per
// node, and factor analysis deletes the whole observation. Both are documented
// treatments. Substituting a zero is not one.

// appendNumericValues appends raw to dst as float64, refusing anything that
// is not a finite number. label names the list in the error and is only
// called when a cell is refused, so a caller looping over many groups does
// not format a name it never prints. position names what the one-based index
// counts: "row" for a list of observations, "condition" for one subject's
// conditions in a repeated-measures design. On error the returned slice is nil.
func appendNumericValues(dst []float64, raw []any, label func() string, position string) ([]float64, error) {
	for i, value := range raw {
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok {
			return nil, fmt.Errorf("%s contains a non-numeric value at %s %d: %v", label(), position, i+1, value)
		}
		if math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil, fmt.Errorf("%s contains a non-finite value at %s %d: %v", label(), position, i+1, converted)
		}
		dst = append(dst, converted)
	}
	return dst, nil
}

// numericValues converts raw values to float64, refusing anything that is not
// a finite number. label names the column or series in the error, so the
// caller can find the offending cell.
func numericValues(raw []any, label string) ([]float64, error) {
	return appendNumericValues(make([]float64, 0, len(raw)), raw, func() string { return label }, "row")
}

// numericSlice reads a DataList under its actor and converts it with
// numericValues. It returns the length separately so a caller checking that
// several series agree in length still gets that length when conversion fails.
func numericSlice(dl insyra.IDataList, label string) ([]float64, int, error) {
	if dl == nil {
		return nil, 0, fmt.Errorf("%s data list is nil", label)
	}
	var raw []any
	var length int
	dl.AtomicDo(func(l *insyra.DataList) {
		length = l.Len()
		raw = l.Data()
	})
	values, err := numericValues(raw, label)
	if err != nil {
		return nil, length, err
	}
	return values, length, nil
}

// requireNumericPair rejects either series holding a value that is not a
// finite number. It exists for the entry points whose internals read values
// through paths that cannot report a failure, where the only place a refusal
// can be raised is before the work starts.
func requireNumericPair(dlX, dlY insyra.IDataList) error {
	if _, _, err := numericSlice(dlX, "x"); err != nil {
		return err
	}
	_, _, err := numericSlice(dlY, "y")
	return err
}
