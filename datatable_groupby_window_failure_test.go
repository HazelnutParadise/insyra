package insyra

import "testing"

// assertGroupedEmptyWithError checks the same shape as assertEmptyWithError, for
// the grouped transforms: a non-nil, empty, still-usable DataList whose Err() is
// set. The error the caller reaches on the result matters as much as the one on
// the table, because the table's is sticky and may already hold an earlier
// failure.
func assertGroupedEmptyWithError(t *testing.T, name string, got *DataList) {
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

// A column or grouping that could not be looked up is a failure the caller
// asked for, so .As hands back an empty list carrying the table's error rather
// than a bare empty list that looks like a valid (if empty) result.
func TestGroupedAsCarriesTheLookupError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("missing column", func(t *testing.T) {
		dt := buildPanelTable()
		got := dt.GroupBy(Name("id")).CumSumCol(Name("missing")).As("x")

		assertGroupedEmptyWithError(t, "CumSumCol", got)
		if err := dt.Err(); err == nil {
			t.Error("expected an error recorded on dt")
		}
	})

	t.Run("missing grouping column", func(t *testing.T) {
		dt := buildPanelTable()
		got := dt.GroupBy(Name("nope")).ShiftCol(Name("price"), 1).As("x")

		assertGroupedEmptyWithError(t, "ShiftCol", got)
		if err := dt.Err(); err == nil {
			t.Error("expected an error recorded on dt")
		}
	})
}

// An argument the per-group transform refuses is the same for every group, so
// one group failing is the whole call failing. Scattering the per-group empty
// results back would instead produce a full-length column of nils that claims
// to be a result.
func TestGroupedInvalidArgumentIsNotAColumnOfNil(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		name string
		call func(dt *DataTable) *DataList
	}{
		{"RollingCol", func(dt *DataTable) *DataList {
			return dt.GroupBy(Name("id")).RollingCol(Name("price"), RollingOptions{Window: 0}).Mean().As("x")
		}},
		{"DiffCol", func(dt *DataTable) *DataList {
			return dt.GroupBy(Name("id")).DiffCol(Name("price"), 0).As("x")
		}},
		{"ShiftCol", func(dt *DataTable) *DataList {
			return dt.GroupBy(Name("id")).ShiftCol(Name("price"), 1, 0.0, 9.0).As("x")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dt := buildPanelTable()
			got := tc.call(dt)

			assertGroupedEmptyWithError(t, tc.name, got)
			if err := dt.Err(); err == nil {
				t.Errorf("%s: expected an error recorded on dt", tc.name)
			}
			if g := got.GetName(); g != "x" {
				t.Errorf("%s: GetName() = %q, want %q", tc.name, g, "x")
			}
		})
	}
}

// Failing the whole call must not touch a valid one.
func TestGroupedValidTransformStillWorks(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dt := buildPanelTable()
	got := dt.GroupBy(Name("id")).ShiftCol(Name("price"), 1).As("prev")

	if n, want := got.Len(), dt.GetCol(Name("price")).Len(); n != want {
		t.Errorf("Len() = %d, want %d", n, want)
	}
	if err := got.Err(); err != nil {
		t.Errorf("expected no error on the result, got %v", err.Message)
	}
	if err := dt.Err(); err != nil {
		t.Errorf("expected no error on dt, got %v", err.Message)
	}
}
