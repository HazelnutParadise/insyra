package insyra

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

// #221 (D-13): Quantile takes 0..1, Percentile 0..100 and Quartile 1..3, and
// all three are the same R type-7 quantile, so they agree with each other, with
// Describe and with pandas.

// The expected values were computed with pandas 3.0.5:
// pd.Series([1, 2, 3, 4, 10]).quantile([0, .25, .5, .75, 1]) and the
// pd.Series([1, 2, 3, 4]).quantile(.25) example of pandas' own documentation.
func TestQuantileMatchesPandas(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, 10)
	for _, tc := range []struct{ p, want float64 }{
		{0, 1}, {0.25, 2}, {0.5, 3}, {0.75, 4}, {1, 10},
	} {
		if got := dl.Quantile(tc.p); got != tc.want {
			t.Errorf("Quantile(%v) = %v, want %v", tc.p, got, tc.want)
		}
	}
	if got := NewDataList(1, 2, 3, 4).Quantile(0.25); got != 1.75 {
		t.Errorf("Quantile(0.25) of [1 2 3 4] = %v, want 1.75", got)
	}
	if err := dl.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQuantilePercentileQuartileAgree(t *testing.T) {
	rng := rand.New(rand.NewSource(221))
	for trial := 0; trial < 50; trial++ {
		n := 1 + rng.Intn(40)
		values := make([]any, n)
		for i := range values {
			values[i] = rng.NormFloat64() * 10
		}
		dl := NewDataList(values...)
		for _, p := range []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1} {
			q := dl.Quantile(p)
			if pc := dl.Percentile(p * 100); pc != q {
				t.Fatalf("n=%d: Percentile(%v) = %v, Quantile(%v) = %v", n, p*100, pc, p, q)
			}
		}
		for k, p := range map[int]float64{1: 0.25, 2: 0.5, 3: 0.75} {
			if got, want := dl.Quartile(k), dl.Quantile(p); got != want {
				t.Fatalf("n=%d: Quartile(%d) = %v, Quantile(%v) = %v", n, k, got, p, want)
			}
		}
		if got, want := dl.Median(), dl.Quantile(0.5); math.Abs(got-want) > 1e-12 {
			t.Fatalf("n=%d: Median = %v, Quantile(0.5) = %v", n, got, want)
		}
		if err := dl.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// Before #221 a NaN was sorted into the data: [1 2 3 NaN] gave Quartile(1)
// NaN, Quartile(3) 2.25 and Median 1.5, while Describe and pandas skip it
// and give 1.5, 2.5 and 2.
func TestQuantileFamilySkipsNaNAndNil(t *testing.T) {
	for _, dl := range []*DataList{
		NewDataList(1, 2, 3, math.NaN()),
		NewDataList(math.NaN(), 1, nil, 2, 3),
	} {
		checks := []struct {
			name      string
			got, want float64
		}{
			{"Quantile(0)", dl.Quantile(0), 1},
			{"Quantile(0.25)", dl.Quantile(0.25), 1.5},
			{"Quantile(0.75)", dl.Quantile(0.75), 2.5},
			{"Quantile(1)", dl.Quantile(1), 3},
			{"Percentile(75)", dl.Percentile(75), 2.5},
			{"Quartile(1)", dl.Quartile(1), 1.5},
			{"Quartile(3)", dl.Quartile(3), 2.5},
			{"IQR", dl.IQR(), 1},
			{"Median", dl.Median(), 2},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("%v: %s = %v, want %v", dl.Data(), c.name, c.got, c.want)
			}
		}
		if err := dl.Err(); err != nil {
			t.Errorf("%v: skipping a missing value is not an error: %v", dl.Data(), err)
		}
	}
}

func TestQuantileAgreesWithDescribe(t *testing.T) {
	col := NewDataList(5, math.NaN(), 1, 9, nil, 4, 4, 7)
	desc := NewDataTable(col).Describe(DescribeOptions{Percentiles: []float64{0.1, 0.25, 0.5, 0.9}})
	rows := map[string]float64{"10%": 0.1, "25%": 0.25, "50%": 0.5, "90%": 0.9}
	for name, p := range rows {
		idx, ok := desc.GetRowIndexByName(name)
		if !ok {
			t.Fatalf("Describe has no row %q", name)
		}
		want, _ := ToFloat64Safe(desc.GetElementByNumberIndex(idx, 0))
		if got := col.Quantile(p); got != want {
			t.Errorf("Quantile(%v) = %v, Describe %s = %v", p, got, name, want)
		}
	}
}

// Percentile(NaN) panicked with "index out of range" before #221.
func TestQuantileRefusesAProbabilityOutOfRange(t *testing.T) {
	cases := []struct {
		name string
		call func(dl *DataList) float64
	}{
		{"Quantile(NaN)", func(dl *DataList) float64 { return dl.Quantile(math.NaN()) }},
		{"Quantile(-0.1)", func(dl *DataList) float64 { return dl.Quantile(-0.1) }},
		{"Quantile(1.1)", func(dl *DataList) float64 { return dl.Quantile(1.1) }},
		{"Quantile(+Inf)", func(dl *DataList) float64 { return dl.Quantile(math.Inf(1)) }},
		{"Percentile(NaN)", func(dl *DataList) float64 { return dl.Percentile(math.NaN()) }},
		{"Percentile(101)", func(dl *DataList) float64 { return dl.Percentile(101) }},
		{"Quartile(0)", func(dl *DataList) float64 { return dl.Quartile(0) }},
		{"Quartile(4)", func(dl *DataList) float64 { return dl.Quartile(4) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dl := NewDataList(1, 2, 3)
			var got float64
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panicked: %v", r)
					}
				}()
				got = tc.call(dl)
			}()
			if !math.IsNaN(got) {
				t.Errorf("got %v, want NaN", got)
			}
			method := tc.name[:strings.Index(tc.name, "(")]
			if err := dl.Err(); err == nil || !strings.Contains(err.Error(), method) {
				t.Errorf("Err() = %v, want an error naming %s", err, method)
			}
		})
	}
}

// Before #221 Percentile recorded an error on a text cell and still returned
// a number, while Quartile only warned. Every member of the family now
// records the error under its own name and returns NaN.
func TestQuantileFamilyRefusesATextCell(t *testing.T) {
	cases := map[string]func(dl *DataList) float64{
		"Quantile":   func(dl *DataList) float64 { return dl.Quantile(0.5) },
		"Percentile": func(dl *DataList) float64 { return dl.Percentile(50) },
		"Quartile":   func(dl *DataList) float64 { return dl.Quartile(2) },
		"IQR":        func(dl *DataList) float64 { return dl.IQR() },
	}
	for method, call := range cases {
		dl := NewDataList(1, 2, "x", 3)
		if got := call(dl); !math.IsNaN(got) {
			t.Errorf("%s = %v, want NaN", method, got)
		}
		err := dl.Err()
		if err == nil {
			t.Errorf("%s: no error for a text cell", method)
			continue
		}
		if !strings.Contains(err.Error(), "DataList."+method+":") {
			t.Errorf("%s: error does not name the call: %v", method, err)
		}
		if !strings.Contains(err.Error(), "row 3") {
			t.Errorf("%s: error does not name the row: %v", method, err)
		}
	}
}

func TestQuantileOfNothingIsNaNWithoutAnError(t *testing.T) {
	for _, dl := range []*DataList{NewDataList(), NewDataList(nil, math.NaN())} {
		if got := dl.Quantile(0.5); !math.IsNaN(got) {
			t.Errorf("%v: Quantile = %v, want NaN", dl.Data(), got)
		}
		if got := dl.IQR(); !math.IsNaN(got) {
			t.Errorf("%v: IQR = %v, want NaN", dl.Data(), got)
		}
		if err := dl.Err(); err != nil {
			t.Errorf("%v: an empty input is not an error: %v", dl.Data(), err)
		}
	}
}

func TestQuantileIsOnTheInterface(t *testing.T) {
	var dl IDataList = NewDataList(1, 2, 3)
	if got := dl.Quantile(0.5); got != 2 {
		t.Errorf("IDataList.Quantile(0.5) = %v, want 2", got)
	}
}
