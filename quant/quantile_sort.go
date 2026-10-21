package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// QuantileTies chooses how QuantileSort places stocks whose signals are equal.
type QuantileTies int

const (
	// QuantileTiesRank gives tied signals their average rank and puts the
	// stock in group ceil(rank * n / m) of m stocks, so equal signals always
	// share a group.
	QuantileTiesRank QuantileTies = iota
	// QuantileTiesQcut cuts at the day's n-quantiles with linear interpolation
	// and puts a stock in the first group whose upper edge it does not exceed,
	// as pandas.qcut does. A day on which two edges coincide cannot be cut and
	// is skipped, as qcut would raise an error.
	QuantileTiesQcut
)

// String returns the constant name of t, or QuantileTies(n) for a value that
// is not one of the constants.
func (t QuantileTies) String() string {
	switch t {
	case QuantileTiesRank:
		return "QuantileTiesRank"
	case QuantileTiesQcut:
		return "QuantileTiesQcut"
	}
	return fmt.Sprintf("QuantileTies(%d)", int(t))
}

// QuantileSortOptions are the optional settings of QuantileSort.
type QuantileSortOptions struct {
	// MinStocks is the fewest stocks with a signal a day needs. A day with
	// fewer is skipped. At least n are always needed. The default is n.
	MinStocks int
	// Ties is how equal signals are placed. The default is QuantileTiesRank.
	Ties QuantileTies
	// WeightCol names a column of positive weights. Group means are weighted
	// by it. Empty means equal weights.
	WeightCol string
	// Demean subtracts the day's mean return over all the stocks that were
	// sorted (weighted when WeightCol is set) from every return, so the group
	// means are excess returns.
	Demean bool
}

// SkippedDay is a date QuantileSort left out and the reason.
type SkippedDay struct {
	Date   any
	Reason string
}

// QuantileSortResult is the result of QuantileSort.
type QuantileSortResult struct {
	// Groups has one row per date and group: columns date, group (1 is the
	// lowest signal), count, then the mean of each return column named after
	// it. A return column with no value in a group on a day is NaN there.
	Groups *insyra.DataTable
	// Spread has one row per date: the date, then for each return column the
	// mean of the highest group minus the mean of the lowest, named
	// "spread_<column>".
	Spread *insyra.DataTable
	// Skipped lists the dates left out, in the order they appear.
	Skipped []SkippedDay
}

// QuantileSort sorts the stocks of every date into n groups by a signal and
// reports each group's mean return and the spread between the top and the
// bottom group, the standard check of whether a signal ranks future returns.
// The table is in long form, one row per stock and date.
//
// dateCol, signalCol, returnCols and WeightCol are column names, or Excel-style
// letters when no column has that name. Dates are grouped by value and appear
// in the output in the order they first appear in the table. A stock with a
// missing or non-finite signal is left out of its day; a missing return is
// left out of that column's means. Dates are never mixed: every day is sorted
// on its own.
func QuantileSort(table insyra.IDataTable, dateCol, signalCol string, returnCols []string, n int, opts ...QuantileSortOptions) (*QuantileSortResult, error) {
	if n < 2 {
		return nil, errors.New("n must be at least 2")
	}
	if len(returnCols) == 0 {
		return nil, errors.New("at least one return column is needed")
	}
	if len(opts) > 1 {
		return nil, errors.New("QuantileSort accepts at most one QuantileSortOptions")
	}
	var o QuantileSortOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Ties != QuantileTiesRank && o.Ties != QuantileTiesQcut {
		return nil, fmt.Errorf("unsupported tie method %v", o.Ties)
	}
	if o.MinStocks < n {
		o.MinStocks = n
	}
	if insyra.IsNilTable(table) {
		return nil, errors.New("table is nil")
	}

	var dates, signals []any
	returns := make([][]any, len(returnCols))
	var weights []any
	var readErr error
	table.AtomicDo(func(dt *insyra.DataTable) {
		find := func(sel, role string) []any {
			if readErr != nil {
				return nil
			}
			data, err := columnData(dt, sel, role)
			readErr = err
			return data
		}
		dates = find(dateCol, "date")
		signals = find(signalCol, "signal")
		for i, c := range returnCols {
			returns[i] = find(c, "return")
		}
		if o.WeightCol != "" {
			weights = find(o.WeightCol, "weight")
		}
	})
	if readErr != nil {
		return nil, readErr
	}
	rows := len(dates)

	sig, err := finiteOrMissing(signals, "signal", rows)
	if err != nil {
		return nil, err
	}
	rets := make([][]float64, len(returnCols))
	for i := range returnCols {
		if rets[i], err = finiteOrMissing(returns[i], returnCols[i], rows); err != nil {
			return nil, err
		}
	}
	var wt []float64
	if weights != nil {
		if wt, err = finiteOrMissing(weights, o.WeightCol, rows); err != nil {
			return nil, err
		}
		for i, w := range wt {
			if !math.IsNaN(w) && w <= 0 {
				return nil, fmt.Errorf("%s holds %v at row index %d; weights must be positive", o.WeightCol, w, i)
			}
		}
	}

	// Group row positions by date, keeping the order dates first appear.
	var order []string
	byDate := map[string][]int{}
	dateValue := map[string]any{}
	for i, d := range dates {
		if d == nil {
			return nil, fmt.Errorf("date is missing at row index %d", i)
		}
		key, err := dateKey(d)
		if err != nil {
			return nil, fmt.Errorf("row index %d: %w", i, err)
		}
		if _, seen := byDate[key]; !seen {
			order = append(order, key)
			dateValue[key] = d
		}
		byDate[key] = append(byDate[key], i)
	}

	res := &QuantileSortResult{}
	gDate, gGroup, gCount := []any{}, []any{}, []any{}
	gMean := make([][]any, len(returnCols))
	sDate := []any{}
	sSpread := make([][]any, len(returnCols))

	for _, key := range order {
		var idx []int
		for _, i := range byDate[key] {
			if !math.IsNaN(sig[i]) {
				idx = append(idx, i)
			}
		}
		if len(idx) < o.MinStocks {
			res.Skipped = append(res.Skipped, SkippedDay{dateValue[key], fmt.Sprintf("%d stocks with a signal, fewer than %d", len(idx), o.MinStocks)})
			continue
		}
		group, reason := assignQuantileGroups(sig, idx, n, o.Ties)
		if reason != "" {
			res.Skipped = append(res.Skipped, SkippedDay{dateValue[key], reason})
			continue
		}

		// The day's mean return per column, for Demean.
		dayMean := make([]float64, len(returnCols))
		if o.Demean {
			for c := range returnCols {
				sum, total := 0.0, 0.0
				for _, i := range idx {
					if math.IsNaN(rets[c][i]) || (wt != nil && math.IsNaN(wt[i])) {
						continue
					}
					w := weightAt(wt, i)
					sum += w * rets[c][i]
					total += w
				}
				dayMean[c] = math.NaN()
				if total > 0 {
					dayMean[c] = sum / total
				}
			}
		}

		counts := make([]int, n)
		sums := make([][]float64, len(returnCols))
		totals := make([][]float64, len(returnCols))
		for c := range sums {
			sums[c], totals[c] = make([]float64, n), make([]float64, n)
		}
		for k, i := range idx {
			g := group[k]
			counts[g]++
			for c := range returnCols {
				r := rets[c][i]
				if math.IsNaN(r) || (wt != nil && math.IsNaN(wt[i])) {
					continue
				}
				if o.Demean {
					r -= dayMean[c]
				}
				w := weightAt(wt, i)
				sums[c][g] += w * r
				totals[c][g] += w
			}
		}
		sDate = append(sDate, dateValue[key])
		for g := 0; g < n; g++ {
			gDate = append(gDate, dateValue[key])
			gGroup = append(gGroup, g+1)
			gCount = append(gCount, counts[g])
			for c := range returnCols {
				gMean[c] = append(gMean[c], groupMean(sums[c][g], totals[c][g]))
			}
		}
		for c := range returnCols {
			sSpread[c] = append(sSpread[c], groupMean(sums[c][n-1], totals[c][n-1])-groupMean(sums[c][0], totals[c][0]))
		}
	}

	groupCols := []*insyra.DataList{
		insyra.NewDataList(gDate...).SetName("date"),
		insyra.NewDataList(gGroup...).SetName("group"),
		insyra.NewDataList(gCount...).SetName("count"),
	}
	spreadCols := []*insyra.DataList{insyra.NewDataList(sDate...).SetName("date")}
	for c, name := range returnCols {
		groupCols = append(groupCols, insyra.NewDataList(gMean[c]...).SetName(name))
		spreadCols = append(spreadCols, insyra.NewDataList(sSpread[c]...).SetName("spread_"+name))
	}
	res.Groups = insyra.NewDataTable(groupCols...)
	res.Spread = insyra.NewDataTable(spreadCols...)
	return res, nil
}

// columnNumber finds a column by name, or by Excel-style letters when no
// column has that name. It returns -1 when neither matches.
func columnNumber(dt *insyra.DataTable, sel string) int {
	names := dt.ColNames()
	for i, name := range names {
		if name == sel {
			return i
		}
	}
	if i, ok := insyra.ColIndexToNumber(sel); ok && i < len(names) {
		return i
	}
	return -1
}

// columnData returns the cells of the column columnNumber finds, or an error
// naming the column's role when it is not found. Call it inside AtomicDo.
func columnData(dt *insyra.DataTable, sel, role string) ([]any, error) {
	i := columnNumber(dt, sel)
	if i < 0 {
		return nil, fmt.Errorf("%s column %q not found", role, sel)
	}
	return dt.GetColByNumber(i).Data(), nil
}

func weightAt(wt []float64, i int) float64 {
	if wt == nil {
		return 1
	}
	return wt[i]
}

func groupMean(sum, total float64) float64 {
	if total == 0 {
		return math.NaN()
	}
	return sum / total
}

// finiteOrMissing reads a column as float64, with nil and NaN as NaN. A value
// that is not a number or is infinite is an error.
func finiteOrMissing(raw []any, label string, rows int) ([]float64, error) {
	if len(raw) != rows {
		return nil, fmt.Errorf("column %s has %d rows, the date column has %d", label, len(raw), rows)
	}
	out := make([]float64, rows)
	for i, v := range raw {
		if v == nil {
			out[i] = math.NaN()
			continue
		}
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, fmt.Errorf("%s contains a non-numeric value at row index %d: %v", label, i, v)
		}
		if math.IsInf(f, 0) {
			return nil, fmt.Errorf("%s contains a non-finite value at row index %d: %v", label, i, f)
		}
		out[i] = f
	}
	return out, nil
}

// assignQuantileGroups returns, for each position in idx, a group from 0 to
// n-1, or a reason the day cannot be cut.
func assignQuantileGroups(sig []float64, idx []int, n int, ties QuantileTies) ([]int, string) {
	m := len(idx)
	order := make([]int, m) // positions into idx, by ascending signal
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return sig[idx[order[a]]] < sig[idx[order[b]]] })

	group := make([]int, m)
	if ties == QuantileTiesRank {
		for start := 0; start < m; {
			end := start
			for end < m && sig[idx[order[end]]] == sig[idx[order[start]]] {
				end++
			}
			avgRank := float64(start+end+1) / 2 // mean of ranks start+1 .. end
			g := int(math.Ceil(avgRank*float64(n)/float64(m))) - 1
			g = max(0, min(n-1, g))
			for k := start; k < end; k++ {
				group[order[k]] = g
			}
			start = end
		}
		return group, ""
	}

	edges := make([]float64, n+1)
	for k := 0; k <= n; k++ {
		pos := float64(m-1) * float64(k) / float64(n)
		lo := int(math.Floor(pos))
		hi := min(lo+1, m-1)
		frac := pos - float64(lo)
		edges[k] = sig[idx[order[lo]]]*(1-frac) + sig[idx[order[hi]]]*frac
	}
	for k := 0; k < n; k++ {
		if edges[k] == edges[k+1] {
			return nil, "the signal has duplicate quantile edges and cannot be cut into groups"
		}
	}
	for pos, i := range idx {
		x := sig[i]
		g := sort.SearchFloat64s(edges[1:], x) // first edge >= x, so the cut is right-closed
		group[pos] = min(g, n-1)
	}
	return group, ""
}
