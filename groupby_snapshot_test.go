package insyra

import (
	"sync"
	"testing"
)

// T-16 (#229): GroupBy copied the parent's column pointers, not their data, so
// Aggregate summed whatever the parent held when Aggregate ran. The groups were
// decided from the rows as they were at GroupBy time, so reading later values
// against those row indices mixes two different tables.
func TestGroupByAggregatesTheSnapshot(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(
		NewDataList("a", "a", "b").SetName("k"),
		NewDataList(1, 2, 3).SetName("v"),
	)
	g := dt.GroupBy("k")
	dt.UpdateElement(0, "B", 100) // after GroupBy; must not reach the groups

	out := g.Aggregate(AggregateConfig{SourceCol: "v", Op: OpSum, As: "s"})
	if err := dt.PopErr(); err != nil {
		t.Fatal(err)
	}
	got := out.GetColByName("s").Data()
	if len(got) != 2 || ToFloat64(got[0]) != 3 || ToFloat64(got[1]) != 3 {
		t.Errorf("sums = %v, want [3 3] (the values when GroupBy ran)", got)
	}
}

// The same defect, seen by the race detector: Aggregate read the parent's
// column data without a lock while another goroutine wrote it. Only meaningful
// under -race.
func TestGroupByAggregateDoesNotRaceParentWrites(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	const n = 200
	keys, vals := make([]any, n), make([]any, n)
	for i := range n {
		keys[i] = i % 5
		vals[i] = i
	}
	dt := NewDataTable(NewDataList(keys...).SetName("k"), NewDataList(vals...).SetName("v"))
	g := dt.GroupBy("k")

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				dt.UpdateElement(i%n, "B", i)
			}
		}
	}()
	for range 50 {
		g.Aggregate(AggregateConfig{SourceCol: "v", Op: OpSum, As: "s"})
	}
	close(stop)
	wg.Wait()
}
