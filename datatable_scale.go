package insyra

import (
	"fmt"
	"math"
	"sort"
)

// ScalerParams reports the fitted parameters for a single scaled column.
// Only the fields relevant to the scaler kind are populated; the rest stay
// at their zero value.
type ScalerParams struct {
	Column string
	Kind   string

	Mean   float64
	Std    float64
	Min    float64
	Max    float64
	Median float64
	Q1     float64
	Q3     float64
	IQR    float64
	MaxAbs float64

	OutputMin float64
	OutputMax float64
}

// Scaler is the shared surface for fitted, reusable feature scalers.
//
// Unlike DataList.Normalize/Standardize (stateless, in-place), a Scaler fits
// parameters once and can Transform/InverseTransform new tables with the same
// parameters, which is the correct way to scale a test set with statistics
// learned from the training set (no data leakage).
type Scaler interface {
	Fit(dt *DataTable, cols ...string) error
	Transform(dt *DataTable) (*DataTable, error)
	FitTransform(dt *DataTable, cols ...string) (*DataTable, error)
	InverseTransform(dt *DataTable) (*DataTable, error)
	Params() map[string]ScalerParams
	Kind() string
}

// DataListScaler is the DataList-oriented counterpart of Scaler.
type DataListScaler interface {
	FitDataList(dl *DataList) error
	TransformDataList(dl *DataList) (*DataList, error)
	FitTransformDataList(dl *DataList) (*DataList, error)
	InverseTransformDataList(dl *DataList) (*DataList, error)
}

// scalerColumn holds the fitted state for one column. Every scaler reduces to
// the affine map y = (x-center)/scale*gain + offset, with the inverse
// x = (y-offset)/gain*scale + center. Degenerate inputs set scale = 1 so the
// transform never divides by zero and never panics.
type scalerColumn struct {
	ref    string
	name   string
	params ScalerParams

	center float64
	scale  float64
	gain   float64
	offset float64
}

// scaler is the embedded base shared by all concrete scalers. The concrete
// type only carries the kind tag and (for min-max) the feature range.
type scaler struct {
	kind       string
	featureMin float64
	featureMax float64

	cols   []scalerColumn
	fitted bool
}

// StandardScaler scales columns to zero mean and unit (sample) standard
// deviation, matching DataList.Standardize's use of the sample stdev.
type StandardScaler struct{ scaler }

// MinMaxScaler scales columns into a [featureMin, featureMax] range.
type MinMaxScaler struct{ scaler }

// RobustScaler centers on the median and scales by the IQR, making it robust
// to outliers.
type RobustScaler struct{ scaler }

// MaxAbsScaler scales each column by its maximum absolute value, preserving
// sign and mapping data into [-1, 1].
type MaxAbsScaler struct{ scaler }

// NewStandardScaler returns an unfitted standard scaler.
func NewStandardScaler() *StandardScaler { return &StandardScaler{scaler{kind: "standard"}} }

// NewMinMaxScaler returns an unfitted min-max scaler targeting the given range.
func NewMinMaxScaler(featureMin, featureMax float64) *MinMaxScaler {
	return &MinMaxScaler{scaler{kind: "minmax", featureMin: featureMin, featureMax: featureMax}}
}

// NewDefaultMinMaxScaler returns a min-max scaler targeting [0, 1].
func NewDefaultMinMaxScaler() *MinMaxScaler { return NewMinMaxScaler(0, 1) }

// NewRobustScaler returns an unfitted robust scaler.
func NewRobustScaler() *RobustScaler { return &RobustScaler{scaler{kind: "robust"}} }

// NewMaxAbsScaler returns an unfitted max-abs scaler.
func NewMaxAbsScaler() *MaxAbsScaler { return &MaxAbsScaler{scaler{kind: "maxabs"}} }

// StandardScale fits a StandardScaler on cols and returns the scaled table.
func (dt *DataTable) StandardScale(cols ...string) (*DataTable, *StandardScaler, error) {
	sc := NewStandardScaler()
	out, err := sc.FitTransform(dt, cols...)
	if err != nil {
		return nil, nil, err
	}
	return out, sc, nil
}

// MinMaxScale fits a MinMaxScaler on cols and returns the scaled table.
func (dt *DataTable) MinMaxScale(featureMin, featureMax float64, cols ...string) (*DataTable, *MinMaxScaler, error) {
	sc := NewMinMaxScaler(featureMin, featureMax)
	out, err := sc.FitTransform(dt, cols...)
	if err != nil {
		return nil, nil, err
	}
	return out, sc, nil
}

// RobustScale fits a RobustScaler on cols and returns the scaled table.
func (dt *DataTable) RobustScale(cols ...string) (*DataTable, *RobustScaler, error) {
	sc := NewRobustScaler()
	out, err := sc.FitTransform(dt, cols...)
	if err != nil {
		return nil, nil, err
	}
	return out, sc, nil
}

// MaxAbsScale fits a MaxAbsScaler on cols and returns the scaled table.
func (dt *DataTable) MaxAbsScale(cols ...string) (*DataTable, *MaxAbsScaler, error) {
	sc := NewMaxAbsScaler()
	out, err := sc.FitTransform(dt, cols...)
	if err != nil {
		return nil, nil, err
	}
	return out, sc, nil
}

// Kind returns the scaler family name ("standard", "minmax", "robust", "maxabs").
func (s *scaler) Kind() string { return s.kind }

// Params returns the fitted parameters keyed by output column name.
func (s *scaler) Params() map[string]ScalerParams {
	out := make(map[string]ScalerParams, len(s.cols))
	for _, c := range s.cols {
		out[c.name] = c.params
	}
	return out
}

// Fit learns scaling parameters from the given columns without modifying dt.
// cols is required; pass at least one column reference (name or Excel-style
// index such as "A").
func (s *scaler) Fit(dt *DataTable, cols ...string) error {
	if dt == nil {
		return fmt.Errorf("%sScaler.Fit: table is nil", s.kind)
	}
	if len(cols) == 0 {
		return fmt.Errorf("%sScaler.Fit: at least one column is required", s.kind)
	}
	fitted := make([]scalerColumn, 0, len(cols))
	var err error
	dt.AtomicDo(func(t *DataTable) {
		seen := map[int]struct{}{}
		for _, ref := range cols {
			idx, label, ok := resolveEncodingColumn(t, ref)
			if !ok {
				err = fmt.Errorf("%sScaler.Fit: column %q not found", s.kind, ref)
				return
			}
			if _, dup := seen[idx]; dup {
				err = fmt.Errorf("%sScaler.Fit: column %q listed more than once", s.kind, ref)
				return
			}
			seen[idx] = struct{}{}

			name := label
			if t.columns[idx].name != "" {
				name = t.columns[idx].name
			}
			var vals []float64
			vals, err = numericColumnValues(t.columns[idx].data, s.kind+"Scaler.Fit")
			if err != nil {
				return
			}
			fitted = append(fitted, s.computeColumn(label, name, vals))
		}
	})
	if err != nil {
		return err
	}
	s.cols = fitted
	s.fitted = true
	return nil
}

// FitTransform fits on cols and immediately returns the scaled table.
func (s *scaler) FitTransform(dt *DataTable, cols ...string) (*DataTable, error) {
	if err := s.Fit(dt, cols...); err != nil {
		return nil, err
	}
	return s.Transform(dt)
}

// Transform applies the fitted parameters to dt and returns a new table.
// The original table is not modified. Unfitted columns pass through unchanged.
// A fitted column missing from dt is an error.
func (s *scaler) Transform(dt *DataTable) (*DataTable, error) {
	return s.apply(dt, false)
}

// InverseTransform restores the original scale of fitted columns and returns a
// new table. Unfitted columns pass through unchanged; fitted columns absent
// from dt are simply skipped (so predictions covering a subset still work).
func (s *scaler) InverseTransform(dt *DataTable) (*DataTable, error) {
	return s.apply(dt, true)
}

func (s *scaler) apply(dt *DataTable, inverse bool) (*DataTable, error) {
	op := "Transform"
	if inverse {
		op = "InverseTransform"
	}
	if !s.fitted {
		return nil, fmt.Errorf("%sScaler.%s: scaler is not fitted", s.kind, op)
	}
	if dt == nil {
		return nil, fmt.Errorf("%sScaler.%s: table is nil", s.kind, op)
	}
	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		colByIndex := map[int]*scalerColumn{}
		for i := range s.cols {
			idx, _, ok := resolveEncodingColumn(t, s.cols[i].ref)
			if !ok {
				if inverse {
					continue
				}
				err = fmt.Errorf("%sScaler.%s: fitted column %q not found", s.kind, op, s.cols[i].ref)
				return
			}
			colByIndex[idx] = &s.cols[i]
		}
		outCols := make([]*DataList, 0, len(t.columns))
		for idx, col := range t.columns {
			c, scaled := colByIndex[idx]
			if !scaled {
				outCols = append(outCols, col.Clone())
				continue
			}
			var transformed *DataList
			transformed, err = s.applyColumn(c, col.name, col.data, inverse, op)
			if err != nil {
				return
			}
			outCols = append(outCols, transformed)
		}
		out.AppendCols(outCols...)
		copyRowNamesNotAtomic(out, t)
		out.name = t.name
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *scaler) applyColumn(c *scalerColumn, name string, data []any, inverse bool, op string) (*DataList, error) {
	dst := NewDataList()
	dst.SetName(name)
	for _, raw := range data {
		if isNilOrNaN(raw) {
			dst.Append(raw)
			continue
		}
		x, ok := ToFloat64Safe(raw)
		if !ok {
			return nil, fmt.Errorf("%sScaler.%s: column %q has non-numeric value %v", s.kind, op, c.name, raw)
		}
		var y float64
		if inverse {
			y = (x-c.offset)/c.gain*c.scale + c.center
		} else {
			y = (x-c.center)/c.scale*c.gain + c.offset
		}
		dst.Append(y)
	}
	return dst, nil
}

// computeColumn turns a column's numeric values into fitted parameters and the
// affine coefficients used by apply. Degenerate spreads set scale = 1.
func (s *scaler) computeColumn(ref, name string, vals []float64) scalerColumn {
	c := scalerColumn{ref: ref, name: name}
	c.params.Column = name
	c.params.Kind = s.kind
	c.gain = 1
	c.offset = 0

	switch s.kind {
	case "standard":
		mean := meanOf(vals)
		std := sampleStdOf(vals, mean)
		c.params.Mean = mean
		c.params.Std = std
		c.center = mean
		c.scale = nonZero(std)
	case "minmax":
		min, max := minMaxOf(vals)
		c.params.Min = min
		c.params.Max = max
		c.params.OutputMin = s.featureMin
		c.params.OutputMax = s.featureMax
		c.center = min
		c.scale = nonZero(max - min)
		c.gain = s.featureMax - s.featureMin
		c.offset = s.featureMin
	case "robust":
		sorted := append([]float64(nil), vals...)
		sort.Float64s(sorted)
		q1 := quantileQuartile(sorted, 0.25)
		median := quantileQuartile(sorted, 0.5)
		q3 := quantileQuartile(sorted, 0.75)
		iqr := q3 - q1
		c.params.Median = median
		c.params.Q1 = q1
		c.params.Q3 = q3
		c.params.IQR = iqr
		c.center = median
		c.scale = nonZero(iqr)
	case "maxabs":
		maxAbs := maxAbsOf(vals)
		c.params.MaxAbs = maxAbs
		c.center = 0
		c.scale = nonZero(maxAbs)
	}
	return c
}

// FitDataList learns scaling parameters from a single DataList.
func (s *scaler) FitDataList(dl *DataList) error {
	if dl == nil {
		return fmt.Errorf("%sScaler.FitDataList: list is nil", s.kind)
	}
	var fitted scalerColumn
	var err error
	dl.AtomicDo(func(d *DataList) {
		var vals []float64
		vals, err = numericColumnValues(d.data, s.kind+"Scaler.FitDataList")
		if err != nil {
			return
		}
		fitted = s.computeColumn(d.name, d.name, vals)
	})
	if err != nil {
		return err
	}
	s.cols = []scalerColumn{fitted}
	s.fitted = true
	return nil
}

// FitTransformDataList fits on dl and returns a new scaled DataList.
func (s *scaler) FitTransformDataList(dl *DataList) (*DataList, error) {
	if err := s.FitDataList(dl); err != nil {
		return nil, err
	}
	return s.TransformDataList(dl)
}

// TransformDataList scales dl using the fitted parameters, returning a new
// DataList. The original list is not modified.
func (s *scaler) TransformDataList(dl *DataList) (*DataList, error) {
	return s.applyList(dl, false)
}

// InverseTransformDataList restores the original scale of dl, returning a new
// DataList.
func (s *scaler) InverseTransformDataList(dl *DataList) (*DataList, error) {
	return s.applyList(dl, true)
}

func (s *scaler) applyList(dl *DataList, inverse bool) (*DataList, error) {
	op := "TransformDataList"
	if inverse {
		op = "InverseTransformDataList"
	}
	if !s.fitted || len(s.cols) == 0 {
		return nil, fmt.Errorf("%sScaler.%s: scaler is not fitted", s.kind, op)
	}
	if dl == nil {
		return nil, fmt.Errorf("%sScaler.%s: list is nil", s.kind, op)
	}
	c := &s.cols[0]
	var out *DataList
	var err error
	dl.AtomicDo(func(d *DataList) {
		out, err = s.applyColumn(c, d.name, d.data, inverse, op)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// numericColumnValues extracts numeric values, skipping nil/NaN. A non-numeric,
// non-missing value is an error (scalers only apply to numeric columns).
func numericColumnValues(data []any, method string) ([]float64, error) {
	vals := make([]float64, 0, len(data))
	for _, raw := range data {
		if isNilOrNaN(raw) {
			continue
		}
		f, ok := ToFloat64Safe(raw)
		if !ok {
			return nil, fmt.Errorf("%s: non-numeric value %v", method, raw)
		}
		vals = append(vals, f)
	}
	return vals, nil
}

func nonZero(v float64) float64 {
	if v == 0 || math.IsNaN(v) {
		return 1
	}
	return v
}

func meanOf(vals []float64) float64 {
	if len(vals) == 0 {
		return math.NaN()
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// sampleStdOf mirrors DataList.Stdev (sample standard deviation, ddof=1).
// Fewer than two values yields 0 (treated as a degenerate, constant column).
func sampleStdOf(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var ss float64
	for _, v := range vals {
		d := v - mean
		ss += d * d
	}
	return math.Sqrt(ss / float64(len(vals)-1))
}

func minMaxOf(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return math.NaN(), math.NaN()
	}
	min, max := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func maxAbsOf(vals []float64) float64 {
	var m float64
	for _, v := range vals {
		if a := math.Abs(v); a > m {
			m = a
		}
	}
	return m
}

// quantileQuartile mirrors DataList.Quartile's p*(n+1) positioning so the
// RobustScaler's quartiles match the library's IQR().
func quantileQuartile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0]
	}
	pos := p * float64(n+1)
	if pos < 1.0 {
		pos = 1.0
	} else if pos > float64(n) {
		pos = float64(n)
	}
	lower := int(math.Floor(pos)) - 1
	upper := int(math.Ceil(pos)) - 1
	if lower < 0 {
		lower = 0
	}
	if upper >= n {
		upper = n - 1
	}
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - math.Floor(pos)
	return sorted[lower] + frac*(sorted[upper]-sorted[lower])
}

// compile-time interface checks
var (
	_ Scaler = (*StandardScaler)(nil)
	_ Scaler = (*MinMaxScaler)(nil)
	_ Scaler = (*RobustScaler)(nil)
	_ Scaler = (*MaxAbsScaler)(nil)

	_ DataListScaler = (*StandardScaler)(nil)
	_ DataListScaler = (*MinMaxScaler)(nil)
	_ DataListScaler = (*RobustScaler)(nil)
	_ DataListScaler = (*MaxAbsScaler)(nil)
)
