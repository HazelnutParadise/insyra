package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// RunsTestOptions are the settings of RunsTest. The zero value splits at 0 with
// no continuity correction.
type RunsTestOptions struct {
	// Threshold splits the values into above and below. The default 0 is the
	// sign of a return (Fama 1965); use the mean or the median for other series.
	Threshold float64
	// Correction applies a continuity correction of 0.5 to the number of runs.
	// Textbook formulas have none, so it is off by default.
	Correction bool
}

// RunsTestResult is the result of RunsTest.
type RunsTestResult struct {
	Runs     int     // the number of runs
	Above    int     // values above the threshold
	Below    int     // values below the threshold
	Dropped  int     // values equal to the threshold, left out before counting runs
	Expected float64 // the expected number of runs when the order is random
	Variance float64 // the variance of the number of runs when the order is random
	Z        float64 // (Runs - Expected) / sqrt(Variance), after the correction if asked
	PValue   float64 // the two-tailed normal p-value of Z
}

// RunsTest tests whether the order of up and down moves is random (Wald and
// Wolfowitz 1940). Each value is marked above or below the threshold; a run is
// an unbroken stretch of one mark, so "++--+-" has four runs. Too few runs
// means the series keeps going the same way (momentum), too many means it keeps
// reversing. Values equal to the threshold are dropped before the runs are
// counted and reported in Dropped.
//
// With n1 values above, n2 below and n = n1 + n2, a random order has
// 2*n1*n2/n + 1 runs on average and a variance of
// 2*n1*n2*(2*n1*n2 - n) / (n^2 * (n - 1)); the p-value is from the standard
// normal distribution, which is an asymptotic result and needs a long series.
// It fails when no value lies above or none below the threshold.
func RunsTest(series insyra.IDataList, opts ...RunsTestOptions) (*RunsTestResult, error) {
	if len(opts) > 1 {
		return nil, fmt.Errorf("RunsTest takes at most one RunsTestOptions, got %d", len(opts))
	}
	var o RunsTestOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if math.IsNaN(o.Threshold) || math.IsInf(o.Threshold, 0) {
		return nil, errors.New("the threshold must be a finite number")
	}
	x, err := testSeries(series, "series")
	if err != nil {
		return nil, err
	}

	res := &RunsTestResult{}
	last := 0 // +1 above, -1 below, 0 none seen yet
	for _, v := range x {
		var mark int
		switch {
		case v > o.Threshold:
			mark = 1
			res.Above++
		case v < o.Threshold:
			mark = -1
			res.Below++
		default:
			res.Dropped++
			continue
		}
		if mark != last {
			res.Runs++
			last = mark
		}
	}
	if res.Above == 0 || res.Below == 0 {
		return nil, fmt.Errorf("the runs test needs values on both sides of the threshold %g, got %d above and %d below", o.Threshold, res.Above, res.Below)
	}

	n1, n2 := float64(res.Above), float64(res.Below)
	n := n1 + n2
	res.Expected = 2*n1*n2/n + 1
	res.Variance = 2 * n1 * n2 * (2*n1*n2 - n) / (n * n * (n - 1))
	if res.Variance <= 0 {
		return nil, errors.New("the runs test needs at least two values on one side of the threshold")
	}
	diff := float64(res.Runs) - res.Expected
	if o.Correction {
		switch {
		case diff > 0:
			diff -= 0.5
		case diff < 0:
			diff += 0.5
		}
	}
	res.Z = diff / math.Sqrt(res.Variance)
	res.PValue = zPValue(res.Z, TwoSided)
	return res, nil
}
