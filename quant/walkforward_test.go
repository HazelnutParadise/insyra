package quant

import (
	"math"
	"reflect"
	"testing"
)

func TestWalkForwardRolling(t *testing.T) {
	var trains, tests [][2]int
	opt := func(a, b int) int {
		trains = append(trains, [2]int{a, b})
		return 0
	}
	eval := func(_ int, a, b int) []float64 {
		tests = append(tests, [2]int{a, b})
		out := make([]float64, b-a)
		for i := range out {
			out[i] = 0.01
		}
		return out
	}

	res, err := WalkForward(10, WalkForwardConfig{TrainSize: 4, TestSize: 2}, opt, eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTrains := [][2]int{{0, 4}, {2, 6}, {4, 8}}
	wantTests := [][2]int{{4, 6}, {6, 8}, {8, 10}}
	if !reflect.DeepEqual(trains, wantTrains) {
		t.Errorf("train windows = %v, want %v", trains, wantTrains)
	}
	if !reflect.DeepEqual(tests, wantTests) {
		t.Errorf("test windows = %v, want %v", tests, wantTests)
	}
	if len(res.Folds) != 3 {
		t.Errorf("got %d folds, want 3", len(res.Folds))
	}
	if len(res.OOSReturns) != 6 {
		t.Errorf("got %d OOS returns, want 6", len(res.OOSReturns))
	}

	// Equity starts at 1.0 and compounds 6 periods of +1%.
	if len(res.Equity) != 7 {
		t.Fatalf("got %d equity points, want 7", len(res.Equity))
	}
	if res.Equity[0] != 1.0 {
		t.Errorf("Equity[0] = %v, want 1.0", res.Equity[0])
	}
	if want := math.Pow(1.01, 6); math.Abs(res.Equity[6]-want) > 1e-12 {
		t.Errorf("final equity = %v, want %v", res.Equity[6], want)
	}

	// Constant returns → zero volatility → Sharpe undefined (error);
	// monotonically rising equity → zero drawdown.
	if _, err := res.Sharpe(0, 252); err == nil {
		t.Error("expected Sharpe error for zero-volatility OOS returns")
	}
	if dd, err := res.MaxDrawdown(); err != nil || dd != 0 {
		t.Errorf("MaxDrawdown = (%v, %v), want (0, nil)", dd, err)
	}
}

func TestWalkForwardAnchored(t *testing.T) {
	var trains [][2]int
	opt := func(a, b int) int {
		trains = append(trains, [2]int{a, b})
		return 0
	}
	eval := func(_ int, a, b int) []float64 { return make([]float64, b-a) }

	_, err := WalkForward(10, WalkForwardConfig{TrainSize: 4, TestSize: 2, Anchored: true}, opt, eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][2]int{{0, 4}, {0, 6}, {0, 8}} // expanding window
	if !reflect.DeepEqual(trains, want) {
		t.Errorf("anchored train windows = %v, want %v", trains, want)
	}
}

func TestWalkForwardShortTail(t *testing.T) {
	var tests [][2]int
	opt := func(_, _ int) int { return 0 }
	eval := func(_ int, a, b int) []float64 {
		tests = append(tests, [2]int{a, b})
		return make([]float64, b-a)
	}

	// n=11, TrainSize=4, TestSize=3: last OOS window [10,11) is short, not dropped.
	res, err := WalkForward(11, WalkForwardConfig{TrainSize: 4, TestSize: 3}, opt, eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][2]int{{4, 7}, {7, 10}, {10, 11}}
	if !reflect.DeepEqual(tests, want) {
		t.Errorf("test windows = %v, want %v", tests, want)
	}
	if len(res.OOSReturns) != 7 {
		t.Errorf("got %d OOS returns, want 7", len(res.OOSReturns))
	}
}

func TestWalkForwardAggregation(t *testing.T) {
	// A varying OOS series so Sharpe/drawdown are well-defined. The
	// "strategy" ignores parameters and emits a fixed pattern per fold.
	pattern := []float64{0.02, -0.01, 0.03}
	opt := func(_, _ int) int { return 0 }
	eval := func(_ int, a, b int) []float64 {
		out := make([]float64, 0, b-a)
		for i := 0; i < b-a; i++ {
			out = append(out, pattern[i%len(pattern)])
		}
		return out
	}

	res, err := WalkForward(10, WalkForwardConfig{TrainSize: 4, TestSize: 2}, opt, eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sharpe and drawdown delegate to the package functions on the
	// stitched OOS series; they must agree with calling those directly.
	wantSharpe, _ := sharpeRatioF64(res.OOSReturns, 0, 252)
	gotSharpe, err := res.Sharpe(0, 252)
	if err != nil || math.Abs(gotSharpe-wantSharpe) > 1e-12 {
		t.Errorf("Sharpe = (%v, %v), want %v", gotSharpe, err, wantSharpe)
	}
	wantDD, _ := maxDrawdownF64(res.Equity)
	gotDD, _ := res.MaxDrawdown()
	if math.Abs(gotDD-wantDD) > 1e-12 {
		t.Errorf("MaxDrawdown = %v, want %v", gotDD, wantDD)
	}
}

func TestWalkForwardErrors(t *testing.T) {
	opt := func(_, _ int) int { return 0 }
	eval := func(_ int, a, b int) []float64 { return make([]float64, b-a) }

	if _, err := WalkForward(0, WalkForwardConfig{TrainSize: 4, TestSize: 2}, opt, eval); err == nil {
		t.Error("expected error for n <= 0")
	}
	if _, err := WalkForward(10, WalkForwardConfig{TrainSize: 0, TestSize: 2}, opt, eval); err == nil {
		t.Error("expected error for non-positive TrainSize")
	}
	if _, err := WalkForward(10, WalkForwardConfig{TrainSize: 4, TestSize: 0}, opt, eval); err == nil {
		t.Error("expected error for non-positive TestSize")
	}
	if _, err := WalkForward(4, WalkForwardConfig{TrainSize: 4, TestSize: 2}, opt, eval); err == nil {
		t.Error("expected error when TrainSize leaves no room for testing")
	}
	if _, err := WalkForward(10, WalkForwardConfig{TrainSize: 4, TestSize: 2}, nil, eval); err == nil {
		t.Error("expected error for nil optimize")
	}
}
