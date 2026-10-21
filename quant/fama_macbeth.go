package quant

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

// FamaMacBethOptions are the optional settings of FamaMacBeth.
type FamaMacBethOptions struct {
	// NeweyWestLag is the lag of the Newey-West t-test on each coefficient's
	// daily series. 0 chooses stats.NeweyWestLag(T) for T estimated days and
	// a positive value is used as given. A negative value is an error.
	NeweyWestLag int
	// NoNeweyWest switches the Newey-West correction off: the t-test then
	// uses lag 0, whose variance divides by T. That equals linearmodels with
	// cov_type="unadjusted" and debiased=False; it is not the one-sample
	// t-test of stats.SingleSampleTTest, which divides by T-1. Setting it
	// together with a non-zero NeweyWestLag is an error.
	NoNeweyWest bool
	// MinObservations is the fewest usable stocks a day needs. A day with
	// fewer is skipped. At least k+2 are always needed for k characteristics,
	// which is also the default.
	MinObservations int
	// Standardize replaces each characteristic by its z-score over the day's
	// usable stocks (sample standard deviation) before the day is regressed.
	Standardize bool
}

// FamaMacBethResult is the result of FamaMacBeth. Names and the slices beside
// it follow the order of charCols.
type FamaMacBethResult struct {
	// Intercept is the mean of the daily intercepts, with its Newey-West
	// standard error, t statistic and p value.
	Intercept, InterceptStdErr, InterceptTValue, InterceptPValue float64

	// Names are the characteristic columns as they were given.
	Names []string
	// Coefs are the means of the daily coefficients.
	Coefs []float64
	// StdErrs are the Newey-West standard errors of Coefs.
	StdErrs []float64
	// TValues are Coefs divided by StdErrs.
	TValues []float64
	// PValues are two-sided, from the t distribution with T-1 degrees of freedom.
	PValues []float64

	// Daily has one row per estimated date: date, intercept, one column per
	// characteristic named as given, r2 and count (the stocks regressed).
	Daily *insyra.DataTable
	// MeanR2 is the mean of the daily R² values that are finite.
	MeanR2 float64
	// Lag is the Newey-West lag used, after any automatic choice.
	Lag int
	// Days is the number of estimated dates, T.
	Days int
	// Skipped lists the dates left out, in the order they appear.
	Skipped []SkippedDay
}

// FamaMacBeth runs a Fama-MacBeth regression: on every date, the return is
// regressed on an intercept and the characteristics across stocks, and each
// coefficient's daily series is then tested for a zero mean with
// stats.NeweyWestMeanTest. It answers whether a characteristic explains the
// cross-section of returns once the other characteristics are controlled for.
// The table is in long form, one row per stock and date, as for QuantileSort.
//
// dateCol, returnCol and charCols are column names, or Excel-style letters
// when no column has that name. Dates are grouped by value and appear in the
// order they first appear. A date is a string, time.Time, bool, integer or
// float (not NaN); any other type, nil included, is an error. A stock is left out of its day when its return or
// any characteristic is missing (nil or NaN). A day with fewer usable stocks
// than MinObservations, or whose characteristics are collinear (including one
// that is constant that day), is listed in Skipped rather than failing the
// call. Fewer than two estimated days is an error, and so is a characteristic named
// date, intercept, r2 or count (the fixed columns of Daily) or the date column, and so is naming one column as both date and return.
//
// The standard errors equal Python's linearmodels.FamaMacBeth with
// cov_type="kernel", kernel="bartlett", the same bandwidth and debiased=False:
// Bartlett weights 1 - j/(lag+1) and no small-sample factor. linearmodels
// reads the p value from the normal distribution in that setting; FamaMacBeth
// uses t with T-1 degrees of freedom, as stats.NeweyWestMeanTest does.
// NoNeweyWest runs the same test with lag 0, which divides the variance by T.
func FamaMacBeth(table insyra.IDataTable, dateCol, returnCol string, charCols []string, opts ...FamaMacBethOptions) (*FamaMacBethResult, error) {
	k := len(charCols)
	if k == 0 {
		return nil, errors.New("at least one characteristic column is needed")
	}
	if len(opts) > 1 {
		return nil, errors.New("FamaMacBeth accepts at most one FamaMacBethOptions")
	}
	var o FamaMacBethOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.NeweyWestLag < 0 {
		return nil, fmt.Errorf("NeweyWestLag is %d; it must be 0 (automatic) or positive, and NoNeweyWest switches the correction off", o.NeweyWestLag)
	}
	if o.NoNeweyWest && o.NeweyWestLag != 0 {
		return nil, fmt.Errorf("NoNeweyWest cannot be combined with NeweyWestLag %d", o.NeweyWestLag)
	}
	if o.MinObservations < k+2 {
		o.MinObservations = k + 2
	}
	if insyra.IsNilTable(table) {
		return nil, errors.New("table is nil")
	}

	var dates, returns []any
	chars := make([][]any, k)
	var readErr error
	table.AtomicDo(func(dt *insyra.DataTable) {
		retNum := columnNumber(dt, returnCol)
		dateNum := columnNumber(dt, dateCol)
		if dateNum >= 0 && dateNum == retNum {
			readErr = fmt.Errorf("the date column %q and the return column %q are the same column", dateCol, returnCol)
			return
		}
		seen := map[int]string{}
		for i, c := range charCols {
			num := columnNumber(dt, c)
			if num < 0 {
				readErr = fmt.Errorf("characteristic column %q not found", c)
				return
			}
			switch c {
			case "date", "intercept", "r2", "count":
				readErr = fmt.Errorf("characteristic column %q is reserved: Daily has columns of that name for its own values", c)
				return
			}
			if num == dateNum {
				readErr = fmt.Errorf("characteristic column %q is the date column", c)
				return
			}
			if num == retNum {
				readErr = fmt.Errorf("characteristic column %q is the return column", c)
				return
			}
			if prev, dup := seen[num]; dup {
				readErr = fmt.Errorf("characteristic columns %q and %q are the same column", prev, c)
				return
			}
			seen[num] = c
			chars[i] = dt.GetColByNumber(num).Data()
		}
		if dates, readErr = columnData(dt, dateCol, "date"); readErr != nil {
			return
		}
		returns, readErr = columnData(dt, returnCol, "return")
	})
	if readErr != nil {
		return nil, readErr
	}
	rows := len(dates)

	ret, err := finiteOrMissing(returns, returnCol, rows)
	if err != nil {
		return nil, err
	}
	x := make([][]float64, k)
	for j, c := range charCols {
		if x[j], err = finiteOrMissing(chars[j], c, rows); err != nil {
			return nil, err
		}
	}

	var order []string
	byDate := map[string][]int{}
	dateValue := map[string]any{}
	for i, d := range dates {
		if d == nil {
			return nil, fmt.Errorf("date is missing at row index %d", i)
		}
		key, err := famaMacBethDateKey(d)
		if err != nil {
			return nil, fmt.Errorf("row index %d: %w", i, err)
		}
		if _, seen := byDate[key]; !seen {
			order = append(order, key)
			dateValue[key] = d
		}
		byDate[key] = append(byDate[key], i)
	}

	res := &FamaMacBethResult{Names: append([]string(nil), charCols...)}
	dDate, dR2, dCount := []any{}, []any{}, []any{}
	coefs := make([][]float64, k+1) // intercept first
	for _, key := range order {
		var idx []int
	rowLoop:
		for _, i := range byDate[key] {
			if math.IsNaN(ret[i]) {
				continue
			}
			for j := range x {
				if math.IsNaN(x[j][i]) {
					continue rowLoop
				}
			}
			idx = append(idx, i)
		}
		if len(idx) < o.MinObservations {
			res.Skipped = append(res.Skipped, SkippedDay{dateValue[key], fmt.Sprintf("%d usable stocks, fewer than %d", len(idx), o.MinObservations)})
			continue
		}
		beta, r2, ok := crossSection(ret, x, idx, o.Standardize)
		if !ok {
			res.Skipped = append(res.Skipped, SkippedDay{dateValue[key], "the characteristics are collinear on this date"})
			continue
		}
		dDate = append(dDate, dateValue[key])
		dR2 = append(dR2, r2)
		dCount = append(dCount, len(idx))
		for j, b := range beta {
			coefs[j] = append(coefs[j], b)
		}
	}

	t := len(dDate)
	if t < 2 {
		return nil, fmt.Errorf("%d dates could be estimated; at least 2 are needed", t)
	}
	lag := o.NeweyWestLag
	switch {
	case o.NoNeweyWest:
		lag = 0
	case lag == 0:
		lag = stats.AutoLag
	}

	tests := make([]*stats.NeweyWestResult, k+1)
	for j := range coefs {
		if tests[j], err = stats.NeweyWestMeanTest(newFloat64DataList(coefs[j]), lag); err != nil {
			return nil, err
		}
	}
	res.Intercept, res.InterceptStdErr = tests[0].Mean, tests[0].StdErr
	res.InterceptTValue, res.InterceptPValue = tests[0].Statistic, tests[0].PValue
	for _, tr := range tests[1:] {
		res.Coefs = append(res.Coefs, tr.Mean)
		res.StdErrs = append(res.StdErrs, tr.StdErr)
		res.TValues = append(res.TValues, tr.Statistic)
		res.PValues = append(res.PValues, tr.PValue)
	}
	res.Lag, res.Days = tests[0].Lag, t

	sum, finite := 0.0, 0
	for _, v := range dR2 {
		if r := v.(float64); !math.IsNaN(r) {
			sum += r
			finite++
		}
	}
	res.MeanR2 = math.NaN()
	if finite > 0 {
		res.MeanR2 = sum / float64(finite)
	}

	cols := []*insyra.DataList{
		insyra.NewDataList(dDate...).SetName("date"),
		newFloat64DataList(coefs[0]).SetName("intercept"),
	}
	for j, name := range charCols {
		cols = append(cols, newFloat64DataList(coefs[j+1]).SetName(name))
	}
	cols = append(cols, insyra.NewDataList(dR2...).SetName("r2"), insyra.NewDataList(dCount...).SetName("count"))
	res.Daily = insyra.NewDataTable(cols...)
	return res, nil
}

// crossSection regresses ret on an intercept and x over the rows idx and
// returns the intercept and slopes and the R², or false when the design
// matrix has a rank below its column count.
func crossSection(ret []float64, x [][]float64, idx []int, standardize bool) ([]float64, float64, bool) {
	n, p := len(idx), len(x)+1
	design := mat.NewDense(n, p, nil)
	y := mat.NewVecDense(n, nil)
	for r, i := range idx {
		design.Set(r, 0, 1)
		y.SetVec(r, ret[i])
	}
	for j, col := range x {
		mean := 0.0
		for _, i := range idx {
			mean += col[i]
		}
		mean /= float64(n)
		scale := 1.0
		if standardize {
			ss := 0.0
			for _, i := range idx {
				d := col[i] - mean
				ss += d * d
			}
			sd := math.Sqrt(ss / float64(n-1))
			if sd == 0 {
				return nil, 0, false
			}
			scale = sd
		}
		for r, i := range idx {
			v := col[i]
			if standardize {
				v = (v - mean) / scale
			}
			design.Set(r, j+1, v)
		}
	}

	var svd mat.SVD
	if !svd.Factorize(design, mat.SVDThin) {
		return nil, 0, false
	}
	values := svd.Values(nil)
	// NumPy's matrix_rank tolerance, the one linearmodels' lstsq rank uses.
	tol := values[0] * float64(max(n, p)) * 0x1p-52
	for _, s := range values {
		if s <= tol {
			return nil, 0, false
		}
	}
	var beta mat.VecDense
	svd.SolveVecTo(&beta, y, p)

	meanY := mat.Sum(y) / float64(n)
	sse, tss := 0.0, 0.0
	for r := 0; r < n; r++ {
		fit := 0.0
		for c := 0; c < p; c++ {
			fit += design.At(r, c) * beta.AtVec(c)
		}
		e := y.AtVec(r) - fit
		d := y.AtVec(r) - meanY
		sse += e * e
		tss += d * d
	}
	r2 := math.NaN()
	if tss > 0 {
		r2 = 1 - sse/tss
	}
	out := make([]float64, p)
	for c := range out {
		out[c] = beta.AtVec(c)
	}
	return out, r2, true
}

// famaMacBethDateKey groups dates by value. A date is a string (kept exactly,
// surrounding whitespace included), a time.Time (its instant, location
// ignored), a bool, an integer or a float (not NaN), keyed by its Go type and
// exact value, so int 1 and float64 1 are different dates. Any other type has
// no key that cannot be shared by two values and is an error.
func famaMacBethDateKey(d any) (string, error) {
	switch v := d.(type) {
	case string:
		return "s:" + v, nil
	case time.Time:
		return fmt.Sprintf("t:%d.%09d", v.Unix(), v.Nanosecond()), nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%T:%d", d, d), nil
	case float32:
		return famaMacBethFloatKey("float32", float64(v))
	case float64:
		return famaMacBethFloatKey("float64", v)
	}
	return "", fmt.Errorf("a date of type %T is not a supported date type; use a string, time.Time, bool, integer or float", d)
}

func famaMacBethFloatKey(name string, v float64) (string, error) {
	if math.IsNaN(v) {
		return "", errors.New("a date is NaN")
	}
	if v == 0 {
		v = 0 // -0 and 0 are the same date
	}
	return name + ":" + strconv.FormatFloat(v, 'g', -1, 64), nil
}
