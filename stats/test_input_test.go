package stats_test

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestParametricTestsRefuseBlankCells(t *testing.T) {
	blank := insyra.NewDataList(1.0, 2.0, nil, 3.0)
	clean := insyra.NewDataList(2.0, 4.0, 9.0, 1.0)
	cases := map[string]func() error{
		"SingleSampleTTest": func() error { _, err := stats.SingleSampleTTest(blank, 0); return err },
		"TwoSampleTTest":    func() error { _, err := stats.TwoSampleTTest(blank, clean, true); return err },
		"TwoSampleTTest2":   func() error { _, err := stats.TwoSampleTTest(clean, blank, false); return err },
		"SingleSampleZTest": func() error { _, err := stats.SingleSampleZTest(blank, 0, 1, stats.TwoSided, 0.95); return err },
		"TwoSampleZTest":    func() error { _, err := stats.TwoSampleZTest(blank, clean, 1, 1, stats.TwoSided, 0.95); return err },
		"FTestForVarianceEquality": func() error {
			_, err := stats.FTestForVarianceEquality(blank, clean)
			return err
		},
		"BartlettTest":    func() error { _, err := stats.BartlettTest([]insyra.IDataList{blank, clean}); return err },
		"LeveneTest":      func() error { _, err := stats.LeveneTest([]insyra.IDataList{blank, clean}); return err },
		"CalculateMoment": func() error { _, err := stats.CalculateMoment(blank, 3, true); return err },
	}
	for name, f := range cases {
		err := f()
		if err == nil || !strings.Contains(err.Error(), "row 3") {
			t.Fatalf("%s: expected error naming row 3, got %v", name, err)
		}
	}
}

func TestParametricTestsCleanInputUnchanged(t *testing.T) {
	clean := insyra.NewDataList(1.0, 2.0, 3.0)
	r, err := stats.SingleSampleTTest(clean, 0)
	if err != nil || r.N != 3 || r.Statistic < 3.46 || r.Statistic > 3.47 {
		t.Fatalf("got n=%d t=%v err=%v", r.N, r.Statistic, err)
	}
}
