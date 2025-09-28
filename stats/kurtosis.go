package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

// KurtosisMethod defines available kurtosis calculation methods.
type KurtosisMethod int

const (
	KurtosisG2           KurtosisMethod = iota + 1 // Type 1: g2 (default)
	KurtosisAdjusted                               // Type 2: adjusted Fisher kurtosis
	KurtosisBiasAdjusted                           // Type 3: bias-adjusted
)

// Kurtosis calculates the kurtosis of the DataList.
// method: 1 = g2, 2 = adjusted Fisher kurtosis, 3 = bias-adjusted.
// Default is KurtosisG2.
// Returns NaN if the data is empty or undefined.
func Kurtosis(data any, method ...KurtosisMethod) float64 {
	d, dLen := insyra.ProcessData(data)
	if dLen == 0 {
		return math.NaN()
	}
	d64 := insyra.SliceToF64(d)
	dl := insyra.NewDataList(d64)

	useMethod := KurtosisG2
	if len(method) > 0 {
		useMethod = method[0]
	}
	if len(method) > 1 {
		return math.NaN()
	}

	switch useMethod {
	case KurtosisG2:
		return calculateKurtType1(dl)
	case KurtosisAdjusted:
		return calculateKurtType2(dl)
	case KurtosisBiasAdjusted:
		return calculateKurtType3(dl)
	default:
		return math.NaN()
	}
}

// ======================== calculation functions ========================

func calculateKurtType1(dl insyra.IDataList) float64 {
	var n, m2, m4 float64
	dl.AtomicDo(func(l *insyra.DataList) {
		n = float64(l.Len())
		if n == 0 {
			return
		}
		m2 = CalculateMoment(l, 2, true)
		m4 = CalculateMoment(l, 4, true)
	})

	if n == 0 || m2 == 0 {
		return math.NaN()
	}

	// g2 = m4 / m2^2 - 3
	return m4/(m2*m2) - 3
}

func calculateKurtType2(dl insyra.IDataList) float64 {
	var n, g2 float64
	dl.AtomicDo(func(l *insyra.DataList) {
		n = float64(l.Len())
		if n < 4 {
			return
		}
		g2 = calculateKurtType1(l)
	})
	if n < 4 {
		return math.NaN()
	}

	// adjusted Fisher kurtosis
	nPlus1 := n + 1
	nMinus1 := n - 1
	nMinus2 := n - 2
	nMinus3 := n - 3

	numerator := (g2*(nPlus1) + 6) * nMinus1
	denominator := nMinus2 * nMinus3

	return numerator / denominator
}

func calculateKurtType3(dl insyra.IDataList) float64 {
	var n, g2 float64
	dl.AtomicDo(func(l *insyra.DataList) {
		n = float64(l.Len())
		if n == 0 {
			return
		}
		g2 = calculateKurtType1(l)
	})

	if n == 0 {
		return math.NaN()
	}
	kurt := g2 + 3 // convert to raw kurtosis

	// bias-adjusted version
	adjustment := (1 - 1/n) * (1 - 1/n)
	return kurt*adjustment - 3
}
