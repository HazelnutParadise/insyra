package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// PivotKind says whether a turning point is a high or a low.
type PivotKind int

const (
	// PivotLow is a trough: the price rose by the threshold after it.
	PivotLow PivotKind = iota
	// PivotHigh is a peak: the price fell by the threshold after it.
	PivotHigh
)

// String returns the constant name of k, or PivotKind(n) for a value that is
// not one of the constants.
func (k PivotKind) String() string {
	switch k {
	case PivotLow:
		return "PivotLow"
	case PivotHigh:
		return "PivotHigh"
	}
	return fmt.Sprintf("PivotKind(%d)", int(k))
}

// Pivot is one turning point found by ZigZag. Index and Price locate the
// extreme; ConfirmedAt is the first row on which the price had moved the
// threshold away from it, the earliest a trader could know about it. Trade on
// ConfirmedAt and describe the series with Index.
type Pivot struct {
	Kind        PivotKind
	Index       int
	Price       float64
	ConfirmedAt int
}

// ZigZag finds the turning points of a price series that reverse by at least
// threshold, a fraction such as 0.10 for 10%. Prices must be positive and
// finite.
//
// It is causal: a pivot is reported at the row that confirms it and uses no
// later price, so recomputing on a series cut short returns exactly the pivots
// of the full series whose ConfirmedAt lies before the cut. Pivots alternate
// between lows and highs. The extreme still being tracked at the end, which has
// not been confirmed, is not reported. A price that ties the running extreme
// leaves the earlier row as the extreme.
func ZigZag(prices insyra.IDataList, threshold float64) ([]Pivot, error) {
	if math.IsNaN(threshold) || threshold <= 0 || threshold >= 1 {
		return nil, errors.New("threshold must be a fraction between 0 and 1, such as 0.10 for 10%")
	}
	p, err := numericSeries(prices, "prices")
	if err != nil {
		return nil, err
	}
	for i, v := range p {
		if v <= 0 {
			return nil, fmt.Errorf("prices holds %v at row index %d; a percentage threshold needs positive prices", v, i)
		}
	}
	if len(p) == 0 {
		return nil, nil
	}

	var pivots []Pivot
	lowIdx, highIdx := 0, 0
	direction := 0 // 0 undecided, +1 rising since a confirmed low, -1 falling since a confirmed high
	for t := 1; t < len(p); t++ {
		switch direction {
		case 0:
			if p[t] < p[lowIdx] {
				lowIdx = t
			}
			if p[t] > p[highIdx] {
				highIdx = t
			}
			if p[t] >= p[lowIdx]*(1+threshold) {
				pivots = append(pivots, Pivot{PivotLow, lowIdx, p[lowIdx], t})
				direction, highIdx = 1, t
			} else if p[t] <= p[highIdx]*(1-threshold) {
				pivots = append(pivots, Pivot{PivotHigh, highIdx, p[highIdx], t})
				direction, lowIdx = -1, t
			}
		case 1:
			if p[t] > p[highIdx] {
				highIdx = t
			} else if p[t] <= p[highIdx]*(1-threshold) {
				pivots = append(pivots, Pivot{PivotHigh, highIdx, p[highIdx], t})
				direction, lowIdx = -1, t
			}
		case -1:
			if p[t] < p[lowIdx] {
				lowIdx = t
			} else if p[t] >= p[lowIdx]*(1+threshold) {
				pivots = append(pivots, Pivot{PivotLow, lowIdx, p[lowIdx], t})
				direction, highIdx = 1, t
			}
		}
	}
	return pivots, nil
}

// RegimeKind says whether a stretch of the price series is a bull or a bear
// market.
type RegimeKind int

const (
	// RegimeBull runs from a low to the next high.
	RegimeBull RegimeKind = iota
	// RegimeBear runs from a high to the next low.
	RegimeBear
)

// String returns the constant name of k, or RegimeKind(n) for a value that is
// not one of the constants.
func (k RegimeKind) String() string {
	switch k {
	case RegimeBull:
		return "RegimeBull"
	case RegimeBear:
		return "RegimeBear"
	}
	return fmt.Sprintf("RegimeKind(%d)", int(k))
}

// Regime is one bull or bear stretch between two consecutive pivots. Start and
// End are the rows of the two extremes, Length is End - Start, Change is the
// price change from the first to the second as a fraction, and ConfirmedAt is
// the row that confirmed the closing pivot, when the regime could first be
// known to have ended.
type Regime struct {
	Kind        RegimeKind
	Start, End  int
	Length      int
	Change      float64
	ConfirmedAt int
}

// RegimeOptions are the optional settings of Regimes.
type RegimeOptions struct {
	// MinPhase is the fewest rows a bull or bear stretch may last. A shorter
	// one is removed together with the two turning points around it, so the
	// stretches on both sides join. 0 keeps every stretch. This is the minimum
	// phase length of Pagan and Sossounov (2003).
	MinPhase int
	// MinCycle is the fewest rows from one low to the next low, or one high to
	// the next high. A turning point that makes a shorter cycle is removed with
	// the one before it. 0 applies no minimum. This is the minimum cycle length
	// of Pagan and Sossounov (2003).
	MinCycle int
}

// Regimes turns the pivots of ZigZag into bull stretches (low to high) and bear
// stretches (high to low). The stretch before the first pivot and after the
// last are not reported, since neither has both ends. Pivots must alternate, as
// ZigZag returns them.
//
// Removing turning points to enforce MinPhase and MinCycle uses the whole list,
// so unlike ZigZag it is not causal: a regime reported as ending at a pivot may
// be merged into a longer one once later pivots are known. Use ConfirmedAt only
// without minimums.
func Regimes(pivots []Pivot, opts ...RegimeOptions) ([]Regime, error) {
	if len(opts) > 1 {
		return nil, errors.New("Regimes accepts at most one RegimeOptions")
	}
	var o RegimeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.MinPhase < 0 || o.MinCycle < 0 {
		return nil, errors.New("MinPhase and MinCycle must not be negative")
	}
	for i := 1; i < len(pivots); i++ {
		if pivots[i].Kind == pivots[i-1].Kind {
			return nil, fmt.Errorf("pivots at list positions %d and %d are both %v; they must alternate", i-1, i, pivots[i].Kind)
		}
		if pivots[i].Index <= pivots[i-1].Index {
			return nil, fmt.Errorf("pivot at list position %d is not after the one before it", i)
		}
	}
	kept := append([]Pivot(nil), pivots...)
	for changed := true; changed; {
		changed = false
		if o.MinPhase > 0 {
			for i := 0; i+1 < len(kept); i++ {
				if kept[i+1].Index-kept[i].Index < o.MinPhase {
					kept = append(kept[:i], kept[i+2:]...)
					changed = true
					break
				}
			}
		}
		if !changed && o.MinCycle > 0 {
			for i := 0; i+2 < len(kept); i++ {
				if kept[i+2].Index-kept[i].Index < o.MinCycle {
					kept = append(kept[:i+1], kept[i+3:]...)
					changed = true
					break
				}
			}
		}
	}
	regimes := make([]Regime, 0, len(kept))
	for i := 0; i+1 < len(kept); i++ {
		a, b := kept[i], kept[i+1]
		kind := RegimeBull
		if a.Kind == PivotHigh {
			kind = RegimeBear
		}
		regimes = append(regimes, Regime{kind, a.Index, b.Index, b.Index - a.Index, b.Price/a.Price - 1, b.ConfirmedAt})
	}
	return regimes, nil
}
