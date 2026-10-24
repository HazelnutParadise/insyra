package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// MomentumOptions are the optional settings of MomentumPortfolios.
type MomentumOptions struct {
	// Skip is the number of months between the end of the formation period and
	// the month the portfolio is first held. 0 (the default) is Jegadeesh and
	// Titman's (1993) baseline; 1 is the convention of Carhart (1997) and of the
	// Fama-French momentum factor, which ranks on months t-12 to t-2.
	Skip int
	// Hold is the number of months a portfolio is held. 0 means 1, a portfolio
	// formed every month and held one month. A larger value overlaps that many
	// portfolios and averages them (Jegadeesh and Titman 1993).
	Hold int
}

// MomentumResult is the result of MomentumPortfolios.
type MomentumResult struct {
	// Groups has one row per reported month: the date, then group_1 to group_n,
	// the return of the equal-weighted group (group_1 the losers, group_n the
	// winners) averaged over the overlapping portfolios. A group with no
	// return in a month, or in any of the overlapping portfolios averaged, is NaN there.
	Groups *insyra.DataTable
	// LongShort is group_n minus group_1 for each row of Groups.
	LongShort *insyra.DataList
	// Skipped lists the months left out because a portfolio they average could
	// not be formed, with the reason. Months before the first full set of
	// overlapping portfolios are not reported and not listed.
	Skipped []SkippedDay
}

// MomentumPortfolios builds Jegadeesh and Titman's (1993) momentum portfolios:
// each month the stocks are ranked by their return over the past formation
// months, skipping the most recent Skip months, and cut into groups; the group
// is held for Hold months, and since a new portfolio is formed every month, Hold
// of them overlap and the month's return is their equal-weighted average. The
// groups' returns and the winners-minus-losers spread tell whether past
// winners keep winning.
//
// The table is in long form, one row per stock and month, like QuantileSort:
// dateCol, stockCol and returnCol are column selectors: an Excel-style index such as "A",
// insyra.Name("price") or an int position, and returnCol holds simple (not log) returns, nil or
// NaN where missing. Months are the dates in the order they first appear and
// must be consecutive; a stock must not appear twice in a month. A stock is ranked
// when every return of its formation period is present, the formation return
// being the compounded product of (1 + r) less 1, and equal signals share a
// group as in QuantileSort. Within a group the stocks are equally weighted, and
// one with no return in a holding month is left out of that month's average. A group that has no
// return at all in one of the overlapping portfolios makes that month's group return NaN. A stock whose compounded formation return overflows is not ranked.
// A month is reported once Hold portfolios have been formed; at least groups
// stocks must be ranked in a month for its portfolio to be formed.
//
// formation and groups have no safe default and are required: formation is the
// number of months ranked on, groups at least 2 (10 for deciles).
func MomentumPortfolios(table insyra.IDataTable, dateCol, stockCol, returnCol any, formation, groups int, opts ...MomentumOptions) (*MomentumResult, error) {
	if formation < 1 {
		return nil, errors.New("formation must be at least 1 month")
	}
	if groups < 2 {
		return nil, errors.New("groups must be at least 2")
	}
	if len(opts) > 1 {
		return nil, errors.New("MomentumPortfolios accepts at most one MomentumOptions")
	}
	var o MomentumOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Skip < 0 {
		return nil, errors.New("the skip must not be negative")
	}
	if o.Hold < 0 {
		return nil, errors.New("the hold must not be negative")
	}
	hold := o.Hold
	if hold == 0 {
		hold = 1
	}
	if insyra.IsNilTable(table) {
		return nil, errors.New("table is nil")
	}

	var dates, stocks, returns []any
	var returnName string
	var readErr error
	table.AtomicDo(func(dt *insyra.DataTable) {
		find := func(sel any, role string) ([]any, string) {
			if readErr != nil {
				return nil, ""
			}
			num, name, err := resolveColumn(dt, sel, role)
			if err != nil {
				readErr = err
				return nil, ""
			}
			return dt.GetColByNumber(num).Data(), name
		}
		dates, _ = find(dateCol, "date")
		stocks, _ = find(stockCol, "stock")
		returns, returnName = find(returnCol, "return")
	})
	if readErr != nil {
		return nil, readErr
	}
	rows := len(dates)
	if len(stocks) != rows {
		return nil, fmt.Errorf("the stock column has %d rows, the date column has %d", len(stocks), rows)
	}
	ret, err := finiteOrMissing(returns, returnName, rows)
	if err != nil {
		return nil, err
	}

	// Months in order of first appearance, and each stock's return by month.
	monthOf := map[string]int{}
	var monthValue []any
	stockOf := map[string]int{}
	type cell struct{ stock, month int }
	cells := make([]cell, rows)
	for i := 0; i < rows; i++ {
		if dates[i] == nil {
			return nil, fmt.Errorf("date is missing at row index %d", i)
		}
		if stocks[i] == nil {
			return nil, fmt.Errorf("stock is missing at row index %d", i)
		}
		dk, err := dateKey(dates[i])
		if err != nil {
			return nil, fmt.Errorf("row index %d: %w", i, err)
		}
		sk, err := dateKey(stocks[i])
		if err != nil {
			return nil, fmt.Errorf("stock at row index %d: %w", i, err)
		}
		m, ok := monthOf[dk]
		if !ok {
			m = len(monthValue)
			monthOf[dk] = m
			monthValue = append(monthValue, dates[i])
		}
		s, ok := stockOf[sk]
		if !ok {
			s = len(stockOf)
			stockOf[sk] = s
		}
		cells[i] = cell{s, m}
	}
	months, nStocks := len(monthValue), len(stockOf)
	first := formation + o.Skip - 1 // the first month a portfolio can be formed
	if months < first+hold+1 {
		return nil, fmt.Errorf("%d months are too few: formation %d, skip %d and hold %d need at least %d", months, formation, o.Skip, hold, first+hold+1)
	}
	r := make([][]float64, nStocks)
	for s := range r {
		r[s] = make([]float64, months)
		for m := range r[s] {
			r[s][m] = math.NaN()
		}
	}
	seen := make([][]bool, nStocks)
	for s := range seen {
		seen[s] = make([]bool, months)
	}
	for i, c := range cells {
		if seen[c.stock][c.month] {
			return nil, fmt.Errorf("a stock appears twice in one month at row index %d", i)
		}
		seen[c.stock][c.month] = true
		r[c.stock][c.month] = ret[i]
	}

	// Cohort f is formed at the end of month f and held in months f+1 .. f+hold.
	// held[f][g][k-1] is group g's mean return in month f+k, NaN if it has none.
	held := make([][][]float64, months)
	skipReason := make([]string, months)
	for f := first; f < months-1; f++ {
		sig := make([]float64, nStocks)
		var idx []int
		for s := 0; s < nStocks; s++ {
			growth, ok := 1.0, true
			for t := f - o.Skip - formation + 1; t <= f-o.Skip; t++ {
				if math.IsNaN(r[s][t]) {
					ok = false
					break
				}
				growth *= 1 + r[s][t]
			}
			if !ok {
				sig[s] = math.NaN()
				continue
			}
			sig[s] = growth - 1
			if math.IsNaN(sig[s]) || math.IsInf(sig[s], 0) {
				// The compounded product overflowed; the stock cannot be ranked.
				sig[s] = math.NaN()
				continue
			}
			idx = append(idx, s)
		}
		if len(idx) < groups {
			skipReason[f] = fmt.Sprintf("%d stocks with a formation return, fewer than %d groups", len(idx), groups)
			continue
		}
		group, reason := assignQuantileGroups(sig, idx, groups, QuantileTiesRank)
		if reason != "" {
			skipReason[f] = reason
			continue
		}
		held[f] = make([][]float64, groups)
		for g := range held[f] {
			held[f][g] = make([]float64, hold)
			for k := range held[f][g] {
				held[f][g][k] = math.NaN()
			}
		}
		for k := 1; k <= hold && f+k < months; k++ {
			sums, counts := make([]float64, groups), make([]int, groups)
			for p, s := range idx {
				if v := r[s][f+k]; !math.IsNaN(v) {
					sums[group[p]] += v
					counts[group[p]]++
				}
			}
			for g := 0; g < groups; g++ {
				if counts[g] > 0 {
					held[f][g][k-1] = sums[g] / float64(counts[g])
				}
			}
		}
	}

	res := &MomentumResult{}
	outDates := []any{}
	cols := make([][]any, groups)
	spread := []any{}
	for m := first + hold; m < months; m++ {
		bad := ""
		for k := 1; k <= hold && bad == ""; k++ {
			if held[m-k] == nil {
				bad = fmt.Sprintf("the portfolio formed at %v was skipped: %s", monthValue[m-k], skipReason[m-k])
			}
		}
		if bad != "" {
			res.Skipped = append(res.Skipped, SkippedDay{monthValue[m], bad})
			continue
		}
		outDates = append(outDates, monthValue[m])
		g1, gn := 0.0, 0.0
		for g := 0; g < groups; g++ {
			// A group is the average of all Hold portfolios; if one of them has no
			// return in this month the average is unknown, not the mean of the rest.
			sum, missing := 0.0, false
			for k := 1; k <= hold; k++ {
				v := held[m-k][g][k-1]
				if math.IsNaN(v) {
					missing = true
					break
				}
				sum += v
			}
			v := math.NaN()
			if !missing {
				v = sum / float64(hold)
			}
			cols[g] = append(cols[g], v)
			if g == 0 {
				g1 = v
			}
			if g == groups-1 {
				gn = v
			}
		}
		spread = append(spread, gn-g1)
	}
	tableCols := []*insyra.DataList{insyra.NewDataList(outDates...).SetName("date")}
	for g := 0; g < groups; g++ {
		tableCols = append(tableCols, insyra.NewDataList(cols[g]...).SetName(fmt.Sprintf("group_%d", g+1)))
	}
	res.Groups = insyra.NewDataTable(tableCols...)
	res.LongShort = insyra.NewDataList(spread...).SetName("long_short")
	return res, nil
}
