package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// wrappedList is an IDataList that is not a *insyra.DataList.
type wrappedList struct{ *insyra.DataList }

func TestNilInterfaceInputsDoNotPanic(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0, 3.0, 4.0)
	var typedNil *insyra.DataList
	calls := map[string]func() error{
		"Correlation":              func() error { _, err := stats.Correlation(nil, good, stats.PearsonCorrelation); return err },
		"Covariance":               func() error { _, err := stats.Covariance(good, nil); return err },
		"PairedTTest":              func() error { _, err := stats.PairedTTest(nil, good); return err },
		"MannWhitneyU":             func() error { _, err := stats.MannWhitneyU(nil, good, stats.TwoSided); return err },
		"PairedWilcoxon":           func() error { _, err := stats.PairedWilcoxon(good, nil, stats.TwoSided); return err },
		"ExponentialReg":           func() error { _, err := stats.ExponentialRegression(nil, good); return err },
		"LogarithmicReg":           func() error { _, err := stats.LogarithmicRegression(good, nil); return err },
		"PolynomialReg":            func() error { _, err := stats.PolynomialRegression(nil, good, 2); return err },
		"TwoSampleTTest":           func() error { _, err := stats.TwoSampleTTest(nil, good, true); return err },
		"TwoSampleTTest typed nil": func() error { _, err := stats.TwoSampleTTest(good, typedNil, false); return err },
		"TwoSampleZTest":           func() error { _, err := stats.TwoSampleZTest(good, nil, 1, 1, stats.TwoSided, 0.95); return err },
		"FTestForVarianceEquality": func() error { _, err := stats.FTestForVarianceEquality(nil, good); return err },
	}
	for name, f := range calls {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked: %v", name, r)
				}
			}()
			if err := f(); err == nil {
				t.Fatalf("%s: expected error for nil input", name)
			}
		}()
	}
}

func TestNonConcreteDataListMatchesConcrete(t *testing.T) {
	a := insyra.NewDataList(10.0, 12.0, 9.0, 11.0, 13.0)
	b := insyra.NewDataList(20.0, 19.0, 21.0, 22.0, 18.5)
	wa := wrappedList{insyra.NewDataList(a.Data()...)}
	wb := wrappedList{insyra.NewDataList(b.Data()...)}

	wantT, err := stats.TwoSampleTTest(a, b, false)
	if err != nil {
		t.Fatal(err)
	}
	gotT, err := stats.TwoSampleTTest(wa, wb, false)
	if err != nil {
		t.Fatal(err)
	}
	if gotT.Statistic != wantT.Statistic || gotT.PValue != wantT.PValue {
		t.Fatalf("TwoSampleTTest: got (%v, %v), want (%v, %v)", gotT.Statistic, gotT.PValue, wantT.Statistic, wantT.PValue)
	}

	wantZ, err := stats.TwoSampleZTest(a, b, 1.5, 1.5, stats.TwoSided, 0.95)
	if err != nil {
		t.Fatal(err)
	}
	gotZ, err := stats.TwoSampleZTest(wa, wb, 1.5, 1.5, stats.TwoSided, 0.95)
	if err != nil {
		t.Fatal(err)
	}
	if gotZ.Statistic != wantZ.Statistic || gotZ.PValue != wantZ.PValue {
		t.Fatalf("TwoSampleZTest: got (%v, %v), want (%v, %v)", gotZ.Statistic, gotZ.PValue, wantZ.Statistic, wantZ.PValue)
	}

	wantF, err := stats.FTestForVarianceEquality(a, b)
	if err != nil {
		t.Fatal(err)
	}
	gotF, err := stats.FTestForVarianceEquality(wa, wb)
	if err != nil {
		t.Fatal(err)
	}
	if gotF.Statistic != wantF.Statistic || gotF.PValue != wantF.PValue {
		t.Fatalf("FTestForVarianceEquality: got (%v, %v), want (%v, %v)", gotF.Statistic, gotF.PValue, wantF.Statistic, wantF.PValue)
	}
}

// A wrapped list must keep a slice cell as one unreadable cell, the way a
// concrete list already does, instead of flattening it into extra observations.
func TestNonConcreteDataListKeepsSliceCells(t *testing.T) {
	withSlice := func(values ...any) wrappedList {
		w := wrappedList{insyra.NewDataList(values...)}
		w.Append([]float64{10, 11})
		return w
	}
	dl := insyra.NewDataList

	cases := []struct {
		name string
		run  func() error
		want string
	}{
		{
			"SingleSampleTTest",
			func() error { _, err := stats.SingleSampleTTest(withSlice(1.0, 2.0, 3.0), 0); return err },
			"data contains a non-numeric value at row 4: [10 11]",
		},
		{
			"PairedTTest",
			func() error {
				_, err := stats.PairedTTest(withSlice(1.0, 2.0, 3.0), dl(1.0, 2.0, 3.0, 4.0))
				return err
			},
			"data1 contains a non-numeric value at row 4: [10 11]",
		},
		{
			"SingleSampleWilcoxon",
			func() error {
				_, err := stats.SingleSampleWilcoxon(withSlice(1.0, 2.0, 3.0), 0, stats.TwoSided)
				return err
			},
			"data contains a non-numeric value at row 4: [10 11]",
		},
		{
			"OneWayANOVA",
			func() error { _, err := stats.OneWayANOVA(withSlice(1.0, 2.0, 3.0), dl(4.0, 5.0, 6.0)); return err },
			"group 1 contains a non-numeric value at row 4: [10 11]",
		},
		{
			"KruskalWallis",
			func() error { _, err := stats.KruskalWallis(withSlice(1.0, 2.0, 3.0), dl(4.0, 5.0, 6.0)); return err },
			"group 1 contains a non-numeric value at row 4: [10 11]",
		},
		{
			"TwoWayANOVA",
			func() error {
				_, err := stats.TwoWayANOVA(2, 2, withSlice(1.0, 2.0, 3.0), dl(3.0, 4.5), dl(5.0, 6.0), dl(7.0, 8.5))
				return err
			},
			"cell (A=1, B=1) contains a non-numeric value at row 4: [10 11]",
		},
		{
			"RepeatedMeasuresANOVA",
			func() error {
				_, err := stats.RepeatedMeasuresANOVA(dl(1.0, 2.0, 3.0), withSlice(2.0, 3.0))
				return err
			},
			"subject 2 contains a non-numeric value at condition 3: [10 11]",
		},
		{
			"FriedmanTest",
			func() error {
				_, err := stats.FriedmanTest(dl(1.0, 2.0, 3.0), withSlice(2.0, 3.0), dl(3.0, 1.0, 2.0))
				return err
			},
			"subject 2 contains a non-numeric value at condition 3: [10 11]",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.run()
			if err == nil {
				t.Fatalf("%s: expected an error naming the slice cell, got nil", c.name)
			}
			if err.Error() != c.want {
				t.Fatalf("%s:\n got %q\nwant %q", c.name, err.Error(), c.want)
			}
		})
	}
}
