package insyra

import (
	"reflect"
	"testing"
)

// assertCarriesError checks the shape every failed window transform owes its
// caller: a non-nil, empty, still-usable DataList whose Err() is set.
func assertCarriesError(t *testing.T, name string, got *DataList) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: returned nil", name)
	}
	if n := got.Len(); n != 0 {
		t.Errorf("%s: expected length 0, got %d", name, n)
	}
	if err := got.Err(); err == nil {
		t.Errorf("%s: expected the returned list to carry an error", name)
	}
}

func TestRollingInvalidOptionsCarryTheError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		label string
		opts  RollingOptions
	}{
		{"Window: 0", RollingOptions{Window: 0}},
		{"MinObs > Window", RollingOptions{Window: 3, MinObs: 5}},
		{"Weights length mismatch", RollingOptions{Window: 2, Weights: []float64{1}}},
	}

	for _, tc := range cases {
		reducers := []struct {
			name string
			call func(r *RollingDataList) *DataList
		}{
			{"Sum", func(r *RollingDataList) *DataList { return r.Sum() }},
			{"Mean", func(r *RollingDataList) *DataList { return r.Mean() }},
			{"Min", func(r *RollingDataList) *DataList { return r.Min() }},
			{"Max", func(r *RollingDataList) *DataList { return r.Max() }},
			{"Median", func(r *RollingDataList) *DataList { return r.Median() }},
			{"Std", func(r *RollingDataList) *DataList { return r.Std() }},
			{"Var", func(r *RollingDataList) *DataList { return r.Var() }},
			{"Apply", func(r *RollingDataList) *DataList {
				return r.Apply(func(w []any) any { return 0 })
			}},
			{"Corr", func(r *RollingDataList) *DataList { return r.Corr(NewDataList(1.0, 2.0, 3.0)) }},
			{"Cov", func(r *RollingDataList) *DataList { return r.Cov(NewDataList(1.0, 2.0, 3.0)) }},
			{"Beta", func(r *RollingDataList) *DataList { return r.Beta(NewDataList(1.0, 2.0, 3.0)) }},
		}

		for _, red := range reducers {
			// Err() is sticky, so every reducer needs a fresh source.
			src := NewDataList(1.0, 2.0, 3.0)
			got := red.call(src.Rolling(tc.opts))
			assertCarriesError(t, tc.label+"/"+red.name, got)
			if err := src.Err(); err == nil {
				t.Errorf("%s/%s: expected an error recorded on the source list", tc.label, red.name)
			}
		}
	}

	// The first failure reports the window check, not a downstream step.
	src := NewDataList(1.0, 2.0, 3.0)
	_ = src.Rolling(RollingOptions{Window: 0}).Mean()
	err := src.Err()
	if err == nil {
		t.Fatal("expected an error recorded on the source list")
	}
	if err.FuncName != "Rolling" {
		t.Errorf("FuncName = %q, want %q", err.FuncName, "Rolling")
	}
	if err.Message != "Window must be > 0" {
		t.Errorf("Message = %q, want %q", err.Message, "Window must be > 0")
	}
}

func TestRollingMisusedReducersCarryTheError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	reducers := []struct {
		name string
		call func(r *RollingDataList) *DataList
	}{
		{"Apply(nil)", func(r *RollingDataList) *DataList { return r.Apply(nil) }},
		{"Corr(nil)", func(r *RollingDataList) *DataList { return r.Corr(nil) }},
		{"Cov(nil)", func(r *RollingDataList) *DataList { return r.Cov(nil) }},
		{"Beta(nil)", func(r *RollingDataList) *DataList { return r.Beta(nil) }},
	}

	for _, red := range reducers {
		src := NewDataList(1.0, 2.0, 3.0)
		got := red.call(src.Rolling(RollingOptions{Window: 2}))
		assertCarriesError(t, red.name, got)
		if err := src.Err(); err == nil {
			t.Errorf("%s: expected an error recorded on the source list", red.name)
		}
	}
}

func TestEWMInvalidDecayCarriesTheError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		label string
		opts  EWMOptions
	}{
		{"no decay", EWMOptions{}},
		{"Alpha out of range", EWMOptions{Alpha: 1.5}},
		{"two decays", EWMOptions{Alpha: 0.5, Span: 3}},
		{"HalfLife negative", EWMOptions{HalfLife: -1}},
	}

	for _, tc := range cases {
		reducers := []struct {
			name string
			call func(e *EWMDataList) *DataList
		}{
			{"Mean", func(e *EWMDataList) *DataList { return e.Mean() }},
			{"Var", func(e *EWMDataList) *DataList { return e.Var() }},
			{"Std", func(e *EWMDataList) *DataList { return e.Std() }},
		}

		for _, red := range reducers {
			src := NewDataList(1.0, 2.0, 3.0)
			got := red.call(src.EWM(tc.opts))
			assertCarriesError(t, tc.label+"/"+red.name, got)
			if err := src.Err(); err == nil {
				t.Errorf("%s/%s: expected an error recorded on the source list", tc.label, red.name)
			}
		}
	}

	src := NewDataList(1.0, 2.0, 3.0)
	_ = src.EWM(EWMOptions{}).Mean()
	err := src.Err()
	if err == nil {
		t.Fatal("expected an error recorded on the source list")
	}
	if want := "exactly one of Alpha, Span, or HalfLife must be specified"; err.Message != want {
		t.Errorf("Message = %q, want %q", err.Message, want)
	}
}

func TestTableWindowBuildersCarryMissingColumnError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		name string
		call func(dt *DataTable) *DataList
	}{
		{"RollingCol", func(dt *DataTable) *DataList {
			return dt.RollingCol(Name("missing"), RollingOptions{Window: 2}).Mean()
		}},
		{"ExpandingCol", func(dt *DataTable) *DataList {
			return dt.ExpandingCol(Name("missing"), 1).Sum()
		}},
		{"EWMCol", func(dt *DataTable) *DataList {
			return dt.EWMCol(Name("missing"), EWMOptions{Alpha: 0.5}).Mean()
		}},
	}

	for _, tc := range cases {
		dt := NewDataTable(NewDataList(1.0, 2.0, 3.0).SetName("a"))
		assertCarriesError(t, tc.name, tc.call(dt))
		if err := dt.Err(); err == nil {
			t.Errorf("%s: expected an error recorded on dt", tc.name)
		}
	}
}

// A window wider than the data is a normal result, not a failure: the positions
// with too few observations are nil and nothing anywhere records an error.
func TestValidWindowWithTooFewObservationsIsStillNil(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	src := NewDataList(1.0, 2.0)
	got := src.Rolling(RollingOptions{Window: 3}).Mean()

	if want := []any{nil, nil}; !reflect.DeepEqual(got.Data(), want) {
		t.Errorf("Data() = %v, want %v", got.Data(), want)
	}
	if err := got.Err(); err != nil {
		t.Errorf("expected no error on the result, got %v", err.Message)
	}
	if err := src.Err(); err != nil {
		t.Errorf("expected no error on the source list, got %v", err.Message)
	}
}
