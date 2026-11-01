package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// The parametric tests below take every observation through numericSlice so a
// blank, text, NaN or Inf cell is an error naming its row instead of a cell
// that DataList.Mean skips while DataList.Len still counts it — which used to
// turn p = 0.074 into p = 0.028 on [1, 2, nil, 3].

// testSeries reads one series for a hypothesis test. label names it in errors.
// The list goes through asDataList first, so any IDataList implementation is
// accepted and a nil or typed nil list reads as empty instead of panicking.
func testSeries(dl insyra.IDataList, label string) ([]float64, error) {
	values, _, err := numericSlice(asDataList(dl), label)
	return values, err
}

// testSeriesPair reads two series for a two-sample test as one step. Reading
// them with two testSeries calls locks each list on its own, so a writer that
// changes both together under insyra.AtomicDoAll can land between the two
// reads and the test then mixes one sample's old state with the other's new
// one. v0.3.2 took both under one AtomicDoAll and so does this.
//
// Only the snapshot is taken under the lock. DataList.Data() returns a copy,
// so the numeric validation — which allocates and formats errors — runs
// afterwards and does not hold two actors while it does.
func testSeriesPair(a, b insyra.IDataList, labelA, labelB string) ([]float64, []float64, error) {
	dlA := asDataList(a)
	dlB := asDataList(b)

	var rawA, rawB []any
	insyra.AtomicDoAll(func() {
		rawA = dlA.Data()
		rawB = dlB.Data()
	}, dlA, dlB)

	valuesA, err := numericValues(rawA, labelA)
	if err != nil {
		return nil, nil, err
	}
	valuesB, err := numericValues(rawB, labelB)
	if err != nil {
		return nil, nil, err
	}
	return valuesA, valuesB, nil
}

// meanOfF64 mirrors DataList.Mean: an in-order sum divided by the count.
func meanOfF64(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// sampleVarianceF64 mirrors DataList.Var: two passes, n-1 denominator, NaN
// below two observations. Same summation order, so results are bit-identical.
func sampleVarianceF64(values []float64) float64 {
	if len(values) < 2 {
		return math.NaN()
	}
	mean := meanOfF64(values)
	var numerator float64
	for _, v := range values {
		numerator += (v - mean) * (v - mean)
	}
	return numerator / float64(len(values)-1)
}

// medianOfF64 mirrors DataList.Median on an all-numeric series.
func medianOfF64(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// eventFlags reads the event indicator, which may be bool or 0/1.
func eventFlags(dl insyra.IDataList, want int) ([]bool, error) {
	if insyra.IsNilList(dl) {
		return nil, errors.New("observed is nil")
	}
	var raw []any
	dl.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })
	if len(raw) != want {
		return nil, fmt.Errorf("durations has %d rows and observed has %d", want, len(raw))
	}
	out := make([]bool, len(raw))
	for i, v := range raw {
		switch x := v.(type) {
		case bool:
			out[i] = x
		default:
			f, ok := insyra.ToFloat64Safe(v)
			if !ok || (f != 0 && f != 1) {
				return nil, fmt.Errorf("observed holds %v at row index %d; it must be true, false, 1 or 0", v, i)
			}
			out[i] = f == 1
		}
	}
	return out, nil
}

// spells reads and checks the durations and the event flags.
func spells(durations, observed insyra.IDataList) ([]int, []bool, error) {
	t, err := testSeries(durations, "durations")
	if err != nil {
		return nil, nil, err
	}
	if len(t) == 0 {
		return nil, nil, errors.New("durations is empty")
	}
	flags, err := eventFlags(observed, len(t))
	if err != nil {
		return nil, nil, err
	}
	d := make([]int, len(t))
	for i, v := range t {
		if v < 1 || v != math.Trunc(v) || v > math.MaxInt32 {
			return nil, nil, fmt.Errorf("durations holds %v at row index %d; a duration must be a whole number of periods, at least 1", v, i)
		}
		d[i] = int(v)
	}
	return d, flags, nil
}

// spellCovariates reads the covariates, one value per spell, and their names.
func spellCovariates(covariates []insyra.IDataList, n int) ([][]float64, []string, error) {
	values := make([][]float64, len(covariates))
	names := make([]string, len(covariates))
	seen := map[string]bool{"Intercept": true, "LogPeriod": true, "Period": true, "Spell": true, "Event": true}
	for j, c := range covariates {
		label := fmt.Sprintf("covariate %d", j+1)
		v, err := testSeries(c, label)
		if err != nil {
			return nil, nil, err
		}
		if len(v) != n {
			return nil, nil, fmt.Errorf("%s has %d values for %d spells", label, len(v), n)
		}
		values[j] = v
		var name string
		c.AtomicDo(func(l *insyra.DataList) { name = l.GetName() })
		if name == "" {
			name = fmt.Sprintf("X%d", j+1)
		}
		if seen[name] || periodTermName(name) {
			return nil, nil, fmt.Errorf("covariate %d is named %q, which another term already uses", j+1, name)
		}
		seen[name] = true
		names[j] = name
	}
	return values, names, nil
}

// periodTermName reports whether name is one of the baseline terms
// Period2, Period3, ... of HazardBaselineDummies.
func periodTermName(name string) bool {
	digits := strings.TrimPrefix(name, "Period")
	if digits == name || digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
