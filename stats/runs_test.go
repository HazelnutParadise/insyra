package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The series and every expected value come from R, counting runs with rle()
// after dropping the values equal to the threshold:
// x <- sin((1:40)*1.7)+0.1*((1:40)%%5-2); x[7] <- 0
var runsRSeries = []float64{
	0.891664810452469, -0.255541102026831, -0.825814682327732, 0.694113351138608, 0.59848711262349,
	-0.799874687593542, 0, 0.959161814856496, 0.596740573130614, -1.16139749187956,
	-0.248999025814199, 0.999792900142669, -0.00863659542407622, -0.771798445743863, 0.159058354022168,
	0.779273061650724, -0.585637339974298, -0.628360767831596, 0.973327889566221, 0.329082686120024,
	-1.00966667183353, -0.294671601500258, 1.08560029879063, 0.240693257349865, -1.19608650311959,
	0.115987266188217, 0.940428962947006, -0.358325454491766, -0.622323540133491, 0.470229175843375,
	0.549612861686201, -0.837627257147028, -0.333765540975684, 1.1494038606867, -0.0108853796491085,
	-1.0981366159327, 0.0680941946844981, 1.08058949177553, -0.120781309043145, -1.09792768068929,
}

func runsList(x []float64) insyra.IDataList {
	vals := make([]any, len(x))
	for i, v := range x {
		vals[i] = v
	}
	return insyra.NewDataList(vals...)
}

func TestRunsTestMatchesR(t *testing.T) {
	cases := []struct {
		name                          string
		opts                          RunsTestOptions
		runs, above, below, dropped   int
		expected, variance, z, pValue float64
	}{
		{"zero threshold", RunsTestOptions{}, 22, 19, 20, 1, 20.4871794871795, 9.48060486522025, 0.491325586731987, 0.623196185714443},
		{"continuity correction", RunsTestOptions{Correction: true}, 22, 19, 20, 1, 20.4871794871795, 9.48060486522025, 0.328938316540907, 0.74220231197184},
		{"threshold 0.05", RunsTestOptions{Threshold: 0.05}, 22, 19, 21, 0, 20.95, 9.69365384615385, 0.337245020100869, 0.735932201910165},
	}
	for _, c := range cases {
		got, err := RunsTest(runsList(runsRSeries), c.opts)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Runs != c.runs || got.Above != c.above || got.Below != c.below || got.Dropped != c.dropped {
			t.Errorf("%s: runs %d above %d below %d dropped %d, R says %d %d %d %d", c.name, got.Runs, got.Above, got.Below, got.Dropped, c.runs, c.above, c.below, c.dropped)
		}
		for label, pair := range map[string][2]float64{"Expected": {got.Expected, c.expected}, "Variance": {got.Variance, c.variance}, "Z": {got.Z, c.z}, "PValue": {got.PValue, c.pValue}} {
			if math.Abs(pair[0]-pair[1]) > 1e-12 {
				t.Errorf("%s: %s %v, R says %v", c.name, label, pair[0], pair[1])
			}
		}
	}
}

func TestRunsTestCountsRunsByHand(t *testing.T) {
	// ++--+- : four runs, three above, three below; E = 2*3*3/6+1 = 4.
	got, err := RunsTest(insyra.NewDataList(1.0, 2.0, -1.0, -2.0, 3.0, -3.0))
	if err != nil {
		t.Fatal(err)
	}
	if got.Runs != 4 || got.Above != 3 || got.Below != 3 || got.Expected != 4 {
		t.Errorf("runs %d above %d below %d expected %v", got.Runs, got.Above, got.Below, got.Expected)
	}
}

func TestRunsTestRefusesBadInput(t *testing.T) {
	for name, call := range map[string]func() error{
		"nil list":   func() error { _, err := RunsTest(nil); return err },
		"empty list": func() error { _, err := RunsTest(insyra.NewDataList()); return err },
		"text cell":  func() error { _, err := RunsTest(insyra.NewDataList(1.0, "x", -1.0, 2.0)); return err },
		"all above":  func() error { _, err := RunsTest(insyra.NewDataList(1.0, 2.0, 3.0)); return err },
		"all on cut": func() error { _, err := RunsTest(insyra.NewDataList(0.0, 0.0, 0.0)); return err },
		"two options": func() error {
			_, err := RunsTest(runsList(runsRSeries), RunsTestOptions{}, RunsTestOptions{})
			return err
		},
		"nan thresold": func() error {
			_, err := RunsTest(runsList(runsRSeries), RunsTestOptions{Threshold: math.NaN()})
			return err
		},
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestRunsTestCorrectionNeverCrossesZero(t *testing.T) {
	// "+---+": two above, three below, three runs. Expected is 2*2*3/5+1 = 3.4,
	// so the runs are 0.4 short, inside the 0.5 correction: the corrected
	// difference must stop at 0, not turn into +0.1.
	x := []float64{1, -1, -1, -1, 1}
	plain, err := RunsTest(runsList(x))
	if err != nil {
		t.Fatal(err)
	}
	if plain.Runs != 3 || plain.Above != 2 || plain.Below != 3 || plain.Z >= 0 {
		t.Fatalf("setup: runs %d above %d below %d z %v", plain.Runs, plain.Above, plain.Below, plain.Z)
	}
	corrected, err := RunsTest(runsList(x), RunsTestOptions{Correction: true})
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Z != 0 || corrected.PValue != 1 {
		t.Errorf("a difference of %v is inside 0.5, so the corrected Z should be 0 and P 1, got Z %v P %v", float64(plain.Runs)-plain.Expected, corrected.Z, corrected.PValue)
	}
}
