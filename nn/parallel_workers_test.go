package nn

import (
	"runtime"
	"testing"
)

func TestParallelWorkerCountFollowsGOMAXPROCS(t *testing.T) {
	previous := runtime.GOMAXPROCS(3)
	defer runtime.GOMAXPROCS(previous)

	if got := parallelWorkerCountForMACs(10, 10, 10); got != 1 {
		t.Errorf("a small product used %d workers, want 1", got)
	}
	if got := parallelWorkerCountForMACs(1000, 1000, 1000); got != 3 {
		t.Errorf("a large product used %d workers with GOMAXPROCS 3, want 3", got)
	}
	if got := parallelWorkerCountForMACs(1<<30, 1<<30, 1<<30); got != 3 {
		t.Errorf("an overflowing product used %d workers, want 3", got)
	}
	runtime.GOMAXPROCS(1)
	if got := parallelWorkerCountForMACs(1000, 1000, 1000); got != 1 {
		t.Errorf("GOMAXPROCS 1 used %d workers, want 1", got)
	}
}
