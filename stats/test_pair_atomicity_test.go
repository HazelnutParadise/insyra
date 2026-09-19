package stats

import (
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// pairAtomicityIterations is how many times each test reads the pair while a
// writer resizes both lists. 20,000 is where the torn read was first measured
// often enough to be unmistakable rather than flaky.
const pairAtomicityIterations = 20000

// resizingPair returns two lists of ten values each and a stop function for a
// writer that keeps appending ten more to both and popping them off again,
// always inside one AtomicDoAll. A reader that locks both lists together
// therefore sees 10 and 10 or 20 and 20, never one of each.
func resizingPair(t *testing.T) (*insyra.DataList, *insyra.DataList, func()) {
	t.Helper()
	base1 := []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	base2 := []any{2.0, 4.0, 5.0, 7.0, 8.0, 11.0, 12.0, 14.0, 15.0, 17.0}
	dl1 := insyra.NewDataList(base1...)
	dl2 := insyra.NewDataList(base2...)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		grown := false
		for {
			select {
			case <-stop:
				return
			default:
			}
			insyra.AtomicDoAll(func() {
				if grown {
					for range base1 {
						dl1.Pop()
						dl2.Pop()
					}
				} else {
					dl1.Append(base1...)
					dl2.Append(base2...)
				}
			}, dl1, dl2)
			grown = !grown
		}
	}()

	return dl1, dl2, func() {
		close(stop)
		wg.Wait()
	}
}

// A two-sample test reads its two samples at one moment or not at all.
// v0.3.2 took both under one insyra.AtomicDoAll; reading them one after the
// other lets a writer that resizes both together land in between, and the
// result then mixes one sample's old length with the other's new one. The
// degrees of freedom say so directly: with ten or twenty observations in each
// list, n1+n2-2 is 18 or 38 and nothing else.
func TestTwoSampleTTestReadsBothSamplesAtOneMoment(t *testing.T) {
	dl1, dl2, stop := resizingPair(t)
	defer stop()

	torn := make(map[float64]int)
	for range pairAtomicityIterations {
		res, err := TwoSampleTTest(dl1, dl2, true)
		if err != nil {
			t.Fatalf("TwoSampleTTest: %v", err)
		}
		if df := *res.DF; df != 18 && df != 38 {
			torn[df]++
		}
	}
	if len(torn) > 0 {
		t.Errorf("TwoSampleTTest saw the two samples at different moments: DF %v in %d of %d runs, want only 18 or 38",
			torn, totalOf(torn), pairAtomicityIterations)
	}
}

// The same for the two-sample z-test, which reports the two sample sizes
// rather than a pooled DF.
func TestTwoSampleZTestReadsBothSamplesAtOneMoment(t *testing.T) {
	dl1, dl2, stop := resizingPair(t)
	defer stop()

	torn := 0
	for range pairAtomicityIterations {
		res, err := TwoSampleZTest(dl1, dl2, 2, 4, TwoSided, 0.95)
		if err != nil {
			t.Fatalf("TwoSampleZTest: %v", err)
		}
		if res.N != *res.N2 {
			torn++
		}
	}
	if torn > 0 {
		t.Errorf("TwoSampleZTest saw the two samples at different moments in %d of %d runs (N != N2)",
			torn, pairAtomicityIterations)
	}
}

// And for the F-test, whose two degrees of freedom are n-1 of each sample in
// whichever order the larger variance put them.
func TestFTestForVarianceEqualityReadsBothSamplesAtOneMoment(t *testing.T) {
	dl1, dl2, stop := resizingPair(t)
	defer stop()

	torn := 0
	for range pairAtomicityIterations {
		res, err := FTestForVarianceEquality(dl1, dl2)
		if err != nil {
			t.Fatalf("FTestForVarianceEquality: %v", err)
		}
		if *res.DF != res.DF2 {
			torn++
		}
	}
	if torn > 0 {
		t.Errorf("FTestForVarianceEquality saw the two samples at different moments in %d of %d runs (DF != DF2 although both samples are always the same length)",
			torn, pairAtomicityIterations)
	}
}

// PairedTTest has always taken both lists under one AtomicDoAll and must keep
// doing so: a torn read there is not a wrong DF but a refusal, because the
// lengths it compares would disagree.
func TestPairedTTestReadsBothSamplesAtOneMoment(t *testing.T) {
	dl1, dl2, stop := resizingPair(t)
	defer stop()

	for range pairAtomicityIterations {
		res, err := PairedTTest(dl1, dl2)
		if err != nil {
			t.Fatalf("PairedTTest saw the two samples at different moments: %v", err)
		}
		if res.N != 10 && res.N != 20 {
			t.Fatalf("PairedTTest N = %d, want 10 or 20", res.N)
		}
	}
}

func totalOf(counts map[float64]int) int {
	total := 0
	for _, n := range counts {
		total += n
	}
	return total
}
