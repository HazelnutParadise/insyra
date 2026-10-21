package insyra

// These tests pin the conclusion of #222: six pairs of DataList methods give the
// same numbers on purely numeric input over the positions both produce, but they
// are not two names for one function. They differ in result length, in what a
// nil or NaN cell does, in which windows they emit, and in whether a call that
// cannot be done fails or returns a usable answer. Difference, MovingAverage,
// MovingStdev and WeightedMovingAverage are deprecated and keep this behaviour
// until they are removed. The differences are tabulated in Docs/DataList.md under
// "Methods that look alike but differ".
//
// Every scenario below builds its own lists, because Err() is sticky and a list
// that has recorded a failure cannot be reused.

import (
	"testing"
)

// assertLegacyFailed is the "failure" half of the contract shared with
// Docs/DataList.md: the call handed back a usable but empty list, and both it
// and the list it was called on recorded why.
func assertLegacyFailed(t *testing.T, result, source *DataList) {
	t.Helper()
	if result == nil {
		t.Fatalf("failed transform returned nil, want a usable empty list")
		return
	}
	if got := result.Len(); got != 0 {
		t.Errorf("failed transform returned a list of length %d, want 0", got)
	}
	if result.Err() == nil {
		t.Errorf("failed transform returned a list with a nil Err()")
	}
	if source.Err() == nil {
		t.Errorf("source list Err() is nil, want the recorded failure")
	}
}

// assertLegacySucceeded is the "not a failure" half: a usable answer, and no
// error on the result or on the list it came from.
func assertLegacySucceeded(t *testing.T, result, source *DataList) {
	t.Helper()
	if result == nil {
		t.Fatalf("transform returned nil, want a usable list")
		return
	}
	if result.Err() != nil {
		t.Errorf("result Err() = %v, want nil", result.Err())
	}
	if source.Err() != nil {
		t.Errorf("source list Err() = %v, want nil", source.Err())
	}
}
