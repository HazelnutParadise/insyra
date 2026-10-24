package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// Expected values are R survival::survfit(Surv(time, event) ~ 1) on the same
// 16 subjects, 5 of them censored, printed with summary().
func TestKaplanMeierMatchesSurvfit(t *testing.T) {
	time := insyra.NewDataList(5.0, 6.0, 6.0, 2.5, 4.0, 4.0, 8.0, 9.0, 9.0, 9.0, 12.0, 15.0, 15.0, 3.0, 3.0, 7.0)
	event := insyra.NewDataList(1, 0, 1, 1, 1, 0, 1, 1, 0, 1, 1, 0, 1, 1, 1, 0)
	got, err := KaplanMeier(time, event)
	if err != nil {
		t.Fatal(err)
	}
	wantTimes := []float64{2.5, 3, 4, 5, 6, 8, 9, 12, 15}
	wantRisk := []int{16, 15, 13, 11, 10, 7, 6, 3, 2}
	wantEvents := []int{1, 2, 1, 1, 1, 1, 2, 1, 1}
	wantS := []float64{0.9375, 0.8125, 0.75, 0.681818181818182, 0.613636363636364, 0.525974025974026, 0.350649350649351, 0.233766233766234, 0.116883116883117}
	wantSE := []float64{0.0605153647844909, 0.097578093724975, 0.108253175473055, 0.117945169932657, 0.12430545208067, 0.133937483412221, 0.134978690986682, 0.131168664176144, 0.105508938586239}
	if len(got.Times) != len(wantTimes) {
		t.Fatalf("times: %v", got.Times)
	}
	for i := range wantTimes {
		if got.Times[i] != wantTimes[i] || got.AtRisk[i] != wantRisk[i] || got.Events[i] != wantEvents[i] ||
			math.Abs(got.Survival[i]-wantS[i]) > 1e-12 || math.Abs(got.StdErr[i]-wantSE[i]) > 1e-12 {
			t.Errorf("row index %d: got time %v risk %d events %d S %v se %v", i, got.Times[i], got.AtRisk[i], got.Events[i], got.Survival[i], got.StdErr[i])
		}
	}
	if got.Median != 9 || got.N != 16 || got.Total != 11 {
		t.Errorf("median %v, N %d, total %d; R says median 9, 16 subjects, 11 events", got.Median, got.N, got.Total)
	}
	if got.Censored[1] != 0 || got.Censored[2] != 1 {
		t.Errorf("censored counts: %v", got.Censored)
	}
}

func TestKaplanMeierMedianAtExactlyHalf(t *testing.T) {
	// R: survfit(Surv(1:4, rep(1, 4))) has S = 0.75, 0.5, 0.25, 0 and median 2.5.
	got, err := KaplanMeier(insyra.NewDataList(1.0, 2.0, 3.0, 4.0), insyra.NewDataList(true, true, true, true))
	if err != nil {
		t.Fatal(err)
	}
	if got.Median != 2.5 {
		t.Errorf("median %v, want 2.5", got.Median)
	}
	if !math.IsNaN(got.StdErr[3]) || got.Survival[3] != 0 {
		t.Errorf("last row: S %v se %v", got.Survival[3], got.StdErr[3])
	}
}

func TestKaplanMeierNeverReachingHalf(t *testing.T) {
	got, err := KaplanMeier(insyra.NewDataList(1.0, 2.0, 3.0), insyra.NewDataList(true, false, false))
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(got.Median) {
		t.Errorf("median %v, want NaN", got.Median)
	}
}

func TestKaplanMeierRefusesBadInput(t *testing.T) {
	good := insyra.NewDataList(1.0, 2.0)
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"nil durations": func() error { _, e := KaplanMeier(nilList, good); return e },
		"nil observed":  func() error { _, e := KaplanMeier(good, nilList); return e },
		"length":        func() error { _, e := KaplanMeier(good, insyra.NewDataList(1)); return e },
		"negative":      func() error { _, e := KaplanMeier(insyra.NewDataList(-1.0, 2.0), insyra.NewDataList(1, 1)); return e },
		"flag 2":        func() error { _, e := KaplanMeier(good, insyra.NewDataList(1, 2)); return e },
		"flag text":     func() error { _, e := KaplanMeier(good, insyra.NewDataList(1, "x")); return e },
		"NaN": func() error {
			_, e := KaplanMeier(insyra.NewDataList(1.0, math.NaN()), insyra.NewDataList(1, 1))
			return e
		},
		"empty": func() error { _, e := KaplanMeier(insyra.NewDataList(), insyra.NewDataList()); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
