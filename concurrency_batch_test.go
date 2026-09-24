package insyra

import (
	"testing"
	"time"
)

// K-8 (#209): a nil instance used to be dereferenced inside AtomicDoAll and
// panic. There is nothing to lock on a nil value; the callback still runs.
func TestAtomicDoAllSkipsNilInstance(t *testing.T) {
	quietLogs(t)

	var nilList *DataList
	var nilTable *DataTable
	var nilIface IDataList
	ran := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("AtomicDoAll panicked on a nil instance: %v", r)
			}
		}()
		AtomicDoAll(func() { ran = true }, NewDataList(1), nilList, nilTable, nilIface, nil)
	}()
	if !ran {
		t.Fatal("the callback did not run")
	}
}

// Skipping the nil instance must not skip the others: a real list passed next
// to a nil one is still locked while the callback runs.
func TestAtomicDoAllStillLocksAlongsideNil(t *testing.T) {
	quietLogs(t)

	var nilList *DataList
	dl := NewDataList(1)
	appended := make(chan struct{})
	AtomicDoAll(func() {
		go func() {
			dl.Append(2)
			close(appended)
		}()
		select {
		case <-appended:
			t.Error("another goroutine wrote the list while AtomicDoAll held it")
		case <-time.After(50 * time.Millisecond):
		}
	}, nilList, dl)
	<-appended
	if n := dl.Len(); n != 2 {
		t.Errorf("Len = %d after the callback returned, want 2", n)
	}
}
