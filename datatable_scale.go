package insyra

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"
)

// ScalerParams reports the fitted parameters for a single scaled or imputed
// column. Only the fields relevant to the fitted kind are populated; the rest
// stay at their zero value.
type ScalerParams struct {
	Column string
	Kind   string
	// Replacement is populated by fitted imputers. It is nil for scalers.
	Replacement any
	// PassThrough reports that a fitted imputer deliberately leaves this
	// column unchanged because its numeric strategy met observed non-numeric
	// values.
	PassThrough bool

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

// computeColumn turns a column's numeric values into fitted parameters; the
// affine coefficients apply needs are derived from them by setAffine.
func (s *scaler) computeColumn(ref, name string, vals []float64) scalerColumn {
	c := scalerColumn{ref: ref, name: name}
	c.params.Column = name
	c.params.Kind = s.kind

	switch s.kind {
	case "standard":
		mean := meanOf(vals)
		c.params.Mean = mean
		c.params.Std = sampleStdOf(vals, mean)
	case "minmax":
		min, max := minMaxOf(vals)
		c.params.Min = min
		c.params.Max = max
		c.params.OutputMin = s.featureMin
		c.params.OutputMax = s.featureMax
	case "robust":
		sorted := append([]float64(nil), vals...)
		sort.Float64s(sorted)
		c.params.Q1 = quantileQuartile(sorted, 0.25)
		c.params.Median = quantileQuartile(sorted, 0.5)
		c.params.Q3 = quantileQuartile(sorted, 0.75)
		c.params.IQR = c.params.Q3 - c.params.Q1
	case "maxabs":
		c.params.MaxAbs = maxAbsOf(vals)
	}
	s.setAffine(&c)
	return c
}

// setAffine derives the four affine coefficients apply uses from the column's
// stored parameters, so a scaler rebuilt from JSON scales exactly like the one
// that was fitted. Degenerate spreads set scale = 1.
func (s *scaler) setAffine(c *scalerColumn) {
	c.gain = 1
	c.offset = 0

	switch s.kind {
	case "standard":
		c.center = c.params.Mean
		c.scale = nonZero(c.params.Std)
	case "minmax":
		c.center = c.params.Min
		c.scale = nonZero(c.params.Max - c.params.Min)
		c.gain = s.featureMax - s.featureMin
		c.offset = s.featureMin
	case "robust":
		c.center = c.params.Median
		c.scale = nonZero(c.params.IQR)
	case "maxabs":
		c.center = 0
		c.scale = nonZero(c.params.MaxAbs)
	}
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

// quantileQuartile computes the RobustScaler's quartiles with the library's
// shared type-7 quantile (see DataList.quantileType7), so they match Quartile /
// IQR / Percentile / Describe and sklearn's RobustScaler (which uses
// np.percentile, i.e. type-7).
func quantileQuartile(sorted []float64, p float64) float64 {
	return quantileType7(sorted, p)
}

// jsonFloat is a float64 that survives encoding/json. Finite values are written
// as the shortest decimal that reads back bit for bit; the values JSON has no
// number for travel as their Go names.
type jsonFloat float64

func (f jsonFloat) MarshalJSON() ([]byte, error) {
	switch v := float64(f); {
	case math.IsNaN(v):
		return []byte(`"NaN"`), nil
	case math.IsInf(v, 1):
		return []byte(`"+Inf"`), nil
	case math.IsInf(v, -1):
		return []byte(`"-Inf"`), nil
	}
	return []byte(strconv.FormatFloat(float64(f), 'g', -1, 64)), nil
}

func (f *jsonFloat) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if len(s) > 0 && s[0] == '"' {
		switch s {
		case `"NaN"`:
			*f = jsonFloat(math.NaN())
			return nil
		case `"+Inf"`:
			*f = jsonFloat(math.Inf(1))
			return nil
		case `"-Inf"`:
			*f = jsonFloat(math.Inf(-1))
			return nil
		}
		return fmt.Errorf("insyra: scaler JSON: %s is not a known non-finite value", s)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("insyra: scaler JSON: %s is not a number", s)
	}
	*f = jsonFloat(v)
	return nil
}

// scalerJSON is the on-disk form of a fitted or unfitted scaler. The concrete
// type is not recoverable from the fields alone, so kind is written out and
// checked on the way back in.
type scalerJSON struct {
	Kind       string             `json:"kind"`
	FeatureMin jsonFloat          `json:"featureMin"`
	FeatureMax jsonFloat          `json:"featureMax"`
	Fitted     bool               `json:"fitted"`
	Columns    []scalerColumnJSON `json:"columns"`
}

type scalerColumnJSON struct {
	Ref       string    `json:"ref"`
	Name      string    `json:"name"`
	Mean      jsonFloat `json:"mean"`
	Std       jsonFloat `json:"std"`
	Min       jsonFloat `json:"min"`
	Max       jsonFloat `json:"max"`
	Median    jsonFloat `json:"median"`
	Q1        jsonFloat `json:"q1"`
	Q3        jsonFloat `json:"q3"`
	IQR       jsonFloat `json:"iqr"`
	MaxAbs    jsonFloat `json:"maxAbs"`
	OutputMin jsonFloat `json:"outputMin"`
	OutputMax jsonFloat `json:"outputMax"`
}

// MarshalJSON writes the scaler's kind, its output range and the fitted
// columns, so UnmarshalJSON on the same scaler type reads back one that
// transforms identically.
func (s *scaler) MarshalJSON() ([]byte, error) {
	out := scalerJSON{
		Kind:       s.kind,
		FeatureMin: jsonFloat(s.featureMin),
		FeatureMax: jsonFloat(s.featureMax),
		Fitted:     s.fitted,
		Columns:    make([]scalerColumnJSON, 0, len(s.cols)),
	}
	for _, c := range s.cols {
		out.Columns = append(out.Columns, scalerColumnJSON{
			Ref:       c.ref,
			Name:      c.name,
			Mean:      jsonFloat(c.params.Mean),
			Std:       jsonFloat(c.params.Std),
			Min:       jsonFloat(c.params.Min),
			Max:       jsonFloat(c.params.Max),
			Median:    jsonFloat(c.params.Median),
			Q1:        jsonFloat(c.params.Q1),
			Q3:        jsonFloat(c.params.Q3),
			IQR:       jsonFloat(c.params.IQR),
			MaxAbs:    jsonFloat(c.params.MaxAbs),
			OutputMin: jsonFloat(c.params.OutputMin),
			OutputMax: jsonFloat(c.params.OutputMax),
		})
	}
	return json.Marshal(out)
}

// unmarshalJSON reads what MarshalJSON wrote into a scaler of the given kind.
// The receiver is left untouched unless everything decodes.
func (s *scaler) unmarshalJSON(data []byte, kind string) error {
	var in scalerJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if in.Kind != kind {
		return fmt.Errorf("insyra: cannot read a %q scaler into a %s scaler", in.Kind, kind)
	}
	if in.Fitted != (len(in.Columns) > 0) {
		return fmt.Errorf("insyra: scaler JSON: fitted=%v but %d columns", in.Fitted, len(in.Columns))
	}

	tmp := scaler{
		kind:       kind,
		featureMin: float64(in.FeatureMin),
		featureMax: float64(in.FeatureMax),
		fitted:     in.Fitted,
	}
	var cols []scalerColumn
	seen := make(map[string]bool, len(in.Columns))
	for _, c := range in.Columns {
		// Fit refuses a column selected twice, so a reference listed twice was
		// not written by Fit, and reading it would keep only the last one.
		if seen[c.Ref] {
			return fmt.Errorf("insyra: scaler JSON: column %q is listed twice", c.Ref)
		}
		seen[c.Ref] = true
		col := scalerColumn{
			ref:  c.Ref,
			name: c.Name,
			params: ScalerParams{
				Column: c.Name,
				Kind:   kind,
				Mean:   float64(c.Mean),
				Std:    float64(c.Std),
				Min:    float64(c.Min),
				Max:    float64(c.Max),
				Median: float64(c.Median),
				Q1:     float64(c.Q1),
				Q3:     float64(c.Q3),
				IQR:    float64(c.IQR),
				MaxAbs: float64(c.MaxAbs),

				OutputMin: float64(c.OutputMin),
				OutputMax: float64(c.OutputMax),
			},
		}
		tmp.setAffine(&col)
		cols = append(cols, col)
	}
	tmp.cols = cols
	*s = tmp
	return nil
}

// UnmarshalJSON reads a scaler written by MarshalJSON. It refuses JSON written
// by another kind of scaler.
func (s *StandardScaler) UnmarshalJSON(data []byte) error {
	return s.unmarshalJSON(data, "standard")
}

// UnmarshalJSON reads a scaler written by MarshalJSON. It refuses JSON written
// by another kind of scaler.
func (s *MinMaxScaler) UnmarshalJSON(data []byte) error {
	return s.unmarshalJSON(data, "minmax")
}

// UnmarshalJSON reads a scaler written by MarshalJSON. It refuses JSON written
// by another kind of scaler.
func (s *RobustScaler) UnmarshalJSON(data []byte) error {
	return s.unmarshalJSON(data, "robust")
}

// UnmarshalJSON reads a scaler written by MarshalJSON. It refuses JSON written
// by another kind of scaler.
func (s *MaxAbsScaler) UnmarshalJSON(data []byte) error {
	return s.unmarshalJSON(data, "maxabs")
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

	_ json.Marshaler   = (*StandardScaler)(nil)
	_ json.Unmarshaler = (*StandardScaler)(nil)
	_ json.Marshaler   = (*MinMaxScaler)(nil)
	_ json.Unmarshaler = (*MinMaxScaler)(nil)
	_ json.Marshaler   = (*RobustScaler)(nil)
	_ json.Unmarshaler = (*RobustScaler)(nil)
	_ json.Marshaler   = (*MaxAbsScaler)(nil)
	_ json.Unmarshaler = (*MaxAbsScaler)(nil)
)
