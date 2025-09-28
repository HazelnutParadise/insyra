package parallel_test

import (
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/parallel"
)

func TestParallelGroup_WithResults(t *testing.T) {
	// Functions that return values
	fn1 := func() int { time.Sleep(10 * time.Millisecond); return 1 }
	fn2 := func() string { time.Sleep(10 * time.Millisecond); return "hello" }

	pg := parallel.GroupUp(fn1, fn2)
	results := pg.Run().AwaitResult()

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0][0] != 1 {
		t.Errorf("Expected first result to be 1, got %v", results[0][0])
	}

	if results[1][0] != "hello" {
		t.Errorf("Expected second result to be 'hello', got %v", results[1][0])
	}
}

func TestParallelGroup_NoResults(t *testing.T) {
	// Functions that do not return values
	counter := 0
	fn1 := func() { time.Sleep(10 * time.Millisecond); counter++ }
	fn2 := func() { time.Sleep(10 * time.Millisecond); counter++ }

	pg := parallel.GroupUp(fn1, fn2)
	pg.Run().AwaitNoResult()

	if counter != 2 {
		t.Errorf("Expected counter to be 2, got %d", counter)
	}
}

func TestParallelGroup_Mixed(t *testing.T) {
	// Mixed functions: one with result, one without
	counter := 0
	fn1 := func() int { time.Sleep(10 * time.Millisecond); return 42 }
	fn2 := func() { time.Sleep(10 * time.Millisecond); counter++ }

	pg := parallel.GroupUp(fn1, fn2)
	results := pg.Run().AwaitResult()

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0] == nil || results[0][0] != 42 {
		t.Errorf("Expected first result to be [42], got %v", results[0])
	}

	if results[1] != nil {
		t.Errorf("Expected second result to be nil, got %v", results[1])
	}

	if counter != 1 {
		t.Errorf("Expected counter to be 1, got %d", counter)
	}
}

func TestParallelGroup_AwaitNoResult(t *testing.T) {
	// Test that AwaitNoResult waits properly
	start := time.Now()
	fn := func() { time.Sleep(50 * time.Millisecond) }

	pg := parallel.GroupUp(fn)
	pg.Run().AwaitNoResult()

	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("Expected at least 40ms elapsed, got %v", elapsed)
	}
}

func TestParallelGroup_NoResults_WithAwaitResult(t *testing.T) {
	// Test using AwaitResult with functions that do not return values
	counter := 0
	fn1 := func() { time.Sleep(10 * time.Millisecond); counter++ }
	fn2 := func() { time.Sleep(10 * time.Millisecond); counter++ }

	pg := parallel.GroupUp(fn1, fn2)
	results := pg.Run().AwaitResult()

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// For functions with no return values, results should be nil or empty slices
	if results[0] != nil {
		t.Errorf("Expected first result to be nil, got %v", results[0])
	}

	if results[1] != nil {
		t.Errorf("Expected second result to be nil, got %v", results[1])
	}

	if counter != 2 {
		t.Errorf("Expected counter to be 2, got %d", counter)
	}
}
