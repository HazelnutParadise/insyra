package stats

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// The two-sample stats methods were rewritten to lock both input lists together
// via insyra.AtomicDoAll. This hammers them concurrently while other goroutines
// mutate the shared inputs — under -race it must report no data race, and it
// must not deadlock (bounded by the timeout).
func TestStress_TwoSampleStatsNoRace(t *testing.T) {
	// Bounded (see the root stress test note): each Append fires a background
	// updateTimestamp goroutine, so mutation is bounded to keep the test reliably
	// completing under -race. Completion proves no deadlock; -race proves no race.
	iters := 120
	appendsPerMutator := 200
	if testing.Short() {
		iters = 20
		appendsPerMutator = 40
	}
	done := make(chan struct{})
	body := func() {
		a := insyra.NewDataList(1.0, 2, 3, 4, 5, 6, 7, 8).SetName("a")
		b := insyra.NewDataList(2.0, 4, 6, 8, 10, 12, 14, 16).SetName("b")

		var wg sync.WaitGroup

		// Bounded mutators: append a fixed number of times, yielding between.
		for g := 0; g < 3; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < appendsPerMutator; i++ {
					if g%2 == 0 {
						a.Append(float64(i))
					} else {
						b.Append(float64(i))
					}
					runtime.Gosched()
				}
			}(g)
		}

		// Workers running every rewritten two-list stats method.
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					_, _ = TwoSampleTTest(a, b, true, 0.95)
					_, _ = TwoSampleTTest(a, b, false)
					_, _ = PairedTTest(a, b)
					_, _ = TwoSampleZTest(a, b, 1, 1, TwoSided, 0.95)
					_, _ = FTestForVarianceEquality(a, b)
					_, _ = MannWhitneyU(a, b, TwoSided)
					_, _ = PairedWilcoxon(a, b, TwoSided)
					_, _ = Correlation(a, b, PearsonCorrelation)
					_, _ = Covariance(a, b)
					_, _ = ExponentialRegression(b, a)
					_, _ = LogarithmicRegression(b, a)
					_, _ = PolynomialRegression(b, a, 2)
				}
			}()
		}

		wg.Wait()
	}

	go func() { body(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		t.Fatal("two-sample stats stress: deadlocked / timed out")
	}
}
