package stats

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// SkewnessMethod defines available skewness calculation methods.
type SkewnessMethod int

const (
	SkewnessG1           SkewnessMethod = iota + 1 // Type 1: G1 (default)
	SkewnessAdjusted                               // Type 2: Adjusted Fisher-Pearson
	SkewnessBiasAdjusted                           // Type 3: Bias-adjusted
)

// Skewness calculates the skewness of a sample using the specified method.
func Skewness(sample any, method ...SkewnessMethod) (float64, error) {
	d, dLen := insyra.ProcessData(sample)
	if dLen == 0 {
		return math.NaN(), errors.New("empty data")
	}

	d64 := insyra.SliceToF64(d)
	dl := insyra.NewDataList(d64)
	n := float64(dl.Len())

	usemethod := SkewnessG1
	if len(method) > 0 {
		usemethod = method[0]
	}
	if len(method) > 1 {
		return math.NaN(), errors.New("too many methods specified")
	}

	if usemethod == SkewnessAdjusted && n < 3 {
		return math.NaN(), errors.New("insufficient data for adjusted method (n < 3)")
	}

	m2, err := CalculateMoment(dl, 2, true)
	if err != nil {
		return math.NaN(), err
	}
	m3, err := CalculateMoment(dl, 3, true)
	if err != nil {
		return math.NaN(), err
	}

	if m2 == 0 {
		return math.NaN(), errors.New("skewness undefined for zero variance")
	}

	// m2^(3/2) = m2 · sqrt(m2). Replaces math.Pow(m2, 1.5) which goes through
	// exp(1.5·log(m2)); the multiplicative form is faster and 1–2 ULPs more
	// accurate. Same shape applied to ((n-1)/n)^(3/2) below.
	g1 := m3 / (m2 * math.Sqrt(m2))

	switch usemethod {
	case SkewnessG1:
		return g1, nil
	case SkewnessAdjusted:
		return g1 * math.Sqrt(n*(n-1)) / (n - 2), nil
	case SkewnessBiasAdjusted:
		ratio := (n - 1) / n
		return g1 * (ratio * math.Sqrt(ratio)), nil
	default:
		return math.NaN(), errors.New("unknown skewness method")
	}
}
