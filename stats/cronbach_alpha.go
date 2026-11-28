package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// CronbachAlphaOptions are the optional settings of CronbachAlpha.
type CronbachAlphaOptions struct {
	// ReverseScored are the items scored in the opposite direction, as column
	// selectors of the table passed to CronbachAlpha. Each is negated before
	// the statistics are computed, which leaves alpha, the SDs and the
	// item-rest correlations as reversing it on its own scale would; each
	// item's Mean is still the mean of its values in the table.
	ReverseScored []any
}

// CronbachAlphaResult is the internal consistency of a scale.
type CronbachAlphaResult struct {
	// Alpha is raw alpha, k/(k-1) * (1 - sum of item variances / variance of
	// the total score), from the item covariances (Cronbach, 1951).
	Alpha float64
	// StandardizedAlpha is k r / (1 + (k-1) r), r being the mean inter-item
	// correlation: the alpha of the items each scaled to variance 1.
	StandardizedAlpha float64
	// MeanInterItemCorrelation is the mean of the correlations between pairs
	// of items.
	MeanInterItemCorrelation float64
	// NumItems is the number of items.
	NumItems int
	// N is the number of rows used: those with every item a finite number.
	N int
	// RowsDropped is the number of rows left out because an item was NaN or
	// ±Inf (listwise deletion).
	RowsDropped int
	// Items holds one entry per item, in the order the items were given.
	Items []CronbachAlphaItem
}

// CronbachAlphaItem is one item's part of a scale.
type CronbachAlphaItem struct {
	// Name is the item's column name.
	Name string
	// Reversed is true for an item in ReverseScored.
	Reversed bool
	// Mean is the mean of the item's values in the table, over the rows used.
	Mean float64
	// SD is the item's standard deviation, with n - 1.
	SD float64
	// ItemRestCorrelation is the correlation of the item with the sum of the
	// other items: SPSS's Corrected Item-Total Correlation, psych's r.drop.
	ItemRestCorrelation float64
	// AlphaIfDeleted is the raw alpha of the other items. It is NaN for a
	// scale of two items, where it would be the alpha of one item.
	AlphaIfDeleted float64
}

// String returns the result as text: "Cronbach's alpha" with one line per
// statistic of the scale, then a table of the items.
func (r *CronbachAlphaResult) String() string {
	if r == nil {
		return "<nil>"
	}
	summary := struct {
		Alpha                    float64
		StandardizedAlpha        float64
		MeanInterItemCorrelation float64
		NumItems                 int
		N                        int
		RowsDropped              int
	}{r.Alpha, r.StandardizedAlpha, r.MeanInterItemCorrelation, r.NumItems, r.N, r.RowsDropped}
	return formatResult("Cronbach's alpha", summary) + "\n" + formatRows("Items", r.Items)
}

// Show prints String to standard output.
func (r *CronbachAlphaResult) Show() { fmt.Println(r.String()) }

// CronbachAlpha computes Cronbach's alpha for a scale whose items are the
// columns items of dt, one row per respondent. items are column selectors of
// dt (see the column selector rule); nil or empty means every column. A row
// with a NaN or ±Inf in any item is left out, as SPSS RELIABILITY does, and
// counted in RowsDropped; a blank numeric cell of a CSV is read as NaN, so a
// skipped question leaves its row out. A cell that is not a number is an
// error naming its column and row, both from 0.
//
// It is an error to give a nil table, fewer than two items, an item or a
// reverse-scored item twice, a reverse-scored item that is not among the
// items, a selector that names no column, an item with no variance over the
// rows used, fewer than two rows used, or more than one options value.
func CronbachAlpha(dt insyra.IDataTable, items []any, opts ...CronbachAlphaOptions) (*CronbachAlphaResult, error) {
	if len(opts) > 1 {
		return nil, fmt.Errorf("CronbachAlpha takes at most one CronbachAlphaOptions, got %d", len(opts))
	}
	var opt CronbachAlphaOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	if dt == nil {
		return nil, errors.New("table is nil")
	}
	if concrete, ok := dt.(*insyra.DataTable); ok && concrete == nil {
		return nil, errors.New("table is nil")
	}

	// The columns are resolved on a copy, so a selector that does not resolve
	// is an error and nothing is recorded on dt's Err().
	snapshot := dt.Clone()
	numRows, numCols := snapshot.Size()
	if len(items) == 0 {
		items = make([]any, numCols)
		for i := range items {
			items[i] = i
		}
	}
	if len(items) < 2 {
		return nil, fmt.Errorf("cronbach alpha needs at least two items, got %d", len(items))
	}

	positions := make([]int, len(items))
	seen := make(map[int]int, len(items))
	for i, sel := range items {
		pos, err := columnPosition(snapshot, sel)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		if first, dup := seen[pos]; dup {
			return nil, fmt.Errorf("item %d names the same column as item %d: %v", i, first, sel)
		}
		seen[pos] = i
		positions[i] = pos
	}
	reversed := make([]bool, len(items))
	for i, sel := range opt.ReverseScored {
		pos, err := columnPosition(snapshot, sel)
		if err != nil {
			return nil, fmt.Errorf("reverse-scored item %d: %w", i, err)
		}
		item, ok := seen[pos]
		if !ok {
			return nil, fmt.Errorf("reverse-scored item %d (%v) is not one of the items", i, sel)
		}
		if reversed[item] {
			return nil, fmt.Errorf("reverse-scored item %d (%v) is given twice", i, sel)
		}
		reversed[item] = true
	}

	k := len(items)
	names := make([]string, k)
	cols := make([][]any, k)
	for i, pos := range positions {
		col := snapshot.GetColByNumber(pos)
		names[i] = col.GetName()
		cols[i] = col.Data()
	}

	// Listwise deletion: a row is used only when every item is a finite
	// number there.
	data := make([][]float64, 0, numRows) // data[row][item]
	dropped := 0
	for row := 0; row < numRows; row++ {
		values := make([]float64, k)
		complete := true
		for i := range cols {
			var cell any
			if row < len(cols[i]) {
				cell = cols[i][row]
			}
			v, ok := insyra.ToFloat64Safe(cell)
			if !ok {
				return nil, fmt.Errorf("non-numeric value at row index %d column index %d (%s): %s", row, positions[i], names[i], utils.SprintBounded(cell))
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				complete = false
			}
			values[i] = v
		}
		if !complete {
			dropped++
			continue
		}
		data = append(data, values)
	}
	n := len(data)
	if n < 2 {
		return nil, fmt.Errorf("cronbach alpha needs at least two complete rows; %d remained after listwise deletion", n)
	}

	means := make([]float64, k)
	for _, values := range data {
		for i, v := range values {
			means[i] += v
		}
	}
	for i := range means {
		means[i] /= float64(n)
	}
	// The covariances of the items as scored: a reversed item is negated,
	// which flips the sign of its covariances and leaves its variance alone.
	sign := make([]float64, k)
	for i := range sign {
		sign[i] = 1
		if reversed[i] {
			sign[i] = -1
		}
	}
	cov := make([][]float64, k)
	for i := range cov {
		cov[i] = make([]float64, k)
	}
	for _, values := range data {
		for i := 0; i < k; i++ {
			di := sign[i] * (values[i] - means[i])
			for j := i; j < k; j++ {
				cov[i][j] += di * sign[j] * (values[j] - means[j])
			}
		}
	}
	for i := 0; i < k; i++ {
		for j := i; j < k; j++ {
			cov[i][j] /= float64(n - 1)
			cov[j][i] = cov[i][j]
		}
	}
	for i := 0; i < k; i++ {
		if cov[i][i] <= 0 {
			return nil, fmt.Errorf("item %d (%s) has no variance over the %d rows used; remove it from the items", i, names[i], n)
		}
	}

	result := &CronbachAlphaResult{
		NumItems:    k,
		N:           n,
		RowsDropped: dropped,
		Items:       make([]CronbachAlphaItem, k),
	}
	all := make([]bool, k)
	for i := range all {
		all[i] = true
	}
	result.Alpha = rawAlpha(cov, all)

	sumR := 0.0
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			sumR += cov[i][j] / math.Sqrt(cov[i][i]*cov[j][j])
		}
	}
	kf := float64(k)
	result.MeanInterItemCorrelation = (sumR - kf) / (kf * (kf - 1))
	result.StandardizedAlpha = kf / (kf - 1) * (1 - kf/sumR)

	for i := 0; i < k; i++ {
		// The rest score is the sum of the other items: its covariance with
		// item i and its variance come from the covariance matrix.
		covRest, varRest := 0.0, 0.0
		for j := 0; j < k; j++ {
			if j == i {
				continue
			}
			covRest += cov[i][j]
			for l := 0; l < k; l++ {
				if l != i {
					varRest += cov[j][l]
				}
			}
		}
		alphaIfDeleted := math.NaN()
		if k > 2 {
			rest := make([]bool, k)
			for j := range rest {
				rest[j] = j != i
			}
			alphaIfDeleted = rawAlpha(cov, rest)
		}
		result.Items[i] = CronbachAlphaItem{
			Name:                names[i],
			Reversed:            reversed[i],
			Mean:                means[i],
			SD:                  math.Sqrt(cov[i][i]),
			ItemRestCorrelation: covRest / math.Sqrt(cov[i][i]*varRest),
			AlphaIfDeleted:      alphaIfDeleted,
		}
	}
	return result, nil
}

// rawAlpha is k/(k-1) * (1 - trace / total) over the items in, from their
// covariance matrix.
func rawAlpha(cov [][]float64, in []bool) float64 {
	k, trace, total := 0, 0.0, 0.0
	for i := range cov {
		if !in[i] {
			continue
		}
		k++
		trace += cov[i][i]
		for j := range cov {
			if in[j] {
				total += cov[i][j]
			}
		}
	}
	kf := float64(k)
	return kf / (kf - 1) * (1 - trace/total)
}

// columnPosition returns the position in dt of the column sel selects: a
// string is an Excel-style index, Name a column name, an int a 0-based
// position counted from the end when negative.
func columnPosition(dt *insyra.DataTable, sel any) (int, error) {
	_, numCols := dt.Size()
	switch v := sel.(type) {
	case int:
		pos := v
		if pos < 0 {
			pos += numCols
		}
		if pos < 0 || pos >= numCols {
			return -1, fmt.Errorf("column position %d is out of range for %d columns", v, numCols)
		}
		return pos, nil
	case insyra.NameSelector:
		for i, name := range dt.ColNames() {
			if name == v.Value() {
				return i, nil
			}
		}
		return -1, fmt.Errorf("column %s does not exist, the table has %d column(s)", v.Value(), numCols)
	case string:
		pos, ok := insyra.ColIndexToNumber(v)
		if !ok || pos < 0 || pos >= numCols {
			return -1, fmt.Errorf("column %s does not exist, the table has %d column(s); use insyra.Name(%q) for a column name", v, numCols, v)
		}
		return pos, nil
	}
	return -1, fmt.Errorf("column selector of type %T; use an Excel-style index such as \"A\", insyra.Name(\"price\"), or an int position", sel)
}
