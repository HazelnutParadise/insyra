package insyra

import (
	"strings"
	"testing"
)

// Asking what went wrong must never crash, even when the value is nil, as a
// lookup that found nothing returns. Only Err() is guarded: every other method
// on a nil value still fails loudly, so the nil cannot travel on unnoticed
// (owner's ruling of 2026-09-26).
func TestErrOnANilReceiverReportsInsteadOfCrashing(t *testing.T) {
	missing := NewDataTable(NewDataList(1).SetName("a")).GetCol(Name("missing"))
	if missing != nil {
		t.Fatalf("GetCol for a missing column returned %v; the test needs a nil list", missing)
	}
	err := missing.Err()
	if err == nil || !strings.Contains(err.Error(), "nil DataList") {
		t.Fatalf("Err() on a nil DataList = %v", err)
	}

	var table *DataTable
	if err := table.Err(); err == nil || !strings.Contains(err.Error(), "nil DataTable") {
		t.Fatalf("Err() on a nil DataTable = %v", err)
	}

	// Each call gets its own value, so one caller changing it cannot change
	// what the next is told.
	first, second := missing.Err(), missing.Err()
	if first == second {
		t.Fatal("Err() on a nil receiver returned a shared value")
	}
}
