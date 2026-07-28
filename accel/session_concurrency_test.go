package accel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// atomicExecutor is a concurrency-safe fake backend, so any race the detector
// reports below belongs to the Session, not to the test double.
type atomicExecutor struct {
	calls atomic.Int64
}

func (e *atomicExecutor) Name() string { return "atomic-fake" }

func (e *atomicExecutor) Execute(_ context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	e.calls.Add(1)
	var sum float64
	for _, value := range req.Values {
		sum += float64(value)
	}
	return ExecuteResponse{
		Value:         sum,
		Transfer:      time.Microsecond,
		Dispatch:      time.Microsecond,
		Readback:      time.Microsecond,
		BytesUploaded: uint64(len(req.Values) * 4),
	}, nil
}

// Regression for the unguarded-session follow-up: before the session mutex,
// concurrent ExecuteProjectedDataset calls raced on s.cache.entries and
// s.reports, and this test failed under -race.
func TestSessionConcurrentExecuteAndReadsAreRaceFree(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	executor := &atomicExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	const writers, readers, rounds = 4, 4, 25
	var wg sync.WaitGroup
	errs := make(chan error, writers*rounds)

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				// Distinct values per goroutine and round, so every dataset has
				// its own fingerprint and the cache map takes real writes.
				values := []float64{float64(w), float64(i), float64(w * i), 1}
				dataset := float64Dataset(fmt.Sprintf("col-%d-%d", w, i), values, nil)
				result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Precision: PrecisionFloat32})
				if err != nil {
					errs <- fmt.Errorf("writer %d round %d: %w", w, i, err)
					return
				}
				want := float64(w) + float64(i) + float64(w*i) + 1
				if got := result.Reductions[fmt.Sprintf("col-%d-%d", w, i)]; got != want {
					errs <- fmt.Errorf("writer %d round %d: got %v want %v", w, i, got, want)
					return
				}
			}
		}(w)
	}
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				_ = session.Report()
				_ = session.Reports()
				_ = session.LastReport()
				_ = session.Devices()
				_ = session.Config()
				_ = session.CacheSnapshot()
				_ = session.PlanShardable()
				_ = session.Closed()
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	if got := executor.calls.Load(); got != writers*rounds {
		t.Fatalf("expected %d executions, got %d", writers*rounds, got)
	}
}

func TestSessionConcurrentProjectionsAreRaceFree(t *testing.T) {
	session := singleDeviceSession(t, Config{})

	const goroutines, rounds = 6, 20
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				dl := insyra.NewDataList(float64(g), float64(i)).SetName(fmt.Sprintf("proj-%d-%d", g, i))
				if _, err := session.ProjectDataList(dl); err != nil {
					t.Errorf("project %d-%d: %v", g, i, err)
					return
				}
				_ = session.CacheSnapshot()
			}
		}(g)
	}
	wg.Wait()
}
