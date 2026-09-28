package stats_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// The paired and rank tests used to check only whether a cell converts, so a
// NaN or ±Inf flowed straight into the arithmetic and came back as a plausible
// looking statistic with no error. A blank or text cell was refused, but with a
// message carrying no position at all — "invalid numeric value in data1" — so
// a reader could not find the cell. Every other one- and two-sample test names
// the series and the one-based row; these four now do the same.
//
// The messages are compared exactly, not by substring. The point of the change
// is that one message shape is shared, and a substring check would let a
// half-converted one through.
func TestPairedAndOneSampleTestsRefuseUnreadableCells(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bad   any
		kind  string
		shown string
	}{
		{"NaN", math.NaN(), "non-finite", "NaN"},
		{"+Inf", math.Inf(1), "non-finite", "+Inf"},
		{"-Inf", math.Inf(-1), "non-finite", "-Inf"},
		{"blank", nil, "non-numeric", "<nil>"},
		{"text", "x", "non-numeric", "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := insyra.NewDataList(1.0, 2.0, tc.bad, 4.0)
			clean := insyra.NewDataList(1.5, 2.5, 3.0, 4.5)
			want := func(label string) string {
				return fmt.Sprintf("%s contains a %s value at row 3: %s", label, tc.kind, tc.shown)
			}

			for _, call := range []struct {
				name     string
				run      func() (resultNil bool, err error)
				want     string
				wantName string
			}{
				{
					"PairedTTest data1",
					func() (bool, error) { r, err := stats.PairedTTest(bad, clean); return r == nil, err },
					want("data1"),
					"PairedTTest",
				},
				{
					"PairedTTest data2",
					func() (bool, error) { r, err := stats.PairedTTest(clean, bad); return r == nil, err },
					want("data2"),
					"PairedTTest",
				},
				{
					"PairedWilcoxon data1",
					func() (bool, error) { r, err := stats.PairedWilcoxon(bad, clean, stats.TwoSided); return r == nil, err },
					want("data1"),
					"PairedWilcoxon",
				},
				{
					"PairedWilcoxon data2",
					func() (bool, error) { r, err := stats.PairedWilcoxon(clean, bad, stats.TwoSided); return r == nil, err },
					want("data2"),
					"PairedWilcoxon",
				},
				{
					"MannWhitneyU data1",
					func() (bool, error) { r, err := stats.MannWhitneyU(bad, clean, stats.TwoSided); return r == nil, err },
					want("data1"),
					"MannWhitneyU",
				},
				{
					"MannWhitneyU data2",
					func() (bool, error) { r, err := stats.MannWhitneyU(clean, bad, stats.TwoSided); return r == nil, err },
					want("data2"),
					"MannWhitneyU",
				},
				{
					"SingleSampleWilcoxon",
					func() (bool, error) {
						r, err := stats.SingleSampleWilcoxon(insyra.NewDataList(1.0, 2.0, tc.bad, 4.0, 6.0), 0, stats.TwoSided)
						return r == nil, err
					},
					want("data"),
					"SingleSampleWilcoxon",
				},
			} {
				t.Run(call.name, func(t *testing.T) {
					resultNil, err := call.run()
					if err == nil {
						t.Fatalf("%s accepted a series holding %v", call.wantName, tc.bad)
					}
					if err.Error() != call.want {
						t.Fatalf("%s error %q, want %q", call.wantName, err.Error(), call.want)
					}
					if !resultNil {
						t.Errorf("%s returned a result alongside an error", call.wantName)
					}
				})
			}
		})
	}
}

// A nil list is a nil interface or a nil pointer inside a non-nil one, and
// they reach the function by different routes. Either way it reads as an empty
// list, so the function's own emptiness check answers it. A typed nil reaching
// AtomicDo directly is a nil-pointer dereference, and inside the goroutine
// OneWayANOVA used to run for the ANOVA families that is unrecoverable and ends
// the host program — so "must not panic" is asserted here, not assumed.
func TestPairedAndOneSampleTestsRefuseNilLists(t *testing.T) {
	var typedNil *insyra.DataList
	clean := insyra.NewDataList(1.5, 2.5, 3.0, 4.5)

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"PairedTTest typed-nil data2", func() error { _, err := stats.PairedTTest(clean, typedNil); return err }},
		{"PairedTTest nil data1", func() error { _, err := stats.PairedTTest(nil, clean); return err }},
		{"PairedWilcoxon typed-nil data1", func() error { _, err := stats.PairedWilcoxon(typedNil, clean, stats.TwoSided); return err }},
		{"MannWhitneyU nil data2", func() error { _, err := stats.MannWhitneyU(clean, nil, stats.TwoSided); return err }},
		{"SingleSampleWilcoxon typed-nil", func() error { _, err := stats.SingleSampleWilcoxon(typedNil, 0, stats.TwoSided); return err }},
		{"SingleSampleWilcoxon nil", func() error { _, err := stats.SingleSampleWilcoxon(nil, 0, stats.TwoSided); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked instead of returning an error: %v", r)
				}
			}()
			if err := tc.run(); err == nil {
				t.Errorf("a nil list was accepted without an error")
			}
		})
	}
}

// The three ANOVA functions read their cells the way the paired and rank tests
// used to: they checked only whether a cell converts, so a NaN or ±Inf flowed
// straight into the arithmetic, and a refused cell was reported with a
// zero-based position. They now read through the same conversion the rest of
// the package uses, which refuses anything that is not a finite number and
// names the series and the one-based position, and they read a nil list as an
// empty one instead of dereferencing it — which in OneWayANOVA happened inside
// a goroutine, where the panic could not be recovered and the host program
// ended. The messages are compared exactly, for the reason the paired and rank
// tests give: one message shape is shared, and a substring check would let a
// half-converted one through.
func TestANOVAFamilyRefusesUnreadableCells(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }

	for _, tc := range []struct {
		name  string
		bad   any
		kind  string
		shown string
	}{
		{"NaN", math.NaN(), "non-finite", "NaN"},
		{"+Inf", math.Inf(1), "non-finite", "+Inf"},
		{"-Inf", math.Inf(-1), "non-finite", "-Inf"},
		{"blank", nil, "non-numeric", "<nil>"},
		{"text", "x", "non-numeric", "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := func(label, position string, at int) string {
				return fmt.Sprintf("%s contains a %s value at %s %d: %s", label, tc.kind, position, at, tc.shown)
			}

			for _, call := range []struct {
				name string
				run  func() (resultNil bool, err error)
				want string
			}{
				{
					"OneWayANOVA",
					func() (bool, error) {
						r, err := stats.OneWayANOVA(dl(1.0, 2.0, 3.0), dl(4.0, 5.0, tc.bad), dl(7.0, 8.0, 9.5))
						return r == nil, err
					},
					want("group 2", "row", 3),
				},
				{
					"TwoWayANOVA",
					func() (bool, error) {
						r, err := stats.TwoWayANOVA(2, 2, dl(1.0, 2.0), dl(3.0, 4.5), dl(5.0, tc.bad), dl(7.0, 8.5))
						return r == nil, err
					},
					want("cell (A=2, B=1)", "row", 2),
				},
				{
					"RepeatedMeasuresANOVA",
					func() (bool, error) {
						r, err := stats.RepeatedMeasuresANOVA(dl(1.0, 2.0, 3.5), dl(2.0, tc.bad, 4.0), dl(3.0, 4.5, 6.0))
						return r == nil, err
					},
					want("subject 2", "condition", 2),
				},
			} {
				t.Run(call.name, func(t *testing.T) {
					resultNil, err := call.run()
					if err == nil {
						t.Fatalf("%s accepted a cell holding %v", call.name, tc.bad)
					}
					if err.Error() != call.want {
						t.Fatalf("%s error %q, want %q", call.name, err.Error(), call.want)
					}
					if !resultNil {
						t.Errorf("%s returned a result alongside an error", call.name)
					}
				})
			}
		})
	}
}

// The refusal itself is also numbered from one now. A group, a cell and a
// subject are all things a reader points at in their data, so "group 0 is
// empty" or "empty cell at A=0, B=0" names a position that does not exist there.
func TestANOVAFamilyNumbersPositionsFromOne(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }

	for _, call := range []struct {
		name string
		run  func() (resultNil bool, err error)
		want string
	}{
		{
			"OneWayANOVA empty group",
			func() (bool, error) {
				r, err := stats.OneWayANOVA(dl(1.0, 2.0), dl())
				return r == nil, err
			},
			"group 2 is empty",
		},
		{
			"TwoWayANOVA empty cell A=1 B=1",
			func() (bool, error) {
				r, err := stats.TwoWayANOVA(2, 2, dl(), dl(3.0, 4.5), dl(5.0, 6.0), dl(7.0, 8.5))
				return r == nil, err
			},
			"empty cell at A=1, B=1",
		},
		{
			"TwoWayANOVA empty cell A=1 B=2",
			func() (bool, error) {
				r, err := stats.TwoWayANOVA(2, 2, dl(1.0, 2.0), dl(), dl(5.0, 6.0), dl(7.0, 8.5))
				return r == nil, err
			},
			"empty cell at A=1, B=2",
		},
		{
			"RepeatedMeasuresANOVA inconsistent subject",
			func() (bool, error) {
				r, err := stats.RepeatedMeasuresANOVA(dl(1.0, 2.0, 3.0), dl(1.0, 2.0))
				return r == nil, err
			},
			"inconsistent condition count at subject 2",
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			resultNil, err := call.run()
			if err == nil {
				t.Fatalf("%s accepted an empty cell without an error", call.name)
			}
			if err.Error() != call.want {
				t.Fatalf("%s error %q, want %q", call.name, err.Error(), call.want)
			}
			if !resultNil {
				t.Errorf("%s returned a result alongside an error", call.name)
			}
		})
	}
}

// A nil list is a nil interface or a nil pointer inside a non-nil one, and they
// reach each function by a different route. Either way it reads as an empty
// list, so the function's own emptiness check answers it. "Must not panic" is
// asserted here rather than assumed: a typed nil that reached AtomicDo directly
// is a nil-pointer dereference, and in OneWayANOVA that happened inside a
// goroutine, where no caller can recover it.
func TestANOVAFamilyRefusesNilLists(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }
	var typedNil *insyra.DataList

	for _, call := range []struct {
		name string
		run  func() error
		want string
	}{
		{
			"OneWayANOVA typed-nil group",
			func() error { _, err := stats.OneWayANOVA(dl(1.0, 2.0), typedNil); return err },
			"group 2 is empty",
		},
		{
			"OneWayANOVA nil group",
			func() error { _, err := stats.OneWayANOVA(nil, dl(1.0, 2.0)); return err },
			"group 1 is empty",
		},
		{
			"TwoWayANOVA typed-nil cell",
			func() error {
				_, err := stats.TwoWayANOVA(2, 2, dl(1.0, 2.0), typedNil, dl(1.0, 2.0), dl(3.0, 4.0))
				return err
			},
			"empty cell at A=1, B=2",
		},
		{
			"RepeatedMeasuresANOVA typed-nil subject",
			func() error { _, err := stats.RepeatedMeasuresANOVA(dl(1.0, 2.0), typedNil); return err },
			"inconsistent condition count at subject 2",
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked instead of returning an error: %v", r)
				}
			}()
			err := call.run()
			if err == nil {
				t.Fatalf("a nil list was accepted without an error")
			}
			if err.Error() != call.want {
				t.Errorf("error %q, want %q", err.Error(), call.want)
			}
		})
	}
}

// The two rank tests and the two variance tests were the last readers left on
// the old contract. KruskalWallis and FriedmanTest checked only whether a cell
// converts, so a NaN or ±Inf reached the ranking and came back as a plausible
// statistic with no error, and a refused cell was reported at a zero-based
// position with no series name at all. LeveneTest and BartlettTest already
// refused a non-finite cell, but named "group 0" for the caller's first group.
// All four now read through the same conversion and name the one-based position,
// and the messages are compared exactly for the reason the tests above give:
// one message shape is shared, so a substring check would let a half-converted
// one through.
func TestRankAndVarianceTestsRefuseUnreadableCells(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }

	for _, tc := range []struct {
		name  string
		bad   any
		kind  string
		shown string
	}{
		{"NaN", math.NaN(), "non-finite", "NaN"},
		{"+Inf", math.Inf(1), "non-finite", "+Inf"},
		{"-Inf", math.Inf(-1), "non-finite", "-Inf"},
		{"blank", nil, "non-numeric", "<nil>"},
		{"text", "x", "non-numeric", "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := tc.bad
			want := func(label, position string, at int) string {
				return fmt.Sprintf("%s contains a %s value at %s %d: %s", label, tc.kind, position, at, tc.shown)
			}

			for _, call := range []struct {
				name string
				run  func() (resultNil bool, err error)
				want string
			}{
				{
					"KruskalWallis group 1",
					func() (bool, error) {
						r, err := stats.KruskalWallis(dl(1.0, 2.0, bad, 4.0), dl(1.0, 2.0, 3.0, 4.0))
						return r == nil, err
					},
					want("group 1", "row", 3),
				},
				{
					"KruskalWallis group 2",
					func() (bool, error) {
						r, err := stats.KruskalWallis(dl(1.0, 2.0, 3.0, 4.0), dl(1.0, bad, 3.0, 4.0))
						return r == nil, err
					},
					want("group 2", "row", 2),
				},
				{
					"FriedmanTest subject 2",
					func() (bool, error) {
						r, err := stats.FriedmanTest(
							dl(1.0, 2.0, 3.0), dl(2.0, bad, 1.0), dl(3.0, 1.0, 2.0), dl(1.0, 3.0, 2.0))
						return r == nil, err
					},
					want("subject 2", "condition", 2),
				},
				{
					"LeveneTest group 2",
					func() (bool, error) {
						r, err := stats.LeveneTest([]insyra.IDataList{dl(1.0, 2.0, 3.0), dl(4.0, bad, 6.5)})
						return r == nil, err
					},
					want("group 2", "row", 2),
				},
				{
					"BartlettTest group 2",
					func() (bool, error) {
						r, err := stats.BartlettTest([]insyra.IDataList{dl(1.0, 2.0, 3.0), dl(4.0, bad, 6.5)})
						return r == nil, err
					},
					want("group 2", "row", 2),
				},
			} {
				t.Run(call.name, func(t *testing.T) {
					resultNil, err := call.run()
					if err == nil {
						t.Fatalf("%s accepted a cell holding %v", call.name, tc.bad)
					}
					if err.Error() != call.want {
						t.Fatalf("%s error %q, want %q", call.name, err.Error(), call.want)
					}
					if !resultNil {
						t.Errorf("%s returned a result alongside an error", call.name)
					}
				})
			}
		})
	}
}

// A group, a subject and a condition are all positions a reader points at in
// their data, so "group 0 is empty" or "subject 1 has 2 observations" names one
// that does not exist there. BartlettTest's zero-variance refusal is the same
// message as its too-few-observations one, and it is numbered the same way.
func TestRankAndVarianceTestsNumberPositionsFromOne(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }

	for _, call := range []struct {
		name string
		run  func() (resultNil bool, err error)
		want string
	}{
		{
			"KruskalWallis empty group",
			func() (bool, error) {
				r, err := stats.KruskalWallis(dl(1.0, 2.0), dl())
				return r == nil, err
			},
			"group 2 is empty",
		},
		{
			"FriedmanTest inconsistent subject",
			func() (bool, error) {
				r, err := stats.FriedmanTest(dl(1.0, 2.0, 3.0), dl(2.0, 1.0))
				return r == nil, err
			},
			"subject 2 has 2 observations, expected 3",
		},
		{
			"BartlettTest zero variance",
			func() (bool, error) {
				r, err := stats.BartlettTest([]insyra.IDataList{dl(1.0, 2.0, 3.0), dl(5.0, 5.0, 5.0)})
				return r == nil, err
			},
			"group 2 must have at least two observations and positive variance",
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			resultNil, err := call.run()
			if err == nil {
				t.Fatalf("%s accepted an empty or degenerate list without an error", call.name)
			}
			if err.Error() != call.want {
				t.Fatalf("%s error %q, want %q", call.name, err.Error(), call.want)
			}
			if !resultNil {
				t.Errorf("%s returned a result alongside an error", call.name)
			}
		})
	}
}

// A nil list reaches a rank test the same two ways it reached OneWayANOVA: a nil
// interface, or a nil pointer inside a non-nil interface. Both read as an empty
// list, so the test's own emptiness or length check answers them — but only
// after asDataList, because KruskalWallis and FriedmanTest read their arguments
// inside a goroutine, where a typed nil dereferenced by AtomicDo is a
// nil-pointer panic that no caller can recover and that ends the host program.
// "Must not panic" is asserted here rather than assumed.
func TestRankTestsRefuseNilLists(t *testing.T) {
	dl := func(values ...any) *insyra.DataList { return insyra.NewDataList(values...) }
	var typedNil *insyra.DataList

	for _, call := range []struct {
		name string
		run  func() error
		want string
	}{
		{
			"KruskalWallis typed-nil group",
			func() error { _, err := stats.KruskalWallis(dl(1.0, 2.0), typedNil); return err },
			"group 2 is empty",
		},
		{
			"KruskalWallis nil group",
			func() error { _, err := stats.KruskalWallis(nil, dl(1.0, 2.0)); return err },
			"group 1 is empty",
		},
		{
			"FriedmanTest typed-nil subject",
			func() error { _, err := stats.FriedmanTest(dl(1.0, 2.0), typedNil); return err },
			"subject 2 has 0 observations, expected 2",
		},
		{
			"FriedmanTest nil subject",
			func() error { _, err := stats.FriedmanTest(typedNil, dl(1.0, 2.0)); return err },
			"each subject must have at least two conditions",
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked instead of returning an error: %v", r)
				}
			}()
			err := call.run()
			if err == nil {
				t.Fatalf("a nil list was accepted without an error")
			}
			if err.Error() != call.want {
				t.Errorf("error %q, want %q", err.Error(), call.want)
			}
		})
	}
}
