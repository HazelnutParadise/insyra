package ta

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// PivotPointsResult holds the floor-trader pivot levels in force on each row.
type PivotPointsResult struct {
	P  *insyra.DataList
	R1 *insyra.DataList
	R2 *insyra.DataList
	R3 *insyra.DataList
	S1 *insyra.DataList
	S2 *insyra.DataList
	S3 *insyra.DataList
}

// PivotPoints returns the floor-trader pivot points for each row from the
// previous row's high H, low L and close C: the pivot P = (H + L + C) / 3, the
// resistances R1 = 2P - L, R2 = P + (H - L), R3 = H + 2(P - L) and the supports
// S1 = 2P - H, S2 = P - (H - L), S3 = L - 2(H - P). They are the levels to
// watch on this row, known at yesterday's close, so row 0 is NaN. (These are
// price levels, not the turning points ZigZag finds, which are Swings.)
func PivotPoints(high, low, close insyra.IDataList) (*PivotPointsResult, error) {
	h, l, c, err := barSeries(high, low, close)
	if err != nil {
		return nil, err
	}
	lines := make([][]float64, 7)
	for k := range lines {
		lines[k] = nanSeries(len(c))
	}
	for i := 1; i < len(c); i++ {
		H, L, C := h[i-1], l[i-1], c[i-1]
		p := (H + L + C) / 3
		lines[0][i], lines[1][i], lines[2][i], lines[3][i] = p, 2*p-L, p+(H-L), H+2*(p-L)
		lines[4][i], lines[5][i], lines[6][i] = 2*p-H, p-(H-L), L-2*(H-p)
	}
	return &PivotPointsResult{
		listOfFloats("P", lines[0]), listOfFloats("R1", lines[1]), listOfFloats("R2", lines[2]), listOfFloats("R3", lines[3]),
		listOfFloats("S1", lines[4]), listOfFloats("S2", lines[5]), listOfFloats("S3", lines[6]),
	}, nil
}

// HeikinAshi returns the Heikin-Ashi bars of a series of daily bars as a table
// with the columns Open, High, Low and Close: the close is (open + high + low +
// close) / 4, the open is the midpoint of the previous Heikin-Ashi bar (the first
// is (open + close) / 2), and the high and low extend to cover them. The bars
// smooth the price, and each depends on all the bars before it. All four series
// must be numeric, finite, positive prices of the same length, and no high may be
// below its low.
func HeikinAshi(open, high, low, close insyra.IDataList) (*insyra.DataTable, error) {
	series := make([][]float64, 4)
	for k, in := range []struct {
		dl    insyra.IDataList
		label string
	}{{open, "open"}, {high, "high"}, {low, "low"}, {close, "close"}} {
		p, err := positiveCloseSeries(in.dl, in.label)
		if err != nil {
			return nil, err
		}
		series[k] = p
	}
	o, h, l, c := series[0], series[1], series[2], series[3]
	if len(o) != len(h) || len(o) != len(l) || len(o) != len(c) {
		return nil, fmt.Errorf("open, high, low and close must have the same length, got %d, %d, %d and %d", len(o), len(h), len(l), len(c))
	}
	for i := range h {
		if h[i] < l[i] {
			return nil, fmt.Errorf("high %v is below low %v at row index %d", h[i], l[i], i)
		}
	}
	ho, hh, hl, hc := make([]any, len(c)), make([]any, len(c)), make([]any, len(c)), make([]any, len(c))
	var prevOpen, prevClose float64
	for i := range c {
		closeHA := (o[i] + h[i] + l[i] + c[i]) / 4
		openHA := (o[i] + c[i]) / 2
		if i > 0 {
			openHA = (prevOpen + prevClose) / 2
		}
		ho[i], hc[i] = openHA, closeHA
		hh[i] = math.Max(h[i], math.Max(openHA, closeHA))
		hl[i] = math.Min(l[i], math.Min(openHA, closeHA))
		prevOpen, prevClose = openHA, closeHA
	}
	return insyra.NewDataTable(
		insyra.NewDataList(ho...).SetName("Open"), insyra.NewDataList(hh...).SetName("High"),
		insyra.NewDataList(hl...).SetName("Low"), insyra.NewDataList(hc...).SetName("Close"),
	), nil
}

// TDSequentialResult holds Tom DeMark's TD Sequential counts. Each is 0 when
// nothing is being counted.
type TDSequentialResult struct {
	BuySetup      *insyra.DataList // 1 to 9: consecutive closes below the close four rows earlier
	SellSetup     *insyra.DataList // 1 to 9: consecutive closes above the close four rows earlier
	BuyCountdown  *insyra.DataList // 1 to 13, after a buy setup
	SellCountdown *insyra.DataList // 1 to 13, after a sell setup
}

// TDSequential returns the setup and countdown counts of TD Sequential on closes
// only.
//
// A buy setup counts consecutive rows whose close is below the close four rows
// earlier, 1 to 9, and a sell setup the mirror image; a row that breaks the run
// resets the count to 0, and a run that reaches 9 starts again from 1 if it goes
// on. A completed setup (9) starts a countdown on the next row: a buy countdown
// counts, not necessarily in a row, the rows whose close is at or below the close
// two rows earlier, up to 13, and a sell countdown the rows whose close is at or
// above the close two rows earlier.
//
// This is not DeMark's original rule set. It differs in two ways: the countdown
// compares the close with the close two rows earlier, where the original compares
// with the low (buy) or high (sell) of two rows earlier, and a setup that goes on
// past 9 starts again from 1. A countdown is cancelled when the opposite setup
// completes, and a second setup of the same side while it runs does not restart
// it. Platforms differ in the finer rules (perfection, qualifiers, recycling);
// this version has none of them, so do not expect it to match a charting
// platform's TD Sequential.
func TDSequential(close insyra.IDataList) (*TDSequentialResult, error) {
	c, err := numericSeries(close, "close")
	if err != nil {
		return nil, err
	}
	n := len(c)
	bs, ss, bc, sc := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	var buy, sell, buyCD, sellCD int
	buyActive, sellActive := false, false
	buyFrom, sellFrom := 0, 0 // the row whose setup started the running countdown
	for i := range c {
		if i >= 4 {
			if c[i] < c[i-4] {
				buy++
			} else {
				buy = 0
			}
			if c[i] > c[i-4] {
				sell++
			} else {
				sell = 0
			}
			if buy > 9 {
				buy = 1
			}
			if sell > 9 {
				sell = 1
			}
		}
		bs[i], ss[i] = float64(buy), float64(sell)
		// A completed setup starts a countdown unless one of the same side is
		// already running; the opposite side's countdown is cancelled.
		if buy == 9 {
			if !buyActive && buyCD != 13 {
				buyActive, buyCD, buyFrom = true, 0, i
			}
			sellActive, sellCD = false, 0
		}
		if sell == 9 {
			if !sellActive && sellCD != 13 {
				sellActive, sellCD, sellFrom = true, 0, i
			}
			buyActive, buyCD = false, 0
		}
		// a countdown starts on the row after its setup completed
		if buyActive && i > buyFrom && c[i] <= c[i-2] {
			if buyCD < 13 {
				buyCD++
			}
		}
		if sellActive && i > sellFrom && c[i] >= c[i-2] {
			if sellCD < 13 {
				sellCD++
			}
		}
		bc[i], sc[i] = float64(buyCD), float64(sellCD)
		if buyCD == 13 {
			buyActive = false
		}
		if sellCD == 13 {
			sellActive = false
		}
	}
	return &TDSequentialResult{listOfFloats("BuySetup", bs), listOfFloats("SellSetup", ss), listOfFloats("BuyCountdown", bc), listOfFloats("SellCountdown", sc)}, nil
}

// PFKind says whether a point-and-figure column is a run of X (rising) or O
// (falling).
type PFKind int

const (
	// PFRising is a column of X, a rise.
	PFRising PFKind = iota
	// PFFalling is a column of O, a fall.
	PFFalling
)

// String returns the constant name of k, or PFKind(n) for a value that is not one
// of the constants.
func (k PFKind) String() string {
	switch k {
	case PFRising:
		return "PFRising"
	case PFFalling:
		return "PFFalling"
	}
	return fmt.Sprintf("PFKind(%d)", int(k))
}

// PFColumn is one column of a point-and-figure chart. StartRow and EndRow are the
// first row that put a box in the column and the last. Top and Bottom are the
// prices of the highest and lowest box (the box's lower edge, a multiple of the
// box size), and Boxes is how many boxes the column holds.
type PFColumn struct {
	Kind     PFKind
	StartRow int
	EndRow   int
	Top      float64
	Bottom   float64
	Boxes    int
}

// PointAndFigure returns the columns of a point-and-figure chart built from
// closes. A price lies in the box floor(price / boxSize). A rising column grows
// when the close reaches a higher box and turns into a falling column when the
// close falls reversal boxes below its top; a falling column is the mirror image.
// The first column starts when the close first leaves the box of the first row.
// A new column starts one box beyond the old column's end, so the two do not
// overlap. A common setting is reversal 3.
func PointAndFigure(close insyra.IDataList, boxSize float64, reversal int) ([]PFColumn, error) {
	if math.IsNaN(boxSize) || math.IsInf(boxSize, 0) || boxSize <= 0 {
		return nil, errors.New("boxSize must be a finite price above 0")
	}
	if reversal < 1 {
		return nil, fmt.Errorf("reversal must be at least 1, got %d", reversal)
	}
	c, err := positiveCloseSeries(close, "close")
	if err != nil {
		return nil, err
	}
	box := func(p float64) int { return int(math.Floor(p / boxSize)) }
	if len(c) == 0 {
		return nil, nil
	}
	b0 := box(c[0])
	var kind PFKind
	var low, high, start int
	started := false
	var out []PFColumn
	flush := func(end int) {
		out = append(out, PFColumn{Kind: kind, StartRow: start, EndRow: end,
			Top: float64(high) * boxSize, Bottom: float64(low) * boxSize, Boxes: high - low + 1})
	}
	last := 0
	for i := 1; i < len(c); i++ {
		b := box(c[i])
		if !started {
			switch {
			case b > b0:
				kind, low, high, start, started = PFRising, b0, b, i, true
			case b < b0:
				kind, low, high, start, started = PFFalling, b, b0, i, true
			default:
				continue
			}
			last = i
			continue
		}
		switch kind {
		case PFRising:
			if b > high {
				high, last = b, i
			} else if b <= high-reversal {
				flush(last)
				kind, high, low, start, last = PFFalling, high-1, b, i, i
			}
		case PFFalling:
			if b < low {
				low, last = b, i
			} else if b >= low+reversal {
				flush(last)
				kind, low, high, start, last = PFRising, low+1, b, i, i
			}
		}
	}
	if started {
		flush(last)
	}
	return out, nil
}
