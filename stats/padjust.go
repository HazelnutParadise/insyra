package stats

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// PAdjustMethod names a multiple-comparison correction for AdjustPValues.
type PAdjustMethod int

const (
	// PAdjustHolm is Holm's step-down correction; it controls the family-wise
	// error rate and is never less powerful than Bonferroni.
	PAdjustHolm PAdjustMethod = iota + 1
	// PAdjustBonferroni multiplies every p-value by the number of tests.
	PAdjustBonferroni
	// PAdjustHochberg is Hochberg's step-up correction (family-wise error
	// rate, for independent or positively dependent tests).
	PAdjustHochberg
	// PAdjustBH is Benjamini-Hochberg; it controls the false discovery rate for
	// independent or positively dependent tests.
	PAdjustBH
	// PAdjustBY is Benjamini-Yekutieli; it controls the false discovery rate
	// under any dependence.
	PAdjustBY
)

// String returns the constant name of m, or PAdjustMethod(n) for a value that
// is not one of the constants.
func (m PAdjustMethod) String() string {
	switch m {
	case PAdjustHolm:
		return "PAdjustHolm"
	case PAdjustBonferroni:
		return "PAdjustBonferroni"
	case PAdjustHochberg:
		return "PAdjustHochberg"
	case PAdjustBH:
		return "PAdjustBH"
	case PAdjustBY:
		return "PAdjustBY"
	}
	return fmt.Sprintf("PAdjustMethod(%d)", int(m))
}

// AdjustPValues corrects a batch of p-values for testing many hypotheses at
// once and returns the adjusted values in the order given, each capped at 1.
// It matches R's p.adjust for the same method.
//
// A NaN is a test that was not run: it stays NaN in the result and is not
// counted in the number of tests. A value that is not a number, infinite, or
// outside [0, 1] is an error, as is an empty or nil list.
func AdjustPValues(pvals insyra.IDataList, method PAdjustMethod) (*insyra.DataList, error) {
	if insyra.IsNilList(pvals) {
		return nil, errors.New("p-value list is nil")
	}
	var raw []any
	pvals.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })
	if len(raw) == 0 {
		return nil, errors.New("p-value list is empty")
	}
	switch method {
	case PAdjustHolm, PAdjustBonferroni, PAdjustHochberg, PAdjustBH, PAdjustBY:
	default:
		return nil, fmt.Errorf("unsupported p-value adjustment method %v", method)
	}

	values := make([]float64, len(raw))
	known := make([]int, 0, len(raw)) // positions of the tests that were run
	for i, v := range raw {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, fmt.Errorf("p-value at row index %d is not a number: %s", i, utils.SprintBounded(v))
		}
		values[i] = f
		if math.IsNaN(f) {
			continue
		}
		if math.IsInf(f, 0) || f < 0 || f > 1 {
			return nil, fmt.Errorf("p-value at row index %d is %v, outside [0, 1]", i, f)
		}
		known = append(known, i)
	}

	out := make([]any, len(values))
	for i := range out {
		out[i] = math.NaN()
	}
	n := len(known)
	if n > 0 {
		p := make([]float64, n)
		for k, i := range known {
			p[k] = values[i]
		}
		adjusted := adjustPValues(p, method)
		for k, i := range known {
			out[i] = adjusted[k]
		}
	}
	return insyra.NewDataList(out...), nil
}

// adjustPValues is R's p.adjust on p-values that are all numbers in [0, 1].
func adjustPValues(p []float64, method PAdjustMethod) []float64 {
	n := len(p)
	fn := float64(n)
	adj := make([]float64, n)
	if method == PAdjustBonferroni {
		for i, v := range p {
			adj[i] = math.Min(1, fn*v)
		}
		return adj
	}

	// order lists positions by p, ascending, ties kept in input order as R's
	// order() does.
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return p[order[a]] < p[order[b]] })

	switch method {
	case PAdjustHolm:
		running := 0.0
		for rank, pos := range order {
			running = math.Max(running, (fn-float64(rank))*p[pos])
			adj[pos] = math.Min(1, running)
		}
	case PAdjustHochberg, PAdjustBH, PAdjustBY:
		scale := 1.0
		if method == PAdjustBY {
			scale = 0
			for k := 1; k <= n; k++ {
				scale += 1 / float64(k)
			}
		}
		// Walk from the largest p-value down, keeping the running minimum.
		running := math.Inf(1)
		for k := n - 1; k >= 0; k-- {
			pos := order[k]
			rank := float64(k + 1) // 1-based ascending rank
			var term float64
			switch method {
			case PAdjustHochberg:
				term = (fn - rank + 1) * p[pos]
			default: // BH and BY
				term = scale * fn / rank * p[pos]
			}
			running = math.Min(running, term)
			adj[pos] = math.Min(1, running)
		}
	}
	return adj
}
